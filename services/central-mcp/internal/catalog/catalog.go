// Package catalog loads the static MCP catalog (packages/registry-index/index.json)
// that the hub and CLI surface as the marketplace. Phase 8 will swap this for
// a remote registry behind the same interface.
package catalog

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/onemcp/central-mcp/internal/manifest"
)

// Load reads and validates a catalog file. Invalid entries are reported
// individually; the loader returns the valid subset and a multi-error.
func Load(path string) ([]manifest.Manifest, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read catalog: %w", err)
	}
	var raw []json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, fmt.Errorf("parse catalog (expected JSON array): %w", err)
	}
	out := make([]manifest.Manifest, 0, len(raw))
	var errs []error
	for i, r := range raw {
		m, err := manifest.Parse(r)
		if err != nil {
			errs = append(errs, fmt.Errorf("entry %d: %w", i, err))
			continue
		}
		out = append(out, *m)
	}
	if len(errs) > 0 {
		return out, fmt.Errorf("%d invalid catalog entries: %v", len(errs), errs)
	}
	return out, nil
}

// Find looks up an entry by id. Returns nil if not present.
func Find(entries []manifest.Manifest, id string) *manifest.Manifest {
	for i := range entries {
		if entries[i].ID == id {
			return &entries[i]
		}
	}
	return nil
}
