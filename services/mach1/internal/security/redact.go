// Package security contains small security helpers shared by router, API, and CLI paths.
package security

import (
	"encoding/json"
	"regexp"
	"strings"
)

var redactors = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b[A-Z0-9._%+\-]+@[A-Z0-9.\-]+\.[A-Z]{2,}\b`),
	regexp.MustCompile(`\b(?:\+?\d[\d .()\-]{7,}\d)\b`),
	regexp.MustCompile(`\b(?:\d[ -]*?){13,19}\b`),
	regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`),
	regexp.MustCompile(`\bgithub_pat_[A-Za-z0-9_]{20,}\b`),
	regexp.MustCompile(`\bgh[pousr]_[A-Za-z0-9_]{20,}\b`),
	regexp.MustCompile(`\beyJ[A-Za-z0-9_\-]{10,}\.[A-Za-z0-9_\-]{10,}\.[A-Za-z0-9_\-]{10,}\b`),
}

// RedactString removes common tokens and personal data from text before it is
// written to logs, persistent storage, or UI event streams.
func RedactString(s string) string {
	if s == "" {
		return s
	}
	out := s
	for _, re := range redactors {
		out = re.ReplaceAllString(out, "[REDACTED]")
	}
	return out
}

// RedactJSON scrubs string leaves in a JSON document while preserving the
// response shape. If the bytes are not valid JSON, it falls back to text redaction.
func RedactJSON(b []byte) []byte {
	if len(b) == 0 {
		return b
	}
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return []byte(RedactString(string(b)))
	}
	scrubJSON(&v)
	out, err := json.Marshal(v)
	if err != nil {
		return []byte(RedactString(string(b)))
	}
	return out
}

func scrubJSON(v *any) {
	switch x := (*v).(type) {
	case string:
		*v = RedactString(x)
	case []any:
		for i := range x {
			scrubJSON(&x[i])
		}
	case map[string]any:
		for k, child := range x {
			lower := strings.ToLower(k)
			if strings.Contains(lower, "token") || strings.Contains(lower, "secret") || strings.Contains(lower, "password") || strings.Contains(lower, "authorization") || strings.Contains(lower, "api_key") {
				x[k] = "[REDACTED]"
				continue
			}
			scrubJSON(&child)
			x[k] = child
		}
	}
}
