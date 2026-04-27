package manifest

import "testing"

func TestCatalogDigestIgnoresTrustFields(t *testing.T) {
	m := &Manifest{ID: "memory", Name: "Memory", Version: "1.0.0", Transport: "stdio", Runtime: "node", Entrypoint: Entrypoint{Command: "npx", Args: []string{"-y", "x"}}, Verification: "community"}
	digest, err := CatalogDigest(m)
	if err != nil {
		t.Fatal(err)
	}
	m.SHA256 = digest
	m.Signature = "detached"
	digest2, err := CatalogDigest(m)
	if err != nil {
		t.Fatal(err)
	}
	if digest != digest2 {
		t.Fatalf("digest changed after adding trust fields: %s != %s", digest, digest2)
	}
	if err := VerifyCatalogDigest(m); err != nil {
		t.Fatalf("verify: %v", err)
	}
}

func TestVerifyCatalogDigestRejectsMismatch(t *testing.T) {
	m := &Manifest{ID: "memory", Name: "Memory", Version: "1.0.0", Transport: "stdio", Runtime: "node", Entrypoint: Entrypoint{Command: "npx", Args: []string{"-y", "x"}}, Verification: "community", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	if err := VerifyCatalogDigest(m); err == nil {
		t.Fatal("expected mismatch")
	}
}
