package main

import (
	"io"
	"log/slog"
	"os"
)

const (
	logFile     = "vibe-music.log"
	logMaxBytes = 5 << 20
)

// setupLog sends every log line (log.* and slog.*) as JSON to path and to
// stderr. The file is 0600 because request logs carry guest names and IPs.
// ponytail: one previous file kept when the current one passes logMaxBytes;
// a size-based rotator if the log ever needs history.
func setupLog(path string) error {
	if st, err := os.Stat(path); err == nil && st.Size() > logMaxBytes {
		_ = os.Rename(path, path+".1")
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(io.MultiWriter(os.Stderr, f), nil)))
	return nil
}
