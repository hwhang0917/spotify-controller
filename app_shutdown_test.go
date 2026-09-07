package main

import (
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/hwhang0917/vibe-music/internal/player"
	"github.com/hwhang0917/vibe-music/internal/server"
	"github.com/hwhang0917/vibe-music/internal/source"
	"github.com/hwhang0917/vibe-music/internal/source/fake"
	"github.com/hwhang0917/vibe-music/internal/store"
)

// An open SSE stream must not hold StopServer until shutdownTimeout expires.
func TestStopServerEndsEventStreams(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "x.db"))
	if err != nil {
		t.Fatal(err)
	}
	p := player.New(player.Options{Sources: []source.Source{fake.New(source.Track{ID: "a"})}})
	a := &App{db: db, player: p, guests: server.NewGuests(db, false, func() {})}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	if _, err := a.StartServer(port); err != nil {
		t.Fatal(err)
	}
	res, err := http.Get("http://127.0.0.1:" + strconv.Itoa(port) + "/api/events")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("events: %d", res.StatusCode)
	}
	done := make(chan struct{})
	go func() { _, _ = http.MaxBytesReader(nil, res.Body, 1<<20).Read(make([]byte, 64)); close(done) }()
	start := time.Now()
	if err := a.StopServer(); err != nil {
		t.Fatal(err)
	}
	if d := time.Since(start); d > time.Second {
		t.Fatalf("StopServer took %v with an open event stream", d)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("event stream still open after StopServer")
	}
}
