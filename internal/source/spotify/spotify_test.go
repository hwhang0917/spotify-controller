package spotify

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/zmb3/spotify/v2"

	"github.com/hwhang0917/vibe-music/internal/source"
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
			Artists:      []spotify.SimpleArtist{{Name: "A", ID: "ar1"}, {Name: "B", ID: "ar2"}},
			ExternalURLs: map[string]string{"spotify": "https://open.spotify.com/track/id1"},
		},
		Album: spotify.SimpleAlbum{ID: "al1", Name: "Album", ReleaseDate: "1981-12-01", Images: []spotify.Image{
			{Height: 640, URL: "big"}, {Height: 300, URL: "mid"}, {Height: 64, URL: "small"},
		}},
	}
	tr := mapTrack(ft)
	if tr.Artist != "A, B" || tr.ArtworkURL != "mid" || tr.ExternalURL == "" || tr.Duration != 200*time.Second {
		t.Fatalf("%+v", tr)
	}
	if tr.ArtistID != "ar1" || tr.AlbumID != "al1" || tr.Year != 1981 {
		t.Fatalf("browse keys: %+v", tr)
	}
	// an album's own track list carries no album: the parent's art and year apply
	st := spotify.SimpleTrack{ID: "id2", Name: "Other"}
	if tr2 := mapSimple(&st, &ft.Album); tr2.ArtworkURL != "mid" || tr2.Year != 1981 || tr2.AlbumID != "al1" || tr2.ArtistID != "" {
		t.Fatalf("mapSimple: %+v", tr2)
	}
	al := mapAlbum(&spotify.SimpleAlbum{ID: "al1", Name: "Album", ReleaseDate: "1981", Artists: []spotify.SimpleArtist{{Name: "A", ID: "ar1"}}})
	if al.Artist != "A" || al.ArtistID != "ar1" || al.Year != 1981 {
		t.Fatalf("mapAlbum: %+v", al)
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

func TestConnectRejectsBadClientIDAndCancels(t *testing.T) {
	opened := 0
	s := New(Options{ClientID: "not-a-real-id", OpenBrowser: func(string) error { opened++; return nil }})
	if err := s.Connect(context.Background()); err != ErrBadClientID || opened != 0 {
		t.Fatalf("bad id: %v opened=%d", err, opened)
	}
	s.SetClientID("0123456789abcdef0123456789abcdef")
	done := make(chan error, 1)
	go func() { done <- s.Connect(context.Background()) }()
	for i := 0; i < 100 && opened == 0; i++ {
		time.Sleep(10 * time.Millisecond)
	}
	s.CancelConnect()
	select {
	case err := <-done:
		if err != ErrLoginCancelled {
			t.Fatalf("cancel: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Connect did not return after cancel")
	}
}

func TestRedirectURIAndBusyPort(t *testing.T) {
	s := New(Options{ClientID: "0123456789abcdef0123456789abcdef", CallbackPort: 27272, OpenBrowser: func(string) error { return nil }})
	if s.RedirectURI() != "http://127.0.0.1:27272/callback" {
		t.Fatalf("redirect: %s", s.RedirectURI())
	}
	// occupy a port, then point Connect at it
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	s.SetCallbackPort(ln.Addr().(*net.TCPAddr).Port)
	if err := s.Connect(context.Background()); !errors.Is(err, ErrCallbackPort) {
		t.Fatalf("busy port: %v", err)
	}
}

func TestWrapAPIPremium(t *testing.T) {
	err := wrapAPI(errors.New("spotify: couldn't decode error: (159) [Active premium subscription required for the owner of the app.]"))
	if !errors.Is(err, ErrPremium) || source.ErrorCode(err, "?") != "spotify_premium_required" {
		t.Fatalf("premium: %v", err)
	}
	if wrapAPI(nil) != nil {
		t.Fatal("nil passes through")
	}
	other := errors.New("boom")
	if wrapAPI(other) != other {
		t.Fatal("unknown errors pass through unchanged")
	}
}
