package youtube

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestServePlayerServesEmbedPage(t *testing.T) {
	url, err := ServePlayer()
	if err != nil {
		t.Fatal(err)
	}
	// a name, not an IP: YouTube refuses licensed music for IP-literal origins
	if !strings.HasPrefix(url, "http://localhost:") {
		t.Fatalf("want http://localhost URL, got %q", url)
	}
	res, err := http.Get(url + "/?blocked=x")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != 200 || !strings.Contains(string(body), "youtube.com/iframe_api") {
		t.Fatalf("status %d, body %q", res.StatusCode, body)
	}
}
