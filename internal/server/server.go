// Package server is the guest-facing HTTP service: a JSON API under /api, a
// Server-Sent Events stream, and the embedded Vue guest UI for everything else.
// Guests never see a Spotify token; every source call goes through the player.
package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/hwhang0917/vibe-music/internal/player"
	"github.com/hwhang0917/vibe-music/internal/source"
)

const (
	guestCookie   = "vm_guest"
	cookieMaxAge  = 365 * 24 * 60 * 60
	maxNameLen    = 24
	searchLimit   = 20
	ssePing       = 30 * time.Second
	requestBodyKB = 4
)

type Server struct {
	player *player.Player
	guests *Guests
}

// Error codes the guest UI translates.
const (
	errNameRequired = "name_required"
	errNameInvalid  = "name_invalid"
	errBlocked      = "blocked"
	errNotInQueue   = "not_in_queue"
	errTrackNeeded  = "track_required"
	errNoPlayer     = "player_not_running"
)

// NewHandler wires the API and serves dist (a built Vite app) as an SPA:
// unknown paths fall back to index.html so client-side routing works.
// guests may be nil for a standalone handler; the app passes its own so the
// admin window can manage them.
func NewHandler(dist fs.FS, p *player.Player, guests *Guests) http.Handler {
	if guests == nil {
		guests = NewGuests(nil, nil)
	}
	s := &Server{player: p, guests: guests}
	r := chi.NewRouter()
	r.Use(middleware.Logger, middleware.Recoverer)

	r.Route("/api", func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		})
		r.Group(func(r chi.Router) {
			r.Use(s.guest)
			r.Get("/state", s.state)
			r.Get("/events", s.events)
			r.Get("/me", s.me)
			r.Post("/me", s.setMe)
			r.Get("/search", s.search)
			r.Get("/artwork/{id}", s.artwork)
			r.Group(func(r chi.Router) {
				r.Use(s.requireName)
				r.Post("/queue", s.request)
				r.Post("/queue/{id}/vote", s.vote)
				r.Post("/skip", s.skip)
			})
		})
	})

	r.NotFound(spa(dist))
	return r
}

type ctxKey int

const guestKey ctxKey = 0

// guest issues the anonymous cookie on first visit and puts the Guest in ctx.
func (s *Server) guest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := ""
		if c, err := r.Cookie(guestCookie); err == nil && len(c.Value) == 32 {
			id = c.Value
		}
		if id == "" {
			id = newGuestID()
			http.SetCookie(w, &http.Cookie{
				Name: guestCookie, Value: id, Path: "/", MaxAge: cookieMaxAge,
				HttpOnly: true, SameSite: http.SameSiteLaxMode,
			})
		}
		if s.guests.isBlocked(id) {
			writeError(w, http.StatusForbidden, errBlocked)
			return
		}
		g := player.Guest{ID: id, Name: s.guests.seen(id)}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), guestKey, g)))
	})
}

func guestFrom(r *http.Request) player.Guest {
	g, _ := r.Context().Value(guestKey).(player.Guest)
	return g
}

func (s *Server) requireName(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if guestFrom(r).Name == "" {
			writeError(w, http.StatusBadRequest, errNameRequired)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) state(w http.ResponseWriter, r *http.Request) {
	if s.player == nil {
		writeError(w, http.StatusServiceUnavailable, errNoPlayer)
		return
	}
	writeJSON(w, http.StatusOK, s.player.State())
}

// events streams player state as SSE `state` events, with a comment ping for keepalive.
func (s *Server) events(w http.ResponseWriter, r *http.Request) {
	if s.player == nil {
		writeError(w, http.StatusServiceUnavailable, errNoPlayer)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}
	g := guestFrom(r)
	kill, n := s.guests.connect(g.ID)
	s.player.SetConnectedGuests(n)
	defer func() { s.player.SetConnectedGuests(s.guests.disconnect(g.ID, kill)) }()

	ch, cancel := s.player.Subscribe()
	defer cancel()

	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	ping := time.NewTicker(ssePing)
	defer ping.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-kill:
			return
		case <-ping.C:
			fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		case st := <-ch:
			data, _ := json.Marshal(st)
			fmt.Fprintf(w, "event: state\ndata: %s\n\n", data)
			flusher.Flush()
		}
	}
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	g := guestFrom(r)
	writeJSON(w, http.StatusOK, map[string]string{"id": g.ID, "name": g.Name})
}

func (s *Server) setMe(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := decode(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, errNameInvalid)
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" || len([]rune(name)) > maxNameLen {
		writeError(w, http.StatusBadRequest, errNameInvalid)
		return
	}
	g := guestFrom(r)
	s.guests.setName(g.ID, name)
	writeJSON(w, http.StatusOK, map[string]string{"id": g.ID, "name": name})
}

func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		writeJSON(w, http.StatusOK, []source.Track{})
		return
	}
	tracks, err := s.player.Search(r.Context(), q, searchLimit)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	if tracks == nil {
		tracks = []source.Track{}
	}
	writeJSON(w, http.StatusOK, tracks)
}

func (s *Server) request(w http.ResponseWriter, r *http.Request) {
	var t source.Track
	if err := decode(r, &t); err != nil || t.ID == "" {
		writeError(w, http.StatusBadRequest, errTrackNeeded)
		return
	}
	if err := s.player.Request(r.Context(), t, guestFrom(r)); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, s.player.State())
}

func (s *Server) vote(w http.ResponseWriter, r *http.Request) {
	if err := s.player.Vote(chi.URLParam(r, "id"), guestFrom(r).ID); err != nil {
		writeError(w, http.StatusNotFound, errNotInQueue)
		return
	}
	writeJSON(w, http.StatusOK, s.player.State())
}

func (s *Server) skip(w http.ResponseWriter, r *http.Request) {
	skipped := s.player.VoteSkip(r.Context(), guestFrom(r).ID)
	writeJSON(w, http.StatusOK, map[string]any{"skipped": skipped, "state": s.player.State()})
}

func (s *Server) artwork(w http.ResponseWriter, r *http.Request) {
	ap, ok := s.player.ActiveSource().(source.ArtworkProvider)
	if !ok {
		http.NotFound(w, r)
		return
	}
	data, mime, err := ap.Artwork(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Write(data)
}

// --- helpers ---

func spa(dist fs.FS) http.HandlerFunc {
	files := http.FileServer(http.FS(dist))
	return func(w http.ResponseWriter, req *http.Request) {
		name := strings.TrimPrefix(path.Clean(req.URL.Path), "/")
		if name != "" {
			if f, err := dist.Open(name); err == nil {
				f.Close()
				files.ServeHTTP(w, req)
				return
			}
		}
		index, err := fs.ReadFile(dist, "index.html")
		if err != nil {
			http.Error(w, "guest UI not built: run `npm run build` in ./web", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(index)
	}
}

func decode(r *http.Request, v any) error {
	defer r.Body.Close()
	if err := json.NewDecoder(http.MaxBytesReader(nil, r.Body, requestBodyKB<<10)).Decode(v); err != nil {
		return errors.New("invalid JSON body")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func newGuestID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
