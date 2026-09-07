// Package source defines the plugin contract every music backend implements.
// The core owns the queue and tells a Source "play this track now"; a Source
// never manages a queue of its own.
package source

import (
	"context"
	"errors"
	"time"
)

// Track is one playable item as shown to guests.
type Track struct {
	// ID is source-scoped and URL-safe (spotify: track ID; local: short hash of path).
	ID       string        `json:"id"`
	Title    string        `json:"title"`
	Artist   string        `json:"artist"`
	Album    string        `json:"album"`
	Duration time.Duration `json:"duration"` // 0 when unknown until played
	// ArtworkURL is absolute (Spotify CDN) or app-relative ("/api/artwork/{id}").
	ArtworkURL string `json:"artworkUrl,omitempty"`
	// ExternalURL links back to the provider (Spotify attribution requirement). Empty for local.
	ExternalURL string `json:"externalUrl,omitempty"`
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
