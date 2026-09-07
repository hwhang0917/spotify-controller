// Package local plays audio files from folders on the host PC through the
// default output device. Pure Go: beep decoders + oto via speaker.
package local

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dhowden/tag"
	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/effects"
	"github.com/gopxl/beep/v2/flac"
	"github.com/gopxl/beep/v2/mp3"
	"github.com/gopxl/beep/v2/speaker"
	"github.com/gopxl/beep/v2/vorbis"
	"github.com/gopxl/beep/v2/wav"

	"github.com/hwhang0917/vibe-music/internal/source"
)

const (
	// speaker.Init runs once per process at a fixed rate; every decoded stream
	// is resampled to it.
	outputRate      = beep.SampleRate(48000)
	speakerBuffer   = 100 * time.Millisecond
	resampleQuality = 4
	idLen           = 12
)

var supported = map[string]func(*os.File) (beep.StreamSeekCloser, beep.Format, error){
	".mp3":  func(f *os.File) (beep.StreamSeekCloser, beep.Format, error) { return mp3.Decode(f) },
	".wav":  func(f *os.File) (beep.StreamSeekCloser, beep.Format, error) { return wav.Decode(f) },
	".flac": func(f *os.File) (beep.StreamSeekCloser, beep.Format, error) { return flac.Decode(f) },
	".ogg":  func(f *os.File) (beep.StreamSeekCloser, beep.Format, error) { return vorbis.Decode(f) },
}

var (
	speakerOnce sync.Once
	speakerErr  error
)

// ErrAudioOutput wraps speaker initialisation failures.
var ErrAudioOutput = errors.New("audio output")

type entry struct {
	path    string
	picture *tag.Picture
}

type Source struct {
	mu      sync.Mutex
	folders []string
	tracks  []source.Track
	byID    map[string]*entry

	// playback (guarded by mu; sample positions additionally by speaker.Lock)
	stream  beep.StreamSeekCloser
	format  beep.Format
	ctrl    *beep.Ctrl
	gain    *effects.Gain
	current *source.Track
	ended   atomic.Bool // set from the speaker goroutine; never take mu there
	volume  int
}

func New(folders []string) *Source {
	return &Source{folders: folders, byID: map[string]*entry{}, volume: 100}
}

func (s *Source) ID() string   { return "local" }
func (s *Source) Name() string { return "Local files" }

// SetFolders replaces the library roots. Call Rescan afterwards.
func (s *Source) SetFolders(folders []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.folders = folders
}

func (s *Source) Activate(ctx context.Context) error {
	speakerOnce.Do(func() {
		speakerErr = speaker.Init(outputRate, outputRate.N(speakerBuffer))
	})
	if speakerErr != nil {
		return fmt.Errorf("%w: %v", ErrAudioOutput, speakerErr)
	}
	_, err := s.Rescan()
	return err
}

func (s *Source) Deactivate(ctx context.Context) error {
	return s.Stop(ctx)
}

// Rescan walks the folders and rebuilds the index. Returns the track count.
// ponytail: synchronous and blocks the caller; fine for a few thousand files.
func (s *Source) Rescan() (int, error) {
	s.mu.Lock()
	folders := s.folders
	s.mu.Unlock()

	var paths []string
	for _, root := range folders {
		err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil // skip unreadable entries, keep walking
			}
			if _, ok := supported[strings.ToLower(filepath.Ext(p))]; ok {
				paths = append(paths, p)
			}
			return nil
		})
		if err != nil {
			return 0, err
		}
	}
	sort.Strings(paths)

	tracks := make([]source.Track, 0, len(paths))
	byID := make(map[string]*entry, len(paths))
	for _, p := range paths {
		id := trackID(p)
		t, pic := readTags(p)
		t.ID, t.Source = id, "local"
		if pic != nil {
			t.ArtworkURL = "/api/artwork/local/" + id
		}
		tracks = append(tracks, t)
		byID[id] = &entry{path: p, picture: pic}
	}

	s.mu.Lock()
	s.tracks, s.byID = tracks, byID
	s.mu.Unlock()
	return len(tracks), nil
}

func trackID(path string) string {
	sum := sha1.Sum([]byte(path))
	return hex.EncodeToString(sum[:])[:idLen]
}

