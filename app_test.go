package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/hwhang0917/vibe-music/internal/player"
	"github.com/hwhang0917/vibe-music/internal/server"
	"github.com/hwhang0917/vibe-music/internal/source"
	"github.com/hwhang0917/vibe-music/internal/source/fake"
	"github.com/hwhang0917/vibe-music/internal/store"
	"github.com/hwhang0917/vibe-music/web"
)

func TestResetClearsGuestTop(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "x.db"))
	if err != nil {
		t.Fatal(err)
	}
	f := fake.New(source.Track{ID: "a", Title: "A"})
	p := player.New(player.Options{Sources: []source.Source{f}})
	_ = p.SetEnabled(t.Context(), "fake", true)
	a := &App{db: db, player: p}
	_ = db.RecordPlay("fake", "a", []byte(`{"id":"a","title":"A","source":"fake"}`), time.Now())
	h := server.NewHandler(web.Dist, p, server.NewGuests(db, false, func() {}), a.topTracks, func(error) {})
	ts := httptest.NewServer(h)
	defer ts.Close()
	get := func() int {
		res, _ := http.Get(ts.URL + "/api/top?source=fake")
		var out []source.Track
		_ = json.NewDecoder(res.Body).Decode(&out)
		return len(out)
	}
	if n := get(); n != 1 {
		t.Fatalf("before reset: %d", n)
	}
	if err := a.ResetPlayHistory(); err != nil {
		t.Fatal(err)
	}
	if n := get(); n != 0 {
		t.Fatalf("after reset: %d", n)
	}
}
