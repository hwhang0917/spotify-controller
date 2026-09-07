// Package youtube plays music videos through the official YouTube IFrame
// player embedded in the admin window. Search uses the YouTube Data API v3
// with the host's own API key. No audio is fetched or decoded here, so this
// stays within YouTube's terms: the player runs on youtube.com's own embed.
//
// Go never touches the player directly. Commands go out through Send (the
// app forwards them as a Wails event); the admin page reports player state
// back through Report.
package youtube

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hwhang0917/vibe-music/internal/source"
)

const (
	apiBase        = "https://www.googleapis.com/youtube/v3"
	musicCategory  = "10"
	watchURL       = "https://www.youtube.com/watch?v="
	httpTimeout    = 10 * time.Second
	reportStale    = 5 * time.Second  // no report for this long after Play: assume still loading
	chartTTL       = 30 * time.Minute // charts move slowly; one unit per region per half hour
	defaultRegion  = "US"
	statePlaying   = 1
	stateEnded     = 0
	stateBuffering = 3
)

// Sentinel errors the UIs translate by code.
var (
	ErrNoAPIKey       = &source.CodedError{Kind: "youtube_api_key", Msg: "youtube: API key is empty"}
	ErrQuotaExceeded  = &source.CodedError{Kind: "youtube_quota", Msg: "youtube: daily API quota exceeded"}
	ErrPlayerNotReady = &source.CodedError{Kind: "youtube_player", Msg: "youtube: player not ready"}
	// ErrKeyRestricted: the key has a website (HTTP referrer) or IP restriction
	// that a server-side caller cannot satisfy.
	ErrKeyRestricted = &source.CodedError{Kind: "youtube_key_restricted", Msg: "youtube: API key rejected because of its application restriction"}
	ErrKeyInvalid    = &source.CodedError{Kind: "youtube_key_invalid", Msg: "youtube: API key invalid or the Data API is not enabled"}
	// ErrChartUnavailable: YouTube has no most-popular chart for that region/category.
	ErrChartUnavailable = &source.CodedError{Kind: "youtube_chart_unavailable", Msg: "youtube: no chart for this region"}
)

// Command is what the embedded player is asked to do.
type Command struct {
	Type    string  `json:"type"` // load, play, pause, stop, seek, volume
	VideoID string  `json:"videoId,omitempty"`
	Seconds float64 `json:"seconds,omitempty"`
	Volume  int     `json:"volume,omitempty"`
}

// Report is what the embedded player says about itself.
type Report struct {
	Ready    bool    `json:"ready"`
	VideoID  string  `json:"videoId"`
	State    int     `json:"state"`    // YT.PlayerState: -1 unstarted, 0 ended, 1 playing, 2 paused, 3 buffering, 5 cued
	Position float64 `json:"position"` // seconds
	Duration float64 `json:"duration"` // seconds
}

type Options struct {
	APIKey string
	Send   func(Command)
	HTTP   *http.Client // optional (tests)
	APIURL string       // optional (tests)
}

type Source struct {
	mu      sync.Mutex
	opts    Options
	http    *http.Client
	api     string
	current string // video we asked to play; "" when idle
	tracks  map[string]source.Track
	report  Report
	at      time.Time
	played  time.Time
	volume  int

	charts map[string]chartEntry // region -> cached chart
}

type chartEntry struct {
	tracks []source.Track
	at     time.Time
}

func New(opts Options) *Source {
	s := &Source{opts: opts, tracks: map[string]source.Track{}, charts: map[string]chartEntry{}, http: opts.HTTP, api: opts.APIURL, volume: 100}
	if s.http == nil {
		s.http = &http.Client{Timeout: httpTimeout}
	}
	if s.api == "" {
		s.api = apiBase
	}
	return s
}

func (s *Source) ID() string   { return "youtube" }
func (s *Source) Name() string { return "YouTube Music" }

func (s *Source) SetAPIKey(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.opts.APIKey = strings.TrimSpace(key)
}

func (s *Source) HasAPIKey() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.opts.APIKey != ""
}

// PlayerReady reports whether the embedded player has announced itself.
func (s *Source) PlayerReady() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.report.Ready
}

// Report is called by the app whenever the embedded player reports state.
func (s *Source) Report(r Report) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.report, s.at = r, time.Now()
}

func (s *Source) send(c Command) {
	if s.opts.Send != nil {
		s.opts.Send(c)
	}
}

func (s *Source) Activate(context.Context) error {
	if !s.HasAPIKey() {
		return ErrNoAPIKey
	}
	s.send(Command{Type: "volume", Volume: s.volume})
	return nil
}

func (s *Source) Deactivate(ctx context.Context) error { return s.Stop(ctx) }

// --- search (Data API v3) ---

type searchResponse struct {
	Items []struct {
		ID struct {
			VideoID string `json:"videoId"`
		} `json:"id"`
	} `json:"items"`
}

