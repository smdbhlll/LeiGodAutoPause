package app

import (
	"testing"

	"leigod-auto-pause/internal/config"
	"leigod-auto-pause/internal/processes"
)

func TestMatchGames(t *testing.T) {
	games := []config.Game{
		{Name: "Game A", Executable: "GameA.exe", Enabled: true},
		{Name: "Game B", Executable: "C:\\Games\\GameB.EXE", Enabled: true},
		{Name: "Disabled", Executable: "off.exe", Enabled: false},
	}
	running := []processes.Process{{Name: "gamea.EXE"}, {Name: "GameB.exe"}, {Name: "off.exe"}}
	matched := matchGames(games, running)
	if len(matched) != 2 || matched[0] != "Game A" || matched[1] != "Game B" {
		t.Fatalf("unexpected matches: %#v", matched)
	}
}

func TestMatchGamesNeedsExactExecutable(t *testing.T) {
	games := []config.Game{{Name: "GTA", Executable: "gta.exe", Enabled: true}}
	running := []processes.Process{{Name: "gta-helper.exe"}}
	if matched := matchGames(games, running); len(matched) != 0 {
		t.Fatalf("partial process name should not match: %#v", matched)
	}
}
