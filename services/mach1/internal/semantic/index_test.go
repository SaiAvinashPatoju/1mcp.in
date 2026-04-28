package semantic

import "testing"

func TestRankRelevance(t *testing.T) {
	idx := Build([]Doc{
		{ID: "fs__read_file", Text: "filesystem read a file's contents from disk"},
		{ID: "fs__write_file", Text: "filesystem write a file to disk"},
		{ID: "memory__create_entities", Text: "knowledge graph memory create new entities"},
		{ID: "github__create_issue", Text: "github create a new issue in a repository"},
		{ID: "fetch__fetch", Text: "fetch a url and convert html to markdown"},
	})
	results := idx.Rank("read a file from disk", 3)
	if len(results) == 0 {
		t.Fatal("expected results")
	}
	if results[0].ID != "fs__read_file" {
		t.Fatalf("expected fs__read_file first, got %v", results)
	}
}

func TestRankEmptyCorpus(t *testing.T) {
	idx := Build(nil)
	if got := idx.Rank("anything", 5); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestRankTopK(t *testing.T) {
	idx := Build([]Doc{
		{ID: "a", Text: "alpha alpha alpha"},
		{ID: "b", Text: "alpha beta"},
		{ID: "c", Text: "beta gamma"},
	})
	results := idx.Rank("alpha", 2)
	if len(results) != 2 {
		t.Fatalf("want 2, got %d", len(results))
	}
}

func TestUnseenTermProducesNoResults(t *testing.T) {
	idx := Build([]Doc{{ID: "a", Text: "hello world"}})
	if got := idx.Rank("xyzzy", 5); got != nil {
		t.Fatalf("expected nil for unseen term, got %v", got)
	}
}
