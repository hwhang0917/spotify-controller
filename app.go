package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/oauth2"

	"github.com/hwhang0917/vibe-music/internal/config"
	"github.com/hwhang0917/vibe-music/internal/player"
	"github.com/hwhang0917/vibe-music/internal/server"
	"github.com/hwhang0917/vibe-music/internal/source"
	"github.com/hwhang0917/vibe-music/internal/source/local"
	"github.com/hwhang0917/vibe-music/internal/source/spotify"
	"github.com/hwhang0917/vibe-music/internal/source/youtube"
	"github.com/hwhang0917/vibe-music/internal/store"
	"github.com/hwhang0917/vibe-music/web"
)

// shutdownTimeout bounds how long StopServer waits for in-flight requests.
const shutdownTimeout = 5 * time.Second

// Wails event names the admin UI listens on.
const (
	stateEvent   = "state"
	guestsEvent  = "guests"
	youtubeEvent = "yt:cmd" // commands for the embedded YouTube player
)

// UI error codes. The admin UI translates "code: detail"; unknown text is
// shown as-is. Keep in sync with frontend/src/i18n.ts err.* keys.
const (
	errPortInUse     = "port_in_use"
	errPortDenied    = "port_denied"
	errServerRunning = "server_running"
	errUnknownSource = "unknown_source"
	errAudioOutput   = "audio_output"
	errSpotifyNoConn = "spotify_not_connected"
	errSpotifyNoID   = "spotify_client_id"
	errSpotifyNoDev  = "spotify_no_device"
	errSpotifyLogin  = "spotify_login_timeout"
	errYouTubeKey    = "youtube_api_key"
	errYouTubeQuota  = "youtube_quota"
	errYouTubePlayer = "youtube_player"
	errSourceOff     = "source_disabled"
)

// Winsock reports its own errno values; Go's syscall.EADDRINUSE/EACCES are
// the POSIX ones and never match on Windows.
const (
	wsaEADDRINUSE syscall.Errno = 10048
	wsaEACCES     syscall.Errno = 10013
)

func isErrno(err error, targets ...syscall.Errno) bool {
	var errno syscall.Errno
	if !errors.As(err, &errno) {
		return false
	}
	for _, t := range targets {
		if errno == t {
			return true
		}
	}
	return false
}

// codeErr formats an error as "code: detail" so the UI can translate it.
func codeErr(code, detail string) error {
	if detail == "" {
		return errors.New(code)
	}
	return fmt.Errorf("%s: %s", code, detail)
}

// uiError maps well-known failures to codes; anything else passes through.
func uiError(err error) error {
	switch {
	case err == nil:
		return nil
	case isErrno(err, syscall.EADDRINUSE, wsaEADDRINUSE):
		return codeErr(errPortInUse, "")
	case isErrno(err, syscall.EACCES, wsaEACCES):
		return codeErr(errPortDenied, "")
	case errors.Is(err, spotify.ErrNotConnected):
		return codeErr(errSpotifyNoConn, "")
	case errors.Is(err, spotify.ErrNoClientID):
		return codeErr(errSpotifyNoID, "")
	case errors.Is(err, spotify.ErrNoDevice):
		return codeErr(errSpotifyNoDev, "")
	case errors.Is(err, spotify.ErrLoginTimeout):
		return codeErr(errSpotifyLogin, "")
	case errors.Is(err, local.ErrAudioOutput):
		return codeErr(errAudioOutput, err.Error())
	case errors.Is(err, youtube.ErrNoAPIKey):
		return codeErr(errYouTubeKey, "")
	case errors.Is(err, youtube.ErrQuotaExceeded):
		return codeErr(errYouTubeQuota, "")
	case errors.Is(err, youtube.ErrPlayerNotReady):
		return codeErr(errYouTubePlayer, "")
	}
	return err
}

// App is the Wails-bound backend for the admin window.
type App struct {
	ctx context.Context

	mu      sync.Mutex
	cfg     config.Config
	db      *store.Store
	srv     *http.Server
	url     string
	player  *player.Player
	guests  *server.Guests
	local   *local.Source
	spotify *spotify.Source
	youtube *youtube.Source
	cancel  context.CancelFunc
}

type ServerStatus struct {
	Running bool   `json:"running"`
	Port    int    `json:"port"`
	URL     string `json:"url"`
}

// SourceStatus is one row in the admin's source picker.
type SourceStatus struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Active  bool   `json:"active"`
	Enabled bool   `json:"enabled"`
	Ready   bool   `json:"ready"`
	Detail  string `json:"detail"`
}

