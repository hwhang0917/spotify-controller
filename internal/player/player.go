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
	votes           map[string]struct{}
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
		return fmt.Errorf("unknown source %q", id)
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
	return errors.New("not in queue")
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

// Skip is the admin override.
func (p *Player) Skip(ctx context.Context) {
	p.mu.Lock()
	defer p.mu.Unlock()
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
		vi, vj := len(p.queue[i].votes), len(p.queue[j].votes)
		if vi != vj {
			return vi > vj
		}
		return p.queue[i].RequestedAt.Before(p.queue[j].RequestedAt)
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
// tick; clients interpolate from Position+At of the last frame.
func (s State) signature() string {
	sig := fmt.Sprintf("%v|%d|%d|%d|%d|", s.Source, s.SkipVotes, s.SkipThreshold, s.Volume, s.Guests)
	if s.NowPlaying != nil {
		sig += fmt.Sprintf("%s|%v|", s.NowPlaying.Track.ID, s.NowPlaying.Playing)
	}
	for _, it := range s.Queue {
		sig += fmt.Sprintf("%s:%d,", it.ID, it.Votes)
	}
	return sig
}

func (p *Player) broadcastIfChanged() {
	s := p.snapshot()
	sig := s.signature()
	if sig == p.lastSig {
		return
	}
	p.lastSig = sig
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
