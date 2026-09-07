package server

import (
	"net/http"
	"net/url"
	"sync"
)

// jar is a minimal cookie jar: one host, keeps the latest value per name.
type jar struct {
	mu      sync.Mutex
	cookies map[string]*http.Cookie
}

func newJar() *jar { return &jar{cookies: map[string]*http.Cookie{}} }

func (j *jar) SetCookies(_ *url.URL, cs []*http.Cookie) {
	j.mu.Lock()
	defer j.mu.Unlock()
	for _, c := range cs {
		j.cookies[c.Name] = c
	}
}

func (j *jar) Cookies(_ *url.URL) []*http.Cookie {
	j.mu.Lock()
	defer j.mu.Unlock()
	out := make([]*http.Cookie, 0, len(j.cookies))
	for _, c := range j.cookies {
		out = append(out, c)
	}
	return out
}
