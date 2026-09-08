package local

import (
	"context"
	"encoding/binary"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/hwhang0917/vibe-music/internal/source"
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
	if all[0].ArtistID != "" { // WAV has no tags: nothing to link
		t.Fatalf("tagless file should have no artist page: %+v", all[0])
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
	mu   sync.Mutex
	m    map[string][]byte
	hits int
}

func (c *memCache) Get(path string, _ int64, _ time.Time) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	d, ok := c.m[path]
	if ok {
		c.hits++
	}
	return d, ok
}
func (c *memCache) Put(path string, _ int64, _ time.Time, d []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[path] = d
}
func (c *memCache) Prune(time.Time) {}

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

func TestBrowse(t *testing.T) {
	mk := func(id, title, artist, album string, year int) source.Track {
		tr := source.Track{ID: id, Source: "local", Title: title, Artist: artist, Album: album, Year: year}
		browseIDs(&tr)
		return tr
	}
	s := New(nil)
	s.tracks = []source.Track{
		mk("1", "Airbag", "Radiohead", "OK Computer", 1997),
		mk("2", "Idioteque", "radiohead ", "Kid A", 2000), // case/space-insensitive artist
		mk("3", "Karma Police", "Radiohead", "OK Computer", 1997),
		mk("4", "Hey", "Pixies", "Doolittle", 1989),
		mk("5", "untagged", "", "", 0),
	}
	if s.tracks[4].ArtistID != "" || s.tracks[4].AlbumID != "" {
		t.Fatal("no artist tag: no browse keys")
	}
	a, err := s.Artist(context.Background(), s.tracks[0].ArtistID)
	if err != nil || a.Name != "Radiohead" || len(a.Tracks) != 3 || len(a.Albums) != 2 || a.Albums[0].Name != "OK Computer" || a.Albums[1].Year != 2000 {
		t.Fatalf("artist: %+v %v", a, err)
	}
	al, err := s.Album(context.Background(), s.tracks[0].AlbumID)
	if err != nil || al.Name != "OK Computer" || al.Artist != "Radiohead" || al.Year != 1997 || len(al.Tracks) != 2 || al.ArtistID != a.ID {
		t.Fatalf("album: %+v %v", al, err)
	}
	if _, err := s.Album(context.Background(), "nope"); err != source.ErrNotFound {
		t.Fatalf("unknown: %v", err)
	}
}

func TestFolders(t *testing.T) {
	sep := string(filepath.Separator)
	root := filepath.Join("music")
	paths := []string{
		filepath.Join(root, "Pixies", "Doolittle", "01 Debaser.mp3"),
		filepath.Join(root, "Radiohead", "Kid A", "01 Everything.mp3"),
		filepath.Join(root, "Radiohead", "OK Computer", "01 Airbag.mp3"),
		filepath.Join(root, "Radiohead", "OK Computer", "02 Paranoid.mp3"),
		filepath.Join(root, "loose.mp3"),
	}
	s := New([]string{root + sep}) // trailing separator must not matter
	for i, p := range paths {
		s.tracks = append(s.tracks, source.Track{ID: shortID(p), Title: filepath.Base(p)})
		_ = i
	}
	s.dirs = buildDirs(s.folders, paths)

	top, err := s.Folder(context.Background(), "")
	if err != nil || top.Count != 5 || len(top.Folders) != 1 || top.Folders[0].Name != "music" || len(top.Path) != 0 {
		t.Fatalf("top: %+v %v", top, err)
	}
	music, err := s.Folder(context.Background(), top.Folders[0].ID)
	if err != nil || len(music.Folders) != 2 || music.Folders[0].Name != "Pixies" || music.Folders[1].Count != 3 || len(music.Tracks) != 1 || music.Tracks[0].Title != "loose.mp3" {
		t.Fatalf("music: %+v %v", music, err)
	}
	if len(music.Path) != 1 || music.Path[0].ID != "" {
		t.Fatalf("path of a root folder: %+v", music.Path)
	}
	rh, _ := s.Folder(context.Background(), music.Folders[1].ID)
	ok, err := s.Folder(context.Background(), rh.Folders[1].ID)
	if err != nil || ok.Name != "OK Computer" || len(ok.Tracks) != 2 || ok.Tracks[1].Title != "02 Paranoid.mp3" || len(ok.Folders) != 0 {
		t.Fatalf("ok computer: %+v %v", ok, err)
	}
	if names := []string{ok.Path[0].Name, ok.Path[1].Name, ok.Path[2].Name}; len(ok.Path) != 3 || names[1] != "music" || names[2] != "Radiohead" {
		t.Fatalf("breadcrumb: %v", names)
	}
	if _, err := s.Folder(context.Background(), "nope"); err != source.ErrNotFound {
		t.Fatalf("unknown: %v", err)
	}
}
