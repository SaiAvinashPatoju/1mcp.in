// Package supervisor owns the lifecycle of every installed-and-enabled MCP.
//
// It performs a brief warmup at Start() to capture each MCP's tools/list, then
// shuts the children down. tools/call requests trigger lazy-start, and an
// idle timer (per-MCP, configurable via manifest.lifecycle.idleShutdownSeconds)
// closes the child when traffic stops.
//
// Tool names exposed to the client are namespaced as "<id>__<toolName>" to
// avoid collisions across MCPs.
package supervisor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/envtmpl"
	"github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/manifest"
	"github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/observability"
	"github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/proto"
	"github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/registry"
	"github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/sandbox"
	"github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/secrets"
	"github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/security"
	"github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/upstream"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
) // NamespaceSep separates upstream id from the original tool name in surfaced
// tool names. Kept in sync with the (now legacy) router constant.
const NamespaceSep = "__"

// Supervisor is goroutine-safe.
type Supervisor struct {
	logger *slog.Logger

	mu    sync.RWMutex
	items map[string]*managed
	order []string

	// warmupTimeout caps initial tools/list discovery per MCP.
	warmupTimeout time.Duration
	registry      *registry.DB
	metrics       *observability.Metrics
}

type managed struct {
	id       string
	manifest *manifest.Manifest
	driver   sandbox.Driver
	command  string
	args     []string
	cwd      string
	env      map[string]string
	idleSec  int

	mu        sync.Mutex
	client    *upstream.Client
	starting  chan struct{} // closed when start completes (success or fail)
	startErr  error
	idleTimer *time.Timer
	tools     []proto.Tool // namespaced
	blocked   map[string]registry.ToolReview
	verified  bool
}

// Options configures a Supervisor at construction time.
type Options struct {
	Logger        *slog.Logger
	WarmupTimeout time.Duration // default 15s
	Registry      *registry.DB
	Metrics       *observability.Metrics
}

// New builds a Supervisor from the registry and secrets store. Entries with
// missing required env are still tracked but will fail at Call() with a clear
// error; we prefer that over silently dropping them.
func New(entries []registry.Entry, getManifest func(id string) (*manifest.Manifest, error), sec *secrets.Store, opts Options) (*Supervisor, error) {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	if opts.WarmupTimeout == 0 {
		opts.WarmupTimeout = 15 * time.Second
	}
	s := &Supervisor{
		logger:        opts.Logger,
		items:         make(map[string]*managed, len(entries)),
		warmupTimeout: opts.WarmupTimeout,
		registry:      opts.Registry,
		metrics:       opts.Metrics,
	}
	for _, e := range entries {
		m, err := getManifest(e.ID)
		if err != nil {
			s.logger.Warn("skip MCP: cannot load manifest", "id", e.ID, "err", err)
			continue
		}
		drv, err := sandbox.Pick(m)
		if err != nil {
			s.logger.Warn("skip MCP: no driver", "id", e.ID, "err", err)
			continue
		}
		// Resolve env: registry non-secret + secrets (secrets win for same key).
		envMap := map[string]string{}
		for k, v := range e.Env {
			envMap[k] = v
		}
		if sec != nil {
			for k, v := range sec.Get(e.ID) {
				envMap[k] = v
			}
		}
		// Apply defaults from manifest envSchema for any keys still unset.
		for _, ev := range m.EnvSchema {
			if _, ok := envMap[ev.Name]; !ok && ev.Default != "" {
				envMap[ev.Name] = ev.Default
			}
		}
		// Build the lookup table used for ${VAR} expansion in command/args.
		// Order: process env (low) <- envMap (high), so users can override
		// system env per-MCP, but unset references can still resolve from the
		// parent process env (e.g. PATH-like vars or test fixtures).
		lookup := map[string]string{}
		for _, kv := range os.Environ() {
			if i := strings.IndexByte(kv, '='); i > 0 {
				lookup[kv[:i]] = kv[i+1:]
			}
		}
		for k, v := range envMap {
			lookup[k] = v
		}
		// Template-expand command/args using the lookup so manifests can
		// reference ${VAR} without the child process doing its own expansion.
		cmd, missingCmd := envtmpl.Expand(e.Command, lookup)
		args, missingArgs := envtmpl.ExpandAll(e.Args, lookup)
		missing := append(missingCmd, missingArgs...)
		if len(missing) > 0 {
			s.logger.Warn("MCP has unresolved ${VAR} refs; tools/call will fail until configured",
				"id", e.ID, "missing", missing)
		}

		mg := &managed{
			id:       e.ID,
			manifest: m,
			driver:   drv,
			command:  cmd,
			args:     args,
			cwd:      e.Cwd,
			env:      envMap,
			idleSec:  m.IdleShutdown(),
			blocked:  map[string]registry.ToolReview{},
		}
		s.items[e.ID] = mg
		s.order = append(s.order, e.ID)
	}
	return s, nil
}

