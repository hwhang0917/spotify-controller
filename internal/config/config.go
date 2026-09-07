// Package config persists host settings: guest port, active source, and the
// per-source settings the admin enters. Lives in the OS user config dir.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
)

// EnvPath overrides the config file location (tests, dev).
const EnvPath = "VIBE_MUSIC_CONFIG"

const (
	DefaultPort      = 5555
	DefaultSkipRatio = 0.5
	appDirName       = "vibe-music"
	fileName         = "config.json"
)

type Config struct {
	Port         int     `json:"port"`
	ActiveSource string  `json:"activeSource"`
	SkipRatio    float64 `json:"skipRatio"`
	Local        Local   `json:"local"`
	Spotify      Spotify `json:"spotify"`
}

type Local struct {
	Folders []string `json:"folders"`
}

type Spotify struct {
	ClientID string        `json:"clientId"`
	DeviceID string        `json:"deviceId,omitempty"`
	Token    *oauth2.Token `json:"token,omitempty"`
	// ponytail: refresh token stored in plaintext (0600). Move to the OS keychain
	// if this ever runs somewhere other than a single trusted host PC.
}

func Default() Config {
	return Config{Port: DefaultPort, ActiveSource: "local", SkipRatio: DefaultSkipRatio}
}

// Path returns the config file location.
func Path() (string, error) {
	if p := os.Getenv(EnvPath); p != "" {
		return p, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("config dir: %w", err)
	}
	return filepath.Join(dir, appDirName, fileName), nil
}

// Load reads the config, creating a default file if none exists.
func Load() (Config, error) {
	p, err := Path()
	if err != nil {
		return Config{}, err
	}
	data, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		c := Default()
		return c, Save(c)
	}
	if err != nil {
		return Config{}, err
	}
	c := Default()
	if err := json.Unmarshal(data, &c); err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", p, err)
	}
	return c, nil
}

// Save writes atomically with 0600 permissions.
func Save(c Config) error {
	p, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}
