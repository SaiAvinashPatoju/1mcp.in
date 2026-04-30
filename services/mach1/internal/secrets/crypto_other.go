//go:build !windows

package secrets

func protectSecretBytes(plaintext []byte) ([]byte, error) {
	return plaintext, nil
}

func unprotectSecretBytes(sealed []byte) ([]byte, error) {
	return sealed, nil
}