// Start performs parallel warmup: for each item, briefly start the upstream,
// fetch tools/list, namespace it, then close the child so it stops consuming
// resources until first Call.
func (s *Supervisor) Start(ctx context.Context) {
	var wg sync.WaitGroup
	for _, id := range s.order {
		mg := s.items[id]
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.warmup(ctx, mg)
		}()
	}
	wg.Wait()
}

func (s *Supervisor) warmup(ctx context.Context, mg *managed) {
	wctx, cancel := context.WithTimeout(ctx, s.warmupTimeout)
	defer cancel()

	c, err := s.startLocked(wctx, mg)
	if err != nil {
		s.logger.Warn("warmup failed", "id", mg.id, "err", err)
		return
	}

	if err := s.ensureToolsVerified(wctx, mg, c); err != nil {
		s.logger.Warn("tool hash verification failed", "id", mg.id, "err", err)
	}

	mg.mu.Lock()
	toolCount := len(mg.tools)
	blockedCount := len(mg.blocked)
	mg.mu.Unlock()
	s.logger.Info("warmup complete", "id", mg.id, "tools", toolCount, "blocked", blockedCount)

	// Immediately schedule idle close so warm processes don't linger.
	s.scheduleIdleLocked(mg)
}

// startLocked spawns mg.client if not running. Concurrent callers wait on
// mg.starting. Returns the live client or an error.
func (s *Supervisor) startLocked(ctx context.Context, mg *managed) (*upstream.Client, error) {
	mg.mu.Lock()
	if mg.client != nil {
		c := mg.client
		s.resetIdleLocked(mg)
		mg.mu.Unlock()
		return c, nil
	}
	if mg.starting != nil {
		ch := mg.starting
		mg.mu.Unlock()
		select {
		case <-ch:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		mg.mu.Lock()
		c, err := mg.client, mg.startErr
		mg.mu.Unlock()
		if err != nil {
			return nil, err
		}
		return c, nil
	}
	mg.starting = make(chan struct{})
	mg.mu.Unlock()

	spec, err := mg.driver.Spec(mg.id, mg.manifest, mg.command, mg.args, mg.cwd, mg.env)
	var c *upstream.Client
	if err == nil {
		c = upstream.New(spec, s.logger)
		err = c.Start(ctx)
	}

	mg.mu.Lock()
	if err != nil {
		mg.startErr = err
		close(mg.starting)
		mg.starting = nil
		mg.mu.Unlock()
		return nil, err
	}
	mg.client = c
	mg.startErr = nil
	mg.verified = false
	close(mg.starting)
	mg.starting = nil
	s.resetIdleLocked(mg)
	mg.mu.Unlock()
	return c, nil
}

// resetIdleLocked must be called with mg.mu held.
func (s *Supervisor) resetIdleLocked(mg *managed) {
	if mg.idleTimer != nil {
		mg.idleTimer.Stop()
	}
	if mg.idleSec <= 0 {
		return
	}
	mg.idleTimer = time.AfterFunc(time.Duration(mg.idleSec)*time.Second, func() {
		s.shutdownIdle(mg)
	})
}

// scheduleIdleLocked starts (or resets) the idle timer with a fresh interval.
func (s *Supervisor) scheduleIdleLocked(mg *managed) {
	mg.mu.Lock()
	defer mg.mu.Unlock()
	s.resetIdleLocked(mg)
}

func (s *Supervisor) shutdownIdle(mg *managed) {
	mg.mu.Lock()
	c := mg.client
	mg.client = nil
	mg.idleTimer = nil
	mg.verified = false
	mg.mu.Unlock()
	if c != nil {
		s.logger.Debug("idle shutdown", "id", mg.id)
		_ = c.Close()
	}
}

// Tools returns the cached, namespaced tool list across all MCPs.
func (s *Supervisor) Tools() []proto.Tool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]proto.Tool, 0, 16)
	for _, id := range s.order {
		mg := s.items[id]
		mg.mu.Lock()
		out = append(out, mg.tools...)
		mg.mu.Unlock()
	}
	return out
}

