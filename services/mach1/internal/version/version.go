// Package version holds the single source of truth for the 1mcp.in version.
// The Version variable is overridden at build time via ldflags:
//
//	go build -ldflags "-X github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/version.Version=v0.4.2"
package version

// Version is the current 1mcp.in version. Overridden at build time via ldflags.
// Defaults to "dev" for local builds.
var Version = "dev"
