// Package source defines the plugin contract every music backend implements.
// The core owns the queue and tells a Source "play this track now"; a Source
// never manages a queue of its own.
package source

import (
	"context"
	"errors"
	"strconv"
	"time"
)

// Track is one playable item as shown to guests.
type Track struct {
	// Source is the ID of the source that can play this track ("local", "spotify", "youtube").
	Source string `json:"source"`
	// ID is source-scoped and URL-safe (spotify: track ID; local: short hash of path).
	ID       string        `json:"id"`
	Title    string        `json:"title"`
	Artist   string        `json:"artist"`
	Album    string        `json:"album"`
	Genre    string        `json:"genre,omitempty"`
	Year     int           `json:"year,omitempty"`
	Duration time.Duration `json:"duration"` // 0 when unknown until played
	// ArtworkURL is absolute (Spotify CDN) or app-relative ("/api/artwork/{id}").
	ArtworkURL string `json:"artworkUrl,omitempty"`
	// ExternalURL links back to the provider (Spotify attribution requirement). Empty for local.
	ExternalURL string `json:"externalUrl,omitempty"`
	// ArtistID and AlbumID are browse keys for Browser sources; empty means
	// the guest UI shows plain text instead of a link.
	ArtistID string `json:"artistId,omitempty"`
	AlbumID  string `json:"albumId,omitempty"`
}

// Playback is a snapshot of what the source is doing right now.
type Playback struct {
	Track    *Track        `json:"track"` // nil when nothing is loaded
	Playing  bool          `json:"playing"`
	Position time.Duration `json:"position"`
	// Ended is the source's judgement that the last Play() has finished.
	Ended bool `json:"-"`
	// At is when the snapshot was taken; clients interpolate Position from it.
	At time.Time `json:"at"`
}

// Source is a music backend. Exactly one is active at a time.
type Source interface {
	ID() string   // "local", "spotify"
	Name() string // "Local files", "Spotify"

	// Activate makes this the live backend (scan library, verify token, ...).
	Activate(ctx context.Context) error
	// Deactivate stops output and releases resources. Never leaves audio running.
	Deactivate(ctx context.Context) error

	Search(ctx context.Context, query string, limit int) ([]Track, error)

	// Play starts trackID immediately, replacing whatever is playing.
	Play(ctx context.Context, trackID string) error
	Pause(ctx context.Context) error
	Resume(ctx context.Context) error
	Stop(ctx context.Context) error
	SetVolume(ctx context.Context, percent int) error
	// Seek moves playback of the current track to pos.
	Seek(ctx context.Context, pos time.Duration) error

	Status(ctx context.Context) (Playback, error)
}

// Charter is optional: sources that can list a popularity chart (e.g. YouTube
// "most popular" in a region) implement it. region is an ISO 3166-1 alpha-2
// code such as "KR"; sources may ignore it.
type Charter interface {
	Chart(ctx context.Context, region string, limit int) ([]Track, error)
}

// Browser is optional: sources that can show an artist or an album page.
type Browser interface {
	Artist(ctx context.Context, id string) (Artist, error)
	Album(ctx context.Context, id string) (Album, error)
}

// Artist is an artist page: its tracks (top or all) and albums (Tracks nil).
type Artist struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	ArtworkURL string  `json:"artworkUrl,omitempty"`
	Tracks     []Track `json:"tracks"`
	Albums     []Album `json:"albums"`
}

// Album is an album page, or an album reference inside an Artist (no Tracks).
type Album struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Artist     string  `json:"artist"`
	ArtistID   string  `json:"artistId,omitempty"`
	Year       int     `json:"year,omitempty"`
	ArtworkURL string  `json:"artworkUrl,omitempty"`
	Tracks     []Track `json:"tracks,omitempty"`
}

// Explorer is optional: sources laid out as folders (local files) let guests
// walk them like a file manager. The empty ID is the top level.
type Explorer interface {
	Folder(ctx context.Context, id string) (Folder, error)
}

// Folder is one directory: where it sits (Path, top level first), its
// subfolders (ID, Name, Count only) and the tracks directly inside it.
type Folder struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Count   int      `json:"count"` // tracks in this folder and below
	Path    []Folder `json:"path,omitempty"`
	Folders []Folder `json:"folders,omitempty"`
	Tracks  []Track  `json:"tracks,omitempty"`
}

// ErrNotFound: no such artist/album/folder at this source (or the source has no pages).
var ErrNotFound = &CodedError{Kind: "not_found", Msg: "not found"}

// YearOf reads the year from a date string such as "2019-04-01" or
// "2019-04-01T10:00:00Z"; 0 when it has none.
func YearOf(date string) int {
	if len(date) < 4 {
		return 0
	}
	y, err := strconv.Atoi(date[:4])
	if err != nil {
		return 0
	}
	return y
}

// ArtworkProvider is optional. Sources whose artwork is not a public URL
// (local files) serve it through the guest API.
type ArtworkProvider interface {
	Artwork(ctx context.Context, trackID string) (data []byte, mime string, err error)
}

// Coder is implemented by errors that carry a stable code for the UIs to
// translate. Unknown errors are shown verbatim.
type Coder interface{ Code() string }

// CodedError is a sentinel with a UI code.
type CodedError struct {
	Kind string // the code, e.g. "youtube_quota"
	Msg  string
}

func (e *CodedError) Error() string { return e.Msg }
func (e *CodedError) Code() string  { return e.Kind }

// ErrorCode returns the code of err (or a wrapped error), else fallback.
func ErrorCode(err error, fallback string) string {
	var c Coder
	if errors.As(err, &c) {
		return c.Code()
	}
	return fallback
}
