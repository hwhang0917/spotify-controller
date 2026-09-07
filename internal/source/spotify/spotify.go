// Package spotify is a remote control for the Spotify desktop client on the
// host PC via the Web API (Connect). It never streams audio itself, so it is a
// non-streaming app under Spotify's developer policy. The host brings their
// own Client ID; auth is PKCE with a loopback redirect; the token never leaves
// this process.
package spotify

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2"

	"github.com/hwhang0917/vibe-music/internal/source"
)

const (
	connectTimeout  = 3 * time.Minute
	callbackPath    = "/callback"
	minArtworkPx    = 300
	trackURIPrefix  = "spotify:track:"
	externalURLKey  = "spotify"
	noDeviceMessage = "no Spotify device found: open Spotify on this PC and press play once"
)

var scopes = []string{
	spotifyauth.ScopeUserReadPlaybackState,
	spotifyauth.ScopeUserModifyPlaybackState,
	spotifyauth.ScopeUserReadCurrentlyPlaying,
}

// Sentinel errors the admin UI maps to translated messages.
var (
	ErrNotConnected = errors.New("spotify: not connected")
	ErrNoClientID   = errors.New("spotify: client ID is empty")
	ErrNoDevice     = errors.New(noDeviceMessage)
	ErrLoginTimeout = errors.New("spotify: login not completed")
)

type Options struct {
	ClientID string
	DeviceID string
	Token    *oauth2.Token
	// SaveToken is called whenever the token changes (PKCE rotates refresh
	// tokens on every refresh; losing one kills the session within an hour).
	SaveToken func(*oauth2.Token)
	// OpenBrowser opens the consent URL (Wails runtime in the app).
	OpenBrowser func(url string) error
}

type Device struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Active bool   `json:"active"`
}

type Source struct {
	mu     sync.Mutex
	opts   Options
	client *spotify.Client

	current spotify.ID // track we last asked to play; "" when idle
	armed   bool       // we have seen `current` actually playing
}

func New(opts Options) *Source {
	s := &Source{opts: opts}
	if opts.Token != nil && opts.ClientID != "" {
		s.client = s.newClient(opts.Token)
	}
	return s
}

func (s *Source) ID() string   { return "spotify" }
func (s *Source) Name() string { return "Spotify" }

func (s *Source) oauthConfig(redirect string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:    s.opts.ClientID,
		RedirectURL: redirect,
		Scopes:      scopes,
		Endpoint:    oauth2.Endpoint{AuthURL: spotifyauth.AuthURL, TokenURL: spotifyauth.TokenURL},
	}
}

func (s *Source) newClient(tok *oauth2.Token) *spotify.Client {
	cfg := s.oauthConfig("")
	ts := &savingTokenSource{
		src:  oauth2.ReuseTokenSource(nil, cfg.TokenSource(context.Background(), tok)),
		last: tok,
		save: s.opts.SaveToken,
	}
	return spotify.New(oauth2.NewClient(context.Background(), ts))
}

// Connected reports whether a token is present.
func (s *Source) Connected() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.client != nil
}

// SetClientID updates the Client ID (admin edit). Requires Connect afterwards.
func (s *Source) SetClientID(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.opts.ClientID = strings.TrimSpace(id)
}

func (s *Source) SetDeviceID(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.opts.DeviceID = id
}

// Connect runs the PKCE flow: loopback listener on an ephemeral port, browser
// consent, code exchange. Blocks until done, cancelled, or timed out.
// The host registers `http://127.0.0.1/callback` (no port) in their dashboard.
func (s *Source) Connect(ctx context.Context) error {
	s.mu.Lock()
	clientID := s.opts.ClientID
	open := s.opts.OpenBrowser
	s.mu.Unlock()
	if clientID == "" {
		return ErrNoClientID
	}
	if open == nil {
		return errors.New("spotify: no browser opener configured")
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	defer ln.Close()
	redirect := fmt.Sprintf("http://%s%s", ln.Addr().String(), callbackPath)
	cfg := s.oauthConfig(redirect)

	verifier := oauth2.GenerateVerifier()
	state := randomState()
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)
	srv := &http.Server{Handler: callbackHandler(state, codeCh, errCh)}
	go srv.Serve(ln)
	defer srv.Close()

	if err := open(cfg.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier))); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()
	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return fmt.Errorf("%w: %v", ErrLoginTimeout, ctx.Err())
	}

	tok, err := cfg.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return fmt.Errorf("spotify: token exchange: %w", err)
	}

	s.mu.Lock()
	s.opts.Token = tok
	s.client = s.newClient(tok)
	save := s.opts.SaveToken
	s.mu.Unlock()
	if save != nil {
		save(tok)
	}
	return nil
}

