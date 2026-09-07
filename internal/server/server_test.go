package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestHandler(t *testing.T) {
	dist := fstest.MapFS{
		"index.html":    {Data: []byte("<html>guest</html>")},
		"assets/app.js": {Data: []byte("console.log(1)")},
	}
	h := NewHandler(dist)

	get := func(p string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, p, nil))
		return rec
	}

	if rec := get("/api/health"); rec.Code != 200 || rec.Body.String() != "{\"status\":\"ok\"}\n" {
		t.Fatalf("health: %d %q", rec.Code, rec.Body.String())
	}
	if rec := get("/assets/app.js"); rec.Code != 200 || rec.Body.String() != "console.log(1)" {
		t.Fatalf("static: %d %q", rec.Code, rec.Body.String())
	}
	// SPA fallback: deep links get index.html
	if rec := get("/vote/abc"); rec.Code != 200 || rec.Body.String() != "<html>guest</html>" {
		t.Fatalf("spa fallback: %d %q", rec.Code, rec.Body.String())
	}
	if rec := get("/"); rec.Code != 200 || rec.Body.String() != "<html>guest</html>" {
		t.Fatalf("root: %d %q", rec.Code, rec.Body.String())
	}
	// Unbuilt UI is a clear 503, not a confusing 404
	if rec := (func() *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		NewHandler(fstest.MapFS{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		return rec
	})(); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("unbuilt: %d", rec.Code)
	}
}
