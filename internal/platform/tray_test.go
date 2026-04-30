package platform

import (
	"context"
	"reflect"
	"testing"
)

func TestTrayManagerStateAndCloseBehavior(t *testing.T) {
	tray := NewTrayManager(true, true, true)
	tray.window = noopWindowOps{}
	tray.Startup(context.Background())

	state := tray.State()
	if !state.Enabled {
		t.Fatal("expected tray to be enabled")
	}
	if !state.Visible {
		t.Fatal("expected window to start visible")
	}

	if !tray.BeforeClose(context.Background()) {
		t.Fatal("expected close to be prevented when tray is enabled")
	}
	if tray.State().Visible {
		t.Fatal("expected close-to-tray to mark the window hidden")
	}

	tray.RequestQuit()
	if tray.BeforeClose(context.Background()) {
		t.Fatal("expected close to continue after quit is requested")
	}
}

func TestTrayManagerDisabledDoesNotPreventClose(t *testing.T) {
	tray := NewTrayManager(false, false, false)
	tray.window = noopWindowOps{}
	if tray.BeforeClose(context.Background()) {
		t.Fatal("expected disabled tray to allow window close")
	}
}

func TestTrayManagerCloseToTrayDisabledDoesNotPreventClose(t *testing.T) {
	tray := NewTrayManager(true, false, true)
	tray.window = noopWindowOps{}
	if tray.BeforeClose(context.Background()) {
		t.Fatal("expected close-to-tray disabled to allow window close")
	}
}

type noopWindowOps struct{}

func (noopWindowOps) Show(context.Context)       {}
func (noopWindowOps) Unminimise(context.Context) {}
func (noopWindowOps) Hide(context.Context)       {}
func (noopWindowOps) Quit(context.Context)       {}

type recordingWindowOps struct {
	calls []string
}

func (r *recordingWindowOps) Show(context.Context) {
	r.calls = append(r.calls, "show")
}

func (r *recordingWindowOps) Unminimise(context.Context) {
	r.calls = append(r.calls, "unminimise")
}

func (r *recordingWindowOps) Hide(context.Context) {
	r.calls = append(r.calls, "hide")
}

func (r *recordingWindowOps) Quit(context.Context) {
	r.calls = append(r.calls, "quit")
}

func TestTrayManagerShowWindowRestoresVisibility(t *testing.T) {
	window := &recordingWindowOps{}
	tray := NewTrayManager(true, true, true)
	tray.window = window
	tray.Startup(context.Background())

	tray.HideWindow()
	tray.ShowWindow()

	if !tray.State().Visible {
		t.Fatal("expected show window to mark the window visible")
	}
	if !reflect.DeepEqual(window.calls, []string{"hide", "unminimise", "show"}) {
		t.Fatalf("unexpected window calls: %v", window.calls)
	}
}
