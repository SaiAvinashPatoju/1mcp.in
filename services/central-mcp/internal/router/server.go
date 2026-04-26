// Package router is the supervisor-backed stdio MCP server. It speaks MCP
// over its own stdio (to the agent client) and delegates lifecycle and
// routing decisions to the supervisor package. Tool name namespacing
// ("<id>__<tool>") is enforced inside supervisor; the router is transport-only.
package router

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"

	"github.com/onemcp/central-mcp/internal/framing"
	"github.com/onemcp/central-mcp/internal/proto"
	"github.com/onemcp/central-mcp/internal/supervisor"
)

// Server speaks MCP over stdio to a single client.
type Server struct {
	logger *slog.Logger
	in     *framing.Reader
	out    *framing.Writer
	sup    *supervisor.Supervisor
}

// New builds a Server.
func New(r io.Reader, w io.Writer, sup *supervisor.Supervisor, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{
		logger: logger,
		in:     framing.NewReader(r),
		out:    framing.NewWriter(w),
		sup:    sup,
	}
}

// Run blocks until EOF or ctx cancellation. Each request runs on its own
// goroutine so a slow upstream cannot stall the connection.
func (s *Server) Run(ctx context.Context) error {
	var wg sync.WaitGroup
	defer wg.Wait()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		raw, err := s.in.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("router read: %w", err)
		}
		buf := make([]byte, len(raw))
		copy(buf, raw)

		var msg proto.Message
		if err := json.Unmarshal(buf, &msg); err != nil {
			s.logger.Warn("decode client message", "err", err)
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.dispatch(ctx, &msg)
		}()
	}
}

func (s *Server) dispatch(ctx context.Context, msg *proto.Message) {
	switch {
	case msg.IsRequest():
		s.handleRequest(ctx, msg)
	case msg.IsNotification():
		// notifications/initialized and friends acknowledged silently in MVP.
	default:
		s.logger.Warn("unexpected message shape", "method", msg.Method)
	}
}

func (s *Server) handleRequest(ctx context.Context, msg *proto.Message) {
	switch msg.Method {
	case "initialize":
		s.replyOK(msg.ID, proto.InitializeResult{
			ProtocolVersion: proto.ProtocolVersion,
			ServerInfo:      proto.Implementation{Name: "onemcp", Version: "0.1.0"},
			Capabilities: proto.ServerCapabilities{
				Tools: &proto.ToolsCapability{ListChanged: false},
			},
		})
	case "tools/list":
		s.replyOK(msg.ID, proto.ListToolsResult{Tools: s.sup.Tools()})
	case "tools/call":
		s.handleToolsCall(ctx, msg)
	case "onemcp/rankTools":
		s.handleRankTools(msg)
	case "ping":
		s.replyOK(msg.ID, struct{}{})
	default:
		s.replyErr(msg.ID, proto.NewError(proto.ErrMethodNotFound, "method not supported in MVP: "+msg.Method, nil))
	}
}

func (s *Server) handleToolsCall(ctx context.Context, msg *proto.Message) {
	var p proto.CallToolParams
	if err := json.Unmarshal(msg.Params, &p); err != nil {
		s.replyErr(msg.ID, proto.NewError(proto.ErrInvalidParams, "invalid tools/call params", err.Error()))
		return
	}
	result, rpcErr := s.sup.Call(ctx, p.Name, p.Arguments)
	if rpcErr != nil {
		s.replyErr(msg.ID, rpcErr)
		return
	}
	s.replyRaw(msg.ID, result)
}

func (s *Server) replyOK(id *proto.ID, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		s.replyErr(id, proto.NewError(proto.ErrInternal, "marshal result", err.Error()))
		return
	}
	s.replyRaw(id, b)
}

func (s *Server) replyRaw(id *proto.ID, result json.RawMessage) {
	if id == nil {
		return
	}
	resp := proto.Message{JSONRPC: proto.Version, ID: id, Result: result}
	if err := s.out.Write(&resp); err != nil {
		s.logger.Error("write response", "err", err)
	}
}

func (s *Server) replyErr(id *proto.ID, e *proto.RPCError) {
	if id == nil {
		return
	}
	resp := proto.Message{JSONRPC: proto.Version, ID: id, Error: e}
	if err := s.out.Write(&resp); err != nil {
		s.logger.Error("write error response", "err", err)
	}
}

// handleRankTools is OneMCP's custom MCP extension for semantic tool surfacing.
// Clients pass {query, k}; we return top-k full Tool entries by relevance.
// Phase 8+ publishes this through the OneMCP SDK so non-VS Code clients can opt in.
func (s *Server) handleRankTools(msg *proto.Message) {
	var p struct {
		Query string `json:"query"`
		K     int    `json:"k"`
	}
	if err := json.Unmarshal(msg.Params, &p); err != nil {
		s.replyErr(msg.ID, proto.NewError(proto.ErrInvalidParams, "invalid params", err.Error()))
		return
	}
	if p.K <= 0 {
		p.K = 5
	}
	s.replyOK(msg.ID, struct {
		Tools []proto.Tool `json:"tools"`
	}{Tools: s.sup.RankTools(p.Query, p.K)})
}
