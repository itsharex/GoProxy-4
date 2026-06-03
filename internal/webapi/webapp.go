package webapi

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"time"

	"gitee.com/jiuhuidalan1/goproxy/internal/config"
	"gitee.com/jiuhuidalan1/goproxy/internal/logger"
	"gitee.com/jiuhuidalan1/goproxy/internal/platform"
	"gitee.com/jiuhuidalan1/goproxy/internal/proxy"
	"gitee.com/jiuhuidalan1/goproxy/internal/stats"
	"gitee.com/jiuhuidalan1/goproxy/internal/store"
	"go.uber.org/zap"
)

const webSource = "webapi"

type WebApp struct {
	mu            sync.Mutex
	configPath    string
	logPath       string
	configManager *config.Manager
	store         *store.Store
	logger        *logger.Manager
	cfg           config.Config
	runtimeCfg    config.Config
	collector     *stats.Collector
	server        *proxy.Server
	hub           *wsHub
	auth          *tokenIssuer
	cancelStats   context.CancelFunc
}

func NewWebApp(configPath, logPath string) (*WebApp, error) {
	manager := config.NewManager(configPath)
	cfg, err := manager.Load()
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("加载配置失败: %w", err)
		}
		cfg = config.Default()
		if err := manager.Save(cfg); err != nil {
			return nil, fmt.Errorf("写入默认配置失败: %w", err)
		}
	}

	configDir := filepath.Dir(configPath)
	dbPath := filepath.Join(configDir, "goproxy.db")
	s, err := store.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	if err := s.ImportFromYAML(configPath); err != nil {
		s.Close()
		return nil, fmt.Errorf("数据迁移失败: %w", err)
	}

	s.FillWebConfig(&cfg)
	s.FillAuthUsers(&cfg)
	s.FillActiveRoute(&cfg)

	logManager, err := logger.NewManager(cfg.Log, logPath)
	if err != nil {
		s.Close()
		return nil, fmt.Errorf("创建日志管理器失败: %w", err)
	}

	collector := stats.NewCollector()
	server := proxy.NewServer(cfg, collector)
	server.SetLogger(logManager)
	if err := applyWebRoutePolicyFromStore(server, s, cfg); err != nil {
		s.Close()
		return nil, fmt.Errorf("加载路由策略失败: %w", err)
	}

	hub := newWSHub()
	go hub.run()

	jwtSecret, _ := s.GetJWTSecret()
	expireHours, _ := s.GetJWTExpireHours()
	if expireHours <= 0 {
		expireHours = 24
	}

	auth := newTokenIssuer(
		func(username, password string) (string, error) {
			u, err := s.GetWebUser(username)
			if err != nil {
				return "", err
			}
			if u == nil {
				return "", errors.New("用户名或密码错误")
			}
			return u.Password, nil
		},
		jwtSecret,
		expireHours,
	)

	app := &WebApp{
		configPath:    configPath,
		logPath:       logPath,
		configManager: manager,
		store:         s,
		logger:        logManager,
		cfg:           cfg,
		runtimeCfg:    cfg,
		collector:     collector,
		server:        server,
		hub:           hub,
		auth:          auth,
	}

	app.bridgeEvents(logManager)

	if cfg.UI.AutoStartProxy {
		if err := app.StartServer(); err != nil {
			logManager.Error(webSource, "自动启动代理服务失败", zap.Error(err))
		}
	}

	return app, nil
}

func (a *WebApp) Hub() *wsHub {
	return a.hub
}

func (a *WebApp) Auth() *tokenIssuer {
	return a.auth
}

func (a *WebApp) GetJWTExpireHours() int {
	hours, err := a.store.GetJWTExpireHours()
	if err != nil || hours <= 0 {
		return 24
	}
	return hours
}

func (a *WebApp) MustChangePwd(username string) bool {
	u, err := a.store.GetWebUser(username)
	if err != nil || u == nil {
		return false
	}
	return u.MustChangePwd
}

