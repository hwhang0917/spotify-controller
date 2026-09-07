package config

import (
	"bytes"
	"errors"
	"os"
)

// Secrets (the Spotify token, the YouTube API key) are sealed at rest. The
// file starts with sealedMagic; a file without it is a plaintext one written
// by an older build and is sealed on first read.
const sealedMagic = "VMSEC1\n"

// loadSecret returns nil, nil when the file does not exist.
func loadSecret(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !bytes.HasPrefix(data, []byte(sealedMagic)) {
		if err := saveSecret(path, data); err != nil { // migrate the legacy plaintext file
			return nil, err
		}
		return data, nil
	}
	return unseal(data[len(sealedMagic):])
}

// saveSecret seals and writes atomically with 0600; nil/empty removes the file.
func saveSecret(path string, plain []byte) error {
	if len(plain) == 0 {
		err := os.Remove(path)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	sealed, err := seal(plain)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append([]byte(sealedMagic), sealed...), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
