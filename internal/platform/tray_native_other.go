//go:build !windows

package platform

func (t *TrayManager) startNativeTray(_ []byte) {}

func (t *TrayManager) stopNativeTray() {}