func NewApp() *App { return &App{} }

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	dir, err := config.Dir()
	if err != nil {
		log.Fatal(err)
	}
	a.db, err = store.Open(filepath.Join(dir, config.DBFile))
	if err != nil {
		log.Fatal(err)
	}
	cfg, err := config.Load(a.db)
	if err != nil {
		log.Println("config:", err)
		cfg = config.Default()
	}
	a.cfg = cfg
	token, err := config.LoadToken()
	if err != nil {
		log.Println("spotify token:", err)
	}

	a.local = local.New(cfg.Local.Folders)
	a.spotify = spotify.New(spotify.Options{
		ClientID:    cfg.Spotify.ClientID,
		DeviceID:    cfg.Spotify.DeviceID,
		Token:       token,
		SaveToken:   a.saveSpotifyToken,
		OpenBrowser: func(u string) error { runtime.BrowserOpenURL(ctx, u); return nil },
	})
	ytKey, err := config.LoadYouTubeKey()
	if err != nil {
		log.Println("youtube key:", err)
	}
	a.youtube = youtube.New(youtube.Options{
		APIKey: ytKey,
		Send:   func(c youtube.Command) { runtime.EventsEmit(a.ctx, youtubeEvent, c) },
	})
	a.player = player.New(player.Options{
		Sources:   []source.Source{a.local, a.spotify, a.youtube},
		SkipRatio: cfg.SkipRatio,
	})
	a.player.SetPersister(func(items []player.PersistedItem) {
		if err := a.db.SaveQueue(toRows(items)); err != nil {
			log.Println("save queue:", err)
		}
	})
	a.player.SetOnPlay(func(sourceID string, t source.Track) {
		track, _ := json.Marshal(t)
		if err := a.db.RecordPlay(sourceID, t.ID, track, time.Now()); err != nil {
			log.Println("record play:", err)
		}
	})
	if rows, err := a.db.LoadQueue(); err == nil {
		a.player.Restore(fromRows(rows))
	} else {
		log.Println("load queue:", err)
	}
	a.guests = server.NewGuests(a.db, cfg.InviteOnly, func() {
		if list, err := a.guests.List(); err == nil {
			runtime.EventsEmit(a.ctx, guestsEvent, list)
		}
	})

	runCtx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel
	go a.player.Run(runCtx)
	go a.forwardState(runCtx)

	if cfg.IsDisabled(cfg.ActiveSource) {
		log.Println("activate: source", cfg.ActiveSource, "is disabled")
	} else if err := a.player.SetSource(runCtx, cfg.ActiveSource); err != nil {
		log.Println("activate", cfg.ActiveSource+":", err)
	}
}

func (a *App) shutdown(ctx context.Context) {
	_ = a.StopServer()
	if a.cancel != nil {
		a.cancel()
	}
	if src := a.player.ActiveSource(); src != nil {
		_ = src.Deactivate(ctx)
	}
	if a.db != nil {
		_ = a.db.Close()
	}
}

// topTracks adapts the store's play history to the server's TopFunc.
func (a *App) topTracks(sourceID string, limit int) ([]source.Track, error) {
	raws, err := a.db.TopTracks(sourceID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]source.Track, 0, len(raws))
	for _, r := range raws {
		var t source.Track
		if json.Unmarshal(r, &t) == nil {
			out = append(out, t)
		}
	}
	return out, nil
}

func toRows(items []player.PersistedItem) []store.QueueRow {
	out := make([]store.QueueRow, 0, len(items))
	for _, it := range items {
		track, _ := json.Marshal(it.Track)
		out = append(out, store.QueueRow{
			ID: it.ID, Track: track, RequestedBy: it.RequestedBy, RequestedByName: it.RequestedByName,
			RequestedAt: it.RequestedAt, Rank: it.Rank, Votes: it.Votes,
		})
	}
	return out
}

func fromRows(rows []store.QueueRow) []player.PersistedItem {
	out := make([]player.PersistedItem, 0, len(rows))
	for _, r := range rows {
		var track source.Track
		if err := json.Unmarshal(r.Track, &track); err != nil {
			continue // a row we cannot read is not worth crashing over
		}
		out = append(out, player.PersistedItem{
			ID: r.ID, Track: track, RequestedBy: r.RequestedBy, RequestedByName: r.RequestedByName,
			RequestedAt: r.RequestedAt, Rank: r.Rank, Votes: r.Votes,
		})
	}
	return out
}

// forwardState pushes player snapshots to the admin window as Wails events.
func (a *App) forwardState(ctx context.Context) {
	ch, cancel := a.player.Subscribe()
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return
		case st := <-ch:
			runtime.EventsEmit(a.ctx, stateEvent, st)
		}
	}
}

