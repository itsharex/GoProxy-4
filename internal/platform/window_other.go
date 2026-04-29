//go:build !windows

package platform

// DisableMaximizeButton is a no-op on non-Windows platforms.
func DisableMaximizeButton(_ string) {}
