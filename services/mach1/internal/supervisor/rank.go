package supervisor

import (
	"strings"

	"github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/proto"
	"github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/semantic"
)

// RankTools returns the top-k tools (by name) most relevant to query, drawn
// from the supervisor's currently-cached tool list. Pure read-side; does not
// trigger upstream starts.
//
// The index is built on-demand from the cached tools so it always reflects
// the most recent warmup state.
func (s *Supervisor) RankTools(query string, k int) []proto.Tool {
	tools := s.Tools()
	if len(tools) == 0 || query == "" {
		return nil
	}
	docs := make([]semantic.Doc, 0, len(tools))
	byName := make(map[string]proto.Tool, len(tools))
	for _, t := range tools {
		// Strip the namespace prefix from the searchable text so token
		// stop-list rules apply to real words, but keep the full namespaced
		// name as the doc ID so the caller can look it up directly.
		idPart, suffix, _ := strings.Cut(t.Name, NamespaceSep)
		text := strings.Join([]string{idPart, suffix, t.Description}, " ")
		docs = append(docs, semantic.Doc{ID: t.Name, Text: text})
		byName[t.Name] = t
	}
	results := semantic.Build(docs).Rank(query, k)
	out := make([]proto.Tool, 0, len(results))
	for _, r := range results {
		if t, ok := byName[r.ID]; ok {
			out = append(out, t)
		}
	}
	return out
}
