package youtube

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hwhang0917/vibe-music/internal/source"
)

func TestParseISODuration(t *testing.T) {
	cases := map[string]time.Duration{"PT4M13S": 4*time.Minute + 13*time.Second, "PT1H2M": 62 * time.Minute, "PT45S": 45 * time.Second, "junk": 0}
	for in, want := range cases {
		if got := parseISODuration(in); got != want {
			t.Errorf("%s: %v want %v", in, got, want)
		}
	}
}

func TestSearchMapsAndKeepsRanking(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("key") != "k" {
			t.Errorf("missing key")
		}
		switch r.URL.Path {
		case "/search":
			if r.URL.Query().Get("videoCategoryId") != "10" || r.URL.Query().Get("q") != "lofi" {
				t.Errorf("query: %v", r.URL.Query())
			}
			w.Write([]byte(`{"items":[{"id":{"videoId":"b"}},{"id":{"videoId":"a"}}]}`))
		case "/videos":
			w.Write([]byte(`{"items":[
				{"id":"a","snippet":{"title":"A","channelTitle":"Ch A","thumbnails":{"medium":{"url":"ma"}}},"contentDetails":{"duration":"PT3M"}},
				{"id":"b","snippet":{"title":"B","channelTitle":"Ch B","thumbnails":{"default":{"url":"db"}}},"contentDetails":{"duration":"PT2M30S"}}]}`))
		}
	}))
	defer srv.Close()

	s := New(Options{APIKey: "k", APIURL: srv.URL})
	got, err := s.Search(context.Background(), "lofi", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != "b" || got[1].ID != "a" {
		t.Fatalf("ranking: %+v", got)
	}
	if got[0].Duration != 150*time.Second || got[0].ArtworkURL != "db" || got[0].ExternalURL != watchURL+"b" || got[0].Artist != "Ch B" {
		t.Fatalf("mapping: %+v", got[0])
	}
	if got[1].ArtworkURL != "ma" {
		t.Fatalf("prefers medium thumbnail: %+v", got[1])
	}

	s.SetAPIKey("")
	if _, err := s.Search(context.Background(), "x", 1); err != ErrNoAPIKey {
		t.Fatalf("no key: %v", err)
	}
}

func TestQuotaError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		w.Write([]byte(`{"error":{"code":403,"errors":[{"reason":"quotaExceeded"}],"message":"quota"}}`))
	}))
	defer srv.Close()
	s := New(Options{APIKey: "k", APIURL: srv.URL})
	if _, err := s.Search(context.Background(), "x", 1); err != ErrQuotaExceeded {
		t.Fatalf("want quota error, got %v", err)
	}
}

func TestPlayerCommandsAndStatus(t *testing.T) {
	var cmds []Command
	s := New(Options{APIKey: "k", Send: func(c Command) { cmds = append(cmds, c) }})
	ctx := context.Background()
	if err := s.Play(ctx, "v1"); err != ErrPlayerNotReady {
		t.Fatalf("play before ready: %v", err)
	}
	s.Report(Report{Ready: true, State: -1})
	if err := s.Activate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := s.Play(ctx, "v1"); err != nil {
		t.Fatal(err)
	}
	if cmds[len(cmds)-1].Type != "load" || cmds[len(cmds)-1].VideoID != "v1" {
		t.Fatalf("cmds: %+v", cmds)
	}
	// stale report (before Play): loading, counts as playing within the grace
	pb, _ := s.Status(ctx)
	if pb.Track == nil || pb.Track.ID != "v1" || !pb.Playing || pb.Ended {
		t.Fatalf("loading: %+v", pb)
	}
	s.Report(Report{Ready: true, VideoID: "v1", State: 1, Position: 12, Duration: 200})
	pb, _ = s.Status(ctx)
	if !pb.Playing || pb.Position != 12*time.Second || pb.Track.Duration != 200*time.Second {
		t.Fatalf("playing: %+v", pb)
	}
	s.Report(Report{Ready: true, VideoID: "v1", State: 2, Position: 12, Duration: 200})
	if pb, _ = s.Status(ctx); pb.Playing || pb.Ended {
		t.Fatalf("paused: %+v", pb)
	}
	s.Report(Report{Ready: true, VideoID: "v1", State: 0, Position: 200, Duration: 200})
	if pb, _ = s.Status(ctx); !pb.Ended {
		t.Fatalf("ended: %+v", pb)
	}
	_ = s.Seek(ctx, 30*time.Second)
	_ = s.SetVolume(ctx, 40)
	_ = s.Stop(ctx)
	if cmds[len(cmds)-3].Seconds != 30 || cmds[len(cmds)-2].Volume != 40 || cmds[len(cmds)-1].Type != "stop" {
		t.Fatalf("cmds: %+v", cmds)
	}
	if pb, _ = s.Status(ctx); pb.Track != nil {
		t.Fatalf("idle after stop: %+v", pb)
	}
	_ = source.Track{}
}

func TestKeyRestrictionError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		w.Write([]byte(`{"error":{"code":403,"errors":[{"reason":"forbidden"}],"message":"Requests from referer <empty> are blocked."}}`))
	}))
	defer srv.Close()
	s := New(Options{APIKey: "k", APIURL: srv.URL})
	_, err := s.Search(context.Background(), "x", 1)
	if err != ErrKeyRestricted || source.ErrorCode(err, "?") != "youtube_key_restricted" {
		t.Fatalf("want ErrKeyRestricted, got %v", err)
	}
}