func (a *WebApp) ChangePassword(username, oldPassword, newPassword string) (string, time.Time, error) {
	u, err := a.store.VerifyWebUser(username, oldPassword)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("旧密码验证失败: %s", err.Error())
	}
	_ = u

	if err := a.store.UpdateWebUserPassword(username, newPassword, false); err != nil {
		return "", time.Time{}, fmt.Errorf("更新密码失败: %w", err)
	}

	newSecret := generateRandomSecret()
	if err := a.store.UpdateJWTSecret(newSecret); err != nil {
		return "", time.Time{}, fmt.Errorf("更新 JWT 密钥失败: %w", err)
	}
	a.auth.SetSecret([]byte(newSecret))

	token, err := a.auth.Authenticate(username, newPassword)
	if err != nil {
		return "", time.Time{}, err
	}

	expireHours := a.GetJWTExpireHours()
	expiresAt := time.Now().Add(time.Duration(expireHours) * time.Hour)
	return token, expiresAt, nil
}

func (a *WebApp) Close() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.cancelStats != nil {
		a.cancelStats()
	}
	if a.server != nil {
		_ = a.server.Stop()
	}
	a.hub.Close()
	if a.logger != nil {
		a.logger.Info(webSource, "Web 服务已关闭")
		_ = a.logger.Close()
	}
	if a.store != nil {
		a.store.Close()
	}
}

func (a *WebApp) GetConfig() config.Config {
	a.mu.Lock()
	defer a.mu.Unlock()
	cfg := a.cfg
	a.store.FillWebConfig(&cfg)
	a.store.FillAuthUsers(&cfg)
	a.store.FillActiveRoute(&cfg)
	return cfg
}

func (a *WebApp) SaveConfig(cfg config.Config) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	oldCfg := a.cfg
	running := a.server != nil && a.server.Status().Running

	var newLogger *logger.Manager
	if oldCfg.Log != cfg.Log {
		var err error
		newLogger, err = logger.NewManager(cfg.Log, a.logPath)
		if err != nil {
			if a.logger != nil {
				a.logger.Warn(webSource, "日志配置无效", zap.Error(err))
			}
			return err
		}
	}

	if err := a.configManager.Save(cfg); err != nil {
		if a.logger != nil {
			a.logger.Warn(webSource, "配置保存失败", zap.Error(err))
		}
		return err
	}

	a.cfg = cfg

	if newLogger != nil {
		oldLogger := a.logger
		a.logger = newLogger
		a.bridgeEvents(newLogger)
		if a.server != nil {
			a.server.SetLogger(newLogger)
		}
		if oldLogger != nil {
			_ = oldLogger.Close()
		}
	}

	if !running {
		a.collector = stats.NewCollector()
		a.server = proxy.NewServer(cfg, a.collector)
		a.server.SetLogger(a.logger)
		if err := a.applyRoutePolicyLocked(cfg); err != nil {
			return err
		}
		a.runtimeCfg = cfg
	} else {
		authUsers, _ := a.store.ListAuthUsers()
		cfg.Auth.Users = authUsers
		if !reflect.DeepEqual(oldCfg.Auth.Enabled, cfg.Auth.Enabled) && a.server != nil {
			a.server.SetAuthConfig(cfg.Auth)
			a.runtimeCfg.Auth = cfg.Auth
		}
	}

	if running && routeConfigChanged(oldCfg, cfg) {
		if err := a.applyRoutePolicyLocked(cfg); err != nil {
			return err
		}
		a.runtimeCfg.Route = cfg.Route
	}

	if a.logger != nil {
		a.logger.Info(webSource, "配置已保存")
		if running && listenerConfigChanged(oldCfg, cfg) {
			a.logger.Warn(webSource, "监听配置已保存，重启服务后生效")
		}
	}

	a.emitStatus()
	return nil
}