func (a *App) saveSpotifyToken(tok *oauth2.Token) {
	if err := config.SaveToken(tok); err != nil {
		log.Println("save token:", err)
	}
}

// saveConfig persists the in-memory config.
func (a *App) saveConfig() error {
	a.mu.Lock()
	cfg := a.cfg
	a.mu.Unlock()
	return config.Save(a.db, cfg)
}

// --- config ---

func (a *App) GetConfig() config.Config {
	a.mu.Lock()
	defer a.mu.Unlock()
	c := a.cfg
	c.YouTube.HasKey = a.youtube.HasAPIKey()
	return c
}

// SaveConfig persists and applies settings. Changing the Spotify Client ID
// requires SpotifyConnect afterwards. InviteOnly is toggled via SetInviteOnly.
func (a *App) SaveConfig(c config.Config) error {
	if c.Local.Folders == nil {
		c.Local.Folders = []string{}
	}
	a.mu.Lock()
	c.InviteOnly = a.cfg.InviteOnly
	a.cfg = c
	a.mu.Unlock()
	if err := config.Save(a.db, c); err != nil {
		return err
	}
	a.local.SetFolders(c.Local.Folders)
	a.spotify.SetClientID(c.Spotify.ClientID)
	a.spotify.SetDeviceID(c.Spotify.DeviceID)
	a.player.SetSkipRatio(c.SkipRatio)
	return nil
}

// --- sources ---

func (a *App) Sources() []SourceStatus {
	active := a.player.ActiveSource()
	cfg := a.GetConfig()
	out := make([]SourceStatus, 0, 2)
	for _, info := range a.player.Sources() {
		st := SourceStatus{ID: info.ID, Name: info.Name, Active: active != nil && active.ID() == info.ID, Enabled: !cfg.IsDisabled(info.ID)}
		switch info.ID {
		case a.local.ID():
			st.Ready = len(cfg.Local.Folders) > 0
			st.Detail = fmt.Sprintf("%d folder(s)", len(cfg.Local.Folders))
		case a.spotify.ID():
			st.Ready = a.spotify.Connected()
			if st.Ready {
				st.Detail = "connected"
			} else {
				st.Detail = "not connected"
			}
		case a.youtube.ID():
			st.Ready = a.youtube.HasAPIKey()
			if st.Ready {
				st.Detail = "API key set"
			} else {
				st.Detail = "API key needed"
			}
		}
		out = append(out, st)
	}
	return out
}

// SetActiveSource switches backends and remembers the choice.
func (a *App) SetActiveSource(id string) error {
	if a.GetConfig().IsDisabled(id) {
		return codeErr(errSourceOff, "")
	}
	if err := a.player.SetSource(a.ctx, id); err != nil {
		return uiError(err)
	}
	a.mu.Lock()
	a.cfg.ActiveSource = id
	a.mu.Unlock()
	return a.saveConfig()
}

// SetSourceEnabled switches a source on or off. Disabling the active source
// stops playback and clears the queue (the UI confirms first).
func (a *App) SetSourceEnabled(id string, enabled bool) error {
	if _, ok := a.sourceByID(id); !ok {
		return codeErr(errUnknownSource, id)
	}
	if !enabled {
		if active := a.player.ActiveSource(); active != nil && active.ID() == id {
			a.player.Deactivate(a.ctx)
		}
	}
	a.mu.Lock()
	a.cfg.SetDisabled(id, !enabled)
	a.mu.Unlock()
	return a.saveConfig()
}

func (a *App) sourceByID(id string) (player.SourceInfo, bool) {
	for _, s := range a.player.Sources() {
		if s.ID == id {
			return s, true
		}
	}
	return player.SourceInfo{}, false
}

func (a *App) LocalRescan() (int, error) { return a.local.Rescan() }

// PickFolder opens the OS directory chooser. Returns "" when cancelled.
func (a *App) PickFolder() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: "Add music folder"})
}

// SpotifyConnect runs the browser consent flow; blocks until finished.
func (a *App) SpotifyConnect() error { return uiError(a.spotify.Connect(a.ctx)) }

func (a *App) SpotifyDisconnect() { a.spotify.Disconnect() }

func (a *App) SpotifyDevices() ([]spotify.Device, error) {
	d, err := a.spotify.Devices(a.ctx)
	return d, uiError(err)
}

// --- guests & invitations ---

func (a *App) Guests() ([]server.GuestInfo, error) { return a.guests.List() }

