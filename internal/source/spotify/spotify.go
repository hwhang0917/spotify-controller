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
	"regexp"
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
	httpTimeout     = 15 * time.Second // a dead network must fail, not hang
	callbackPath    = "/callback"
	minArtworkPx    = 300
	browseAlbums    = 20 // albums+singles shown on an artist page
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
	ErrNotConnected   = &source.CodedError{Kind: "spotify_not_connected", Msg: "spotify: not connected"}
	ErrNoClientID     = &source.CodedError{Kind: "spotify_client_id", Msg: "spotify: client ID is empty"}
	ErrBadClientID    = &source.CodedError{Kind: "spotify_client_id_invalid", Msg: "spotify: client ID must be 32 hex characters"}
	ErrNoDevice       = &source.CodedError{Kind: "spotify_no_device", Msg: noDeviceMessage}
	ErrLoginTimeout   = &source.CodedError{Kind: "spotify_login_timeout", Msg: "spotify: login not completed"}
	ErrLoginCancelled = &source.CodedError{Kind: "spotify_login_cancelled", Msg: "spotify: login cancelled"}
	ErrCallbackPort   = &source.CodedError{Kind: "spotify_callback_port", Msg: "spotify: callback port is in use"}
	ErrPremium        = &source.CodedError{Kind: "spotify_premium_required", Msg: "spotify: the account that owns the app needs an active Premium subscription"}
)

// wrapAPI maps Web API failures the host can act on to coded errors; the
// library's own text (e.g. "couldn't decode error: (159) [...]") stays for
// anything else.
func wrapAPI(err error) error {
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "premium"):
		return fmt.Errorf("%w: %v", ErrPremium, err)
	}
	return err
}

// clientIDPattern: Spotify client IDs are 32 lowercase hex characters.
var clientIDPattern = regexp.MustCompile(`^[0-9a-f]{32}$`)

type Options struct {
	ClientID string
	DeviceID string
	// CallbackPort is where the loopback redirect listens; 0 picks a free port
	// (tests). The dashboard needs the registered URI to include this port.
	CallbackPort int
	Token        *oauth2.Token
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

	cancelConnect context.CancelFunc // set while Connect waits for the browser
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
	// oauth2 builds its clients from the one in this context value
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, &http.Client{Timeout: httpTimeout})
	ts := &savingTokenSource{
		src:  oauth2.ReuseTokenSource(nil, cfg.TokenSource(ctx, tok)),
		last: tok,
		save: s.opts.SaveToken,
	}
	return spotify.New(oauth2.NewClient(ctx, ts))
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

func (s *Source) SetCallbackPort(port int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.opts.CallbackPort = port
}

// RedirectURI is what the host must register in the Spotify dashboard.
func (s *Source) RedirectURI() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return fmt.Sprintf("http://127.0.0.1:%d%s", s.opts.CallbackPort, callbackPath)
}

func (s *Source) SetDeviceID(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.opts.DeviceID = id
}

// Connect runs the PKCE flow: loopback listener on the configured port,
// browser consent, code exchange. Blocks until done, cancelled, or timed out.
// The host registers RedirectURI() in their dashboard.
func (s *Source) Connect(ctx context.Context) error {
	s.mu.Lock()
	clientID := s.opts.ClientID
	open := s.opts.OpenBrowser
	port := s.opts.CallbackPort
	s.mu.Unlock()
	if clientID == "" {
		return ErrNoClientID
	}
	if !clientIDPattern.MatchString(clientID) {
		return ErrBadClientID // catches typos before Spotify shows INVALID_CLIENT in a tab we cannot see
	}
	if open == nil {
		return errors.New("spotify: no browser opener configured")
	}

	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCallbackPort, err)
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
	s.mu.Lock()
	s.cancelConnect = cancel
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.cancelConnect = nil
		s.mu.Unlock()
	}()
	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		return err
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.Canceled) {
			return ErrLoginCancelled
		}
		return ErrLoginTimeout
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

// CancelConnect aborts a Connect that is waiting for the browser.
func (s *Source) CancelConnect() {
	s.mu.Lock()
	cancel := s.cancelConnect
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
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
		return nil, wrapAPI(err)
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
		return wrapAPI(err)
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
		return nil, wrapAPI(err)
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
		return wrapAPI(err)
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
	return wrapAPI(c.PauseOpt(ctx, s.playOpts()))
}

func (s *Source) Resume(ctx context.Context) error {
	c, err := s.getClient()
	if err != nil {
		return err
	}
	return wrapAPI(c.PlayOpt(ctx, s.playOpts()))
}