func (a *WebApp) StartServer() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.server == nil || !sameRuntimeConfig(a.cfg, a.runtimeCfg) {
		a.collector = stats.NewCollector()
		a.server = proxy.NewServer(a.cfg, a.collector)
		a.server.SetLogger(a.logger)
		if err := a.applyRoutePolicyLocked(a.cfg); err != nil {
			return err
		}
		a.runtimeCfg = a.cfg
	}

	ctx := context.Background()
	if err := a.server.Start(ctx); err != nil {
		if a.logger != nil {
			a.logger.Error(webSource, "代理服务启动失败", zap.Error(err))
		}
		return err
	}

	if a.logger != nil {
		status := a.server.Status()
		a.logger.Info(webSource, "代理服务已启动",
			zap.String("socks5Addr", status.SOCKS5Addr),
			zap.String("httpAddr", status.HTTPAddr),
		)
	}

	a.emitStatus()
	return nil
}

func (a *WebApp) StopServer() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.server == nil {
		return nil
	}
	if err := a.server.Stop(); err != nil {
		if a.logger != nil {
			a.logger.Error(webSource, "代理服务停止失败", zap.Error(err))
		}
		return err
	}

	if a.logger != nil {
		a.logger.Info(webSource, "代理服务已停止")
	}
	a.emitStatus()
	return nil
}

func (a *WebApp) GetServerStatus() proxy.Status {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.server == nil {
		return proxy.Status{}
	}
	return a.server.Status()
}

func (a *WebApp) GetStats() stats.Stats {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.server == nil {
		return stats.Stats{}
	}
	return a.server.Stats()
}

func (a *WebApp) GetActiveConnections() []proxy.ConnectionSnapshot {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.server == nil {
		return nil
	}
	return a.server.ActiveConnections()
}

func (a *WebApp) GetRecentLogs(n int) []logger.Entry {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.logger == nil {
		return []logger.Entry{}
	}
	entries := a.logger.Recent(n)
	if entries == nil {
		return []logger.Entry{}
	}
	return entries
}

func (a *WebApp) ClearLogs() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	oldLogger := a.logger
	if oldLogger != nil {
		if err := oldLogger.Close(); err != nil {
			return fmt.Errorf("关闭日志文件失败: %w", err)
		}
	}
	if a.logPath != "" {
		_ = os.Remove(a.logPath)
	}

	newLogger, err := logger.NewManager(a.cfg.Log, a.logPath)
	if err != nil {
		return fmt.Errorf("重新创建日志管理器失败: %w", err)
	}
	a.logger = newLogger
	a.bridgeEvents(newLogger)
	if a.server != nil {
		a.server.SetLogger(newLogger)
	}
	return nil
}

func (a *WebApp) SetAuthEnabled(enabled bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	cfg := a.cfg
	cfg.Auth.Enabled = enabled
	return a.saveAuthConfigLocked(cfg)
}

func (a *WebApp) AddUser(username, password string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	hash, err := proxy.HashPassword(password)
	if err != nil {
		return err
	}
	if err := a.store.AddAuthUser(username, hash); err != nil {
		return err
	}

	cfg := a.cfg
	cfg.Auth.Users = nil
	users, _ := a.store.ListAuthUsers()
	cfg.Auth.Users = users
	return a.saveAuthConfigLocked(cfg)
}

func (a *WebApp) RemoveUser(username string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.store.RemoveAuthUser(username); err != nil {
		return err
	}

	cfg := a.cfg
	users, _ := a.store.ListAuthUsers()
	cfg.Auth.Users = users
	return a.saveAuthConfigLocked(cfg)
}

func (a *WebApp) ResetUserPassword(username, password string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	hash, err := proxy.HashPassword(password)
	if err != nil {
		return err
	}
	if err := a.store.UpdateAuthUserPassword(username, hash); err != nil {
		return err
	}

	cfg := a.cfg
	users, _ := a.store.ListAuthUsers()
	cfg.Auth.Users = users
	return a.saveAuthConfigLocked(cfg)
}

func (a *WebApp) ListRouteFiles() ([]config.RouteFileInfo, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.store.ListRouteRuleSets()
}

