// Package clients provides JSONC (JSON with Comments) parsing support.
// JSONC is used by OpenCode and some VS Code configurations.
package clients

import (
	"regexp"
	"strings"
)

var (
	// blockCommentRegex matches /* */ style comments (multiline)
	blockCommentRegex = regexp.MustCompile(`(?s)/\*.*?\*/`)
	// lineCommentRegex matches // style comments to end of line
	lineCommentRegex = regexp.MustCompile(`//.*$`)
	// trailingCommasRegex matches trailing commas before } or ]
	trailingCommasRegex = regexp.MustCompile(`,(\s*[}\]])`)
)

// StripJSONC removes comments from JSONC content and returns clean JSON.
// It handles:
//   - /* */ block comments (multiline)
//   - // line comments
//   - Trailing commas (common in JSONC)
func StripJSONC(input string) string {
	// Remove block comments first (they can contain // patterns)
	result := blockCommentRegex.ReplaceAllString(input, "")
	// Remove line comments
	result = lineCommentRegex.ReplaceAllString(result, "")
	// Remove trailing commas (not strictly comments but common in JSONC)
	result = trailingCommasRegex.ReplaceAllString(result, "$1")
	// Clean up whitespace while preserving structure
	return strings.TrimSpace(result)
}

// IsJSONC detects if content likely contains JSONC features (comments).
func IsJSONC(input string) bool {
	return strings.Contains(input, "//") || strings.Contains(input, "/*")
}