func callbackHandler(state string, codeCh chan<- string, errCh chan<- error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != callbackPath {
			http.NotFound(w, r)
			return
		}
		q := r.URL.Query()
		switch {
		case q.Get("state") != state:
			http.Error(w, "state mismatch", http.StatusBadRequest)
			errCh <- errors.New("spotify: state mismatch in callback")
		case q.Get("error") != "":
			http.Error(w, q.Get("error"), http.StatusBadRequest)
			errCh <- fmt.Errorf("spotify: %s", q.Get("error"))
		default:
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(w, "<p>vibe-music is connected to Spotify. You can close this tab.</p>")
			codeCh <- q.Get("code")
		}
	})
}

// Disconnect forgets the token.
func (s *Source) Disconnect() {
	s.mu.Lock()
	s.opts.Token, s.client = nil, nil
	s.current, s.armed = "", false
	save := s.opts.SaveToken
	s.mu.Unlock()
	if save != nil {
		save(nil)
	}
}

func (s *Source) Devices(ctx context.Context) ([]Device, error) {
	c, err := s.getClient()
	if err != nil {
		return nil, err
	}
	devs, err := c.PlayerDevices(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Device, 0, len(devs))
	for _, d := range devs {
		out = append(out, Device{ID: string(d.ID), Name: d.Name, Type: d.Type, Active: d.Active})
	}
	return out, nil
}

func (s *Source) getClient() (*spotify.Client, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.client == nil {
		return nil, ErrNotConnected
	}
	return s.client, nil
}

func (s *Source) playOpts() *spotify.PlayOptions {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.opts.DeviceID == "" {
		return &spotify.PlayOptions{}
	}
	id := spotify.ID(s.opts.DeviceID)
	return &spotify.PlayOptions{DeviceID: &id}
}

// Activate verifies there is a device to control, moves playback to the
// configured one, and turns repeat off (repeat-track would defeat end detection).
func (s *Source) Activate(ctx context.Context) error {
	c, err := s.getClient()
	if err != nil {
		return err
	}
	devs, err := c.PlayerDevices(ctx)
	if err != nil {
		return err
	}
	if len(devs) == 0 {
		return ErrNoDevice
	}
	s.mu.Lock()
	want := spotify.ID(s.opts.DeviceID)
	s.mu.Unlock()
	found := false
	for _, d := range devs {
		if want != "" && d.ID == want {
			found = true
			if !d.Active {
				if err := c.TransferPlayback(ctx, d.ID, false); err != nil {
					return err
				}
			}
		}
	}
	if !found {
		// configured device is gone (or none configured): follow the active one
		s.SetDeviceID("")
		for _, d := range devs {
			if d.Active {
				s.SetDeviceID(string(d.ID))
			}
		}
	}
	if err := c.RepeatOpt(ctx, "off", s.playOpts()); err != nil {
		log.Println("spotify: repeat off:", err)
	}
	return nil
}

// Deactivate pauses; it is the host's own client, so never stop or transfer.
func (s *Source) Deactivate(ctx context.Context) error {
	err := s.Pause(ctx)
	s.mu.Lock()
	s.current, s.armed = "", false
	s.mu.Unlock()
	return err
}

func (s *Source) Search(ctx context.Context, q string, limit int) ([]source.Track, error) {
	c, err := s.getClient()
	if err != nil {
		return nil, err
	}
	res, err := c.Search(ctx, q, spotify.SearchTypeTrack, spotify.Limit(limit), spotify.Market(spotify.MarketFromToken))
	if err != nil {
		return nil, err
	}
	if res.Tracks == nil {
		return nil, nil
	}
	out := make([]source.Track, 0, len(res.Tracks.Tracks))
	for i := range res.Tracks.Tracks {
		out = append(out, mapTrack(&res.Tracks.Tracks[i]))
	}
	return out, nil
}

