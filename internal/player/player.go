// Package player is the core: it owns the queue and votes, drives whichever
// Source is active, and fans state out to subscribers (guest SSE, admin window).
package player

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/hwhang0917/vibe-music/internal/source"
)

// playGrace is how long after Play() we ignore "nothing loaded" from Status():
// Spotify's player state lags a poll or two behind the command.
// ponytail: fixed grace; make it per-source if a backend needs more.
const playGrace = 3 * time.Second

// Guest is whoever is acting: a cookie ID plus the display name they chose.
type Guest struct {
	ID   string
	Name string
}

type QueueItem struct {
	ID              string       `json:"id"`
	Track           source.Track `json:"track"`
	RequestedBy     string       `json:"-"`
	RequestedByName string       `json:"requestedBy"`
	RequestedAt     time.Time    `json:"requestedAt"`
	Votes           int          `json:"votes"`
	// Mine is set per recipient by the server (the requester's cookie ID must not leak).
	Mine bool `json:"mine,omitempty"`
	// rank > 0 means the admin placed this item by hand; ranked items come
	// first in rank order and votes no longer move them. Unranked requests
	// below still sort by votes. ponytail: "admin curates the top, guests vote
	// on the rest"; a full manual mode can replace this if it confuses people.
	rank  int
	votes map[string]struct{}
}

// Event is a one-shot notice attached to a single state frame so UIs can
// toast what the host just did.
type Event struct {
	Type     string        `json:"type"` // "seek", "queue_moved", "queue_removed", "skipped"
	Title    string        `json:"title,omitempty"`
	Position time.Duration `json:"position,omitempty"`
}

type NowPlaying struct {
	Track       source.Track  `json:"track"`
	Playing     bool          `json:"playing"`
	Position    time.Duration `json:"position"`
	At          time.Time     `json:"at"`
	RequestedBy string        `json:"requestedBy,omitempty"`
}

type SourceInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// State is the snapshot guests and the admin window render.
type State struct {
	Source        *SourceInfo `json:"source"`
	NowPlaying    *NowPlaying `json:"nowPlaying"`
	Queue         []QueueItem `json:"queue"`
	SkipVotes     int         `json:"skipVotes"`
	SkipThreshold int         `json:"skipThreshold"`
	Volume        int         `json:"volume"`
	Guests        int         `json:"guests"`
	Event         *Event      `json:"event,omitempty"`
}

type Options struct {
	Sources      []source.Source
	PollInterval time.Duration // default 1s
	SkipRatio    float64       // fraction of connected guests needed to skip, default 0.5
}

type Player struct {
	mu        sync.Mutex
	sources   map[string]source.Source
	order     []SourceInfo
	active    source.Source
	queue     []*QueueItem
	now       source.Playback
	requester string // display name of who requested the current track
	skipVotes map[string]struct{}
	guests    int
	volume    int
	lastPlay  time.Time
	subs      map[chan State]struct{}
	lastSig   string
	pending   *Event // attached to the next broadcast, which is then forced
	persist   func([]PersistedItem)
	lastQueue string
	poll      time.Duration
	skipRatio float64
}

func New(opts Options) *Player {
	p := &Player{
		sources:   map[string]source.Source{},
		skipVotes: map[string]struct{}{},
		subs:      map[chan State]struct{}{},
		volume:    100,
		poll:      opts.PollInterval,
		skipRatio: opts.SkipRatio,
	}
	if p.poll == 0 {
		p.poll = time.Second
	}
	if p.skipRatio == 0 {
		p.skipRatio = 0.5
	}
	for _, s := range opts.Sources {
		p.sources[s.ID()] = s
		p.order = append(p.order, SourceInfo{ID: s.ID(), Name: s.Name()})
	}
	return p
}

// Run polls the active source until ctx is done.
func (p *Player) Run(ctx context.Context) {
	t := time.NewTicker(p.poll)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			p.Tick(ctx)
		}
	}
}

