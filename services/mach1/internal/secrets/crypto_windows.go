package secrets

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

const dpapiPrefix = "dpapi:"

func protectSecretBytes(plaintext []byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return plaintext, nil
	}
	in := bytesToBlob(plaintext)
	var out windows.DataBlob
	if err := windows.CryptProtectData(&in, nil, nil, 0, nil, 0, &out); err != nil {
		return nil, err
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))

	protected := unsafe.Slice(out.Data, int(out.Size))
	encoded := base64.StdEncoding.EncodeToString(protected)
	return []byte(dpapiPrefix + encoded), nil
}

func unprotectSecretBytes(sealed []byte) ([]byte, error) {
	if len(sealed) == 0 {
		return sealed, nil
	}
	if !bytes.HasPrefix(sealed, []byte(dpapiPrefix)) {
		return sealed, nil
	}
	ciphertext, err := base64.StdEncoding.DecodeString(string(sealed[len(dpapiPrefix):]))
	if err != nil {
		return nil, fmt.Errorf("decode DPAPI payload: %w", err)
	}
	if len(ciphertext) == 0 {
		return nil, nil
	}
	in := bytesToBlob(ciphertext)
	var description *uint16
	var out windows.DataBlob
	if err := windows.CryptUnprotectData(&in, &description, nil, 0, nil, 0, &out); err != nil {
		return nil, err
	}
	if description != nil {
		windows.LocalFree(windows.Handle(unsafe.Pointer(description)))
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))

	plaintext := unsafe.Slice(out.Data, int(out.Size))
	return append([]byte(nil), plaintext...), nil
}

func bytesToBlob(b []byte) windows.DataBlob {
	if len(b) == 0 {
		return windows.DataBlob{}
	}
	return windows.DataBlob{Size: uint32(len(b)), Data: &b[0]}
}
