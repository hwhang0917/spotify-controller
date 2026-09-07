package local

import (
	"context"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// writeWAV writes a valid 16-bit mono PCM file of n samples of silence.
func writeWAV(t *testing.T, path string, n int) {
	t.Helper()
	const rate = 44100
	data := make([]byte, n*2)
	buf := make([]byte, 0, 44+len(data))
	put32 := func(v uint32) { buf = binary.LittleEndian.AppendUint32(buf, v) }
	put16 := func(v uint16) { buf = binary.LittleEndian.AppendUint16(buf, v) }
	buf = append(buf, "RIFF"...)
	put32(uint32(36 + len(data)))
	buf = append(buf, "WAVEfmt "...)
	put32(16)
	put16(1) // PCM
	put16(1) // mono
	put32(rate)
	put32(rate * 2)
	put16(2)
	put16(16)
	buf = append(buf, "data"...)
	put32(uint32(len(data)))
	buf = append(buf, data...)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScanAndSearch(t *testing.T) {
	root := t.TempDir()
	writeWAV(t, filepath.Join(root, "Morning Song.wav"), 4410)
	writeWAV(t, filepath.Join(root, "nested", "evening-song.wav"), 4410)
	os.WriteFile(filepath.Join(root, "notes.txt"), []byte("x"), 0o644)

	s := New([]string{root})
	n, err := s.Rescan()
	if err != nil || n != 2 {
		t.Fatalf("rescan: n=%d err=%v", n, err)
	}

	all, _ := s.Search(context.Background(), "", 0)
	if len(all) != 2 || all[0].Title != "Morning Song" || all[1].Title != "evening-song" {
		t.Fatalf("index: %+v", all)
	}
	if all[0].Duration == 0 || all[0].Duration > 2*time.Second {
		t.Fatalf("duration should come from the scan: %v", all[0].Duration)
	}
	hits, _ := s.Search(context.Background(), "SONG", 1)
	if len(hits) != 1 {
		t.Fatalf("limit: %+v", hits)
	}
	none, _ := s.Search(context.Background(), "jazz", 0)
	if len(none) != 0 {
		t.Fatalf("expected no hits: %+v", none)
	}

	// IDs are stable across rescans and URL-safe
	id := all[0].ID
	s.Rescan()
	again, _ := s.Search(context.Background(), "Morning", 0)
	if again[0].ID != id || len(id) != idLen {
		t.Fatalf("id changed: %s -> %s", id, again[0].ID)
	}

	if _, _, err := s.Artwork(context.Background(), id); err == nil {
		t.Fatal("WAV has no artwork; expected error")
	}
	if err := s.Play(context.Background(), "nope"); err == nil {
		t.Fatal("unknown track should error")
	}
}

func TestGainMapping(t *testing.T) {
	if gainFor(100) != 0 || gainFor(0) != -1 || gainFor(50) != -0.5 {
		t.Fatalf("gain: %v %v %v", gainFor(100), gainFor(0), gainFor(50))
	}
}

type memCache struct {
	m    map[string][]byte
	hits int
}

func (c *memCache) Get(path string, _ int64, _ time.Time) ([]byte, bool) {
	d, ok := c.m[path]
	if ok {
		c.hits++
	}
	return d, ok
}
func (c *memCache) Put(path string, _ int64, _ time.Time, d []byte) { c.m[path] = d }
func (c *memCache) Prune(time.Time)                                 {}

// A second scan answers from the cache and reports progress to the end.
func TestRescanUsesCacheAndReportsProgress(t *testing.T) {
	root := t.TempDir()
	writeWAV(t, filepath.Join(root, "a.wav"), 4410)
	writeWAV(t, filepath.Join(root, "b.wav"), 4410)
	s := New([]string{root})
	c := &memCache{m: map[string][]byte{}}
	s.SetCache(c)
	var last [2]int
	s.SetProgress(func(done, total int) { last = [2]int{done, total} })
	if _, err := s.Rescan(); err != nil || c.hits != 0 || len(c.m) != 2 {
		t.Fatalf("first scan: hits=%d cached=%d err=%v", c.hits, len(c.m), err)
	}
	if _, err := s.Rescan(); err != nil || c.hits != 2 {
		t.Fatalf("second scan should hit the cache: hits=%d err=%v", c.hits, err)
	}
	if last != [2]int{2, 2} {
		t.Fatalf("progress: %v", last)
	}
	all, _ := s.Search(context.Background(), "", 0)
	if len(all) != 2 || all[0].Duration == 0 {
		t.Fatalf("cached index lost data: %+v", all)
	}
}