// Tick is one poll iteration. Status() runs outside the lock: for Spotify it
// is a network call and must not stall guest requests.
func (p *Player) Tick(ctx context.Context) {
	p.mu.Lock()
	src := p.active
	p.mu.Unlock()
	if src == nil {
		return
	}
	st, err := src.Status(ctx)
	if err != nil {
		log.Println("player: status:", err)
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	if p.active != src {
		return // switched while we were polling
	}
	p.now = st
	idle := st.Track == nil && !st.Playing && time.Since(p.lastPlay) > playGrace
	if st.Ended || (idle && len(p.queue) > 0) {
		p.advance(ctx)
	}
	p.broadcastIfChanged()
}

func (p *Player) Sources() []SourceInfo { return p.order }

func (p *Player) ActiveSource() source.Source {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.active
}

// SetSource switches backends. Sources are mutually exclusive, so the queue and
// votes are cleared and the old source is fully deactivated first.
func (p *Player) SetSource(ctx context.Context, id string) error {
	next, ok := p.sources[id]
	if !ok {
		return fmt.Errorf("unknown_source: %s", id)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.active != nil {
		if err := p.active.Deactivate(ctx); err != nil {
			log.Println("player: deactivate:", err)
		}
	}
	p.active = nil
	p.queue = nil
	p.now = source.Playback{}
	p.requester = ""
	p.skipVotes = map[string]struct{}{}
	if err := next.Activate(ctx); err != nil {
		p.broadcastIfChanged()
		return err
	}
	p.active = next
	_ = next.SetVolume(ctx, p.volume)
	p.broadcastIfChanged()
	return nil
}

func (p *Player) Search(ctx context.Context, q string, limit int) ([]source.Track, error) {
	src := p.ActiveSource()
	if src == nil {
		return nil, errors.New("no active source")
	}
	return src.Search(ctx, q, limit)
}

// Request queues a track. Requesting something already queued counts as an upvote.
// If nothing is playing the track starts right away.
func (p *Player) Request(ctx context.Context, t source.Track, g Guest) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.active == nil {
		return errors.New("no active source")
	}
	for _, it := range p.queue {
		if it.Track.ID == t.ID {
			it.votes[g.ID] = struct{}{}
			p.sortQueue()
			p.broadcastIfChanged()
			return nil
		}
	}
	p.queue = append(p.queue, &QueueItem{
		ID:              newID(),
		Track:           t,
		RequestedBy:     g.ID,
		RequestedByName: g.Name,
		RequestedAt:     time.Now(),
		votes:           map[string]struct{}{g.ID: {}},
	})
	p.sortQueue()
	if p.now.Track == nil && !p.now.Playing && time.Since(p.lastPlay) > playGrace {
		p.advance(ctx)
	}
	p.broadcastIfChanged()
	return nil
}

func (p *Player) Vote(itemID, guestID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, it := range p.queue {
		if it.ID == itemID {
			it.votes[guestID] = struct{}{}
			p.sortQueue()
			p.broadcastIfChanged()
			return nil
		}
	}
	return ErrNotInQueue
}

// ErrNotOwner is returned when a guest tries to remove someone else's request.
var ErrNotOwner = errors.New("not your request")

// ErrNotInQueue is returned for unknown queue items.
var ErrNotInQueue = errors.New("not in queue")

// Remove drops a queued item. Guests may only remove their own requests;
// admin passes admin=true to remove anything.
func (p *Player) Remove(itemID, guestID string, admin bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i, it := range p.queue {
		if it.ID != itemID {
			continue
		}
		if !admin && it.RequestedBy != guestID {
			return ErrNotOwner
		}
		p.queue = append(p.queue[:i], p.queue[i+1:]...)
		if admin {
			p.pending = &Event{Type: "queue_removed", Title: it.Track.Title}
		}
		p.broadcastIfChanged()
		return nil
	}
	return ErrNotInQueue
}

// VoteSkip records a skip vote and advances when the threshold is met.
// Returns whether the track was skipped.
func (p *Player) VoteSkip(ctx context.Context, guestID string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.now.Track == nil {
		return false
	}
	p.skipVotes[guestID] = struct{}{}
	skipped := len(p.skipVotes) >= p.skipThreshold()
	if skipped {
		p.advance(ctx)
	}
	p.broadcastIfChanged()
	return skipped
}

// Move places a queued item at index (0 = next up). All current items become
// admin-ranked in their resulting order; see QueueItem.rank.
func (p *Player) Move(itemID string, index int) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	from := -1
	for i, it := range p.queue {
		if it.ID == itemID {
			from = i
		}
	}
	if from < 0 {
		return ErrNotInQueue
	}
	index = max(0, min(len(p.queue)-1, index))
	it := p.queue[from]
	rest := append(append([]*QueueItem{}, p.queue[:from]...), p.queue[from+1:]...)
	p.queue = append(append(append([]*QueueItem{}, rest[:index]...), it), rest[index:]...)
	for i, q := range p.queue {
		q.rank = i + 1
	}
	p.pending = &Event{Type: "queue_moved", Title: it.Track.Title}
	p.broadcastIfChanged()
	return nil
}

