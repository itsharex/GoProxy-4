//go:build !windows && !darwin

package platform

func supportsTrayMenu() bool {
	return false
}

func trayHideDescription() string {
	return "This platform keeps the process active by hiding the window when supported by Wails."
}

func (t *TrayManager) updateNativeTray() {}
