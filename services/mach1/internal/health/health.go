// Package health provides a lightweight health checker that wraps the
// supervisor. It is kept separate so the metatools package can depend on it
// without pulling in all supervisor internals.
package health

import (
	"context"
	"log/slog"

	"github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/supervisor"
)

// Checker aggregates health signals for individual MCPs.
type Checker struct {
	sup    *supervisor.Supervisor
	logger *slog.Logger
}

// New builds a Checker.
func New(sup *supervisor.Supervisor, logger *slog.Logger) *Checker {
	if logger == nil {
		logger = slog.Default()
	}
	return &Checker{sup: sup, logger: logger}
}

// Result is the normalized health output for one MCP.
type Result struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	ProcessOK bool   `json:"process_ok"`
	AuthOK    bool   `json:"auth_ok"`
	Message   string `json:"message,omitempty"`
}

// Check delegates to the supervisor and maps the result.
func (c *Checker) Check(ctx context.Context, mcpID string) (*Result, error) {
	hr, err := c.sup.HealthCheck(ctx, mcpID)
	if err != nil {
		return nil, err
	}
	return &Result{
		ID:        hr.ID,
		Status:    hr.Status,
		ProcessOK: hr.ProcessOK,
		AuthOK:    hr.AuthOK,
		Message:   hr.Message,
	}, nil
}
