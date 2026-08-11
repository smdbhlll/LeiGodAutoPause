//go:build windows

package gui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"leigod-auto-pause/internal/config"
	"leigod-auto-pause/internal/processes"
)

func TestCandidateCategory(t *testing.T) {
	tests := []struct {
		game config.Game
		want string
	}{
		{config.Game{Executable: "Game.exe", Path: `D:\\SteamLibrary\\steamapps\\common\\Game\\Game.exe`}, "游戏"},
		{config.Game{Executable: "steam.exe", Path: `C:\\Program Files (x86)\\Steam\\steam.exe`}, "游戏平台"},
		{config.Game{Executable: "msedge.exe", Path: `C:\\Program Files\\Microsoft\\Edge\\msedge.exe`}, "浏览器"},
		{config.Game{Executable: "code.exe", Path: `C:\\Apps\\Code.exe`}, "应用工具"},
		{config.Game{Name: "Installed", Executable: "installed.exe", Source: "Steam"}, "已安装游戏"},
	}
	for _, test := range tests {
		if got := candidateCategory(test.game); got != test.want {
			t.Errorf("category for %s: got %q, want %q", test.game.Executable, got, test.want)
		}
	}
}

type fakeScanner struct{ items []processes.Process }

func (scanner fakeScanner) List() ([]processes.Process, error) { return scanner.items, nil }

func TestResolveLeiGodPath(t *testing.T) {
	directory := t.TempDir()
	executable := filepath.Join(directory, "leigod.exe")
	if err := os.WriteFile(executable, []byte("test"), 0600); err != nil {
		t.Fatal(err)
	}
	if got := resolveLeiGodPath(`"`+executable+`"`, nil); got != executable {
		t.Fatalf("configured path: got %q", got)
	}
	if got := resolveLeiGodPath("", fakeScanner{items: []processes.Process{{Name: "leigod.exe", Path: executable}}}); got != executable {
		t.Fatalf("process path: got %q", got)
	}
}

func TestSystemDarkModeOverride(t *testing.T) {
	t.Setenv("LEIGOD_THEME", "dark")
	if !systemDarkMode() {
		t.Fatal("dark override was not applied")
	}
	t.Setenv("LEIGOD_THEME", "light")
	if systemDarkMode() {
		t.Fatal("light override was not applied")
	}
}

func TestMixChannel(t *testing.T) {
	if got := mixChannel(20, 120, 50); got != 70 {
		t.Fatalf("midpoint: got %d, want 70", got)
	}
}

func TestToggleProgressUsesTimeBasedEasing(t *testing.T) {
	started := time.Date(2026, 8, 12, 9, 0, 0, 0, time.UTC)
	window := &window{
		toggleStates: map[uintptr]bool{1: true},
		toggleAnimation: map[uintptr]toggleMotion{
			1: {from: 0, to: 1, started: started, dur: 200 * time.Millisecond},
		},
	}
	if got := window.toggleProgress(1, started); got != 0 {
		t.Fatalf("initial progress: got %f, want 0", got)
	}
	if got := window.toggleProgress(1, started.Add(100*time.Millisecond)); got != 0.5 {
		t.Fatalf("midpoint progress: got %f, want 0.5", got)
	}
	if got := window.toggleProgress(1, started.Add(240*time.Millisecond)); got != 1 {
		t.Fatalf("completed progress: got %f, want 1", got)
	}
}
