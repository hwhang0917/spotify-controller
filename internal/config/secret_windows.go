//go:build windows

package config

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// On Windows secrets go through DPAPI: the key is derived from the logged-in
// user's credentials by the OS, so nothing usable is stored beside the file
// and another user account on the same PC cannot read it.
func seal(plain []byte) ([]byte, error)  { return dpapi(plain, true) }
func unseal(blob []byte) ([]byte, error) { return dpapi(blob, false) }

func dpapi(in []byte, protect bool) ([]byte, error) {
	if len(in) == 0 {
		return nil, nil
	}
	inBlob := windows.DataBlob{Size: uint32(len(in)), Data: &in[0]}
	var out windows.DataBlob
	var err error
	if protect {
		err = windows.CryptProtectData(&inBlob, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out)
	} else {
		err = windows.CryptUnprotectData(&inBlob, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out)
	}
	if err != nil {
		return nil, err
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	return append([]byte(nil), unsafe.Slice(out.Data, out.Size)...), nil
}
