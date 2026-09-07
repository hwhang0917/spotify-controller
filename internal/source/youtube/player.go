package youtube

import (
	_ "embed"
	"fmt"
	"net"
	"net/http"
)

//go:embed player.html
var playerHTML []byte

// ServePlayer serves the embedded IFrame player page on a random loopback port
// and returns its URL. YouTube refuses embeds without an HTTP Referer (error
// 153), and the wails:// origin the admin window uses on Linux and macOS sends
// none, so the player is framed from an http origin instead. The URL uses the
// name "localhost", not 127.0.0.1: YouTube answers "This video is unavailable"
// for licensed music when the embedding origin is an IP literal.
func ServePlayer() (string, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	port := ln.Addr().(*net.TCPAddr).Port
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(playerHTML)
	})}
	go srv.Serve(ln) // ponytail: lives for the process; nothing to shut down
	// localhost may resolve to ::1 first; best effort, the webview falls back to IPv4 otherwise
	if ln6, err := net.Listen("tcp", fmt.Sprintf("[::1]:%d", port)); err == nil {
		go srv.Serve(ln6)
	}
	return fmt.Sprintf("http://localhost:%d", port), nil
}
