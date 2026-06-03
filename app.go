package main

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gitee.com/jiuhuidalan1/goproxy/internal/config"
	"gitee.com/jiuhuidalan1/goproxy/internal/logger"
	"gitee.com/jiuhuidalan1/goproxy/internal/platform"
	"gitee.com/jiuhuidalan1/goproxy/internal/proxy"
	"gitee.com/jiuhuidalan1/goproxy/internal/stats"
	"gitee.com/jiuhuidalan1/goproxy/internal/store"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"go.uber.org/zap"
)

const appSource = "app"

//go:embed build/windows/icon.ico
var trayIcon []byte

// App is the Wails binding layer between the desktop UI and backend services.
type App struct {
	mu sync.Mutex

	ctx context.Context

	configPath    string
	logPath       string
	configManager *config.Manager
	store         *store.Store
	logger        *logger.Manager

	cfg        config.Config
	runtimeCfg config.Config
	collector  *stats.Collector
	server     *proxy.Server
	tray       *platform.TrayManager
}

// NewApp creates the desktop application using platform-specific paths.
func NewApp() (*App, error) {
	configPath, err := platform.ConfigPath()
	if err != nil {
		return nil, err
	}
	logPath, err := platform.LogPath()
	if err != nil {
		return nil, err
	}
	return NewAppWithPaths(configPath, logPath)
}

// NewAppWithPaths creates the application with explicit paths, primarily for tests.
func NewAppWithPaths(configPath, logPath string) (*App, error) {
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

	// 确保在出现错误时关闭数据库连接
	var closeDBOnError bool
	defer func() {
		if closeDBOnError && s != nil {
			s.Close()
		}
	}()

	if err := s.ImportFromYAML(configPath); err != nil {
		closeDBOnError = true
		return nil, fmt.Errorf("数据迁移失败: %w", err)
	}

	s.FillWebConfig(&cfg)
	s.FillAuthUsers(&cfg)
	s.FillActiveRoute(&cfg)

	logManager, err := logger.NewManager(cfg.Log, logPath)
	if err != nil {
		closeDBOnError = true
		return nil, fmt.Errorf("创建日志管理器失败: %w", err)
	}

	collector := stats.NewCollector()
	server := proxy.NewServer(cfg, collector)
	server.SetLogger(logManager)
	if err := applyAppRoutePolicy(server, s, cfg); err != nil {
		closeDBOnError = true
		return nil, fmt.Errorf("加载路由策略失败: %w", err)
	}

	// 成功返回，不关闭数据库连接，将其所有权转移给 App
	return &App{
		configPath:    configPath,
		logPath:       logPath,
		configManager: manager,
		store:         s,
		logger:        logManager,
		cfg:           cfg,
		runtimeCfg:    cfg,
		collector:     collector,
		server:        server,
		tray:          platform.NewTrayManager(cfg.UI.ShowTrayIcon, cfg.UI.CloseToTray, cfg.UI.TrayStatusAndIP),
	}, nil
}

func (a *App) startup(ctx context.Context) {
	cfg := a.cfg
	a.mu.Lock()
	a.ctx = ctx

	go platform.DisableMaximizeButton(appTitle)

	if a.tray != nil {
		a.tray.Startup(ctx)
		a.tray.StartNative(trayIcon, a.trayActions())
	}
	if a.logger != nil {
		a.subscribeLoggerLocked(a.logger)
		a.logger.Info(appSource, "应用已启动", zap.String("configPath", a.configPath))
	}
	a.emitStatusLocked()
	a.mu.Unlock()
	if cfg.UI.AutoStartProxy {
		_ = a.StartServer()
	}
	if cfg.UI.StartMinimized && cfg.UI.ShowTrayIcon {
		a.HideToTray()
	}
	go a.emitStatsLoop(ctx)
}

func (a *App) shutdown(ctx context.Context) {
	a.mu.Lock()
	server := a.server
	logManager := a.logger
	tray := a.tray
	s := a.store
	a.mu.Unlock()

	if server != nil {
		_ = server.Stop()
	}
	if tray != nil {
		tray.StopNative()
	}
	if logManager != nil {
		logManager.Info(appSource, "应用正在退出")
		_ = logManager.Close()
	}
	if s != nil {
		s.Close()
	}
}

// GetConfig returns the current complete configuration.
func (a *App) GetConfig() config.Config {
	a.mu.Lock()
	defer a.mu.Unlock()
	cfg := a.cfg
	a.store.FillWebConfig(&cfg)
	a.store.FillAuthUsers(&cfg)
	a.store.FillActiveRoute(&cfg)
	return cfg
}