// readTags returns what the file's tags say, falling back to the filename.
// dhowden/tag has no WAV support, so WAV always takes the fallback.
func readTags(path string) (source.Track, *tag.Picture) {
	t := source.Track{Title: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))}
	f, err := os.Open(path)
	if err != nil {
		return t, nil
	}
	defer f.Close()
	m, err := tag.ReadFrom(f)
	if err != nil {
		return t, nil
	}
	if m.Title() != "" {
		t.Title = m.Title()
	}
	t.Artist, t.Album = m.Artist(), m.Album()
	return t, m.Picture()
}

func (s *Source) Search(_ context.Context, q string, limit int) ([]source.Track, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	q = strings.ToLower(strings.TrimSpace(q))
	var out []source.Track
	for _, t := range s.tracks {
		if q == "" || strings.Contains(strings.ToLower(t.Title+" "+t.Artist+" "+t.Album), q) {
			out = append(out, t)
			if limit > 0 && len(out) == limit {
				break
			}
		}
	}
	return out, nil
}

func (s *Source) Artwork(_ context.Context, id string) ([]byte, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.byID[id]
	if !ok || e.picture == nil {
		return nil, "", errors.New("no artwork")
	}
	return e.picture.Data, e.picture.MIMEType, nil
}

func (s *Source) Play(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.byID[id]
	if !ok {
		return fmt.Errorf("unknown track %q", id)
	}
	f, err := os.Open(e.path)
	if err != nil {
		return err
	}
	stream, format, err := supported[strings.ToLower(filepath.Ext(e.path))](f)
	if err != nil {
		f.Close()
		return fmt.Errorf("decode %s: %w", filepath.Base(e.path), err)
	}

	s.stopLocked()

	var st beep.Streamer = stream
	if format.SampleRate != outputRate {
		st = beep.Resample(resampleQuality, format.SampleRate, outputRate, stream)
	}
	s.gain = &effects.Gain{Streamer: st, Gain: gainFor(s.volume)}
	s.ctrl = &beep.Ctrl{Streamer: s.gain}
	s.stream, s.format = stream, format
	s.ended.Store(false)

	t := s.trackLocked(id)
	t.Duration = format.SampleRate.D(stream.Len())
	s.current = &t

	speaker.Play(beep.Seq(s.ctrl, beep.Callback(func() { s.ended.Store(true) })))
	return nil
}

func (s *Source) trackLocked(id string) source.Track {
	for _, t := range s.tracks {
		if t.ID == id {
			return t
		}
	}
	return source.Track{ID: id, Source: "local"}
}

// gainFor maps 0..100% to beep's Gain, where output = sample * (1 + Gain).
func gainFor(pct int) float64 { return float64(pct)/100 - 1 }

func (s *Source) Pause(context.Context) error  { return s.setPaused(true) }
func (s *Source) Resume(context.Context) error { return s.setPaused(false) }

func (s *Source) setPaused(paused bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ctrl == nil {
		return nil
	}
	speaker.Lock()
	s.ctrl.Paused = paused
	speaker.Unlock()
	return nil
}

func (s *Source) Stop(context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopLocked()
	return nil
}

func (s *Source) stopLocked() {
	if s.stream != nil {
		speaker.Clear()
		s.stream.Close()
	}
	s.stream, s.ctrl, s.gain, s.current = nil, nil, nil, nil
	s.ended.Store(false)
}

func (s *Source) Seek(_ context.Context, pos time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stream == nil {
		return nil
	}
	n := max(0, min(s.stream.Len()-1, s.format.SampleRate.N(pos)))
	speaker.Lock()
	err := s.stream.Seek(n)
	speaker.Unlock()
	return err
}

func (s *Source) SetVolume(_ context.Context, pct int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.volume = pct
	if s.gain != nil {
		speaker.Lock()
		s.gain.Gain = gainFor(pct)
		speaker.Unlock()
	}
	return nil
}

func (s *Source) Status(context.Context) (source.Playback, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	pb := source.Playback{At: time.Now()}
	if s.current == nil {
		return pb, nil
	}
	if s.ended.Load() {
		pb.Track, pb.Ended = s.current, true
		return pb, nil
	}
	speaker.Lock()
	pos := s.stream.Position()
	paused := s.ctrl.Paused
	speaker.Unlock()
	pb.Track = s.current
	pb.Playing = !paused
	pb.Position = s.format.SampleRate.D(pos)
	return pb, nil
}
