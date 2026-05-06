//go:build windows

package platform

import (
	"fmt"
	"strings"

	"github.com/getlantern/systray"
)

var currentNativeMenu nativeTrayMenu

func supportsTrayMenu() bool {
	return true
}

func trayHideDescription() string {
	return "Windows 使用通知区域托盘图标保持后台运行。"
}

func (t *TrayManager) setNativeMenu(menu nativeTrayMenu) {
	t.mu.Lock()
	currentNativeMenu = menu
	t.mu.Unlock()
}

func (t *TrayManager) updateNativeTray() {
	t.mu.Lock()
	running := t.serverRunning
	showStatusIP := t.showStatusIP
	menu := currentNativeMenu
	localIPs := append([]string(nil), t.localIPs...)
	socksAddr := t.socksAddr
	httpAddr := t.httpAddr
	t.mu.Unlock()

	if menu.start == nil || menu.stop == nil {
		return
	}

	systray.SetTooltip(trayTooltip(running, showStatusIP, localIPs, socksAddr, httpAddr))

	if running {
		menu.start.Disable()
		menu.stop.Enable()
		return
	}
	menu.start.Enable()
	menu.stop.Disable()
}

func emptyAsDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}

func trayIPSummary(localIPs []string) string {
	if len(localIPs) == 0 {
		return "未检测到"
	}
	first := strings.TrimSpace(localIPs[0])
	if first == "" {
		return "未检测到"
	}
	if len(localIPs) == 1 {
		return first
	}
	return fmt.Sprintf("%s 等 %d 个", first, len(localIPs))
}

func trayTooltip(running, showDetails bool, localIPs []string, socksAddr, httpAddr string) string {
	state := "停"
	if running {
		state = "运行"
	}
	if !showDetails {
		return "GoProxy " + state
	}
	lines := []string{
		"GoProxy " + state,
		"IP：" + trayIPSummary(localIPs),
		"S5：" + emptyAsDash(socksAddr),
		"HTTP：" + emptyAsDash(httpAddr),
	}
	return strings.Join(lines, "\n")
}
