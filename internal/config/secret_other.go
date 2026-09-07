//go:build !windows

package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
)

// Off Windows there is no DPAPI, so secrets are AES-256-GCM sealed with a
// random key kept in the data dir (0600).
// ponytail: the key sits beside the data, so this protects against a copied
// file, not against the same user account; the OS keychain (Keychain Services,
// libsecret) is the upgrade if macOS/Linux hosts ever matter.
const keyFile = "secret.key"

func sealKey() ([]byte, error) {
	dir, err := Dir()
	if err != nil {
		return nil, err
	}
	p := filepath.Join(dir, keyFile)
	if k, err := os.ReadFile(p); err == nil && len(k) == 32 {
		return k, nil
	}
	k := make([]byte, 32)
	if _, err := rand.Read(k); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	return k, os.WriteFile(p, k, 0o600)
}

func gcm() (cipher.AEAD, error) {
	k, err := sealKey()
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(k)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func seal(plain []byte) ([]byte, error) {
	g, err := gcm()
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, g.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return g.Seal(nonce, nonce, plain, nil), nil
}

func unseal(blob []byte) ([]byte, error) {
	g, err := gcm()
	if err != nil {
		return nil, err
	}
	if len(blob) < g.NonceSize() {
		return nil, errors.New("secret: truncated")
	}
	return g.Open(nil, blob[:g.NonceSize()], blob[g.NonceSize():], nil)
}
