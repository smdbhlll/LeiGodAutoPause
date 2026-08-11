package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"leigod-auto-pause/internal/secret"
)

const currentVersion = 1

const (
	CloseAsk  = "ask"
	CloseTray = "tray"
	CloseExit = "exit"
)

type Game struct {
	Name       string `json:"name"`
	Executable string `json:"executable"`
	Path       string `json:"path,omitempty"`
	Source     string `json:"source,omitempty"`
	Enabled    bool   `json:"enabled"`
}

type Settings struct {
	Version          int    `json:"version"`
	Monitoring       bool   `json:"monitoring"`
	AutoPause        bool   `json:"autoPause"`
	AutoStart        bool   `json:"autoStart"`
	SilentStart      bool   `json:"silentStart"`
	StartMinimized   bool   `json:"startMinimized"`
	CloseAction      string `json:"closeAction"`
	CheckIntervalSec int    `json:"checkIntervalSec"`
	GracePeriodSec   int    `json:"gracePeriodSec"`
	LeiGodPath       string `json:"leiGodPath"`
	Username         string `json:"username"`
	EncryptedToken   string `json:"encryptedToken,omitempty"`
	EncryptedPassMD5 string `json:"encryptedPasswordMd5,omitempty"`
	Games            []Game `json:"games"`
}

type PublicSettings struct {
	Monitoring       bool   `json:"monitoring"`
	AutoPause        bool   `json:"autoPause"`
	AutoStart        bool   `json:"autoStart"`
	SilentStart      bool   `json:"silentStart"`
	StartMinimized   bool   `json:"startMinimized"`
	CloseAction      string `json:"closeAction"`
	CheckIntervalSec int    `json:"checkIntervalSec"`
	GracePeriodSec   int    `json:"gracePeriodSec"`
	LeiGodPath       string `json:"leiGodPath"`
	Username         string `json:"username"`
	HasToken         bool   `json:"hasToken"`
	HasPassword      bool   `json:"hasPassword"`
	Games            []Game `json:"games"`
}

type Store struct {
	mu       sync.RWMutex
	path     string
	settings Settings
}

func DataDir() (string, error) {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		return "", errors.New("LOCALAPPDATA is not available")
	}
	return filepath.Join(base, "LeiGodAutoPause"), nil
}

func NewStore(dataDir string) (*Store, error) {
	s := &Store{path: filepath.Join(dataDir, "config.json")}
	s.settings = defaults()
	data, err := os.ReadFile(s.path)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &s.settings); err != nil {
			backup := s.path + ".invalid-" + time.Now().Format("20060102-150405")
			_ = os.Rename(s.path, backup)
			s.settings = defaults()
		}
	}
	s.normalizeLocked()
	if err := s.saveLocked(); err != nil {
		return nil, err
	}
	return s, nil
}

func defaults() Settings {
	return Settings{
		Version:          currentVersion,
		Monitoring:       true,
		AutoPause:        true,
		CloseAction:      CloseAsk,
		CheckIntervalSec: 3,
		GracePeriodSec:   30,
		LeiGodPath:       `C:\Program Files (x86)\LeiGod_Acc\leigod.exe`,
		Games:            []Game{},
	}
}

func (s *Store) Snapshot() Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	copy := s.settings
	copy.Games = append([]Game(nil), s.settings.Games...)
	return copy
}

func (s *Store) Public() PublicSettings {
	v := s.Snapshot()
	return PublicSettings{
		Monitoring: v.Monitoring, AutoPause: v.AutoPause,
		AutoStart: v.AutoStart, SilentStart: v.SilentStart, StartMinimized: v.StartMinimized,
		CloseAction:      v.CloseAction,
		CheckIntervalSec: v.CheckIntervalSec, GracePeriodSec: v.GracePeriodSec,
		LeiGodPath: v.LeiGodPath, Username: v.Username,
		HasToken: v.EncryptedToken != "", HasPassword: v.EncryptedPassMD5 != "", Games: v.Games,
	}
}

func (s *Store) Update(fn func(*Settings) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := fn(&s.settings); err != nil {
		return err
	}
	s.normalizeLocked()
	return s.saveLocked()
}

func (s *Store) Credentials() (token, username, passwordMD5 string, err error) {
	v := s.Snapshot()
	if v.EncryptedToken != "" {
		token, err = secret.Unprotect(v.EncryptedToken)
		if err != nil {
			return "", "", "", err
		}
	}
	if v.EncryptedPassMD5 != "" {
		passwordMD5, err = secret.Unprotect(v.EncryptedPassMD5)
		if err != nil {
			return "", "", "", err
		}
	}
	return token, v.Username, passwordMD5, nil
}

func (s *Store) SetToken(token string) error {
	encrypted := ""
	var err error
	if token != "" {
		encrypted, err = secret.Protect(token)
		if err != nil {
			return err
		}
	}
	return s.Update(func(v *Settings) error { v.EncryptedToken = encrypted; return nil })
}

func (s *Store) SetPasswordMD5(hash string) error {
	encrypted := ""
	var err error
	if hash != "" {
		encrypted, err = secret.Protect(hash)
		if err != nil {
			return err
		}
	}
	return s.Update(func(v *Settings) error { v.EncryptedPassMD5 = encrypted; return nil })
}

func (s *Store) normalizeLocked() {
	s.settings.Version = currentVersion
	if s.settings.CheckIntervalSec < 2 {
		s.settings.CheckIntervalSec = 2
	}
	if s.settings.CheckIntervalSec > 60 {
		s.settings.CheckIntervalSec = 60
	}
	if s.settings.GracePeriodSec < 0 {
		s.settings.GracePeriodSec = 0
	}
	if s.settings.GracePeriodSec > 3600 {
		s.settings.GracePeriodSec = 3600
	}
	if s.settings.Games == nil {
		s.settings.Games = []Game{}
	}
	if s.settings.CloseAction != CloseAsk && s.settings.CloseAction != CloseTray && s.settings.CloseAction != CloseExit {
		s.settings.CloseAction = CloseAsk
	}
}

func (s *Store) saveLocked() error {
	data, err := json.MarshalIndent(s.settings, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
