package server

import (
	"crypto/rand"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/hwhang0917/vibe-music/internal/store"
)

// Guests is the server's view of who is here. Names, blocks, admissions and
// invitations live in the store (keyed by hashed cookie ID); open SSE
// connections and the plaintext of invitation codes created this session are
// memory only.
// ponytail: identity is a cookie, so a blocked guest can clear cookies and come
// back as someone new. Invite-only mode closes that hole for new arrivals.
type Guests struct {
	mu         sync.Mutex
	db         *store.Store
	conns      map[string]map[chan struct{}]struct{} // guest id -> kill channels
	known      map[string]bool                       // seen this session (first-sight notification)
	codes      map[int64]string                      // invitation id -> plaintext, this session only
	inviteOnly bool
	onChange   func() // called after any change, outside the lock
}

// GuestInfo is what the admin sees.
type GuestInfo struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Connections int       `json:"connections"`
	Blocked     bool      `json:"blocked"`
	Admitted    bool      `json:"admitted"`
	LastSeen    time.Time `json:"lastSeen"`
}

// Invitation is a store.Invitation plus the code itself when this session created it.
type Invitation struct {
	store.Invitation
	Code string `json:"code,omitempty"`
}

func NewGuests(db *store.Store, inviteOnly bool, onChange func()) *Guests {
	return &Guests{
		db:         db,
		conns:      map[string]map[chan struct{}]struct{}{},
		known:      map[string]bool{},
		codes:      map[int64]string{},
		inviteOnly: inviteOnly,
		onChange:   onChange,
	}
}

func (g *Guests) changed() {
	if g.onChange != nil {
		g.onChange()
	}
}

// seen records a visit and returns the stored guest. A first sighting this
// session notifies so the admin sees the guest before they pick a name.
func (g *Guests) seen(id string) (store.Guest, error) {
	gu, err := g.db.Seen(id, time.Now())
	if err != nil {
		return store.Guest{}, err
	}
	g.mu.Lock()
	first := !g.known[id]
	g.known[id] = true
	g.mu.Unlock()
	if first {
		g.changed()
	}
	return gu, nil
}

func (g *Guests) setName(id, name string) error {
	err := g.db.SetGuestName(id, name)
	g.changed()
	return err
}

// --- invite-only ---

func (g *Guests) InviteOnly() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.inviteOnly
}

// SetInviteOnly toggles the mode. Turning it on admits everyone already known,
// so the room is not emptied; newcomers need a code from then on.
func (g *Guests) SetInviteOnly(on bool) error {
	g.mu.Lock()
	g.inviteOnly = on
	g.mu.Unlock()
	var err error
	if on {
		err = g.db.AdmitAll()
	}
	g.changed()
	return err
}

func (g *Guests) Admit(id string) error {
	err := g.db.SetGuestAdmitted(id, true)
	g.changed()
	return err
}

// Redeem admits the guest if the code is valid.
func (g *Guests) Redeem(code, id string) error {
	if _, err := g.db.Seen(id, time.Now()); err != nil {
		return err
	}
	if err := g.db.RedeemInvitation(strings.TrimSpace(code), time.Now()); err != nil {
		return err
	}
	return g.Admit(id)
}

// CreateInvitation returns the plaintext code (shown to the admin) and its record.
func (g *Guests) CreateInvitation(ttl time.Duration) (Invitation, error) {
	code := newCode()
	inv, err := g.db.CreateInvitation(code, time.Now().Add(ttl))
	if err != nil {
		return Invitation{}, err
	}
	g.mu.Lock()
	g.codes[inv.ID] = code
	g.mu.Unlock()
	g.changed()
	return Invitation{Invitation: inv, Code: code}, nil
}

func (g *Guests) Invitations() ([]Invitation, error) {
	list, err := g.db.Invitations()
	if err != nil {
		return nil, err
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	out := make([]Invitation, 0, len(list))
	for _, in := range list {
		out = append(out, Invitation{Invitation: in, Code: g.codes[in.ID]})
	}
	return out, nil
}

func (g *Guests) RevokeInvitation(id int64) error {
	err := g.db.RevokeInvitation(id)
	g.changed()
	return err
}

// codeAlphabet avoids look-alikes (0/O, 1/I/L) since codes get read aloud or typed.
const codeAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

// newCode returns e.g. "K7PM-3QXD".
func newCode() string {
	b := make([]byte, 8)
	rand.Read(b)
	out := make([]byte, 0, 9)
	for i, v := range b {
		if i == 4 {
			out = append(out, '-')
		}
		out = append(out, codeAlphabet[int(v)%len(codeAlphabet)])
	}
	return string(out)
}

// --- connections ---

// connect registers an SSE connection. Returns a kill channel (closed on kick)
// and the number of distinct connected guests.
func (g *Guests) connect(id string) (kill chan struct{}, connected int) {
	g.mu.Lock()
	kill = make(chan struct{})
	if g.conns[id] == nil {
		g.conns[id] = map[chan struct{}]struct{}{}
	}
	g.conns[id][kill] = struct{}{}
	connected = len(g.conns)
	g.mu.Unlock()
	g.changed()
	return kill, connected
}

func (g *Guests) disconnect(id string, kill chan struct{}) int {
	g.mu.Lock()
	if c, ok := g.conns[id]; ok {
		delete(c, kill)
		if len(c) == 0 {
			delete(g.conns, id)
		}
	}
	n := len(g.conns)
	g.mu.Unlock()
	g.changed()
	return n
}

// Connected returns the number of distinct guests with an open stream.
func (g *Guests) Connected() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return len(g.conns)
}

// --- admin ---

// List returns every guest the store knows, most recent first.
func (g *Guests) List() ([]GuestInfo, error) {
	rows, err := g.db.Guests()
	if err != nil {
		return nil, err
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	out := make([]GuestInfo, 0, len(rows))
	for _, r := range rows {
		out = append(out, GuestInfo{
			ID: r.ID, Name: r.Name, Connections: len(g.conns[r.ID]),
			Blocked: r.Blocked, Admitted: r.Admitted, LastSeen: r.LastSeen,
		})
	}
	return out, nil
}

// Kick closes the guest's open streams. They can reconnect.
func (g *Guests) Kick(id string) {
	g.mu.Lock()
	for kill := range g.conns[id] {
		close(kill)
	}
	delete(g.conns, id)
	g.mu.Unlock()
	g.changed()
}

// Remove kicks the guest and forgets them; they start over at the name gate.
func (g *Guests) Remove(id string) error {
	g.Kick(id)
	g.mu.Lock()
	delete(g.known, id)
	g.mu.Unlock()
	err := g.db.DeleteGuest(id)
	g.changed()
	return err
}

// SetBlocked adds or removes the guest from the block list; blocking also kicks.
func (g *Guests) SetBlocked(id string, blocked bool) error {
	err := g.db.SetGuestBlocked(id, blocked)
	if blocked {
		g.Kick(id)
	}
	g.changed()
	return err
}

var errNoStore = errors.New("server: guests need a store")