func (s *Source) Stop(ctx context.Context) error {
	err := s.Pause(ctx)
	s.mu.Lock()
	s.current, s.armed = "", false
	s.mu.Unlock()
	return err
}

func (s *Source) Seek(ctx context.Context, pos time.Duration) error {
	c, err := s.getClient()
	if err != nil {
		return err
	}
	return wrapAPI(c.SeekOpt(ctx, int(pos.Milliseconds()), s.playOpts()))
}

func (s *Source) SetVolume(ctx context.Context, pct int) error {
	c, err := s.getClient()
	if err != nil {
		return err
	}
	return wrapAPI(c.VolumeOpt(ctx, pct, s.playOpts()))
}

func (s *Source) Status(ctx context.Context) (source.Playback, error) {
	c, err := s.getClient()
	if err != nil {
		return source.Playback{}, err
	}
	cp, err := c.PlayerCurrentlyPlaying(ctx)
	if err != nil {
		return source.Playback{}, wrapAPI(err)
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

func mapTrack(ft *spotify.FullTrack) source.Track { return mapSimple(&ft.SimpleTrack, &ft.Album) }

// mapSimple maps a track with the album it belongs to (album tracks from the
// API carry no album of their own, so the parent's art and year are used).
func mapSimple(st *spotify.SimpleTrack, al *spotify.SimpleAlbum) source.Track {
	names := make([]string, 0, len(st.Artists))
	for _, a := range st.Artists {
		names = append(names, a.Name)
	}
	t := source.Track{
		Source:      "spotify",
		ID:          string(st.ID),
		Title:       st.Name,
		Artist:      strings.Join(names, ", "),
		Album:       al.Name,
		Year:        source.YearOf(al.ReleaseDate),
		Duration:    time.Duration(st.Duration) * time.Millisecond,
		ArtworkURL:  pickImage(al.Images),
		ExternalURL: st.ExternalURLs[externalURLKey],
		AlbumID:     string(al.ID),
	}
	if len(st.Artists) > 0 {
		t.ArtistID = string(st.Artists[0].ID)
	}
	return t
}

func mapAlbum(al *spotify.SimpleAlbum) source.Album {
	a := source.Album{ID: string(al.ID), Name: al.Name, Year: source.YearOf(al.ReleaseDate), ArtworkURL: pickImage(al.Images)}
	if len(al.Artists) > 0 {
		a.Artist, a.ArtistID = al.Artists[0].Name, string(al.Artists[0].ID)
	}
	return a
}

// Artist implements source.Browser: the artist's top tracks and their albums.
func (s *Source) Artist(ctx context.Context, id string) (source.Artist, error) {
	c, err := s.getClient()
	if err != nil {
		return source.Artist{}, err
	}
	ar, err := c.GetArtist(ctx, spotify.ID(id))
	if err != nil {
		return source.Artist{}, wrapAPI(err)
	}
	// top-tracks takes the market as "country"; from_token works there like in Search
	top, err := c.GetArtistsTopTracks(ctx, spotify.ID(id), spotify.MarketFromToken)
	if err != nil {
		return source.Artist{}, wrapAPI(err)
	}
	albums, err := c.GetArtistAlbums(ctx, spotify.ID(id), []spotify.AlbumType{spotify.AlbumTypeAlbum, spotify.AlbumTypeSingle},
		spotify.Market(spotify.MarketFromToken), spotify.Limit(browseAlbums))
	if err != nil {
		return source.Artist{}, wrapAPI(err)
	}
	out := source.Artist{ID: id, Name: ar.Name, ArtworkURL: pickImage(ar.Images)}
	for i := range top {
		out.Tracks = append(out.Tracks, mapTrack(&top[i]))
	}
	for i := range albums.Albums {
		out.Albums = append(out.Albums, mapAlbum(&albums.Albums[i]))
	}
	return out, nil
}

// Album implements source.Browser. First page of tracks only.
// ponytail: albums over 50 tracks are cut; page if anyone notices.
func (s *Source) Album(ctx context.Context, id string) (source.Album, error) {
	c, err := s.getClient()
	if err != nil {
		return source.Album{}, err
	}
	fa, err := c.GetAlbum(ctx, spotify.ID(id), spotify.Market(spotify.MarketFromToken))
	if err != nil {
		return source.Album{}, wrapAPI(err)
	}
	out := mapAlbum(&fa.SimpleAlbum)
	for i := range fa.Tracks.Tracks {
		out.Tracks = append(out.Tracks, mapSimple(&fa.Tracks.Tracks[i], &fa.SimpleAlbum))
	}
	return out, nil
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