// Seek moves the current track to pos and tells everyone.
func (p *Player) Seek(ctx context.Context, pos time.Duration) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.active == nil || p.now.Track == nil {
		return errors.New("nothing playing")
	}
	if err := p.active.Seek(ctx, pos); err != nil {
		return err
	}
	if st, err := p.active.Status(ctx); err == nil {
		p.now = st
	} else {
		p.now.Position, p.now.At = pos, time.Now()
	}
	p.pending = &Event{Type: "seek", Position: pos}
	p.broadcastIfChanged()
	return nil
}

// Skip is the admin override.
func (p *Player) Skip(ctx context.Context) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.now.Track != nil {
		p.pending = &Event{Type: "skipped", Title: p.now.Track.Title}
	}
	p.advance(ctx)
	p.broadcastIfChanged()
}

func (p *Player) Pause(ctx context.Context) error {
	return p.control(func(s source.Source) error { return s.Pause(ctx) })
}

func (p *Player) Resume(ctx context.Context) error {
	return p.control(func(s source.Source) error { return s.Resume(ctx) })
}

func (p *Player) SetVolume(ctx context.Context, pct int) error {
	pct = max(0, min(100, pct))
	p.mu.Lock()
	p.volume = pct
	p.mu.Unlock()
	return p.control(func(s source.Source) error { return s.SetVolume(ctx, pct) })
}

func (p *Player) control(fn func(source.Source) error) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.active == nil {
		return errors.New("no active source")
	}
	err := fn(p.active)
	if st, serr := p.active.Status(context.Background()); serr == nil {
		p.now = st
	}
	p.broadcastIfChanged()
	return err
}

// RemoveGuest drops everything a guest contributed: their queued requests,
// their upvotes, and their skip vote. Used when the admin removes or blocks them.
func (p *Player) RemoveGuest(guestID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	kept := p.queue[:0]
	for _, it := range p.queue {
		delete(it.votes, guestID)
		if it.RequestedBy != guestID {
			kept = append(kept, it)
		}
	}
	p.queue = kept
	delete(p.skipVotes, guestID)
	p.sortQueue()
	p.broadcastIfChanged()
}

// SetSkipRatio changes the fraction of connected guests needed to skip.
func (p *Player) SetSkipRatio(r float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if r > 0 && r <= 1 {
		p.skipRatio = r
	}
	p.broadcastIfChanged()
}

// SetConnectedGuests updates the denominator for the skip threshold.
func (p *Player) SetConnectedGuests(n int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.guests = n
	p.broadcastIfChanged()
}

func (p *Player) State() State {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.snapshot()
}

// PersistedItem is a queue entry as stored across relaunches.
type PersistedItem struct {
	ID              string
	Track           source.Track
	RequestedBy     string
	RequestedByName string
	RequestedAt     time.Time
	Rank            int
	Votes           []string
}

// SetPersister registers a callback that receives the whole queue whenever it
// changes. It runs under the player lock, so it must be quick (a local
// SQLite write is). ponytail: synchronous; hand it a channel if it ever blocks.
func (p *Player) SetPersister(fn func([]PersistedItem)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.persist = fn
}

// Restore loads a previously persisted queue (at startup, before Run).
func (p *Player) Restore(items []PersistedItem) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.queue = p.queue[:0]
	for _, it := range items {
		votes := make(map[string]struct{}, len(it.Votes))
		for _, v := range it.Votes {
			votes[v] = struct{}{}
		}
		p.queue = append(p.queue, &QueueItem{
			ID: it.ID, Track: it.Track, RequestedBy: it.RequestedBy, RequestedByName: it.RequestedByName,
			RequestedAt: it.RequestedAt, rank: it.Rank, votes: votes,
		})
	}
	p.sortQueue()
	p.lastQueue = p.queueSig()
	p.broadcastIfChanged()
}

func (p *Player) persisted() []PersistedItem {
	out := make([]PersistedItem, 0, len(p.queue))
	for _, it := range p.queue {
		votes := make([]string, 0, len(it.votes))
		for v := range it.votes {
			votes = append(votes, v)
		}
		sort.Strings(votes)
		out = append(out, PersistedItem{
			ID: it.ID, Track: it.Track, RequestedBy: it.RequestedBy, RequestedByName: it.RequestedByName,
			RequestedAt: it.RequestedAt, Rank: it.rank, Votes: votes,
		})
	}
	return out
}

