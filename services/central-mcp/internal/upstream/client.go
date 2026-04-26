// Package upstream manages a single child MCP process: spawn, JSON-RPC client
// over its stdio, lazy-start, and idle shutdown.
//
// A Client is goroutine-safe. Call Start() once; concurrent Call() invocations
// are multiplexed by request id.
package upstream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"github.com/onemcp/central-mcp/internal/framing"
	"github.com/onemcp/central-mcp/internal/proto"
)

// Spec describes how to launch a child MCP. Mirrors a subset of the manifest
// entrypoint so the registry can hand it to us directly.
type Spec struct {
	ID      string            // stable id, used as tool prefix
	Command string            // executable path
	Args    []string          // process args
	Env     map[string]string // additional env (merged onto os.Environ at spawn)
	Cwd     string
}

// Client is a JSON-RPC client over a child process's stdio.
type Client struct {
	spec   Spec
	logger *slog.Logger

	mu       sync.Mutex
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	stdout   io.ReadCloser
	writer   *framing.Writer
	pending  map[string]chan *proto.Message
	nextID   atomic.Uint64
	started  bool
	closed   bool
	exitCh   chan struct{}
	tools    []proto.Tool // cached after initialize
	toolsErr error
}

func New(spec Spec, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}
	return &Client{
		spec:    spec,
		logger:  logger.With("mcp", spec.ID),
		pending: make(map[string]chan *proto.Message),
		exitCh:  make(chan struct{}),
	}
}

// Start spawns the child, performs the MCP initialize handshake, and caches
// the upstream tools list. Subsequent calls are no-ops.
func (c *Client) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.started {
		c.mu.Unlock()
		return nil
	}
	if c.closed {
		c.mu.Unlock()
		return errors.New("upstream: client closed")
	}

	cmd := exec.CommandContext(context.Background(), c.spec.Command, c.spec.Args...)
	if c.spec.Cwd != "" {
		cmd.Dir = c.spec.Cwd
	}
	if len(c.spec.Env) > 0 {
		env := make([]string, 0, len(c.spec.Env))
		for k, v := range c.spec.Env {
			env = append(env, k+"="+v)
		}
		// inherit parent env, append overrides last so they win
		cmd.Env = append(cmd.Environ(), env...)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		c.mu.Unlock()
		return fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		c.mu.Unlock()
		return fmt.Errorf("stdout pipe: %w", err)
	}
	// Forward child stderr to our logger.
	stderr, err := cmd.StderrPipe()
	if err != nil {
		c.mu.Unlock()
		return fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		c.mu.Unlock()
		return fmt.Errorf("start child: %w", err)
	}

	c.cmd = cmd
	c.stdin = stdin
	c.stdout = stdout
	c.writer = framing.NewWriter(stdin)
	c.started = true
	c.mu.Unlock()

	go c.readLoop()
	go c.drainStderr(stderr)
	go c.waitProc()

	if err := c.handshake(ctx); err != nil {
		_ = c.Close()
		return err
	}
	return nil
}

func (c *Client) drainStderr(r io.ReadCloser) {
	defer r.Close()
	br := framing.NewReader(r) // reuse: it's just a line scanner
	for {
		line, err := br.Read()
		if err != nil {
			return
		}
		if len(line) > 0 {
			c.logger.Debug("child stderr", "line", string(line))
		}
	}
}

func (c *Client) waitProc() {
	if err := c.cmd.Wait(); err != nil {
		c.logger.Warn("child exited", "err", err)
	} else {
		c.logger.Info("child exited cleanly")
	}
	close(c.exitCh)
	c.failPending(errors.New("upstream: child process exited"))
}

func (c *Client) failPending(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for id, ch := range c.pending {
		select {
		case ch <- &proto.Message{Error: proto.NewError(proto.ErrInternal, err.Error(), nil)}:
		default:
		}
		delete(c.pending, id)
	}
}

func (c *Client) readLoop() {
	r := framing.NewReader(c.stdout)
	for {
		raw, err := r.Read()
		if err != nil {
			if err != io.EOF {
				c.logger.Warn("read loop", "err", err)
			}
			return
		}
		var msg proto.Message
		if err := json.Unmarshal(raw, &msg); err != nil {
			c.logger.Warn("decode upstream message", "err", err, "raw", truncate(raw, 256))
			continue
		}
		// Notifications from the child are ignored for Phase 1 (no listChanged
		// propagation yet). Responses route to the pending waiter.
		if msg.IsResponse() {
			id := msg.ID.String()
			c.mu.Lock()
			ch, ok := c.pending[id]
			if ok {
				delete(c.pending, id)
			}
			c.mu.Unlock()
			if ok {
				ch <- &msg
			}
		}
	}
}

