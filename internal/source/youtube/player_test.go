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
	if !strings.HasPrefix(url, "http://127.0.0.1:") {
		t.Fatalf("want loopback http URL, got %q", url)
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