// SaveConfig validates and persists configuration changes.
func (a *App) SaveConfig(cfg config.Config) error {
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
				a.logger.Warn(appSource, "日志配置无效", zap.Error(err))
			}
			return err
		}
	}

	if err := a.configManager.Save(cfg); err != nil {
		if a.logger != nil {
			a.logger.Warn(appSource, "配置保存失败", zap.Error(err))
		}
		return err
	}

	a.cfg = cfg
	if a.tray != nil {
		a.tray.SetEnabled(cfg.UI.ShowTrayIcon)
		a.tray.SetCloseToTray(cfg.UI.CloseToTray)
		a.tray.SetStatusIPVisible(cfg.UI.TrayStatusAndIP)
		if cfg.UI.ShowTrayIcon {
			a.tray.StartNative(trayIcon, a.trayActions())
		} else {
			a.tray.StopNative()
		}
	}
	if newLogger != nil {
		oldLogger := a.logger
		a.logger = newLogger
		a.subscribeLoggerLocked(newLogger)
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
	}

	if running && routeConfigChanged(oldCfg, cfg) {
		if err := a.applyRoutePolicyLocked(cfg); err != nil {
			return err
		}
		a.runtimeCfg.Route = cfg.Route
	}

	if a.logger != nil {
		a.logger.Info(appSource, "配置已保存")
		if running && listenerConfigChanged(oldCfg, cfg) {
			a.logger.Warn(appSource, "监听配置已保存，重启服务后生效")
		}
	}

	a.emitStatusLocked()
	return nil
}

// StartServer starts the proxy server using the current configuration.
func (a *App) StartServer() error {
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

	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	if err := a.server.Start(ctx); err != nil {
		if a.logger != nil {
			a.logger.Error(appSource, "代理服务启动失败", zap.Error(err))
		}
		return err
	}

	if a.logger != nil {
		status := a.server.Status()
		a.logger.Info(appSource, "代理服务已启动",
			zap.String("socks5Addr", status.SOCKS5Addr),
			zap.String("httpAddr", status.HTTPAddr),
		)
	}

	a.emitStatusLocked()
	return nil
}

// StopServer stops all listeners and active proxy connections.
func (a *App) StopServer() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.server == nil {
		return nil
	}
	if err := a.server.Stop(); err != nil {
		if a.logger != nil {
			a.logger.Error(appSource, "代理服务停止失败", zap.Error(err))
		}
		return err
	}

	if a.logger != nil {
		a.logger.Info(appSource, "代理服务已停止")
	}
	a.emitStatusLocked()
	return nil
}

// GetServerStatus returns the current server status.
func (a *App) GetServerStatus() proxy.Status {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.server == nil {
		return proxy.Status{}
	}
	return a.server.Status()
}

// GetStats returns a snapshot of current proxy counters.
func (a *App) GetStats() stats.Stats {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.server == nil {
		return stats.Stats{}
	}
	return a.server.Stats()
}

// GetActiveConnections returns current active proxy connection details.
func (a *App) GetActiveConnections() []proxy.ConnectionSnapshot {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.server == nil {
		return nil
	}
	return a.server.ActiveConnections()
}

// GetRecentLogs returns the newest n log entries from the in-memory ring buffer.
func (a *App) GetRecentLogs(n int) []logger.Entry {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.logger == nil {
		return nil
	}
	return a.logger.Recent(n)
}

// ClearLogs clears the in-memory UI logs and removes the current runtime log file.
func (a *App) ClearLogs() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	oldLogger := a.logger
	if oldLogger != nil {
		if err := oldLogger.Close(); err != nil {
			return fmt.Errorf("关闭日志文件失败: %w", err)
		}
	}
	if a.logPath != "" {
		if err := os.Remove(a.logPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("删除日志文件失败: %w", err)
		}
	}

	newLogger, err := logger.NewManager(a.cfg.Log, a.logPath)
	if err != nil {
		return fmt.Errorf("重新创建日志管理器失败: %w", err)
	}
	a.logger = newLogger
	a.subscribeLoggerLocked(newLogger)
	if a.server != nil {
		a.server.SetLogger(newLogger)
	}
	return nil
}

// GetTrayState returns the current tray/window integration state.
func (a *App) GetTrayState() platform.TrayState {
	a.mu.Lock()
	tray := a.tray
	a.mu.Unlock()
	if tray == nil {
		return platform.TrayState{}
	}
	return tray.State()
}

// GetLocalIPAddresses returns IPv4 addresses from active local network adapters.
func (a *App) GetLocalIPAddresses() ([]string, error) {
	return platform.LocalIPAddresses()
}

// GetNetworkInterfaces returns local adapters for route outbound binding.
func (a *App) GetNetworkInterfaces() ([]platform.NetworkInterface, error) {
	return platform.NetworkInterfaces()
}

// ListRouteFiles lists all available .rule files.
func (a *App) ListRouteFiles() ([]config.RouteFileInfo, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.store.ListRouteRuleSets()
}

// LoadRouteFile loads one route rule file.
func (a *App) LoadRouteFile(name string) (config.RouteRuleSet, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.store.LoadRouteRuleSet(name)
}

// SaveRouteFile validates and saves one route rule file.
func (a *App) SaveRouteFile(name string, set config.RouteRuleSet) error {
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

// CreateRouteFile creates a new route rule file.
func (a *App) CreateRouteFile(name string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.store.CreateRouteRuleSet(name)
}

// DeleteRouteFile removes a non-active route rule file.
func (a *App) DeleteRouteFile(name string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	active, _ := a.store.GetActiveRouteFileName()
	if name == active {
		return errors.New("当前正在使用的规则文件不能删除")
	}
	return a.store.DeleteRouteRuleSet(name)
}

// SetActiveRouteFile switches the active route policy file for new connections.
func (a *App) SetActiveRouteFile(name string) error {
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
		a.logger.Info(appSource, "当前规则文件已切换", zap.String("file", name))
	}
	return nil
}