// Call sends a request and waits for the matching response, or ctx.Done().
func (c *Client) Call(ctx context.Context, method string, params any) (*proto.Message, error) {
	if !c.started {
		return nil, errors.New("upstream: not started")
	}
	seq := c.nextID.Add(1)
	rpcID := proto.NewStringID(fmt.Sprintf("u-%d", seq))
	// Map keys must match what readLoop sees on the response (raw JSON bytes
	// including surrounding quotes for string ids), otherwise responses are
	// dropped silently and the caller deadlocks until ctx expires.
	key := rpcID.String()

	var rawParams json.RawMessage
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("marshal params: %w", err)
		}
		rawParams = b
	}
	req := proto.Message{
		JSONRPC: proto.Version,
		ID:      &rpcID,
		Method:  method,
		Params:  rawParams,
	}

	ch := make(chan *proto.Message, 1)
	c.mu.Lock()
	c.pending[key] = ch
	c.mu.Unlock()

	if err := c.writer.Write(&req); err != nil {
		c.mu.Lock()
		delete(c.pending, key)
		c.mu.Unlock()
		return nil, fmt.Errorf("write request: %w", err)
	}

	select {
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.pending, key)
		c.mu.Unlock()
		return nil, ctx.Err()
	case <-c.exitCh:
		return nil, errors.New("upstream: child exited before response")
	case resp := <-ch:
		if resp.Error != nil {
			return resp, resp.Error
		}
		return resp, nil
	}
}

// Notify sends a JSON-RPC notification (no id, no response expected).
func (c *Client) Notify(method string, params any) error {
	var rawParams json.RawMessage
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return err
		}
		rawParams = b
	}
	return c.writer.Write(&proto.Message{
		JSONRPC: proto.Version,
		Method:  method,
		Params:  rawParams,
	})
}

func (c *Client) handshake(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	resp, err := c.Call(ctx, "initialize", proto.InitializeParams{
		ProtocolVersion: proto.ProtocolVersion,
		ClientInfo:      proto.Implementation{Name: "onemcp-central", Version: "0.1.0"},
		Capabilities:    json.RawMessage(`{}`),
	})
	if err != nil {
		return fmt.Errorf("initialize: %w", err)
	}
	_ = resp // we don't currently use server info; reserved for capability negotiation
	if err := c.Notify("notifications/initialized", struct{}{}); err != nil {
		return fmt.Errorf("initialized notify: %w", err)
	}

	// Cache tools list.
	tctx, tcancel := context.WithTimeout(ctx, 10*time.Second)
	defer tcancel()
	tresp, err := c.Call(tctx, "tools/list", struct{}{})
	if err != nil {
		// Some MCPs only expose resources; tolerate by caching empty.
		c.logger.Warn("tools/list failed; continuing with empty tool set", "err", err)
		c.toolsErr = err
		return nil
	}
	var lr proto.ListToolsResult
	if len(tresp.Result) > 0 {
		if err := json.Unmarshal(tresp.Result, &lr); err != nil {
			return fmt.Errorf("decode tools/list: %w", err)
		}
	}
	c.mu.Lock()
	c.tools = lr.Tools
	c.mu.Unlock()
	c.logger.Info("upstream ready", "tools", len(lr.Tools))
	return nil
}

// Tools returns the cached tools list (read-only snapshot).
func (c *Client) Tools() []proto.Tool {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]proto.Tool, len(c.tools))
	copy(out, c.tools)
	return out
}

// ID returns the upstream id (used as tool name prefix).
func (c *Client) ID() string { return c.spec.ID }

// Close terminates the child process.
func (c *Client) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	cmd := c.cmd
	stdin := c.stdin
	c.mu.Unlock()

	if stdin != nil {
		_ = stdin.Close()
	}
	if cmd != nil && cmd.Process != nil {
		// Give it a beat to exit on stdin close, then kill.
		done := make(chan struct{})
		go func() { <-c.exitCh; close(done) }()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			_ = cmd.Process.Kill()
		}
	}
	return nil
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "..."
}
