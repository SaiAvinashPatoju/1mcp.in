// Package composition executes a sequential chain of tool calls, stopping on
// the first error and returning partial results.
package composition

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/proto"
)

// Engine runs tool-call steps sequentially.
type Engine struct {
	caller func(ctx context.Context, toolName string, args json.RawMessage) (json.RawMessage, *proto.RPCError)
	logger *slog.Logger
}

// New builds an Engine. The caller must route to supervisor.Call or meta.Handle
// depending on the tool name prefix.
func New(caller func(ctx context.Context, toolName string, args json.RawMessage) (json.RawMessage, *proto.RPCError), logger *slog.Logger) *Engine {
	if logger == nil {
		logger = slog.Default()
	}
	return &Engine{caller: caller, logger: logger}
}

// Step is one tool invocation.
type Step struct {
	Tool string          `json:"tool"`
	Args json.RawMessage `json:"args"`
}

// StepResult records the outcome of a single step.
type StepResult struct {
	Tool   string          `json:"tool"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *proto.RPCError `json:"error,omitempty"`
}

// Result is the full output of a composition run.
type Result struct {
	Steps []StepResult `json:"results"`
}

// Run executes steps sequentially. It stops at the first error and returns
// the partial Result.
func (e *Engine) Run(ctx context.Context, steps []Step) (*Result, error) {
	out := &Result{Steps: make([]StepResult, 0, len(steps))}
	for _, step := range steps {
		raw, rpcErr := e.caller(ctx, step.Tool, step.Args)
		sr := StepResult{Tool: step.Tool, Result: raw, Error: rpcErr}
		out.Steps = append(out.Steps, sr)
		if rpcErr != nil {
			e.logger.Warn("compose step failed", "tool", step.Tool, "err", rpcErr.Message)
			return out, nil
		}
	}
	return out, nil
}