type videosResponse struct {
	Items []struct {
		ID      string `json:"id"`
		Snippet struct {
			Title        string `json:"title"`
			ChannelTitle string `json:"channelTitle"`
			Thumbnails   map[string]struct {
				URL string `json:"url"`
			} `json:"thumbnails"`
		} `json:"snippet"`
		ContentDetails struct {
			Duration string `json:"duration"`
		} `json:"contentDetails"`
	} `json:"items"`
}

type apiError struct {
	Error struct {
		Code   int `json:"code"`
		Errors []struct {
			Reason string `json:"reason"`
		} `json:"errors"`
		Message string `json:"message"`
	} `json:"error"`
}

func (s *Source) get(ctx context.Context, path string, q url.Values, out any) error {
	s.mu.Lock()
	key := s.opts.APIKey
	s.mu.Unlock()
	if key == "" {
		return ErrNoAPIKey
	}
	q.Set("key", key)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.api+path+"?"+q.Encode(), nil)
	if err != nil {
		return err
	}
	res, err := s.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		var e apiError
		_ = json.NewDecoder(res.Body).Decode(&e)
		msg := strings.ToLower(e.Error.Message)
		reasons := ""
		for _, r := range e.Error.Errors {
			reasons += r.Reason + " "
		}
		// Every mapped error still carries Google's status, reasons and message
		// (wrapped), so a raw view shows exactly what was refused and why.
		detail := fmt.Errorf("youtube: HTTP %d %s%s", res.StatusCode, strings.TrimSpace(reasons+" "), e.Error.Message)
		wrap := func(sentinel *source.CodedError) error { return fmt.Errorf("%w: %v", sentinel, detail) }
		for _, r := range e.Error.Errors {
			switch r.Reason {
			case "quotaExceeded", "dailyLimitExceeded":
				return wrap(ErrQuotaExceeded)
			case "keyInvalid":
				return wrap(ErrKeyInvalid)
			case "videoChartNotFound":
				return wrap(ErrChartUnavailable)
			}
		}
		switch {
		case strings.Contains(msg, "referer") || strings.Contains(msg, "referrer") || strings.Contains(msg, "ip address") || strings.Contains(msg, "api_key_http_referrer_blocked") || strings.Contains(msg, "api_key_ip_address_blocked"):
			return wrap(ErrKeyRestricted)
		case strings.Contains(msg, "api key not valid"):
			return wrap(ErrKeyInvalid)
		}
		return detail
	}
	return json.NewDecoder(res.Body).Decode(out)
}

// Search finds music videos, then fetches durations. Two API calls: search
// costs 100 quota units, videos.list 1, so about 100 searches/day on the
// default 10,000-unit quota. The guest UI debounces to make that last.
func (s *Source) Search(ctx context.Context, query string, limit int) ([]source.Track, error) {
	var sr searchResponse
	err := s.get(ctx, "/search", url.Values{
		"part": {"snippet"}, "type": {"video"}, "videoCategoryId": {musicCategory},
		"maxResults": {strconv.Itoa(limit)}, "q": {query},
	}, &sr)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(sr.Items))
	for _, it := range sr.Items {
		if it.ID.VideoID != "" {
			ids = append(ids, it.ID.VideoID)
		}
	}
	if len(ids) == 0 {
		return []source.Track{}, nil
	}
	var vr videosResponse
	if err := s.get(ctx, "/videos", url.Values{"part": {"snippet,contentDetails"}, "id": {strings.Join(ids, ",")}}, &vr); err != nil {
		return nil, err
	}
	byID := map[string]source.Track{}
	for _, t := range s.remember(vr) {
		byID[t.ID] = t
	}
	out := make([]source.Track, 0, len(ids))
	for _, id := range ids { // keep search ranking
		if t, ok := byID[id]; ok {
			out = append(out, t)
		}
	}
	return out, nil
}

// remember maps a videos.list response to tracks and caches them for Status.
func (s *Source) remember(vr videosResponse) []source.Track {
	out := make([]source.Track, 0, len(vr.Items))
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, v := range vr.Items {
		t := source.Track{
			Source:      "youtube",
			ID:          v.ID,
			Title:       v.Snippet.Title,
			Artist:      v.Snippet.ChannelTitle,
			Duration:    parseISODuration(v.ContentDetails.Duration),
			ExternalURL: watchURL + v.ID,
		}
		for _, k := range []string{"medium", "high", "default"} {
			if th, ok := v.Snippet.Thumbnails[k]; ok {
				t.ArtworkURL = th.URL
				break
			}
		}
		s.tracks[v.ID] = t
		out = append(out, t)
	}
	return out
}

// Test exercises the key the way the app uses it: one search and one chart
// call. It returns a short human summary, or the first failure verbatim.
func (s *Source) Test(ctx context.Context, region string) (string, error) {
	hint := s.keyHint()
	tracks, err := s.Search(ctx, "music", 1)
	if err != nil {
		return "", fmt.Errorf("search (key %s): %w", hint, err)
	}
	s.mu.Lock()
	delete(s.charts, strings.ToUpper(strings.TrimSpace(region))) // force a live call
	s.mu.Unlock()
	chart, err := s.Chart(ctx, region, 1)
	if err != nil {
		return "", fmt.Errorf("chart (key %s): %w", hint, err)
	}
	return fmt.Sprintf("key %s: search ok (%d), chart ok (%d)", hint, len(tracks), len(chart)), nil
}