func (a *App) trayActions() platform.TrayActions {
	return platform.TrayActions{
		ShowWindow:      a.ShowWindow,
		StartServer:     a.StartServer,
		StopServer:      a.StopServer,
		Quit:            a.QuitApp,
		IsServerRunning: func() bool { return a.GetServerStatus().Running },
		LocalIPs: func() []string {
			ips, _ := platform.LocalIPAddresses()
			return ips
		},
		SOCKS5Addr: func() string { return a.GetServerStatus().SOCKS5Addr },
		HTTPAddr:   func() string { return a.GetServerStatus().HTTPAddr },
	}
}

// ShowWindow restores the main window from the tray/background state.
func (a *App) ShowWindow() {
	a.mu.Lock()
	tray := a.tray
	a.mu.Unlock()
	if tray != nil {
		tray.ShowWindow()
	}
}

// HideToTray hides the main window when tray integration is enabled.
func (a *App) HideToTray() {
	a.mu.Lock()
	tray := a.tray
	a.mu.Unlock()
	if tray != nil {
		tray.HideWindow()
	}
}

// QuitApp exits the desktop application from the tray/menu command path.
func (a *App) QuitApp() {
	a.mu.Lock()
	tray := a.tray
	a.mu.Unlock()
	if tray != nil {
		tray.RequestQuit()
		return
	}
	if a.ctx != nil {
		runtime.Quit(a.ctx)
	}
}

// SetAuthEnabled enables or disables proxy authentication.
func (a *App) SetAuthEnabled(enabled bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	cfg := a.cfg
	cfg.Auth.Enabled = enabled
	return a.saveAuthConfigLocked(cfg)
}

// AddUser creates a bcrypt-backed proxy user.
func (a *App) AddUser(username, password string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return errors.New("请输入用户名")
	}
	if password == "" {
		return errors.New("请输入密码")
	}

	hash, err := proxy.HashPassword(password)
	if err != nil {
		return err
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.store.AddAuthUser(username, hash); err != nil {
		return err
	}

	cfg := a.cfg
	users, _ := a.store.ListAuthUsers()
	cfg.Auth.Users = users
	return a.saveAuthConfigLocked(cfg)
}

// RemoveUser deletes a proxy user by username.
func (a *App) RemoveUser(username string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return errors.New("请输入用户名")
	}

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

// ResetUserPassword replaces a user's bcrypt password hash.
func (a *App) ResetUserPassword(username, password string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return errors.New("请输入用户名")
	}
	if password == "" {
		return errors.New("请输入密码")
	}

	hash, err := proxy.HashPassword(password)
	if err != nil {
		return err
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.store.UpdateAuthUserPassword(username, hash); err != nil {
		return err
	}

	cfg := a.cfg
	users, _ := a.store.ListAuthUsers()
	cfg.Auth.Users = users
	return a.saveAuthConfigLocked(cfg)
}

func (a *App) saveAuthConfigLocked(cfg config.Config) error {
	if err := a.configManager.Save(cfg); err != nil {
		return err
	}
	a.cfg = cfg
	if a.server != nil {
		a.server.SetAuthConfig(cfg.Auth)
		a.runtimeCfg.Auth = cfg.Auth
	}
	a.emitStatusLocked()
	return nil
}

func (a *App) beforeClose(ctx context.Context) bool {
	a.mu.Lock()
	tray := a.tray
	a.mu.Unlock()
	if tray == nil {
		return false
	}
	return tray.BeforeClose(ctx)
}

func (a *App) emitStatusLocked() {
	if a.ctx == nil || a.server == nil {
		return
	}
	status := a.server.Status()
	if a.tray != nil {
		ips, _ := platform.LocalIPAddresses()
		a.tray.SetServerStatus(status.Running, ips, status.SOCKS5Addr, status.HTTPAddr)
	}
	runtime.EventsEmit(a.ctx, "proxy:status", status)
}

func (a *App) emitStatsLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.mu.Lock()
			emitCtx := a.ctx
			server := a.server
			a.mu.Unlock()
			if emitCtx == nil || server == nil {
				continue
			}
			runtime.EventsEmit(emitCtx, "proxy:stats", server.TickStats())
		}
	}
}

func (a *App) subscribeLoggerLocked(logManager *logger.Manager) {
	if a.ctx == nil || logManager == nil {
		return
	}
	emitCtx := a.ctx
	logManager.Subscribe(func(entry logger.Entry) {
		runtime.EventsEmit(emitCtx, "proxy:log", entry)
	})
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

func (a *App) applyRoutePolicyLocked(cfg config.Config) error {
	return applyAppRoutePolicy(a.server, a.store, cfg)
}

func applyAppRoutePolicy(server *proxy.Server, s *store.Store, cfg config.Config) error {
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