func (s *Source) Play(ctx context.Context, id string) error {
	c, err := s.getClient()
	if err != nil {
		return err
	}
	opts := s.playOpts()
	opts.URIs = []spotify.URI{spotify.URI(trackURIPrefix + id)}
	if err := c.PlayOpt(ctx, opts); err != nil {
		return err
	}
	s.mu.Lock()
	s.current, s.armed = spotify.ID(id), false
	s.mu.Unlock()
	return nil
}

func (s *Source) Pause(ctx context.Context) error {
	c, err := s.getClient()
	if err != nil {
		return err
	}
	return c.PauseOpt(ctx, s.playOpts())
}

func (s *Source) Resume(ctx context.Context) error {
	c, err := s.getClient()
	if err != nil {
		return err
	}
	return c.PlayOpt(ctx, s.playOpts())
}

func (s *Source) Stop(ctx context.Context) error {
	err := s.Pause(ctx)
	s.mu.Lock()
	s.current, s.armed = "", false
	s.mu.Unlock()
	return err
}

func (s *Source) SetVolume(ctx context.Context, pct int) error {
	c, err := s.getClient()
	if err != nil {
		return err
	}
	return c.VolumeOpt(ctx, pct, s.playOpts())
}

func (s *Source) Status(ctx context.Context) (source.Playback, error) {
	c, err := s.getClient()
	if err != nil {
		return source.Playback{}, err
	}
	cp, err := c.PlayerCurrentlyPlaying(ctx)
	if err != nil {
		return source.Playback{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	pb, armed := mapStatus(cp, s.current, s.armed)
	s.armed = armed
	return pb, nil
}

// mapStatus turns Spotify's player snapshot into a Playback and decides whether
// the track we started has ended. `armed` flips on once we have seen `current`
// playing; before that a poll may still show the previous track at progress 0.
// ponytail: poll-based end detection, ~1s gap between tracks; push-to-queue
// lookahead if the gap annoys people.
func mapStatus(cp *spotify.CurrentlyPlaying, current spotify.ID, armed bool) (source.Playback, bool) {
	pb := source.Playback{At: time.Now()}
	if cp == nil || cp.Item == nil { // 204: nothing active
		pb.Ended = armed && current != ""
		return pb, armed && !pb.Ended
	}
	t := mapTrack(cp.Item)
	pb.Track = &t
	pb.Playing = cp.Playing
	pb.Position = time.Duration(cp.Progress) * time.Millisecond
	if current == "" {
		return pb, false // host is playing their own thing; not ours to end
	}
	if cp.Item.ID == current {
		if cp.Playing {
			return pb, true
		}
		pb.Ended = armed && cp.Progress == 0
		return pb, armed && !pb.Ended
	}
	// A different track: either autoplay took over after ours finished (armed),
	// or Spotify has not caught up with our Play yet (not armed).
	pb.Ended = armed
	return pb, false
}

func mapTrack(ft *spotify.FullTrack) source.Track {
	names := make([]string, 0, len(ft.Artists))
	for _, a := range ft.Artists {
		names = append(names, a.Name)
	}
	return source.Track{
		ID:          string(ft.ID),
		Title:       ft.Name,
		Artist:      strings.Join(names, ", "),
		Album:       ft.Album.Name,
		Duration:    time.Duration(ft.Duration) * time.Millisecond,
		ArtworkURL:  pickImage(ft.Album.Images),
		ExternalURL: ft.ExternalURLs[externalURLKey],
	}
}

// pickImage returns the smallest image at least minArtworkPx tall, else the largest.
func pickImage(imgs []spotify.Image) string {
	best := ""
	bestH := 0
	for _, im := range imgs {
		h := int(im.Height)
		if h >= minArtworkPx && (bestH < minArtworkPx || h < bestH) {
			best, bestH = im.URL, h
		} else if bestH < minArtworkPx && h > bestH {
			best, bestH = im.URL, h
		}
	}
	return best
}

// savingTokenSource persists the token whenever it changes.
type savingTokenSource struct {
	mu   sync.Mutex
	src  oauth2.TokenSource
	last *oauth2.Token
	save func(*oauth2.Token)
}

func (t *savingTokenSource) Token() (*oauth2.Token, error) {
	tok, err := t.src.Token()
	if err != nil {
		return nil, err
	}
	t.mu.Lock()
	changed := t.last == nil || tok.AccessToken != t.last.AccessToken || tok.RefreshToken != t.last.RefreshToken
	t.last = tok
	save := t.save
	t.mu.Unlock()
	if changed && save != nil {
		save(tok)
	}
	return tok, nil
}

func randomState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