// SetInviteOnly toggles invitation-only access. Turning it on keeps everyone
// already known admitted; newcomers need a code.
func (a *App) SetInviteOnly(on bool) error {
	if err := a.guests.SetInviteOnly(on); err != nil {
		return err
	}
	a.mu.Lock()
	a.cfg.InviteOnly = on
	a.mu.Unlock()
	return a.saveConfig()
}

// AdmitGuest lets a specific guest in without a code.
func (a *App) AdmitGuest(id string) error { return a.guests.Admit(id) }

// CreateInvitation makes a code valid for ttlMinutes; the plaintext is only
// available in the returned value and in this session's list.
func (a *App) CreateInvitation(ttlMinutes int) (server.Invitation, error) {
	if ttlMinutes <= 0 {
		return server.Invitation{}, errors.New("ttl must be positive")
	}
	return a.guests.CreateInvitation(time.Duration(ttlMinutes) * time.Minute)
}

func (a *App) Invitations() ([]server.Invitation, error) { return a.guests.Invitations() }

func (a *App) RevokeInvitation(id int64) error { return a.guests.RevokeInvitation(id) }

// KickGuest drops the guest's live connections; they can reconnect.
func (a *App) KickGuest(id string) { a.guests.Kick(id) }

// RemoveGuest forgets the guest and everything they queued or voted for.
func (a *App) RemoveGuest(id string) error {
	a.player.RemoveGuest(id)
	return a.guests.Remove(id)
}

// BlockGuest toggles the (persisted) block. Blocking also removes the guest's
// requests and votes.
func (a *App) BlockGuest(id string, blocked bool) error {
	if blocked {
		a.player.RemoveGuest(id)
	}
	return a.guests.SetBlocked(id, blocked)
}

// --- queue (admin override) ---

// RemoveQueueItem drops any queued item regardless of who requested it.
func (a *App) RemoveQueueItem(id string) error { return a.player.Remove(id, "", true) }

// MoveQueueItem places a queued item at index (0 = next up).
func (a *App) MoveQueueItem(id string, index int) error { return a.player.Move(id, index) }

// Seek moves the current track to the given millisecond offset.
func (a *App) Seek(ms int) error {
	return uiError(a.player.Seek(a.ctx, time.Duration(ms)*time.Millisecond))
}

// --- youtube ---

// SetYouTubeAPIKey stores the Data API key (empty removes it).
func (a *App) SetYouTubeAPIKey(key string) error {
	if err := config.SaveYouTubeKey(key); err != nil {
		return err
	}
	a.youtube.SetAPIKey(key)
	return nil
}

// YouTubeReport receives the embedded player's state from the admin page.
func (a *App) YouTubeReport(r youtube.Report) { a.youtube.Report(r) }

// --- playback (admin override) ---

func (a *App) GetState() player.State  { return a.player.State() }
func (a *App) Pause() error            { return a.player.Pause(a.ctx) }
func (a *App) Resume() error           { return a.player.Resume(a.ctx) }
func (a *App) Skip()                   { a.player.Skip(a.ctx) }
func (a *App) SetVolume(pct int) error { return a.player.SetVolume(a.ctx, pct) }

// --- guest server ---

// StartServer starts the guest HTTP server and returns the LAN URL to share.
func (a *App) StartServer(port int) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.srv != nil {
		return a.url, codeErr(errServerRunning, "")
	}
	if port <= 0 {
		port = config.DefaultPort
	}
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return "", uiError(err)
	}
	srv := &http.Server{Handler: server.NewHandler(web.Dist, a.player, a.guests, a.topTracks)}
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Println("guest server:", err)
		}
	}()
	a.srv = srv
	a.url = fmt.Sprintf("http://%s:%d", lanIP(), port)
	a.cfg.Port = port
	if err := config.Save(a.db, a.cfg); err != nil {
		log.Println("save config:", err)
	}
	return a.url, nil
}

// StopServer gracefully shuts the guest server down. No-op if not running.
func (a *App) StopServer() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.srv == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	err := a.srv.Shutdown(ctx)
	a.srv, a.url = nil, ""
	return err
}

func (a *App) Status() ServerStatus {
	a.mu.Lock()
	defer a.mu.Unlock()
	return ServerStatus{Running: a.srv != nil, Port: a.cfg.Port, URL: a.url}
}

// lanIP returns the first private IPv4 on a non-loopback interface, falling
// back to loopback so the URL is still openable on the host.
func lanIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, addr := range addrs {
		ipnet, ok := addr.(*net.IPNet)
		if !ok || ipnet.IP.To4() == nil {
			continue
		}
		if ipnet.IP.IsPrivate() {
			return ipnet.IP.String()
		}
	}
	return "127.0.0.1"
}
