package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"leigod-auto-pause/internal/app"
	"leigod-auto-pause/internal/config"
	"leigod-auto-pause/internal/gui"
	"leigod-auto-pause/internal/leigod"
	"leigod-auto-pause/internal/processes"
)

func main() {
	autoStarted := flag.Bool("autostart", false, "started by Windows sign-in")
	flag.Parse()

	dataDir, err := config.DataDir()
	if err != nil {
		fatal(err)
	}
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		fatal(err)
	}
	logFile, err := os.OpenFile(filepath.Join(dataDir, "app.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err == nil {
		defer logFile.Close()
		log.SetOutput(logFile)
	}
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	store, err := config.NewStore(dataDir)
	if err != nil {
		fatal(err)
	}
	client := leigod.NewClient(store)
	monitor := app.NewMonitor(store, processes.NewScanner(), client)
	monitor.Start()
	defer monitor.Stop()
	if err := gui.Run(store, monitor, client, processes.NewScanner(), *autoStarted); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	log.Printf("fatal: %v", err)
	gui.ShowError("雷神自动暂停", err.Error())
	fmt.Fprintln(os.Stderr, "LeiGod Auto Pause:", err)
	os.Exit(1)
}
