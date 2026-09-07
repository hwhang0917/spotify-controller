package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/hwhang0917/vibe-music/internal/server"
	"github.com/hwhang0917/vibe-music/web"
)

// DefaultPort is what the admin UI pre-fills for the guest server.
const DefaultPort = 5555

// shutdownTimeout bounds how long StopServer waits for in-flight requests.
const shutdownTimeout = 5 * time.Second

// App is the Wails-bound backend for the admin window.
type App struct {
	ctx context.Context

	mu  sync.Mutex
	srv *http.Server
	url string
}

// ServerStatus is what the admin UI renders.
type ServerStatus struct {
	Running bool   `json:"running"`
	Port    int    `json:"port"`
	URL     string `json:"url"`
}

func NewApp() *App { return &App{} }

func (a *App) startup(ctx context.Context) { a.ctx = ctx }

func (a *App) shutdown(ctx context.Context) { _ = a.StopServer() }

// StartServer starts the guest HTTP server on the given port and returns the
// LAN URL guests should open.
func (a *App) StartServer(port int) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.srv != nil {
		return a.url, errors.New("server already running")
	}
	if port <= 0 {
		port = DefaultPort
	}

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return "", err
	}
	srv := &http.Server{Handler: server.NewHandler(web.Dist)}
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Println("guest server:", err)
		}
	}()

	a.srv = srv
	a.url = fmt.Sprintf("http://%s:%d", lanIP(), port)
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
	return ServerStatus{Running: a.srv != nil, Port: DefaultPort, URL: a.url}
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
