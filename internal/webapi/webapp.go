package webapi

import (
	"context"
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
	"go.uber.org/zap"
)

const webSource = "webapi"

type WebApp struct {
	mu            sync.Mutex
	configPath    string
	logPath       string
	configManager *config.Manager
	routeManager  *config.RouteFileManager
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

	routeManager := config.NewRouteFileManager(filepath.Dir(configPath))
	activeFile, err := routeManager.EnsureActive(cfg.Route.ActiveFile)
	if err != nil {
		return nil, fmt.Errorf("初始化规则文件失败: %w", err)
	}
	if cfg.Route.ActiveFile != activeFile {
		cfg.Route.ActiveFile = activeFile
		if err := manager.Save(cfg); err != nil {
			return nil, fmt.Errorf("保存当前规则文件回退失败: %w", err)
		}
	}

	logManager, err := logger.NewManager(cfg.Log, logPath)
	if err != nil {
		return nil, fmt.Errorf("创建日志管理器失败: %w", err)
	}

	collector := stats.NewCollector()
	server := proxy.NewServer(cfg, collector)
	server.SetLogger(logManager)
	if err := applyWebRoutePolicy(server, routeManager, cfg); err != nil {
		return nil, fmt.Errorf("加载路由策略失败: %w", err)
	}

	hub := newWSHub()
	go hub.run()

	webPassword := cfg.Web.Password
	if webPassword == "" {
		webPassword, err = proxy.HashPassword("admin")
		if err != nil {
			return nil, fmt.Errorf("生成默认面板密码失败: %w", err)
		}
		cfg.Web.Password = webPassword
		if err := manager.Save(cfg); err != nil {
			return nil, fmt.Errorf("保存面板默认密码失败: %w", err)
		}
		logManager.Info(webSource, "已生成默认 Web 面板密码")
		logManager.Info(webSource, "默认用户名: admin, 默认密码: admin, 请尽快修改")
	}

	auth := newTokenIssuer(cfg.Web.Username, webPassword, cfg.Web.JWTSecret, cfg.Web.JWTExpireHours)

	app := &WebApp{
		configPath:    configPath,
		logPath:       logPath,
		configManager: manager,
		routeManager:  routeManager,
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
}

func (a *WebApp) GetConfig() config.Config {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.cfg
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
	} else if !reflect.DeepEqual(oldCfg.Auth, cfg.Auth) && a.server != nil {
		a.server.SetAuthConfig(cfg.Auth)
		a.runtimeCfg.Auth = cfg.Auth
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

	cfg := a.cfg
	for _, user := range cfg.Auth.Users {
		if user.Username == username {
			return fmt.Errorf("用户 %q 已存在", username)
		}
	}
	cfg.Auth.Users = append(cfg.Auth.Users, config.AuthUser{Username: username, Password: hash})
	return a.saveAuthConfigLocked(cfg)
}

func (a *WebApp) RemoveUser(username string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	cfg := a.cfg
	next := cfg.Auth.Users[:0]
	found := false
	for _, user := range cfg.Auth.Users {
		if user.Username == username {
			found = true
			continue
		}
		next = append(next, user)
	}
	if !found {
		return fmt.Errorf("用户 %q 不存在", username)
	}
	cfg.Auth.Users = next
	return a.saveAuthConfigLocked(cfg)
}

func (a *WebApp) ResetUserPassword(username, password string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	hash, err := proxy.HashPassword(password)
	if err != nil {
		return err
	}

	cfg := a.cfg
	found := false
	for i := range cfg.Auth.Users {
		if cfg.Auth.Users[i].Username == username {
			cfg.Auth.Users[i].Password = hash
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("用户 %q 不存在", username)
	}
	return a.saveAuthConfigLocked(cfg)
}

func (a *WebApp) ListRouteFiles() ([]config.RouteFileInfo, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.routeManager.List(a.cfg.Route.ActiveFile)
}

func (a *WebApp) LoadRouteFile(name string) (config.RouteRuleSet, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.routeManager.Load(name)
}

func (a *WebApp) SaveRouteFile(name string, set config.RouteRuleSet) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.routeManager.Save(name, set); err != nil {
		return err
	}
	if name == a.cfg.Route.ActiveFile {
		if err := a.applyRoutePolicyLocked(a.cfg); err != nil {
			return err
		}
	}
	return nil
}

func (a *WebApp) CreateRouteFile(name string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.routeManager.Create(name)
}

func (a *WebApp) DeleteRouteFile(name string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if name == a.cfg.Route.ActiveFile {
		return errors.New("当前正在使用的规则文件不能删除")
	}
	return a.routeManager.Delete(name)
}

func (a *WebApp) SetActiveRouteFile(name string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, err := a.routeManager.Load(name); err != nil {
		return err
	}
	cfg := a.cfg
	cfg.Route.ActiveFile = name
	if err := a.configManager.Save(cfg); err != nil {
		return err
	}
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
	return applyWebRoutePolicy(a.server, a.routeManager, cfg)
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

func applyWebRoutePolicy(server *proxy.Server, routeManager *config.RouteFileManager, cfg config.Config) error {
	if server == nil || routeManager == nil {
		return nil
	}
	if !cfg.Route.Enabled {
		server.SetRoutePolicy(false, config.RouteRuleSet{})
		return nil
	}
	set, err := routeManager.Load(cfg.Route.ActiveFile)
	if err != nil {
		return err
	}
	server.SetRoutePolicy(true, set)
	return nil
}

func listenerConfigChanged(a, b config.Config) bool {
	return a.Server.SOCKS5 != b.Server.SOCKS5 || a.Server.HTTP != b.Server.HTTP
}

func sameRuntimeConfig(a, b config.Config) bool {
	return a.Server == b.Server && reflect.DeepEqual(a.Auth, b.Auth) && a.Relay == b.Relay
}

func routeConfigChanged(a, b config.Config) bool {
	return a.Route != b.Route
}
