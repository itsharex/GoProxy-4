package platform

import (
	"context"
	"runtime"
	"sync"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// TrayState describes the platform tray/window integration exposed to the UI.
type TrayState struct {
	Enabled         bool   `json:"enabled"`
	Visible         bool   `json:"visible"`
	Platform        string `json:"platform"`
	SupportsMenu    bool   `json:"supportsMenu"`
	NativeStarted   bool   `json:"nativeStarted"`
	HideDescription string `json:"hideDescription"`
}

// TrayActions contains application callbacks invoked from native tray menus.
type TrayActions struct {
	ShowWindow      func()
	StartServer     func() error
	StopServer      func() error
	Quit            func()
	IsServerRunning func() bool
	LocalIPs        func() []string
	SOCKS5Addr      func() string
	HTTPAddr        func() string
}

// TrayManager centralizes desktop window actions that differ by platform.
type TrayManager struct {
	mu            sync.Mutex
	ctx           context.Context
	enabled       bool
	closeToTray   bool
	showStatusIP  bool
	visible       bool
	quit          bool
	nativeStarted bool
	serverRunning bool
	localIPs      []string
	socksAddr     string
	httpAddr      string
	actions       TrayActions
	window        windowOps
}

// NewTrayManager creates a tray manager with platform defaults.
func NewTrayManager(enabled, closeToTray, showStatusIP bool) *TrayManager {
	return &TrayManager{
		enabled:      enabled,
		closeToTray:  closeToTray,
		showStatusIP: showStatusIP,
		visible:      true,
		window:       wailsWindowOps{},
	}
}

// Startup attaches the Wails context used by runtime window calls.
func (t *TrayManager) Startup(ctx context.Context) {
	t.mu.Lock()
	t.ctx = ctx
	t.mu.Unlock()
}

// StartNative creates the platform tray icon and menu when supported.
func (t *TrayManager) StartNative(icon []byte, actions TrayActions) {
	t.mu.Lock()
	t.actions = actions
	if !t.enabled || t.nativeStarted {
		t.mu.Unlock()
		return
	}
	t.nativeStarted = true
	t.mu.Unlock()
	t.startNativeTray(icon)
}

// StopNative removes the platform tray icon when it is running.
func (t *TrayManager) StopNative() {
	t.stopNativeTray()
}

// SetEnabled updates whether close-to-tray behavior is active.
func (t *TrayManager) SetEnabled(enabled bool) {
	t.mu.Lock()
	t.enabled = enabled
	t.mu.Unlock()
	t.updateNativeTray()
}

// SetCloseToTray updates whether closing the window should hide it to tray.
func (t *TrayManager) SetCloseToTray(enabled bool) {
	t.mu.Lock()
	t.closeToTray = enabled
	t.mu.Unlock()
}

// SetStatusIPVisible updates native tray status/IP menu visibility.
func (t *TrayManager) SetStatusIPVisible(enabled bool) {
	t.mu.Lock()
	t.showStatusIP = enabled
	t.mu.Unlock()
	t.updateNativeTray()
}

// ShowWindow restores the main window.
func (t *TrayManager) ShowWindow() {
	t.mu.Lock()
	ctx := t.ctx
	t.visible = true
	t.mu.Unlock()
	if ctx != nil {
		t.window.Unminimise(ctx)
		t.window.Show(ctx)
	}
}

// HideWindow hides the main window while keeping the process alive.
func (t *TrayManager) HideWindow() {
	t.mu.Lock()
	ctx := t.ctx
	if t.enabled {
		t.visible = false
	}
	enabled := t.enabled
	t.mu.Unlock()
	if enabled && ctx != nil {
		t.window.Hide(ctx)
	}
}

// RequestQuit marks the application as intentionally quitting.
func (t *TrayManager) RequestQuit() {
	t.mu.Lock()
	ctx := t.ctx
	t.quit = true
	t.mu.Unlock()
	if ctx != nil {
		t.window.Quit(ctx)
	}
	t.stopNativeTray()
}

// BeforeClose implements close-to-tray when the tray integration is enabled.
func (t *TrayManager) BeforeClose(ctx context.Context) bool {
	t.mu.Lock()
	if t.quit || !t.enabled || !t.closeToTray {
		t.mu.Unlock()
		return false
	}
	t.ctx = ctx
	t.visible = false
	t.mu.Unlock()
	t.window.Hide(ctx)
	return true
}

// State returns the current tray/window state.
func (t *TrayManager) State() TrayState {
	t.mu.Lock()
	defer t.mu.Unlock()
	return TrayState{
		Enabled:         t.enabled,
		Visible:         t.visible,
		Platform:        runtime.GOOS,
		SupportsMenu:    supportsTrayMenu(),
		NativeStarted:   t.nativeStarted,
		HideDescription: trayHideDescription(),
	}
}

// SetServerRunning updates tray menu state after server lifecycle changes.
func (t *TrayManager) SetServerRunning(running bool) {
	t.mu.Lock()
	t.serverRunning = running
	t.mu.Unlock()
	t.updateNativeTray()
}

// SetServerStatus updates the tray status snapshot without calling back into App.
func (t *TrayManager) SetServerStatus(running bool, localIPs []string, socksAddr, httpAddr string) {
	t.mu.Lock()
	t.serverRunning = running
	t.localIPs = append(t.localIPs[:0], localIPs...)
	t.socksAddr = socksAddr
	t.httpAddr = httpAddr
	t.mu.Unlock()
	t.updateNativeTray()
}

func (t *TrayManager) runTrayAction(action func() error) {
	if action == nil {
		return
	}
	if err := action(); err != nil {
		return
	}
	if t.actions.IsServerRunning != nil {
		t.SetServerRunning(t.actions.IsServerRunning())
	}
}

type windowOps interface {
	Show(context.Context)
	Unminimise(context.Context)
	Hide(context.Context)
	Quit(context.Context)
}

type wailsWindowOps struct{}

func (wailsWindowOps) Show(ctx context.Context) {
	wailsruntime.WindowShow(ctx)
}

func (wailsWindowOps) Unminimise(ctx context.Context) {
	wailsruntime.WindowUnminimise(ctx)
}

func (wailsWindowOps) Hide(ctx context.Context) {
	wailsruntime.WindowHide(ctx)
}

func (wailsWindowOps) Quit(ctx context.Context) {
	wailsruntime.Quit(ctx)
}
