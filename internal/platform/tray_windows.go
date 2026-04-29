//go:build windows

package platform

import "github.com/getlantern/systray"

var currentNativeMenu nativeTrayMenu

func supportsTrayMenu() bool {
	return true
}

func trayHideDescription() string {
	return "Windows uses a notification-area style background window."
}

func (t *TrayManager) setNativeMenu(menu nativeTrayMenu) {
	t.mu.Lock()
	currentNativeMenu = menu
	t.mu.Unlock()
}

func (t *TrayManager) updateNativeTray() {
	t.mu.Lock()
	running := t.serverRunning
	menu := currentNativeMenu
	t.mu.Unlock()

	if menu.start == nil || menu.stop == nil {
		return
	}
	if running {
		menu.start.Disable()
		menu.stop.Enable()
		systray.SetTooltip("ProxyServer - 服务运行中")
		return
	}
	menu.start.Enable()
	menu.stop.Disable()
	systray.SetTooltip("ProxyServer - 服务已停止")
}
