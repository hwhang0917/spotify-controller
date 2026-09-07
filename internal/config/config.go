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
	"strings"

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
	youtubeKeyFile   = "youtube-api-key"
	settingsKey      = "config"
)

type Config struct {
	Port       int     `json:"port"`
	SkipRatio  float64 `json:"skipRatio"`
	InviteOnly bool    `json:"inviteOnly"`
	// Enabled lists the sources the admin switched on (Use).
	Enabled []string `json:"enabled"`
	// Kept only to migrate settings written before Enabled existed.
	ActiveSource string  `json:"activeSource,omitempty"`
	Local        Local   `json:"local"`
	Spotify      Spotify `json:"spotify"`
	// YouTube.HasKey is derived at read time; the key itself is in its own file.
	YouTube YouTube `json:"youtube"`
}

type YouTube struct {
	HasKey bool `json:"hasKey"`
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
	c := Config{Port: DefaultPort, SkipRatio: DefaultSkipRatio}
	c.normalize()
	return c
}

// normalize keeps slices non-nil so they serialize as [] rather than null,
// which the UIs would otherwise have to guard against everywhere.
func (c *Config) normalize() {
	if c.Local.Folders == nil {
		c.Local.Folders = []string{}
	}
	if c.Enabled == nil {
		// nothing stored: pre-Enabled settings name a single active source,
		// otherwise start with local files
		c.Enabled = []string{"local"}
		if c.ActiveSource != "" {
			c.Enabled = []string{c.ActiveSource}
		}
	}
	c.ActiveSource = ""
}

// IsEnabled reports whether the admin switched a source on.
func (c Config) IsEnabled(id string) bool {
	for _, e := range c.Enabled {
		if e == id {
			return true
		}
	}
	return false
}

// SetEnabled adds or removes id from the enabled list.
func (c *Config) SetEnabled(id string, on bool) {
	kept := make([]string, 0, len(c.Enabled)+1)
	for _, e := range c.Enabled {
		if e != id {
			kept = append(kept, e)
		}
	}
	if on {
		kept = append(kept, id)
	}
	c.Enabled = kept
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
	c.Enabled = nil // let the stored value (or the migration) decide
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

// LoadYouTubeKey returns "" when none is stored.
func LoadYouTubeKey() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(dir, youtubeKeyFile))
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	return strings.TrimSpace(string(data)), err
}

// SaveYouTubeKey writes the key 0600; an empty key removes the file.
func SaveYouTubeKey(key string) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	p := filepath.Join(dir, youtubeKeyFile)
	if strings.TrimSpace(key) == "" {
		err := os.Remove(p)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	return os.WriteFile(p, []byte(strings.TrimSpace(key)), 0o600)
}
