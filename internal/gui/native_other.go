//go:build !windows

package gui

import (
	"errors"

	"leigod-auto-pause/internal/app"
	"leigod-auto-pause/internal/config"
	"leigod-auto-pause/internal/leigod"
	"leigod-auto-pause/internal/processes"
)

func Run(*config.Store, *app.Monitor, *leigod.Client, processes.Scanner, bool) error {
	return errors.New("native interface is supported on Windows only")
}

func ShowError(string, string) {}
