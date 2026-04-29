//go:build windows

package platform

import "github.com/getlantern/systray"

type nativeTrayMenu struct {
	show  *systray.MenuItem
	start *systray.MenuItem
	stop  *systray.MenuItem
	quit  *systray.MenuItem
}

func (t *TrayManager) startNativeTray(icon []byte) {
	go systray.Run(func() {
		systray.SetIcon(icon)
		systray.SetTitle("ProxyServer")
		systray.SetTooltip("ProxyServer")

		menu := nativeTrayMenu{
			show:  systray.AddMenuItem("显示窗口", "显示 ProxyServer 主窗口"),
			start: systray.AddMenuItem("启动服务", "启动代理服务"),
			stop:  systray.AddMenuItem("停止服务", "停止代理服务"),
			quit:  systray.AddMenuItem("退出", "退出 ProxyServer"),
		}
		t.setNativeMenu(menu)
		t.updateNativeTray()

		go t.watchNativeMenu(menu)
	}, func() {
		t.mu.Lock()
		t.nativeStarted = false
		t.mu.Unlock()
	})
}

func (t *TrayManager) watchNativeMenu(menu nativeTrayMenu) {
	for {
		select {
		case <-menu.show.ClickedCh:
			if t.actions.ShowWindow != nil {
				t.actions.ShowWindow()
			}
		case <-menu.start.ClickedCh:
			t.runTrayAction(t.actions.StartServer)
		case <-menu.stop.ClickedCh:
			t.runTrayAction(t.actions.StopServer)
		case <-menu.quit.ClickedCh:
			if t.actions.Quit != nil {
				t.actions.Quit()
			}
			return
		}
	}
}

func (t *TrayManager) stopNativeTray() {
	if t.State().NativeStarted {
		systray.Quit()
	}
}