func (p *Player) queueSig() string {
	sig := ""
	for _, it := range p.queue {
		sig += fmt.Sprintf("%s:%d:%d,", it.ID, it.rank, len(it.votes))
	}
	return sig
}

// Subscribe returns a channel receiving every state change. Slow readers drop
// frames rather than block the player.
func (p *Player) Subscribe() (<-chan State, func()) {
	ch := make(chan State, 8)
	p.mu.Lock()
	p.subs[ch] = struct{}{}
	ch <- p.snapshot()
	p.mu.Unlock()
	return ch, func() {
		p.mu.Lock()
		delete(p.subs, ch)
		p.mu.Unlock()
	}
}

// --- internals (caller holds p.mu) ---

// advance plays the queue head or stops when the queue is empty.
func (p *Player) advance(ctx context.Context) {
	p.skipVotes = map[string]struct{}{}
	if p.active == nil {
		return
	}
	if len(p.queue) == 0 {
		_ = p.active.Stop(ctx)
		p.now = source.Playback{At: time.Now()}
		p.requester = ""
		return
	}
	head := p.queue[0]
	p.queue = p.queue[1:]
	if err := p.active.Play(ctx, head.Track.ID); err != nil {
		log.Println("player: play:", err)
		return
	}
	p.lastPlay = time.Now()
	p.requester = head.RequestedByName
	p.now = source.Playback{Track: &head.Track, Playing: true, At: p.lastPlay}
}

func (p *Player) sortQueue() {
	sort.SliceStable(p.queue, func(i, j int) bool {
		a, b := p.queue[i], p.queue[j]
		if (a.rank > 0) != (b.rank > 0) {
			return a.rank > 0 // admin-ranked first
		}
		if a.rank > 0 {
			return a.rank < b.rank
		}
		if va, vb := len(a.votes), len(b.votes); va != vb {
			return va > vb
		}
		return a.RequestedAt.Before(b.RequestedAt)
	})
}

func (p *Player) skipThreshold() int {
	return max(1, int(math.Ceil(p.skipRatio*float64(p.guests))))
}

func (p *Player) snapshot() State {
	s := State{
		Queue:         make([]QueueItem, 0, len(p.queue)),
		SkipVotes:     len(p.skipVotes),
		SkipThreshold: p.skipThreshold(),
		Volume:        p.volume,
		Guests:        p.guests,
	}
	if p.active != nil {
		s.Source = &SourceInfo{ID: p.active.ID(), Name: p.active.Name()}
	}
	if p.now.Track != nil {
		s.NowPlaying = &NowPlaying{
			Track:       *p.now.Track,
			Playing:     p.now.Playing,
			Position:    p.now.Position,
			At:          p.now.At,
			RequestedBy: p.requester,
		}
	}
	for _, it := range p.queue {
		c := *it
		c.Votes = len(it.votes)
		s.Queue = append(s.Queue, c)
	}
	return s
}

// signature excludes Position so a playing track does not broadcast every
// tick; clients interpolate from Position+At of the last frame. Duration is
// included because local files only learn it once decoding starts.
func (s State) signature() string {
	sig := fmt.Sprintf("%v|%d|%d|%d|%d|", s.Source, s.SkipVotes, s.SkipThreshold, s.Volume, s.Guests)
	if s.NowPlaying != nil {
		sig += fmt.Sprintf("%s|%v|%d|", s.NowPlaying.Track.ID, s.NowPlaying.Playing, s.NowPlaying.Track.Duration)
	}
	for _, it := range s.Queue {
		sig += fmt.Sprintf("%s:%d,", it.ID, it.Votes)
	}
	return sig
}

func (p *Player) broadcastIfChanged() {
	if q := p.queueSig(); q != p.lastQueue {
		p.lastQueue = q
		if p.persist != nil {
			p.persist(p.persisted())
		}
	}
	s := p.snapshot()
	sig := s.signature()
	if sig == p.lastSig && p.pending == nil {
		return
	}
	p.lastSig = sig
	s.Event, p.pending = p.pending, nil
	for ch := range p.subs {
		select {
		case ch <- s:
		default:
		}
	}
}

func newID() string {
	b := make([]byte, 6)
	rand.Read(b)
	return hex.EncodeToString(b)
}
