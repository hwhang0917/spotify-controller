package youtube

import (
	_ "embed"
	"net"
	"net/http"
)

//go:embed player.html
var playerHTML []byte

// ServePlayer serves the embedded IFrame player page on a random loopback port
// and returns its URL. YouTube refuses embeds without an HTTP Referer (error
// 153), and the wails:// origin the admin window uses on Linux and macOS sends
// none, so the player is framed from an http origin instead.
func ServePlayer() (string, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(playerHTML)
	})}
	go srv.Serve(ln) // ponytail: lives for the process; nothing to shut down
	return "http://" + ln.Addr().String(), nil
}
