// Package config holds host settings. Settings persist as one JSON blob in the
// store; the Spotify token lives in a separate 0600 file next to the database
// so the database itself never contains a credential.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
)

// EnvDir overrides the data directory (tests, dev).
const EnvDir = "VIBE_MUSIC_DIR"

const (
	DefaultPort      = 5555
	DefaultSkipRatio = 0.5
	appDirName       = "vibe-music"
	DBFile           = "vibe-music.db"
	tokenFile        = "spotify-token.json"
	settingsKey      = "config"
)

type Config struct {
	Port         int     `json:"port"`
	ActiveSource string  `json:"activeSource"`
	SkipRatio    float64 `json:"skipRatio"`
	InviteOnly   bool    `json:"inviteOnly"`
	Local        Local   `json:"local"`
	Spotify      Spotify `json:"spotify"`
}

type Local struct {
	Folders []string `json:"folders"`
}

type Spotify struct {
	ClientID string `json:"clientId"`
	DeviceID string `json:"deviceId,omitempty"`
}

// KV is what config needs from the store.
type KV interface {
	Get(key string) ([]byte, error)
	Set(key string, v []byte) error
}

func Default() Config {
	c := Config{Port: DefaultPort, ActiveSource: "local", SkipRatio: DefaultSkipRatio}
	c.normalize()
	return c
}

// normalize keeps slices non-nil so they serialize as [] rather than null,
// which the UIs would otherwise have to guard against everywhere.
func (c *Config) normalize() {
	if c.Local.Folders == nil {
		c.Local.Folders = []string{}
	}
}

// Dir returns the data directory (created on demand).
func Dir() (string, error) {
	dir := os.Getenv(EnvDir)
	if dir == "" {
		base, err := os.UserConfigDir()
		if err != nil {
			return "", fmt.Errorf("config dir: %w", err)
		}
		dir = filepath.Join(base, appDirName)
	}
	return dir, os.MkdirAll(dir, 0o700)
}

// Load reads settings from kv, saving defaults when none exist.
func Load(kv KV) (Config, error) {
	data, err := kv.Get(settingsKey)
	if err != nil {
		c := Default()
		return c, Save(kv, c)
	}
	c := Default()
	if err := json.Unmarshal(data, &c); err != nil {
		return Config{}, fmt.Errorf("parse settings: %w", err)
	}
	c.normalize()
	return c, nil
}

func Save(kv KV, c Config) error {
	c.normalize()
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return kv.Set(settingsKey, data)
}

// LoadToken returns nil, nil when no token is stored.
// ponytail: plaintext file, 0600, same protection as the database itself.
// Encrypting it with a key stored beside it would add nothing; the OS
// keychain is the upgrade path if this ever runs on a shared machine.
func LoadToken() (*oauth2.Token, error) {
	dir, err := Dir()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(dir, tokenFile))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var tok oauth2.Token
	if err := json.Unmarshal(data, &tok); err != nil {
		return nil, err
	}
	return &tok, nil
}

// SaveToken writes atomically with 0600; nil removes the file.
func SaveToken(tok *oauth2.Token) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	p := filepath.Join(dir, tokenFile)
	if tok == nil {
		err := os.Remove(p)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	data, err := json.Marshal(tok)
	if err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}
