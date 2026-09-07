package spotify

import (
	"errors"
	"testing"
	"time"

	"github.com/zmb3/spotify/v2"
	"golang.org/x/oauth2"
)

func playing(id string, playing bool, progressMs int) *spotify.CurrentlyPlaying {
	return &spotify.CurrentlyPlaying{
		Playing:  playing,
		Progress: spotify.Numeric(progressMs),
		Item:     &spotify.FullTrack{SimpleTrack: spotify.SimpleTrack{ID: spotify.ID(id), Name: id}},
	}
}

func TestMapStatusEndDetection(t *testing.T) {
	cases := []struct {
		name      string
		cp        *spotify.CurrentlyPlaying
		current   spotify.ID
		armed     bool
		wantEnded bool
		wantArmed bool
	}{
		{"204 with nothing of ours", nil, "", false, false, false},
		{"204 after our track played", &spotify.CurrentlyPlaying{}, "a", true, true, false},
		{"host playing their own stuff", playing("x", true, 5000), "", false, false, false},
		{"stale old track right after Play", playing("old", false, 0), "a", false, false, false},
		{"our track starts playing -> armed", playing("a", true, 1200), "a", false, false, true},
		{"user paused mid-track", playing("a", false, 30000), "a", true, false, true},
		{"normal end: stopped at 0", playing("a", false, 0), "a", true, true, false},
		{"pause at 0 before ever playing is not an end", playing("a", false, 0), "a", false, false, false},
		{"autoplay took over", playing("z", true, 3000), "a", true, true, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			pb, armed := mapStatus(c.cp, c.current, c.armed)
			if pb.Ended != c.wantEnded || armed != c.wantArmed {
				t.Fatalf("ended=%v armed=%v, want %v %v", pb.Ended, armed, c.wantEnded, c.wantArmed)
			}
		})
	}
	pb, _ := mapStatus(playing("a", true, 1500), "a", true)
	if pb.Track == nil || pb.Track.ID != "a" || !pb.Playing || pb.Position != 1500*time.Millisecond {
		t.Fatalf("mapping: %+v", pb)
	}
}

func TestMapTrackAndArtwork(t *testing.T) {
	ft := &spotify.FullTrack{
		SimpleTrack: spotify.SimpleTrack{
			ID: "id1", Name: "Song", Duration: 200000,
			Artists:      []spotify.SimpleArtist{{Name: "A"}, {Name: "B"}},
			ExternalURLs: map[string]string{"spotify": "https://open.spotify.com/track/id1"},
		},
		Album: spotify.SimpleAlbum{Name: "Album", Images: []spotify.Image{
			{Height: 640, URL: "big"}, {Height: 300, URL: "mid"}, {Height: 64, URL: "small"},
		}},
	}
	tr := mapTrack(ft)
	if tr.Artist != "A, B" || tr.ArtworkURL != "mid" || tr.ExternalURL == "" || tr.Duration != 200*time.Second {
		t.Fatalf("%+v", tr)
	}
	if pickImage([]spotify.Image{{Height: 64, URL: "s"}, {Height: 128, URL: "m"}}) != "m" {
		t.Fatal("should fall back to the largest when none reach the minimum")
	}
	if pickImage(nil) != "" {
		t.Fatal("no images -> empty")
	}
}

type stubTS struct {
	tok *oauth2.Token
	err error
}

func (s stubTS) Token() (*oauth2.Token, error) { return s.tok, s.err }

func TestSavingTokenSourceSavesOnlyOnChange(t *testing.T) {
	saves := 0
	initial := &oauth2.Token{AccessToken: "a1", RefreshToken: "r1"}
	stub := &stubTS{tok: initial}
	ts := &savingTokenSource{src: stub, last: initial, save: func(*oauth2.Token) { saves++ }}

	ts.Token()
	ts.Token()
	if saves != 0 {
		t.Fatalf("unchanged token saved %d times", saves)
	}
	stub.tok = &oauth2.Token{AccessToken: "a2", RefreshToken: "r2"}
	ts.Token()
	ts.Token()
	if saves != 1 {
		t.Fatalf("rotated token saved %d times, want 1", saves)
	}
	stub.err = errors.New("boom")
	if _, err := ts.Token(); err == nil || saves != 1 {
		t.Fatal("error must propagate without saving")
	}
}
