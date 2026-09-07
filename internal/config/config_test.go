package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/oauth2"
)

type mapKV map[string][]byte

func (m mapKV) Get(k string) ([]byte, error) {
	v, ok := m[k]
	if !ok {
		return nil, os.ErrNotExist
	}
	return v, nil
}
func (m mapKV) Set(k string, v []byte) error { m[k] = v; return nil }

func TestLoadDefaultsThenRoundTrips(t *testing.T) {
	kv := mapKV{}
	c, err := Load(kv)
	if err != nil {
		t.Fatal(err)
	}
	if c.Port != DefaultPort || !c.IsEnabled("local") || c.Local.Folders == nil || c.InviteOnly || c.Spotify.CallbackPort != DefaultSpotifyCallbackPort {
		t.Fatalf("defaults: %+v", c)
	}
	if _, ok := kv[settingsKey]; !ok {
		t.Fatal("defaults should be saved")
	}

	c.Local.Folders = []string{"/music"}
	c.Spotify.ClientID = "abc"
	c.InviteOnly = true
	if err := Save(kv, c); err != nil {
		t.Fatal(err)
	}
	c2, err := Load(kv)
	if err != nil || c2.Local.Folders[0] != "/music" || c2.Spotify.ClientID != "abc" || !c2.InviteOnly || c2.SkipRatio != DefaultSkipRatio {
		t.Fatalf("round trip: %+v %v", c2, err)
	}
	// null slices in stored JSON normalize to []
	kv[settingsKey] = []byte(`{"local":{"folders":null}}`)
	if c3, _ := Load(kv); c3.Local.Folders == nil {
		t.Fatal("null folders should normalize to []")
	}
}

func TestTokenFile(t *testing.T) {
	t.Setenv(EnvDir, filepath.Join(t.TempDir(), "data"))
	if tok, err := LoadToken(); tok != nil || err != nil {
		t.Fatalf("missing token: %v %v", tok, err)
	}
	if err := SaveToken(&oauth2.Token{AccessToken: "a", RefreshToken: "r"}); err != nil {
		t.Fatal(err)
	}
	dir, _ := Dir()
	if fi, err := os.Stat(filepath.Join(dir, tokenFile)); err != nil || fi.Mode().Perm() != 0o600 {
		t.Fatalf("token perms: %v %v", fi, err)
	}
	tok, err := LoadToken()
	if err != nil || tok.RefreshToken != "r" {
		t.Fatalf("load: %+v %v", tok, err)
	}
	if err := SaveToken(nil); err != nil {
		t.Fatal(err)
	}
	if tok, _ := LoadToken(); tok != nil {
		t.Fatal("token should be removed")
	}
	if err := SaveToken(nil); err != nil {
		t.Fatal("removing twice should be fine")
	}
}

func TestYouTubeKeyFile(t *testing.T) {
	t.Setenv(EnvDir, filepath.Join(t.TempDir(), "data"))
	if k, err := LoadYouTubeKey(); k != "" || err != nil {
		t.Fatalf("missing: %q %v", k, err)
	}
	if err := SaveYouTubeKey("  AIza-test \n"); err != nil {
		t.Fatal(err)
	}
	if k, _ := LoadYouTubeKey(); k != "AIza-test" {
		t.Fatalf("trimmed: %q", k)
	}
	if err := SaveYouTubeKey(""); err != nil {
		t.Fatal(err)
	}
	if k, _ := LoadYouTubeKey(); k != "" {
		t.Fatal("removed")
	}
}

func TestEnabledSourcesAndMigration(t *testing.T) {
	c := Default()
	if !c.IsEnabled("local") || c.IsEnabled("spotify") {
		t.Fatalf("defaults: %+v", c)
	}
	c.SetEnabled("spotify", true)
	c.SetEnabled("spotify", true)
	c.SetEnabled("local", false)
	if c.IsEnabled("local") || !c.IsEnabled("spotify") || len(c.Enabled) != 1 {
		t.Fatalf("set: %v", c.Enabled)
	}
	// settings written before Enabled existed: activeSource becomes the enabled one
	kv := mapKV{settingsKey: []byte(`{"activeSource":"spotify"}`)}
	m, err := Load(kv)
	if err != nil || !m.IsEnabled("spotify") || m.IsEnabled("local") || m.ActiveSource != "" {
		t.Fatalf("migration: %+v %v", m, err)
	}
	kv = mapKV{settingsKey: []byte(`{"enabled":[]}`)}
	if m, _ := Load(kv); len(m.Enabled) != 0 {
		t.Fatalf("explicit empty stays empty: %+v", m)
	}
}

// Secrets are sealed on disk; a plaintext file from an older build still loads
// and is sealed on that first read.
func TestSecretsSealedAndLegacyMigrates(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(EnvDir, dir)
	if err := SaveYouTubeKey("AIzaTEST"); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(dir, youtubeKeyFile))
	if !strings.HasPrefix(string(raw), sealedMagic) || strings.Contains(string(raw), "AIzaTEST") {
		t.Fatalf("not sealed: %q", raw)
	}
	if k, err := LoadYouTubeKey(); err != nil || k != "AIzaTEST" {
		t.Fatalf("round trip: %q %v", k, err)
	}
	// legacy plaintext
	os.WriteFile(filepath.Join(dir, youtubeKeyFile), []byte("AIzaOLD\n"), 0o600)
	if k, err := LoadYouTubeKey(); err != nil || k != "AIzaOLD" {
		t.Fatalf("legacy: %q %v", k, err)
	}
	raw, _ = os.ReadFile(filepath.Join(dir, youtubeKeyFile))
	if !strings.HasPrefix(string(raw), sealedMagic) {
		t.Fatalf("legacy file not migrated: %q", raw)
	}
	if err := SaveYouTubeKey(""); err != nil {
		t.Fatal(err)
	}
	if k, _ := LoadYouTubeKey(); k != "" {
		t.Fatalf("removed key still loads: %q", k)
	}
}
