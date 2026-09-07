// Package local plays audio files from folders on the host PC through the
// default output device. Pure Go: beep decoders + oto via speaker.
package local

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
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

// entry is the per-track file info. Artwork is read from the file on demand
// rather than held for every track.
type entry struct {
	path       string
	hasPicture bool
}

type Source struct {
	mu       sync.Mutex
	folders  []string
	tracks   []source.Track
	byID     map[string]*entry
	cache    Cache
	progress func(done, total int)

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

// Cache remembers what a file indexed to (tags and duration) keyed by path,
// size and mtime, so a rescan only opens files that are new or changed.
// Reading an MP3's duration means reading the whole file, so this is what
// keeps startup quick for a large library.
type Cache interface {
	Get(path string, size int64, mtime time.Time) ([]byte, bool)
	Put(path string, size int64, mtime time.Time, data []byte)
	Prune(before time.Time) // forget files not seen since the scan started
}

// SetCache installs the index cache (nil disables caching).
func (s *Source) SetCache(c Cache) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache = c
}

// SetProgress installs a callback that receives (done, total) while a rescan runs.
func (s *Source) SetProgress(fn func(done, total int)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.progress = fn
}

// indexed is what the cache stores per file.
type indexed struct {
	Track      source.Track `json:"track"`
	HasPicture bool         `json:"hasPicture"`
}

// scanWorkers bounds concurrent file reads; the work is I/O, not CPU.
const scanWorkers = 4

// Rescan walks the folders and rebuilds the index: tags, duration, and
// whether there is embedded artwork. Returns the track count.
func (s *Source) Rescan() (int, error) {
	s.mu.Lock()
	folders, cache, progress := s.folders, s.cache, s.progress
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
	if progress != nil {
		progress(0, len(paths))
	}

	started := time.Now()
	results := make([]indexed, len(paths))
	var done atomic.Int64
	var wg sync.WaitGroup
	sem := make(chan struct{}, scanWorkers)
	for i, p := range paths {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, p string) {
			defer func() { <-sem; wg.Done() }()
			results[i] = index(p, cache)
			if progress != nil {
				progress(int(done.Add(1)), len(paths))
			}
		}(i, p)
	}
	wg.Wait()
	if cache != nil {
		cache.Prune(started)
	}

	tracks := make([]source.Track, 0, len(paths))
	byID := make(map[string]*entry, len(paths))
	for i, p := range paths {
		id := trackID(p)
		t := results[i].Track
		t.ID, t.Source = id, "local"
		if results[i].HasPicture {
			t.ArtworkURL = "/api/artwork/local/" + id
		}
		tracks = append(tracks, t)
		byID[id] = &entry{path: p, hasPicture: results[i].HasPicture}
	}

	s.mu.Lock()
	s.tracks, s.byID = tracks, byID
	s.mu.Unlock()
	return len(tracks), nil
}

// index returns the cached record for a file or reads tags and duration.
func index(path string, cache Cache) indexed {
	st, err := os.Stat(path)
	if err != nil {
		t, _ := readTags(path)
		return indexed{Track: t}
	}
	if cache != nil {
		if data, ok := cache.Get(path, st.Size(), st.ModTime()); ok {
			var ix indexed
			if json.Unmarshal(data, &ix) == nil {
				return ix
			}
		}
	}
	t, pic := readTags(path)
	t.Duration = duration(path)
	ix := indexed{Track: t, HasPicture: pic != nil}
	if cache != nil {
		if data, err := json.Marshal(ix); err == nil {
			cache.Put(path, st.Size(), st.ModTime(), data)
		}
	}
	return ix
}

// duration opens the file with its decoder and asks for the sample count.
// FLAC, WAV and OGG answer from headers; MP3 has to read every frame header.
func duration(path string) time.Duration {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()
	stream, format, err := supported[strings.ToLower(filepath.Ext(path))](f)
	if err != nil {
		return 0
	}
	defer stream.Close()
	return format.SampleRate.D(stream.Len())
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
	t.Artist, t.Album, t.Genre, t.Year = m.Artist(), m.Album(), m.Genre(), m.Year()
	return t, m.Picture()
}

func (s *Source) Search(_ context.Context, q string, limit int) ([]source.Track, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	q = strings.ToLower(strings.TrimSpace(q))
	var out []source.Track
	for _, t := range s.tracks {
		if q == "" || strings.Contains(strings.ToLower(fmt.Sprintf("%s %s %s %s %d", t.Title, t.Artist, t.Album, t.Genre, t.Year)), q) {
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
	if !ok || !e.hasPicture {
		return nil, "", errors.New("no artwork")
	}
	_, pic := readTags(e.path)
	if pic == nil {
		return nil, "", errors.New("no artwork")
	}
	return pic.Data, pic.MIMEType, nil
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
