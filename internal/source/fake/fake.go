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
	mu      sync.Mutex
	Library []source.Track
	Calls   []string // "activate", "play:<id>", "pause", ...
	Current *source.Track
	Playing bool
	Ended   bool
	Volume  int
	Err     error // returned from every method when set
}

func New(tracks ...source.Track) *Source { return &Source{Library: tracks, Volume: 100} }

func (f *Source) ID() string   { return "fake" }
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
	f.Current = &source.Track{ID: id}
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

func (f *Source) Status(context.Context) (source.Playback, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.Err != nil {
		return source.Playback{}, f.Err
	}
	return source.Playback{Track: f.Current, Playing: f.Playing, Ended: f.Ended, At: time.Now()}, nil
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
