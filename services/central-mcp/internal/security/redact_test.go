package security

import (
	"strings"
	"testing"
)

func TestRedactString(t *testing.T) {
	in := "email dev@example.com token ghp_abcdefghijklmnopqrstuvwxyz123456 jwt eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0In0.abcdefghi123456789"
	out := RedactString(in)
	for _, leaked := range []string{"dev@example.com", "ghp_abcdefghijklmnopqrstuvwxyz123456", "eyJhbGci"} {
		if strings.Contains(out, leaked) {
			t.Fatalf("leaked %q in %q", leaked, out)
		}
	}
}

func TestRedactJSON(t *testing.T) {
	out := string(RedactJSON([]byte(`{"token":"ghp_abcdefghijklmnopqrstuvwxyz123456","nested":{"email":"dev@example.com"}}`)))
	if strings.Contains(out, "ghp_") || strings.Contains(out, "dev@example.com") {
		t.Fatalf("secret leaked: %s", out)
	}
}
