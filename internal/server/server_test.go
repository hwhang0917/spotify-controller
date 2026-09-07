package server

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/hwhang0917/vibe-music/internal/player"
	"github.com/hwhang0917/vibe-music/internal/source"
	"github.com/hwhang0917/vibe-music/internal/source/fake"
)

var dist = fstest.MapFS{
	"index.html":    {Data: []byte("<html>guest</html>")},
	"assets/app.js": {Data: []byte("console.log(1)")},
}

func newTestServer(t *testing.T) (*httptest.Server, *fake.Source) {
	t.Helper()
	f := fake.New(
		source.Track{ID: "a", Title: "Alpha", Artist: "X"},
		source.Track{ID: "b", Title: "Beta", Artist: "Y"},
	)
	p := player.New(player.Options{Sources: []source.Source{f}})
	if err := p.SetSource(context.Background(), "fake"); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(NewHandler(dist, p))
	t.Cleanup(ts.Close)
	return ts, f
}

// client is a guest with a cookie jar.
type client struct {
	t    *testing.T
	base string
	http *http.Client
}

func newClient(t *testing.T, base string) *client {
	jar, _ := (&http.Client{}).Jar, error(nil)
	_ = jar
	c := &http.Client{}
	c.Jar = newJar()
	return &client{t: t, base: base, http: c}
}

func (c *client) do(method, p string, body string) (*http.Response, map[string]any) {
	req, _ := http.NewRequest(method, c.base+p, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer res.Body.Close()
	var out map[string]any
	json.NewDecoder(res.Body).Decode(&out)
	return res, out
}

func TestStaticAndSPA(t *testing.T) {
	ts, _ := newTestServer(t)
	c := newClient(t, ts.URL)
	for _, p := range []string{"/", "/vote/abc"} {
		res, _ := c.do("GET", p, "")
		if res.StatusCode != 200 {
			t.Fatalf("%s: %d", p, res.StatusCode)
		}
	}
	res, _ := c.do("GET", "/assets/app.js", "")
	if res.StatusCode != 200 {
		t.Fatalf("static: %d", res.StatusCode)
	}
	rec := httptest.NewRecorder()
	NewHandler(fstest.MapFS{}, nil).ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("unbuilt UI: %d", rec.Code)
	}
}

func TestGuestFlow(t *testing.T) {
	ts, f := newTestServer(t)
	kim := newClient(t, ts.URL)
	lee := newClient(t, ts.URL)

	// cookie issued on first call; write actions need a name first
	res, me := kim.do("GET", "/api/me", "")
	if res.StatusCode != 200 || me["id"] == "" || me["name"] != "" {
		t.Fatalf("me: %d %v", res.StatusCode, me)
	}
	res, _ = kim.do("POST", "/api/queue", `{"id":"a"}`)
	if res.StatusCode != 400 {
		t.Fatalf("request without name: %d", res.StatusCode)
	}
	res, _ = kim.do("POST", "/api/me", `{"name":"   "}`)
	if res.StatusCode != 400 {
		t.Fatalf("blank name: %d", res.StatusCode)
	}
	kim.do("POST", "/api/me", `{"name":"Kim"}`)
	lee.do("POST", "/api/me", `{"name":"Lee"}`)

	// search goes through the player, never straight to the source
	res, _ = kim.do("GET", "/api/search?q=alp", "")
	if res.StatusCode != 200 || f.Calls[len(f.Calls)-1] != "search:alp" {
		t.Fatalf("search: %d %v", res.StatusCode, f.Calls)
	}

	// first request plays, second queues with requester name
	kim.do("POST", "/api/queue", `{"id":"a","title":"Alpha"}`)
	res, st := lee.do("POST", "/api/queue", `{"id":"b","title":"Beta"}`)
	if res.StatusCode != 200 {
		t.Fatalf("request: %d", res.StatusCode)
	}
	queue := st["queue"].([]any)
	if len(queue) != 1 || queue[0].(map[string]any)["requestedBy"] != "Lee" {
		t.Fatalf("queue: %v", queue)
	}
	np := st["nowPlaying"].(map[string]any)
	if np["requestedBy"] != "Kim" {
		t.Fatalf("now playing: %v", np)
	}

	// vote on the queued item
	itemID := queue[0].(map[string]any)["id"].(string)
	res, st = kim.do("POST", "/api/queue/"+itemID+"/vote", "")
	if res.StatusCode != 200 || st["queue"].([]any)[0].(map[string]any)["votes"].(float64) != 2 {
		t.Fatalf("vote: %d %v", res.StatusCode, st["queue"])
	}
	res, _ = kim.do("POST", "/api/queue/nope/vote", "")
	if res.StatusCode != 404 {
		t.Fatalf("vote unknown: %d", res.StatusCode)
	}

	// skip: no guests connected -> threshold 1
	res, out := lee.do("POST", "/api/skip", "")
	if res.StatusCode != 200 || out["skipped"] != true {
		t.Fatalf("skip: %d %v", res.StatusCode, out)
	}
	if got := f.Played(); len(got) != 2 || got[1] != "b" {
		t.Fatalf("played: %v", got)
	}

	// artwork: fake is not an ArtworkProvider
	res, _ = kim.do("GET", "/api/artwork/a", "")
	if res.StatusCode != 404 {
		t.Fatalf("artwork: %d", res.StatusCode)
	}
}

func TestEventsCountsDistinctGuests(t *testing.T) {
	ts, _ := newTestServer(t)
	kim := newClient(t, ts.URL)
	kim.do("GET", "/api/me", "") // get a cookie

	open := func() (*http.Response, *bufio.Reader) {
		req, _ := http.NewRequest("GET", ts.URL+"/api/events", nil)
		res, err := kim.http.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		return res, bufio.NewReader(res.Body)
	}
	readEvent := func(r *bufio.Reader) map[string]any {
		var data string
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				t.Fatal(err)
			}
			if strings.HasPrefix(line, "data: ") {
				data = strings.TrimPrefix(line, "data: ")
			}
			if line == "\n" && data != "" {
				var st map[string]any
				json.Unmarshal([]byte(data), &st)
				return st
			}
		}
	}

	res1, r1 := open()
	defer res1.Body.Close()
	if ct := res1.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("content-type: %s", ct)
	}
	first := readEvent(r1) // initial snapshot
	if _, ok := first["queue"]; !ok {
		t.Fatalf("first event should be a state snapshot: %v", first)
	}

	// second tab, same guest: still 1 connected guest
	res2, _ := open()
	defer res2.Body.Close()
	_, st := kim.do("GET", "/api/state", "")
	if st["guests"].(float64) != 1 {
		t.Fatalf("guests = %v, want 1 for two tabs of one guest", st["guests"])
	}

	lee := newClient(t, ts.URL)
	req, _ := http.NewRequest("GET", ts.URL+"/api/events", nil)
	res3, err := lee.http.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res3.Body.Close()
	_, st = kim.do("GET", "/api/state", "")
	if st["guests"].(float64) != 2 {
		t.Fatalf("guests = %v, want 2", st["guests"])
	}
}