// Call routes a namespaced tool name to the right MCP, lazy-starting it if
// needed. The result bytes are forwarded verbatim to preserve any extension
// fields the upstream returned.
func (s *Supervisor) Call(ctx context.Context, toolName string, args json.RawMessage) (json.RawMessage, *proto.RPCError) {
	start := time.Now()
	id, suffix, ok := strings.Cut(toolName, NamespaceSep)
	if !ok {
		return nil, proto.NewError(proto.ErrInvalidParams, "tool name missing upstream prefix", toolName)
	}
	ctx, span := otel.Tracer("github.com/SaiAvinashPatoju/1mcp.in/services/mach1/router").Start(ctx, "mcp.tool_call")
	span.SetAttributes(attribute.String("mcp.server", id), attribute.String("mcp.tool", suffix))
	success := false
	defer func() {
		span.SetAttributes(attribute.Bool("mcp.success", success))
		span.End()
		if s.metrics != nil {
			s.metrics.Record(observability.ToolCall{MCPID: id, ToolName: suffix, Success: success, Duration: time.Since(start)})
		}
	}()
	s.mu.RLock()
	mg := s.items[id]
	s.mu.RUnlock()
	if mg == nil {
		return nil, proto.NewError(proto.ErrMethodNotFound, "unknown upstream", id)
	}
	c, err := s.startLocked(ctx, mg)
	if err != nil {
		return nil, proto.NewError(proto.ErrInternal, "start upstream: "+err.Error(), nil)
	}
	if err := s.ensureToolsVerified(ctx, mg, c); err != nil {
		return nil, proto.NewError(proto.ErrInternal, "tool definition verification: "+err.Error(), nil)
	}
	mg.mu.Lock()
	if review, ok := mg.blocked[suffix]; ok {
		mg.mu.Unlock()
		return nil, proto.NewError(proto.ErrInternal, "tool definition changed and is pending review", map[string]string{
			"mcp":           review.MCPID,
			"tool":          review.ToolName,
			"approved_hash": review.ApprovedHash,
			"current_hash":  review.CurrentHash,
		})
	}
	mg.mu.Unlock()
	resp, err := c.Call(ctx, "tools/call", proto.CallToolParams{Name: suffix, Arguments: args})
	// Reset idle timer regardless of success.
	mg.mu.Lock()
	s.resetIdleLocked(mg)
	mg.mu.Unlock()

	if err != nil {
		if resp != nil && resp.Error != nil {
			if len(resp.Error.Data) > 0 {
				resp.Error.Data = security.RedactJSON(resp.Error.Data)
			}
			resp.Error.Message = security.RedactString(resp.Error.Message)
			return nil, resp.Error
		}
		return nil, proto.NewError(proto.ErrInternal, "upstream call: "+err.Error(), nil)
	}
	success = true
	return resp.Result, nil
}

func (s *Supervisor) ensureToolsVerified(ctx context.Context, mg *managed, c *upstream.Client) error {
	mg.mu.Lock()
	if mg.verified {
		mg.mu.Unlock()
		return nil
	}
	mg.mu.Unlock()

	raw := c.Tools()
	statuses := map[string]registry.ToolReview{}
	if s.registry != nil && len(raw) > 0 {
		defs := make([]registry.ToolDefinition, 0, len(raw))
		for _, t := range raw {
			defs = append(defs, registry.ToolDefinition{Name: t.Name, Description: t.Description, InputSchema: t.InputSchema})
		}
		verified, err := s.registry.VerifyToolDefinitions(ctx, mg.id, defs)
		if err != nil {
			return err
		}
		statuses = verified
	}

	tools := make([]proto.Tool, 0, len(raw))
	blocked := map[string]registry.ToolReview{}
	for _, t := range raw {
		if anns, ok := mg.manifest.ToolAnnotations[t.Name]; ok {
			t.Annotations = &proto.ToolAnnotations{ReadOnly: anns.ReadOnly, Destructive: anns.Destructive, Idempotent: anns.Idempotent}
		}
		if review, ok := statuses[t.Name]; ok && review.Status == registry.ToolStatusPendingReview {
			blocked[t.Name] = review
			s.logger.Warn("tool definition changed; blocking pending approval", "mcp", mg.id, "tool", t.Name, "approved_hash", review.ApprovedHash, "current_hash", review.CurrentHash)
			continue
		}
		t.Name = mg.id + NamespaceSep + t.Name
		tools = append(tools, t)
	}

	mg.mu.Lock()
	mg.tools = tools
	mg.blocked = blocked
	mg.verified = true
	mg.mu.Unlock()
	return nil
}

// Close stops every running upstream.
func (s *Supervisor) Close() error {
	var errs []error
	for _, id := range s.order {
		mg := s.items[id]
		mg.mu.Lock()
		c := mg.client
		mg.client = nil
		if mg.idleTimer != nil {
			mg.idleTimer.Stop()
		}
		mg.mu.Unlock()
		if c != nil {
			if err := c.Close(); err != nil {
				errs = append(errs, fmt.Errorf("close %s: %w", id, err))
			}
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// IDs returns the supervised MCP ids in stable order (testing helper).
func (s *Supervisor) IDs() []string {
	out := make([]string, len(s.order))
	copy(out, s.order)
	return out
}
