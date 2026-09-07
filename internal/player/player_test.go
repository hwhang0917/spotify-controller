package player

import (
	"context"
	"testing"
	"time"

	"github.com/hwhang0917/vibe-music/internal/source"
	"github.com/hwhang0917/vibe-music/internal/source/fake"
)

var (
	a  = source.Track{ID: "a", Title: "A"}
	b  = source.Track{ID: "b", Title: "B"}
	c  = source.Track{ID: "c", Title: "C"}
	g1 = Guest{ID: "g1", Name: "Kim"}
	g2 = Guest{ID: "g2", Name: "Lee"}
	g3 = Guest{ID: "g3", Name: "Park"}
)

func setup(t *testing.T) (*Player, *fake.Source) {
	t.Helper()
	f := fake.New(a, b, c)
	p := New(Options{Sources: []source.Source{f}})
	if err := p.SetSource(context.Background(), "fake"); err != nil {
		t.Fatal(err)
	}
	return p, f
}

func queueIDs(s State) []string {
	var ids []string
	for _, it := range s.Queue {
		ids = append(ids, it.Track.ID)
	}
	return ids
}

func eq(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestFirstRequestPlaysImmediately(t *testing.T) {
	p, f := setup(t)
	ctx := context.Background()
	_ = p.Request(ctx, a, g1)
	eq(t, f.Played(), []string{"a"})
	s := p.State()
	if s.NowPlaying == nil || s.NowPlaying.Track.ID != "a" || s.NowPlaying.RequestedBy != "Kim" {
		t.Fatalf("now playing: %+v", s.NowPlaying)
	}
	if len(s.Queue) != 0 {
		t.Fatalf("queue should be empty, got %v", queueIDs(s))
	}
}

func TestVotesOrderQueueThenRequestTime(t *testing.T) {
	p, _ := setup(t)
	ctx := context.Background()
	_ = p.Request(ctx, a, g1) // playing
	_ = p.Request(ctx, b, g1)
	time.Sleep(time.Millisecond)
	_ = p.Request(ctx, c, g2)
	eq(t, queueIDs(p.State()), []string{"b", "c"})

	// duplicate request = upvote, idempotent per guest
	_ = p.Request(ctx, c, g3)
	_ = p.Request(ctx, c, g3)
	s := p.State()
	eq(t, queueIDs(s), []string{"c", "b"})
	if s.Queue[0].Votes != 2 {
		t.Fatalf("votes = %d", s.Queue[0].Votes)
	}

	// explicit vote on b by two more guests overtakes
	bID := s.Queue[1].ID
	_ = p.Vote(bID, g2.ID)
	_ = p.Vote(bID, g3.ID)
	_ = p.Vote(bID, g3.ID)
	s = p.State()
	eq(t, queueIDs(s), []string{"b", "c"})
	if s.Queue[0].Votes != 3 {
		t.Fatalf("votes = %d", s.Queue[0].Votes)
	}
	if err := p.Vote("nope", g1.ID); err == nil {
		t.Fatal("vote on unknown item should fail")
	}
}

func TestAutoAdvanceOnEnded(t *testing.T) {
	p, f := setup(t)
	ctx := context.Background()
	_ = p.Request(ctx, a, g1)
	_ = p.Request(ctx, b, g1)
	p.Tick(ctx)
	eq(t, f.Played(), []string{"a"}) // still playing a
	f.FinishTrack()
	p.Tick(ctx)
	eq(t, f.Played(), []string{"a", "b"})
	f.FinishTrack()
	p.Tick(ctx)
	eq(t, f.Played(), []string{"a", "b"}) // queue empty -> stop
	if p.State().NowPlaying != nil {
		t.Fatal("expected nothing playing")
	}
	if f.Calls[len(f.Calls)-1] != "stop" {
		t.Fatalf("expected stop, calls=%v", f.Calls)
	}
}

func TestSkipThreshold(t *testing.T) {
	p, f := setup(t)
	ctx := context.Background()
	_ = p.Request(ctx, a, g1)
	_ = p.Request(ctx, b, g1)

	p.SetConnectedGuests(4) // ceil(0.5*4) = 2
	if p.State().SkipThreshold != 2 {
		t.Fatalf("threshold = %d", p.State().SkipThreshold)
	}
	if p.VoteSkip(ctx, g1.ID) {
		t.Fatal("one vote should not skip")
	}
	if p.VoteSkip(ctx, g1.ID) {
		t.Fatal("same guest twice should not skip")
	}
	if !p.VoteSkip(ctx, g2.ID) {
		t.Fatal("second guest should skip")
	}
	eq(t, f.Played(), []string{"a", "b"})
	if p.State().SkipVotes != 0 {
		t.Fatal("skip votes should reset on advance")
	}

	p.SetConnectedGuests(0) // min threshold 1
	if p.State().SkipThreshold != 1 {
		t.Fatalf("threshold = %d", p.State().SkipThreshold)
	}
	if !p.VoteSkip(ctx, g3.ID) {
		t.Fatal("single vote should skip with no guests connected")
	}
	if p.VoteSkip(ctx, g3.ID) {
		t.Fatal("nothing playing: no skip")
	}
}

func TestSetSourceClearsQueueAndDeactivatesFirst(t *testing.T) {
	f1 := fake.New(a, b)
	f2 := &named{fake.New(c), "two"}
	p := New(Options{Sources: []source.Source{f1, f2}})
	ctx := context.Background()
	_ = p.SetSource(ctx, "fake")
	_ = p.Request(ctx, a, g1)
	_ = p.Request(ctx, b, g1)
	if err := p.SetSource(ctx, "two"); err != nil {
		t.Fatal(err)
	}
	s := p.State()
	if len(s.Queue) != 0 || s.NowPlaying != nil || s.Source.ID != "two" {
		t.Fatalf("state after switch: %+v", s)
	}
	if f1.Calls[len(f1.Calls)-1] != "deactivate" {
		t.Fatalf("f1 calls: %v", f1.Calls)
	}
	if f2.Calls[0] != "activate" {
		t.Fatalf("f2 calls: %v", f2.Calls)
	}
	if err := p.SetSource(ctx, "nope"); err == nil {
		t.Fatal("unknown source should error")
	}
}

func TestBroadcastOnlyOnChange(t *testing.T) {
	p, _ := setup(t)
	ctx := context.Background()
	ch, cancel := p.Subscribe()
	defer cancel()
	<-ch // initial snapshot
	p.Tick(ctx)
	select {
	case s := <-ch:
		t.Fatalf("unexpected broadcast: %+v", s)
	default:
	}
	_ = p.Request(ctx, a, g1)
	select {
	case s := <-ch:
		if s.NowPlaying == nil || s.NowPlaying.Track.ID != "a" {
			t.Fatalf("bad state: %+v", s)
		}
	default:
		t.Fatal("expected broadcast after request")
	}
}

// named wraps a fake with a different ID so two can coexist.
type named struct {
	*fake.Source
	id string
}

func (n *named) ID() string { return n.id }
