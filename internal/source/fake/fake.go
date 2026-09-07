// Package fake is an in-memory Source for tests: a fixed library, a
// scriptable Status, and a log of calls.
package fake

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/hwhang0917/vibe-music/internal/source"
)

type Source struct {
	mu       sync.Mutex
	id       string
	Library  []source.Track
	Calls    []string // "activate", "play:<id>", "pause", ...
	Current  *source.Track
	Playing  bool
	Ended    bool
	Position time.Duration
	Volume   int
	Err      error // returned from every method when set
}

func New(tracks ...source.Track) *Source {
	f := &Source{Volume: 100, id: "fake"}
	for _, t := range tracks {
		t.Source = f.id
		f.Library = append(f.Library, t)
	}
	return f
}

// WithID gives the fake a different source ID (for multi-source tests).
func (f *Source) WithID(id string) *Source {
	f.id = id
	for i := range f.Library {
		f.Library[i].Source = id
	}
	return f
}

func (f *Source) ID() string   { return f.id }
func (f *Source) Name() string { return "Fake" }

func (f *Source) log(s string) error {
	f.Calls = append(f.Calls, s)
	return f.Err
}

func (f *Source) Activate(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.log("activate")
}

func (f *Source) Deactivate(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Current, f.Playing = nil, false
	return f.log("deactivate")
}

func (f *Source) Search(_ context.Context, q string, limit int) ([]source.Track, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.log("search:" + q); err != nil {
		return nil, err
	}
	var out []source.Track
	for _, t := range f.Library {
		if strings.Contains(strings.ToLower(t.Title+" "+t.Artist), strings.ToLower(q)) {
			out = append(out, t)
			if len(out) == limit {
				break
			}
		}
	}
	return out, nil
}

func (f *Source) Play(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.log("play:" + id); err != nil {
		return err
	}
	for i := range f.Library {
		if f.Library[i].ID == id {
			t := f.Library[i]
			f.Current, f.Playing, f.Ended = &t, true, false
			return nil
		}
	}
	f.Current = &source.Track{ID: id, Source: f.id}
	f.Playing, f.Ended = true, false
	return nil
}

func (f *Source) Pause(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Playing = false
	return f.log("pause")
}

func (f *Source) Resume(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Playing = f.Current != nil
	return f.log("resume")
}

func (f *Source) Stop(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Current, f.Playing = nil, false
	return f.log("stop")
}

func (f *Source) SetVolume(_ context.Context, pct int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Volume = pct
	return f.log("volume")
}

func (f *Source) Seek(_ context.Context, pos time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Position = pos
	return f.log("seek")
}

func (f *Source) Status(context.Context) (source.Playback, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.Err != nil {
		return source.Playback{}, f.Err
	}
	return source.Playback{Track: f.Current, Playing: f.Playing, Ended: f.Ended, Position: f.Position, At: time.Now()}, nil
}

// FinishTrack simulates the current track reaching its end.
func (f *Source) FinishTrack() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Playing, f.Ended = false, true
}

// Played returns the IDs passed to Play, in order.
func (f *Source) Played() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var ids []string
	for _, c := range f.Calls {
		if strings.HasPrefix(c, "play:") {
			ids = append(ids, strings.TrimPrefix(c, "play:"))
		}
	}
	return ids
}

// Artist implements source.Browser over Library by ArtistID.
func (f *Source) Artist(_ context.Context, id string) (source.Artist, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.log("artist:" + id); err != nil {
		return source.Artist{}, err
	}
	a := source.Artist{ID: id}
	seen := map[string]bool{}
	for _, t := range f.Library {
		if t.ArtistID != id {
			continue
		}
		a.Name = t.Artist
		a.Tracks = append(a.Tracks, t)
		if t.AlbumID != "" && !seen[t.AlbumID] {
			seen[t.AlbumID] = true
			a.Albums = append(a.Albums, source.Album{ID: t.AlbumID, Name: t.Album, Artist: t.Artist, ArtistID: id, Year: t.Year})
		}
	}
	if len(a.Tracks) == 0 {
		return source.Artist{}, source.ErrNotFound
	}
	return a, nil
}

// Album implements source.Browser over Library by AlbumID.
func (f *Source) Album(_ context.Context, id string) (source.Album, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.log("album:" + id); err != nil {
		return source.Album{}, err
	}
	a := source.Album{ID: id}
	for _, t := range f.Library {
		if t.AlbumID == id {
			a.Name, a.Artist, a.ArtistID, a.Year = t.Album, t.Artist, t.ArtistID, t.Year
			a.Tracks = append(a.Tracks, t)
		}
	}
	if len(a.Tracks) == 0 {
		return source.Album{}, source.ErrNotFound
	}
	return a, nil
}
