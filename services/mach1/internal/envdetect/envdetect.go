// Package envdetect scans the host environment and maps common aliases to
// canonical env var names declared in MCP manifests.
package envdetect

import (
	"strings"

	"github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/manifest"
)

// Detector holds the catalog used to resolve env schemas.
type Detector struct {
	catalog []manifest.Manifest
}

// New creates a Detector.
func New(catalog []manifest.Manifest) *Detector {
	return &Detector{catalog: catalog}
}

// DetectForMCP returns a map of canonical_name -> value found in hostEnv for
// the given MCP id. It respects manifest aliases.
func (d *Detector) DetectForMCP(mcpID string, hostEnv []string) map[string]string {
	var m *manifest.Manifest
	for i := range d.catalog {
		if d.catalog[i].ID == mcpID {
			m = &d.catalog[i]
			break
		}
	}
	if m == nil {
		return nil
	}
	envMap := parseHostEnv(hostEnv)
	out := map[string]string{}
	for _, ev := range m.EnvSchema {
		if v, ok := envMap[ev.Name]; ok {
			out[ev.Name] = v
			continue
		}
		for _, alias := range ev.Aliases {
			if v, ok := envMap[alias]; ok {
				out[ev.Name] = v
				break
			}
		}
	}
	return out
}

// DetectAll runs DetectForMCP across every catalog entry.
func (d *Detector) DetectAll(hostEnv []string) map[string]map[string]string {
	out := map[string]map[string]string{}
	for i := range d.catalog {
		id := d.catalog[i].ID
		found := d.DetectForMCP(id, hostEnv)
		if len(found) > 0 {
			out[id] = found
		}
	}
	return out
}

// EnvMatch records one discovered env variable.
type EnvMatch struct {
	MCP    string `json:"mcp"`
	Var    string `json:"var"`
	Source string `json:"source"`
}

// EnvMissing records one missing required env variable.
type EnvMissing struct {
	MCP    string `json:"mcp"`
	Var    string `json:"var"`
	Reason string `json:"reason"`
}

// EnvReport is the result of a detection pass for a single MCP.
type EnvReport struct {
	Configured []EnvMatch   `json:"configured"`
	Missing    []EnvMissing `json:"missing"`
}

// Report returns a structured report for one MCP.
func (d *Detector) Report(mcpID string, hostEnv []string) EnvReport {
	var m *manifest.Manifest
	for i := range d.catalog {
		if d.catalog[i].ID == mcpID {
			m = &d.catalog[i]
			break
		}
	}
	if m == nil {
		return EnvReport{}
	}
	envMap := parseHostEnv(hostEnv)
	report := EnvReport{}
	for _, ev := range m.EnvSchema {
		if _, ok := envMap[ev.Name]; ok {
			report.Configured = append(report.Configured, EnvMatch{MCP: mcpID, Var: ev.Name, Source: ev.Name})
			continue
		}
		found := false
		for _, alias := range ev.Aliases {
			if _, ok := envMap[alias]; ok {
				report.Configured = append(report.Configured, EnvMatch{MCP: mcpID, Var: ev.Name, Source: alias})
				found = true
				break
			}
		}
		if !found && ev.Required {
			report.Missing = append(report.Missing, EnvMissing{MCP: mcpID, Var: ev.Name, Reason: "not found in host environment"})
		}
	}
	return report
}

func parseHostEnv(hostEnv []string) map[string]string {
	out := map[string]string{}
	for _, kv := range hostEnv {
		if i := strings.IndexByte(kv, '='); i > 0 {
			out[kv[:i]] = kv[i+1:]
		}
	}
	return out
}
