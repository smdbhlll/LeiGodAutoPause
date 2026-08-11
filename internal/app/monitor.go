package app

import (
	"context"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"leigod-auto-pause/internal/config"
	"leigod-auto-pause/internal/leigod"
	"leigod-auto-pause/internal/processes"
)

type MonitorStatus struct {
	Phase          string   `json:"phase"`
	Message        string   `json:"message"`
	RunningGames   []string `json:"runningGames"`
	CountdownUntil string   `json:"countdownUntil,omitempty"`
	LastPauseAt    string   `json:"lastPauseAt,omitempty"`
	LastError      string   `json:"lastError,omitempty"`
	LastScanAt     string   `json:"lastScanAt,omitempty"`
}

type Monitor struct {
	store    *config.Store
	scanner  processes.Scanner
	client   *leigod.Client
	mu       sync.RWMutex
	status   MonitorStatus
	hadGame  bool
	deadline time.Time
	wake     chan struct{}
	stop     chan struct{}
	done     chan struct{}
}

func NewMonitor(store *config.Store, scanner processes.Scanner, client *leigod.Client) *Monitor {
	return &Monitor{store: store, scanner: scanner, client: client, wake: make(chan struct{}, 1), stop: make(chan struct{}), done: make(chan struct{}), status: MonitorStatus{Phase: "idle", Message: "等待游戏运行"}}
}

func (m *Monitor) Start() { go m.loop() }
func (m *Monitor) Stop()  { close(m.stop); <-m.done }
func (m *Monitor) Wake() {
	select {
	case m.wake <- struct{}{}:
	default:
	}
}

func (m *Monitor) Status() MonitorStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	copy := m.status
	copy.RunningGames = append([]string(nil), m.status.RunningGames...)
	return copy
}

func (m *Monitor) loop() {
	defer close(m.done)
	for {
		settings := m.store.Snapshot()
		interval := time.Duration(settings.CheckIntervalSec) * time.Second
		if interval < 2*time.Second {
			interval = 2 * time.Second
		}
		m.tick(settings)
		timer := time.NewTimer(interval)
		select {
		case <-timer.C:
		case <-m.wake:
			if !timer.Stop() {
				<-timer.C
			}
		case <-m.stop:
			if !timer.Stop() {
				<-timer.C
			}
			return
		}
	}
}

func (m *Monitor) tick(settings config.Settings) {
	now := time.Now()
	if !settings.Monitoring {
		m.setStatus(MonitorStatus{Phase: "disabled", Message: "监控已关闭", LastScanAt: now.Format(time.RFC3339)})
		m.hadGame, m.deadline = false, time.Time{}
		return
	}
	items, err := m.scanner.List()
	if err != nil {
		m.setStatus(MonitorStatus{Phase: "error", Message: "无法扫描进程", LastError: err.Error(), LastScanAt: now.Format(time.RFC3339)})
		return
	}
	running := matchGames(settings.Games, items)
	if len(running) > 0 {
		m.hadGame = true
		m.deadline = time.Time{}
		m.setStatus(MonitorStatus{Phase: "playing", Message: "检测到游戏运行", RunningGames: running, LastScanAt: now.Format(time.RFC3339), LastPauseAt: m.Status().LastPauseAt})
		return
	}
	if !m.hadGame {
		m.setStatus(MonitorStatus{Phase: "idle", Message: "等待游戏运行", LastScanAt: now.Format(time.RFC3339), LastPauseAt: m.Status().LastPauseAt})
		return
	}
	if m.deadline.IsZero() {
		m.deadline = now.Add(time.Duration(settings.GracePeriodSec) * time.Second)
	}
	if now.Before(m.deadline) {
		m.setStatus(MonitorStatus{Phase: "countdown", Message: "游戏已退出，等待自动暂停", CountdownUntil: m.deadline.Format(time.RFC3339), LastScanAt: now.Format(time.RFC3339), LastPauseAt: m.Status().LastPauseAt})
		return
	}
	m.hadGame, m.deadline = false, time.Time{}
	if !settings.AutoPause {
		m.setStatus(MonitorStatus{Phase: "idle", Message: "游戏已退出，自动暂停未开启", LastScanAt: now.Format(time.RFC3339), LastPauseAt: m.Status().LastPauseAt})
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	message, pauseErr := m.client.Pause(ctx)
	cancel()
	if pauseErr != nil {
		log.Printf("automatic pause failed: %v", pauseErr)
		m.setStatus(MonitorStatus{Phase: "error", Message: "自动暂停失败", LastError: pauseErr.Error(), LastScanAt: now.Format(time.RFC3339), LastPauseAt: m.Status().LastPauseAt})
		return
	}
	m.setStatus(MonitorStatus{Phase: "paused", Message: message, LastScanAt: now.Format(time.RFC3339), LastPauseAt: time.Now().Format(time.RFC3339)})
}

func (m *Monitor) RecordManualPause(message string, err error) {
	status := m.Status()
	status.RunningGames = nil
	status.CountdownUntil = ""
	if err != nil {
		status.Phase, status.Message, status.LastError = "error", "手动暂停失败", err.Error()
	} else {
		status.Phase, status.Message, status.LastError, status.LastPauseAt = "paused", message, "", time.Now().Format(time.RFC3339)
	}
	m.setStatus(status)
}

func (m *Monitor) setStatus(status MonitorStatus) { m.mu.Lock(); m.status = status; m.mu.Unlock() }

func matchGames(games []config.Game, running []processes.Process) []string {
	byName := make(map[string]bool, len(running))
	for _, item := range running {
		byName[strings.ToLower(filepath.Base(item.Name))] = true
	}
	result := []string{}
	for _, game := range games {
		if !game.Enabled {
			continue
		}
		exe := strings.ToLower(filepath.Base(strings.TrimSpace(game.Executable)))
		if exe != "" && byName[exe] {
			result = append(result, game.Name)
		}
	}
	return result
}