func (a *WebApp) LoadRouteFile(name string) (config.RouteRuleSet, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.store.LoadRouteRuleSet(name)
}

func (a *WebApp) SaveRouteFile(name string, set config.RouteRuleSet) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.store.SaveRouteRuleSet(name, set); err != nil {
		return err
	}

	active, _ := a.store.GetActiveRouteFileName()
	if name == active {
		if err := a.applyRoutePolicyLocked(a.cfg); err != nil {
			return err
		}
	}
	return nil
}

func (a *WebApp) CreateRouteFile(name string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.store.CreateRouteRuleSet(name)
}

func (a *WebApp) DeleteRouteFile(name string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	active, _ := a.store.GetActiveRouteFileName()
	if name == active {
		return errors.New("当前正在使用的规则文件不能删除")
	}
	return a.store.DeleteRouteRuleSet(name)
}

func (a *WebApp) SetActiveRouteFile(name string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.store.SetActiveRouteRuleSet(name); err != nil {
		return err
	}
	cfg := a.cfg
	cfg.Route.ActiveFile = name
	a.cfg = cfg
	if err := a.applyRoutePolicyLocked(cfg); err != nil {
		return err
	}
	a.runtimeCfg.Route = cfg.Route
	if a.logger != nil {
		a.logger.Info(webSource, "当前规则文件已切换", zap.String("file", name))
	}
	return nil
}

func (a *WebApp) GetLocalIPAddresses() ([]string, error) {
	return platform.LocalIPAddresses()
}

func (a *WebApp) GetNetworkInterfaces() ([]platform.NetworkInterface, error) {
	return platform.NetworkInterfaces()
}

func (a *WebApp) saveAuthConfigLocked(cfg config.Config) error {
	if err := a.configManager.Save(cfg); err != nil {
		return err
	}
	a.cfg = cfg
	if a.server != nil {
		a.server.SetAuthConfig(cfg.Auth)
		a.runtimeCfg.Auth = cfg.Auth
	}
	a.emitStatus()
	return nil
}

func (a *WebApp) applyRoutePolicyLocked(cfg config.Config) error {
	return applyWebRoutePolicyFromStore(a.server, a.store, cfg)
}

func (a *WebApp) bridgeEvents(logManager *logger.Manager) {
	if logManager == nil || a.hub == nil {
		return
	}
	logManager.Subscribe(func(entry logger.Entry) {
		a.hub.Emit("proxy:log", entry)
	})
}

func (a *WebApp) emitStatus() {
	if a.server == nil {
		return
	}
	status := a.server.Status()
	a.hub.Emit("proxy:status", status)
}

func (a *WebApp) StartStatsLoop(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	a.cancelStats = cancel

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.mu.Lock()
			server := a.server
			a.mu.Unlock()
			if server == nil {
				continue
			}
			snapshot := server.TickStats()
			a.hub.Emit("proxy:stats", snapshot)
		}
	}
}

func applyWebRoutePolicyFromStore(server *proxy.Server, s *store.Store, cfg config.Config) error {
	if server == nil {
		return nil
	}
	if !cfg.Route.Enabled {
		server.SetRoutePolicy(false, config.RouteRuleSet{})
		return nil
	}
	activeFile, err := s.GetActiveRouteFileName()
	if err != nil || activeFile == "" {
		server.SetRoutePolicy(false, config.RouteRuleSet{})
		return nil
	}
	set, err := s.LoadRouteRuleSet(activeFile)
	if err != nil {
		return err
	}
	server.SetRoutePolicy(true, set)
	return nil
}

func generateRandomSecret() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func listenerConfigChanged(a, b config.Config) bool {
	return a.Server.SOCKS5 != b.Server.SOCKS5 || a.Server.HTTP != b.Server.HTTP
}

func sameRuntimeConfig(a, b config.Config) bool {
	return a.Server == b.Server && a.Relay == b.Relay
}

func routeConfigChanged(a, b config.Config) bool {
	return a.Route != b.Route
}
