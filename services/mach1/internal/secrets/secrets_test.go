package secrets

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestStoreRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secrets.json")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := store.Set("github", "GITHUB_TOKEN", "ghp_test_secret"); err != nil {
		t.Fatalf("set: %v", err)
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	got := reopened.Get("github")["GITHUB_TOKEN"]
	if got != "ghp_test_secret" {
		t.Fatalf("secret mismatch: %q", got)
	}
}

func TestStoreDoesNotWritePlaintextOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("DPAPI encryption is Windows-only")
	}

	path := filepath.Join(t.TempDir(), "secrets.json")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := store.Set("github", "GITHUB_TOKEN", "ghp_test_secret"); err != nil {
		t.Fatalf("set: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read raw: %v", err)
	}
	if bytes.Contains(raw, []byte("ghp_test_secret")) {
		t.Fatal("secret file contains plaintext secret")
	}
}
