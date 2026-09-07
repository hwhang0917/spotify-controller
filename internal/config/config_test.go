package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCreatesDefaultThenRoundTrips(t *testing.T) {
	p := filepath.Join(t.TempDir(), "sub", "config.json")
	t.Setenv(EnvPath, p)

	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.Port != DefaultPort || c.ActiveSource != "local" {
		t.Fatalf("defaults: %+v", c)
	}
	if fi, err := os.Stat(p); err != nil || fi.Mode().Perm() != 0o600 {
		t.Fatalf("stat: %v %v", fi, err)
	}

	c.Local.Folders = []string{"/music"}
	c.Spotify.ClientID = "abc"
	if err := Save(c); err != nil {
		t.Fatal(err)
	}
	c2, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c2.Local.Folders[0] != "/music" || c2.Spotify.ClientID != "abc" || c2.SkipRatio != DefaultSkipRatio {
		t.Fatalf("round trip: %+v", c2)
	}
}
