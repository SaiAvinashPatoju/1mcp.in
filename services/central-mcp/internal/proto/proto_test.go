package proto

import (
	"encoding/json"
	"testing"
)

// TestIDRoundTrip locks the contract relied on by upstream.Client: the
// pending-request map key must match between request build (NewStringID)
// and response read (msg.ID.String()). Both sides use ID.String(); this
// test catches accidental drift in either direction.
func TestIDRoundTrip(t *testing.T) {
	out := NewStringID("u-42")
	wireMsg := Message{JSONRPC: Version, ID: &out, Method: "x"}
	b, err := json.Marshal(&wireMsg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back Message
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.ID == nil {
		t.Fatal("decoded ID is nil")
	}
	if back.ID.String() != out.String() {
		t.Fatalf("id mismatch: out=%s back=%s", out.String(), back.ID.String())
	}
}

func TestNumericIDRoundTrip(t *testing.T) {
	var orig ID
	if err := json.Unmarshal([]byte("17"), &orig); err != nil {
		t.Fatal(err)
	}
	if orig.String() != "17" {
		t.Fatalf("got %q", orig.String())
	}
	b, err := json.Marshal(orig)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "17" {
		t.Fatalf("got %q", string(b))
	}
}

func TestNullIDIsNull(t *testing.T) {
	var id ID
	if err := json.Unmarshal([]byte("null"), &id); err != nil {
		t.Fatal(err)
	}
	if !id.IsNull() {
		t.Fatal("expected null")
	}
}
