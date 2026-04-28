package registry

import (
	"context"
	"testing"
)

func TestVerifyToolDefinitionsMarksChangedToolPending(t *testing.T) {
	db, err := Open(t.TempDir() + "/registry.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	defs := []ToolDefinition{{Name: "read", Description: "safe", InputSchema: []byte(`{"type":"object"}`)}}
	first, err := db.VerifyToolDefinitions(ctx, "fs", defs)
	if err != nil {
		t.Fatal(err)
	}
	if first["read"].Status != ToolStatusApproved {
		t.Fatalf("first status = %s", first["read"].Status)
	}
	defs[0].Description = "quietly unsafe"
	second, err := db.VerifyToolDefinitions(ctx, "fs", defs)
	if err != nil {
		t.Fatal(err)
	}
	if second["read"].Status != ToolStatusPendingReview {
		t.Fatalf("second status = %s", second["read"].Status)
	}
	pending, err := db.ListPendingToolReviews(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 || pending[0].ToolName != "read" {
		t.Fatalf("pending = %#v", pending)
	}
	if err := db.ApproveToolDefinition(ctx, "fs", "read"); err != nil {
		t.Fatal(err)
	}
	pending, err = db.ListPendingToolReviews(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 0 {
		t.Fatalf("expected no pending reviews, got %#v", pending)
	}
}
