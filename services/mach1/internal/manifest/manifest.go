// Package manifest is the Go mirror of packages/mcp-manifest/manifest.schema.json.
// Hand-written for now; codegen lands in Phase 8 once the schema is stable.
package manifest

import (
	"encoding/json"
	"fmt"
	"regexp"
)

var (
	idPattern  = regexp.MustCompile(`^[a-z0-9][a-z0-9\-_]{1,62}[a-z0-9]$`)
	verPattern = regexp.MustCompile(`^\d+\.\d+\.\d+(?:[-+][A-Za-z0-9.-]+)?$`)
	envPattern = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)
)

// Manifest mirrors manifest.schema.json. JSON tags must match exactly.
type Manifest struct {
	ID              string                     `json:"id"`
	Name            string                     `json:"name"`
	Version         string                     `json:"version"`
	Description     string                     `json:"description,omitempty"`
	Homepage        string                     `json:"homepage,omitempty"`
	Author          string                     `json:"author,omitempty"`
	License         string                     `json:"license,omitempty"`
	Tags            []string                   `json:"tags,omitempty"`
	Transport       string                     `json:"transport"`
	Runtime         string                     `json:"runtime"`
	Entrypoint      Entrypoint                 `json:"entrypoint"`
	EnvSchema       []EnvVar                   `json:"envSchema,omitempty"`
	Permissions     *Permissions               `json:"permissions,omitempty"`
	ToolAnnotations map[string]ToolAnnotations `json:"toolAnnotations,omitempty"`
	Capabilities    []string                   `json:"capabilities,omitempty"`
	Verification    string                     `json:"verification,omitempty"`
	SHA256          string                     `json:"sha256,omitempty"`
	Signature       string                     `json:"signature,omitempty"`
	EmbeddingText   string                     `json:"embeddingText,omitempty"`
	Lifecycle       *Lifecycle                 `json:"lifecycle,omitempty"`
}

// Entrypoint is a discriminated union: command-style OR docker-image-style.
// Both shapes are decoded; the runtime field on Manifest determines which is
// expected. We surface helpers so callers don't switch on string fields.
type Entrypoint struct {
	// command-style
	Command string   `json:"command,omitempty"`
	Args    []string `json:"args,omitempty"`
	Cwd     string   `json:"cwd,omitempty"`
	// docker-style
	Image  string  `json:"image,omitempty"`
	Mounts []Mount `json:"mounts,omitempty"`
}

type Mount struct {
	Source   string `json:"source"`
	Target   string `json:"target"`
	ReadOnly bool   `json:"readOnly,omitempty"`
}

type EnvVar struct {
	Name        string `json:"name"`
	Label       string `json:"label,omitempty"`
	Description string `json:"description,omitempty"`
	Secret      bool   `json:"secret,omitempty"`
	Required    bool   `json:"required,omitempty"`
	Default     string `json:"default,omitempty"`
}

type Permissions struct {
	Network    bool     `json:"network,omitempty"`
	Filesystem *FSPerms `json:"filesystem,omitempty"`
}

type FSPerms struct {
	Read  []string `json:"read,omitempty"`
	Write []string `json:"write,omitempty"`
}

type Lifecycle struct {
	IdleShutdownSeconds int  `json:"idleShutdownSeconds,omitempty"`
	Autostart           bool `json:"autostart,omitempty"`
}

type ToolAnnotations struct {
	ReadOnly    bool `json:"readOnly,omitempty"`
	Destructive bool `json:"destructive,omitempty"`
	Idempotent  bool `json:"idempotent,omitempty"`
}

// Validate enforces the structural rules of the JSON Schema. We do this in
// addition to (not instead of) JSON unmarshalling because we want clear,
// per-field error messages for hub UI feedback.
func (m *Manifest) Validate() error {
	if !idPattern.MatchString(m.ID) {
		return fmt.Errorf("invalid id %q", m.ID)
	}
	if m.Name == "" {
		return fmt.Errorf("name required")
	}
	if !verPattern.MatchString(m.Version) {
		return fmt.Errorf("invalid version %q", m.Version)
	}
	switch m.Transport {
	case "stdio", "sse", "http":
	default:
		return fmt.Errorf("invalid transport %q", m.Transport)
	}
	switch m.Verification {
	case "", "anthropic-official", "1mcp.in-verified", "community":
	default:
		return fmt.Errorf("invalid verification %q", m.Verification)
	}
	switch m.Runtime {
	case "node", "python", "docker", "binary":
	default:
		return fmt.Errorf("invalid runtime %q", m.Runtime)
	}
	if m.Runtime == "docker" {
		if m.Entrypoint.Image == "" {
			return fmt.Errorf("docker runtime requires entrypoint.image")
		}
	} else {
		if m.Entrypoint.Command == "" {
			return fmt.Errorf("%s runtime requires entrypoint.command", m.Runtime)
		}
	}
	for _, e := range m.EnvSchema {
		if !envPattern.MatchString(e.Name) {
			return fmt.Errorf("invalid env var name %q", e.Name)
		}
	}
	return nil
}

// Parse decodes JSON bytes into a validated Manifest.
func Parse(b []byte) (*Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("decode manifest: %w", err)
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return &m, nil
}

// IdleShutdown returns the configured idle shutdown duration in seconds with
// a sensible default of 60 when unset.
func (m *Manifest) IdleShutdown() int {
	if m.Lifecycle == nil || m.Lifecycle.IdleShutdownSeconds <= 0 {
		return 60
	}
	return m.Lifecycle.IdleShutdownSeconds
}
