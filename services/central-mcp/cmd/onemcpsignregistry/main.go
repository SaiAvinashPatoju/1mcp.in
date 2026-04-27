package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/onemcp/central-mcp/internal/manifest"
)

func main() {
	path := flag.String("catalog", "../../packages/registry-index/index.json", "registry-index catalog path")
	check := flag.Bool("check", false, "verify without rewriting")
	flag.Parse()

	b, err := os.ReadFile(*path)
	must(err)
	var entries []manifest.Manifest
	must(json.Unmarshal(b, &entries))

	changed := false
	seen := map[string]bool{}
	for i := range entries {
		m := &entries[i]
		must(m.Validate())
		if seen[m.ID] {
			fatalf("duplicate id %s", m.ID)
		}
		seen[m.ID] = true
		if m.Verification == "" {
			m.Verification = defaultVerification(*m)
			changed = true
		}
		digest, err := manifest.CatalogDigest(m)
		must(err)
		if m.SHA256 != digest {
			if *check {
				fatalf("%s sha256 mismatch: expected %s got %s", m.ID, m.SHA256, digest)
			}
			m.SHA256 = digest
			changed = true
		}
	}
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].ID < entries[j].ID })
	if *check {
		fmt.Printf("catalog verified: %d entries\n", len(entries))
		return
	}
	if changed {
		out, err := json.MarshalIndent(entries, "", "  ")
		must(err)
		out = append(out, '\n')
		must(os.WriteFile(*path, out, 0o644))
	}
	fmt.Printf("catalog signed: %d entries\n", len(entries))
}

func defaultVerification(m manifest.Manifest) string {
	for _, tag := range m.Tags {
		if tag == "official" {
			if strings.Contains(m.Homepage, "modelcontextprotocol/servers") || strings.Contains(m.Homepage, "github.com/github/github-mcp-server") {
				return "anthropic-official"
			}
			return "onemcp-verified"
		}
	}
	return "community"
}

func must(err error) {
	if err != nil {
		fatalf("%v", err)
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