// keyHint identifies the stored key without revealing it, so the admin can
// tell which console key a test result is talking about.
func (s *Source) keyHint() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := s.opts.APIKey
	if len(k) < 8 {
		return fmt.Sprintf("%d chars", len(k))
	}
	return fmt.Sprintf("%s…%s, %d chars", k[:6], k[len(k)-2:], len(k))
}

// Chart returns the most popular music videos in a region (videos.list with
// chart=mostPopular, 1 quota unit), cached per region for chartTTL.
func (s *Source) Chart(ctx context.Context, region string, limit int) ([]source.Track, error) {
	region = strings.ToUpper(strings.TrimSpace(region))
	if region == "" {
		region = defaultRegion
	}
	s.mu.Lock()
	c, ok := s.charts[region]
	s.mu.Unlock()
	if ok && time.Since(c.at) < chartTTL && len(c.tracks) >= limit {
		return c.tracks[:limit], nil
	}
	var vr videosResponse
	fetch := func(region string) error {
		return s.get(ctx, "/videos", url.Values{
			"part": {"snippet,contentDetails"}, "chart": {"mostPopular"},
			"videoCategoryId": {musicCategory}, "regionCode": {region}, "maxResults": {strconv.Itoa(limit)},
		}, &vr)
	}
	err := fetch(region)
	if errors.Is(err, ErrChartUnavailable) && region != defaultRegion {
		err = fetch(defaultRegion) // no music chart for that region: show the global one
	}
	if err != nil {
		return nil, err
	}
	tracks := s.remember(vr)
	s.mu.Lock()
	s.charts[region] = chartEntry{tracks: tracks, at: time.Now()}
	s.mu.Unlock()
	return tracks, nil
}

var isoDur = regexp.MustCompile(`^PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?$`)

// parseISODuration reads the ISO 8601 subset YouTube uses, e.g. PT4M13S.
func parseISODuration(s string) time.Duration {
	m := isoDur.FindStringSubmatch(s)
	if m == nil {
		return 0
	}
	h, _ := strconv.Atoi(m[1])
	mi, _ := strconv.Atoi(m[2])
	sec, _ := strconv.Atoi(m[3])
	return time.Duration(h)*time.Hour + time.Duration(mi)*time.Minute + time.Duration(sec)*time.Second
}

// --- playback (via the embedded player) ---

func (s *Source) Play(_ context.Context, id string) error {
	s.mu.Lock()
	if !s.report.Ready {
		s.mu.Unlock()
		return ErrPlayerNotReady
	}
	s.current, s.played = id, time.Now()
	s.mu.Unlock()
	s.send(Command{Type: "load", VideoID: id})
	return nil
}

func (s *Source) Pause(context.Context) error  { s.send(Command{Type: "pause"}); return nil }
func (s *Source) Resume(context.Context) error { s.send(Command{Type: "play"}); return nil }

func (s *Source) Stop(context.Context) error {
	s.mu.Lock()
	s.current = ""
	s.mu.Unlock()
	s.send(Command{Type: "stop"})
	return nil
}

func (s *Source) Seek(_ context.Context, pos time.Duration) error {
	s.mu.Lock()
	// the player will report the new position within a poll; until then
	// answer with what we asked for so nobody snaps back to the old spot
	s.report.Position = pos.Seconds()
	s.at = time.Now()
	s.mu.Unlock()
	s.send(Command{Type: "seek", Seconds: pos.Seconds()})
	return nil
}

func (s *Source) SetVolume(_ context.Context, pct int) error {
	s.mu.Lock()
	s.volume = pct
	s.mu.Unlock()
	s.send(Command{Type: "volume", Volume: pct})
	return nil
}

func (s *Source) Status(context.Context) (source.Playback, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return mapStatus(s.report, s.at, s.current, s.played, s.tracks, time.Now()), nil
}

// mapStatus turns the last player report into a Playback. A report for a
// different video right after Play is the old one still unloading; a stale
// report (none since Play) means the player is loading.
func mapStatus(r Report, reportedAt time.Time, current string, playedAt time.Time, tracks map[string]source.Track, now time.Time) source.Playback {
	pb := source.Playback{At: now}
	if current == "" {
		return pb
	}
	t, ok := tracks[current]
	if !ok {
		t = source.Track{Source: "youtube", ID: current, ExternalURL: watchURL + current}
	}
	pb.Track = &t
	if r.VideoID != current || reportedAt.Before(playedAt) {
		pb.Playing = now.Sub(playedAt) < reportStale
		return pb
	}
	if r.Duration > 0 {
		pb.Track.Duration = time.Duration(r.Duration * float64(time.Second))
	}
	pb.Position = time.Duration(r.Position * float64(time.Second))
	pb.Playing = r.State == statePlaying || r.State == stateBuffering
	pb.Ended = r.State == stateEnded
	return pb
}
