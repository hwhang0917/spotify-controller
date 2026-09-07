package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/oauth2"

	"github.com/hwhang0917/vibe-music/internal/config"
	"github.com/hwhang0917/vibe-music/internal/player"
	"github.com/hwhang0917/vibe-music/internal/server"
	"github.com/hwhang0917/vibe-music/internal/source"
	"github.com/hwhang0917/vibe-music/internal/source/local"
	"github.com/hwhang0917/vibe-music/internal/source/spotify"
	"github.com/hwhang0917/vibe-music/web"
)

// shutdownTimeout bounds how long StopServer waits for in-flight requests.
const shutdownTimeout = 5 * time.Second

// Wails event names the admin UI listens on.
const (
	stateEvent  = "state"
	guestsEvent = "guests"
)

// App is the Wails-bound backend for the admin window.
type App struct {
	ctx context.Context

	mu      sync.Mutex
	cfg     config.Config
	srv     *http.Server
	url     string
	player  *player.Player
	guests  *server.Guests
	local   *local.Source
	spotify *spotify.Source
	cancel  context.CancelFunc
}

type ServerStatus struct {
	Running bool   `json:"running"`
	Port    int    `json:"port"`
	URL     string `json:"url"`
}

// SourceStatus is one row in the admin's source picker.
type SourceStatus struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
	Ready  bool   `json:"ready"`
	Detail string `json:"detail"`
}

func NewApp() *App { return &App{} }

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	cfg, err := config.Load()
	if err != nil {
		log.Println("config:", err)
		cfg = config.Default()
	}
	a.cfg = cfg

	a.local = local.New(cfg.Local.Folders)
	a.spotify = spotify.New(spotify.Options{
		ClientID:    cfg.Spotify.ClientID,
		DeviceID:    cfg.Spotify.DeviceID,
		Token:       cfg.Spotify.Token,
		SaveToken:   a.saveSpotifyToken,
		OpenBrowser: func(u string) error { runtime.BrowserOpenURL(ctx, u); return nil },
	})
	a.player = player.New(player.Options{
		Sources:   []source.Source{a.local, a.spotify},
		SkipRatio: cfg.SkipRatio,
	})
	a.guests = server.NewGuests(cfg.Blocked, func() {
		runtime.EventsEmit(a.ctx, guestsEvent, a.guests.List())
	})

	runCtx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel
	go a.player.Run(runCtx)
	go a.forwardState(runCtx)

	if err := a.player.SetSource(runCtx, cfg.ActiveSource); err != nil {
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
	a.mu.Lock()
	a.cfg.Spotify.Token = tok
	cfg := a.cfg
	a.mu.Unlock()
	if err := config.Save(cfg); err != nil {
		log.Println("save token:", err)
	}
}

// --- config ---

func (a *App) GetConfig() config.Config {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.cfg
}

// SaveConfig persists and applies settings. Changing the Spotify Client ID
// requires SpotifyConnect afterwards; the token is kept as-is here.
func (a *App) SaveConfig(c config.Config) error {
	if c.Local.Folders == nil {
		c.Local.Folders = []string{}
	}
	a.mu.Lock()
	c.Spotify.Token = a.cfg.Spotify.Token
	a.cfg = c
	a.mu.Unlock()
	if err := config.Save(c); err != nil {
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
		st := SourceStatus{ID: info.ID, Name: info.Name, Active: active != nil && active.ID() == info.ID}
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
		}
		out = append(out, st)
	}
	return out
}

// SetActiveSource switches backends and remembers the choice.
func (a *App) SetActiveSource(id string) error {
	if err := a.player.SetSource(a.ctx, id); err != nil {
		return err
	}
	a.mu.Lock()
	a.cfg.ActiveSource = id
	cfg := a.cfg
	a.mu.Unlock()
	return config.Save(cfg)
}

func (a *App) LocalRescan() (int, error) { return a.local.Rescan() }

// SpotifyConnect runs the browser consent flow; blocks until finished.
func (a *App) SpotifyConnect() error { return a.spotify.Connect(a.ctx) }

func (a *App) SpotifyDisconnect() { a.spotify.Disconnect() }

func (a *App) SpotifyDevices() ([]spotify.Device, error) { return a.spotify.Devices(a.ctx) }

// --- guests ---

func (a *App) Guests() []server.GuestInfo { return a.guests.List() }

// KickGuest drops the guest's live connections; they can reconnect.
func (a *App) KickGuest(id string) { a.guests.Kick(id) }

// RemoveGuest forgets the guest and everything they queued or voted for.
func (a *App) RemoveGuest(id string) {
	a.guests.Remove(id)
	a.player.RemoveGuest(id)
}

// BlockGuest toggles the block list and persists it. Blocking also removes
// the guest's requests and votes.
func (a *App) BlockGuest(id string, blocked bool) error {
	a.guests.SetBlocked(id, blocked)
	if blocked {
		a.player.RemoveGuest(id)
	}
	a.mu.Lock()
	a.cfg.Blocked = a.guests.Blocked()
	cfg := a.cfg
	a.mu.Unlock()
	return config.Save(cfg)
}

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
		return a.url, errors.New("server already running")
	}
	if port <= 0 {
		port = config.DefaultPort
	}
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return "", err
	}
	srv := &http.Server{Handler: server.NewHandler(web.Dist, a.player, a.guests)}
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Println("guest server:", err)
		}
	}()
	a.srv = srv
	a.url = fmt.Sprintf("http://%s:%d", lanIP(), port)
	a.cfg.Port = port
	if err := config.Save(a.cfg); err != nil {
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
