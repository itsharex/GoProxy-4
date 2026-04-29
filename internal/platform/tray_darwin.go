//go:build darwin

package platform

func supportsTrayMenu() bool {
	return true
}

func trayHideDescription() string {
	return "macOS keeps the app process active and restores the window from the dock/menu command."
}

func (t *TrayManager) setNativeMenu(_ struct{}) {}

func (t *TrayManager) updateNativeTray() {}
