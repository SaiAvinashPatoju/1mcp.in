package manifest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// CatalogDigest returns the SHA256 digest maintainers sign for marketplace
// entries. Mutable trust fields are excluded so maintainers can add the digest
// and detached signature without changing the value being verified.
func CatalogDigest(m *Manifest) (string, error) {
	if m == nil {
		return "", fmt.Errorf("nil manifest")
	}
	copy := *m
	copy.SHA256 = ""
	copy.Signature = ""
	b, err := json.Marshal(copy)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func VerifyCatalogDigest(m *Manifest) error {
	if m == nil {
		return fmt.Errorf("nil manifest")
	}
	if m.SHA256 == "" {
		return fmt.Errorf("missing sha256 for marketplace entry %q", m.ID)
	}
	got, err := CatalogDigest(m)
	if err != nil {
		return err
	}
	if got != m.SHA256 {
		return fmt.Errorf("sha256 mismatch for %s: expected %s got %s", m.ID, m.SHA256, got)
	}
	return nil
}
