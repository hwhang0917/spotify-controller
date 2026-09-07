package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func open(t *testing.T) *Store {
	t.Helper()
	p := filepath.Join(t.TempDir(), "sub", "vibe.db")
	s, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	if fi, err := os.Stat(p); err != nil || fi.Mode().Perm() != 0o600 {
		t.Fatalf("db perms: %v %v", fi, err)
	}
	return s
}

func TestSettingsRoundTrip(t *testing.T) {
	s := open(t)
	if _, err := s.Get("config"); err != ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
	if err := s.Set("config", []byte(`{"port":1}`)); err != nil {
		t.Fatal(err)
	}
	if err := s.Set("config", []byte(`{"port":2}`)); err != nil {
		t.Fatal(err)
	}
	v, err := s.Get("config")
	if err != nil || string(v) != `{"port":2}` {
		t.Fatalf("get: %s %v", v, err)
	}
}

func TestGuests(t *testing.T) {
	s := open(t)
	id := Hash("cookie-1")
	now := time.Now()
	g, err := s.Seen(id, now)
	if err != nil || g.ID != id || g.Name != "" || g.Blocked || g.Admitted {
		t.Fatalf("seen: %+v %v", g, err)
	}
	s.SetGuestName(id, "Kim")
	s.SetGuestAdmitted(id, true)
	g, _ = s.Seen(id, now.Add(time.Second))
	if g.Name != "Kim" || !g.Admitted || !g.LastSeen.After(g.FirstSeen) {
		t.Fatalf("after updates: %+v", g)
	}
	// blocking an id never seen creates the row so the block persists
	s.SetGuestBlocked(Hash("stranger"), true)
	list, _ := s.Guests()
	if len(list) != 2 {
		t.Fatalf("list: %+v", list)
	}
	s.DeleteGuest(id)
	list, _ = s.Guests()
	if len(list) != 1 || !list[0].Blocked {
		t.Fatalf("after delete: %+v", list)
	}
	s.AdmitAll()
	list, _ = s.Guests()
	if !list[0].Admitted {
		t.Fatal("admit all")
	}
}

func TestInvitations(t *testing.T) {
	s := open(t)
	now := time.Now()
	in, err := s.CreateInvitation("ABCD-EFGH", now.Add(time.Hour))
	if err != nil || in.Label != "EFGH" || in.ID == 0 {
		t.Fatalf("create: %+v %v", in, err)
	}
	if err := s.RedeemInvitation("wrong", "g1", now); err != ErrInvalidInvitation {
		t.Fatalf("wrong code: %v", err)
	}
	if err := s.RedeemInvitation("ABCD-EFGH", "g1", now); err != nil {
		t.Fatal(err)
	}
	// single use: the same guest may reopen the link, anyone else is refused
	if err := s.RedeemInvitation("ABCD-EFGH", "g1", now); err != nil {
		t.Fatalf("same guest again: %v", err)
	}
	if err := s.RedeemInvitation("ABCD-EFGH", "g2", now); err != ErrInvalidInvitation {
		t.Fatalf("second guest: %v", err)
	}
	if err := s.RedeemInvitation("ABCD-EFGH", "g1", now.Add(2*time.Hour)); err != ErrInvalidInvitation {
		t.Fatalf("expired: %v", err)
	}
	list, _ := s.Invitations()
	if len(list) != 1 || list[0].Uses != 1 {
		t.Fatalf("list: %+v", list)
	}
	s.RevokeInvitation(in.ID)
	if err := s.RedeemInvitation("ABCD-EFGH", "g1", now); err != ErrInvalidInvitation {
		t.Fatalf("revoked: %v", err)
	}
	// the code itself is never stored
	var stored string
	s.db.QueryRow(`SELECT code_hash FROM invitations`).Scan(&stored)
	if stored == "ABCD-EFGH" || stored != Hash("ABCD-EFGH") {
		t.Fatalf("code stored in the clear? %q", stored)
	}
}

func TestQueueRoundTrip(t *testing.T) {
	s := open(t)
	at := time.Now().Truncate(time.Millisecond)
	items := []QueueRow{
		{ID: "q1", Track: json.RawMessage(`{"id":"a","title":"A"}`), RequestedBy: "g1", RequestedByName: "Kim", RequestedAt: at, Rank: 1, Votes: []string{"g1", "g2"}},
		{ID: "q2", Track: json.RawMessage(`{"id":"b"}`), RequestedBy: "g2", RequestedByName: "Lee", RequestedAt: at, Votes: []string{"g2"}},
	}
	if err := s.SaveQueue(items); err != nil {
		t.Fatal(err)
	}
	got, err := s.LoadQueue()
	if err != nil || len(got) != 2 || got[0].ID != "q1" || got[0].Rank != 1 || len(got[0].Votes) != 2 || string(got[1].Track) != `{"id":"b"}` || !got[0].RequestedAt.Equal(at) {
		t.Fatalf("load: %+v %v", got, err)
	}
	if err := s.SaveQueue(nil); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.LoadQueue(); len(got) != 0 {
		t.Fatalf("expected empty, got %+v", got)
	}
}

func TestPlayHistory(t *testing.T) {
	s := open(t)
	now := time.Now()
	a, b := json.RawMessage(`{"id":"a","title":"A"}`), json.RawMessage(`{"id":"b","title":"B"}`)
	s.RecordPlay("local", "a", a, now)
	s.RecordPlay("local", "b", b, now.Add(time.Minute))
	s.RecordPlay("local", "a", json.RawMessage(`{"id":"a","title":"A2"}`), now.Add(2*time.Minute))
	s.RecordPlay("spotify", "z", json.RawMessage(`{"id":"z"}`), now)

	top, err := s.TopTracks("local", 10)
	if err != nil || len(top) != 2 || string(top[0]) != `{"id":"a","title":"A2"}` || string(top[1]) != string(b) {
		t.Fatalf("top: %s %v", top, err)
	}
	if top, _ := s.TopTracks("local", 1); len(top) != 1 {
		t.Fatal("limit")
	}
	if top, _ := s.TopTracks("youtube", 10); len(top) != 0 {
		t.Fatal("other source should be empty")
	}
	if err := s.ClearPlays(); err != nil {
		t.Fatal(err)
	}
	if top, _ := s.TopTracks("local", 10); len(top) != 0 {
		t.Fatalf("history should be gone: %s", top)
	}
}
