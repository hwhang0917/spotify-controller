package server

import (
	"sort"
	"sync"
	"time"
)

// Guests tracks who is here: cookie ID -> display name, open SSE connections,
// and a block list. In-memory except the block list, which the app persists.
// ponytail: identity is a cookie, so a blocked guest can clear cookies and come
// back as someone new. Invites/RBAC close that hole later.
type Guests struct {
	mu       sync.Mutex
	guests   map[string]*guest
	blocked  map[string]struct{}
	onChange func() // called after any change, outside the lock
}

type guest struct {
	name     string
	lastSeen time.Time
	conns    map[chan struct{}]struct{} // one kill channel per SSE connection
}

// GuestInfo is what the admin sees.
type GuestInfo struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Connections int       `json:"connections"`
	Blocked     bool      `json:"blocked"`
	LastSeen    time.Time `json:"lastSeen"`
}

func NewGuests(blocked []string, onChange func()) *Guests {
	g := &Guests{guests: map[string]*guest{}, blocked: map[string]struct{}{}, onChange: onChange}
	for _, id := range blocked {
		g.blocked[id] = struct{}{}
	}
	return g
}

func (g *Guests) changed() {
	if g.onChange != nil {
		g.onChange()
	}
}

func (g *Guests) get(id string) *guest {
	gu, ok := g.guests[id]
	if !ok {
		gu = &guest{conns: map[chan struct{}]struct{}{}}
		g.guests[id] = gu
	}
	gu.lastSeen = time.Now()
	return gu
}

// seen records a visit and returns the name (empty if none yet).
func (g *Guests) seen(id string) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.get(id).name
}

func (g *Guests) setName(id, name string) {
	g.mu.Lock()
	g.get(id).name = name
	g.mu.Unlock()
	g.changed()
}

func (g *Guests) isBlocked(id string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	_, ok := g.blocked[id]
	return ok
}

// connect registers an SSE connection. Returns a kill channel (closed on kick)
// and the number of distinct connected guests.
func (g *Guests) connect(id string) (kill chan struct{}, connected int) {
	g.mu.Lock()
	kill = make(chan struct{})
	g.get(id).conns[kill] = struct{}{}
	connected = g.connectedLocked()
	g.mu.Unlock()
	g.changed()
	return kill, connected
}

func (g *Guests) disconnect(id string, kill chan struct{}) int {
	g.mu.Lock()
	if gu, ok := g.guests[id]; ok {
		delete(gu.conns, kill)
	}
	n := g.connectedLocked()
	g.mu.Unlock()
	g.changed()
	return n
}

func (g *Guests) connectedLocked() int {
	n := 0
	for _, gu := range g.guests {
		if len(gu.conns) > 0 {
			n++
		}
	}
	return n
}

// Connected returns the number of distinct guests with an open stream.
func (g *Guests) Connected() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.connectedLocked()
}

// List returns every guest seen, blocked ones included, most recent first.
func (g *Guests) List() []GuestInfo {
	g.mu.Lock()
	defer g.mu.Unlock()
	out := make([]GuestInfo, 0, len(g.guests))
	for id, gu := range g.guests {
		_, blocked := g.blocked[id]
		out = append(out, GuestInfo{ID: id, Name: gu.name, Connections: len(gu.conns), Blocked: blocked, LastSeen: gu.lastSeen})
	}
	for id := range g.blocked {
		if _, seen := g.guests[id]; !seen {
			out = append(out, GuestInfo{ID: id, Blocked: true})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].LastSeen.After(out[j].LastSeen) })
	return out
}

// Kick closes the guest's open streams. They can reconnect.
func (g *Guests) Kick(id string) {
	g.mu.Lock()
	if gu, ok := g.guests[id]; ok {
		for kill := range gu.conns {
			close(kill)
		}
		gu.conns = map[chan struct{}]struct{}{}
	}
	g.mu.Unlock()
	g.changed()
}

// Remove kicks the guest and forgets their name; they start over at the name gate.
func (g *Guests) Remove(id string) {
	g.Kick(id)
	g.mu.Lock()
	delete(g.guests, id)
	g.mu.Unlock()
	g.changed()
}

// SetBlocked adds or removes the guest from the block list; blocking also kicks.
func (g *Guests) SetBlocked(id string, blocked bool) {
	g.mu.Lock()
	if blocked {
		g.blocked[id] = struct{}{}
	} else {
		delete(g.blocked, id)
	}
	g.mu.Unlock()
	if blocked {
		g.Kick(id)
	}
	g.changed()
}

func (g *Guests) Blocked() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	out := make([]string, 0, len(g.blocked))
	for id := range g.blocked {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}
