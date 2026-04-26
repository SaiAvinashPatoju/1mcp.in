// Package envtmpl expands ${VAR} references inside command/args at launch time.
//
// Lookup order (later wins):
//  1. process env (parent, e.g. PATH)
//  2. registry env_json (per-MCP non-secret config)
//  3. secrets store (per-MCP secret config)
//
// Unknown variables expand to "" and are reported in MissingVars so the
// supervisor can fail fast with a clear message instead of spawning a child
// that will crash with a confusing stack.
package envtmpl

import (
	"strings"
)

// Expand walks s and replaces ${NAME} occurrences using values. NAME must
// match [A-Za-z_][A-Za-z0-9_]*. Unknown names are recorded in missing.
func Expand(s string, values map[string]string) (out string, missing []string) {
	if !strings.Contains(s, "${") {
		return s, nil
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		if i+1 < len(s) && s[i] == '$' && s[i+1] == '{' {
			end := strings.IndexByte(s[i+2:], '}')
			if end >= 0 {
				name := s[i+2 : i+2+end]
				if isValidName(name) {
					if v, ok := values[name]; ok {
						b.WriteString(v)
					} else {
						missing = append(missing, name)
					}
					i += 2 + end + 1
					continue
				}
			}
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String(), missing
}

// ExpandAll runs Expand over each string in args. The aggregated missing slice
// is deduplicated.
func ExpandAll(args []string, values map[string]string) (out []string, missing []string) {
	out = make([]string, len(args))
	seen := map[string]struct{}{}
	for i, a := range args {
		v, m := Expand(a, values)
		out[i] = v
		for _, k := range m {
			if _, ok := seen[k]; !ok {
				seen[k] = struct{}{}
				missing = append(missing, k)
			}
		}
	}
	return
}

func isValidName(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		ok := (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r == '_' ||
			(i > 0 && r >= '0' && r <= '9')
		if !ok {
			return false
		}
	}
	return true
}
