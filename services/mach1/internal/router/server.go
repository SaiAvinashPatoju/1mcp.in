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

	"github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/framing"
	"github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/proto"
	"github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/security"
	"github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/supervisor"
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
			if resp := s.Handle(ctx, &msg); resp != nil {
				if err := s.out.Write(resp); err != nil {
					s.logger.Error("write response", "err", err)
				}
			}
		}()
	}
}

func (s *Server) Handle(ctx context.Context, msg *proto.Message) *proto.Message {
	switch {
	case msg.IsRequest():
		return s.handleRequest(ctx, msg)
	case msg.IsNotification():
		// notifications/initialized and friends acknowledged silently in MVP.
		return nil
	default:
		s.logger.Warn("unexpected message shape", "method", msg.Method)
		return nil
	}
}

func (s *Server) handleRequest(ctx context.Context, msg *proto.Message) *proto.Message {
	switch msg.Method {
	case "initialize":
		return s.ok(msg.ID, proto.InitializeResult{
			ProtocolVersion: proto.ProtocolVersion,
			ServerInfo:      proto.Implementation{Name: "mach1", Version: "0.1.0"},
			Capabilities: proto.ServerCapabilities{
				Tools: &proto.ToolsCapability{ListChanged: false},
			},
		})
	case "tools/list":
		return s.ok(msg.ID, proto.ListToolsResult{Tools: s.sup.Tools()})
	case "tools/call":
		return s.handleToolsCall(ctx, msg)
	case "mach1/rankTools":
		return s.handleRankTools(msg)
	case "ping":
		return s.ok(msg.ID, struct{}{})
	default:
		return s.err(msg.ID, proto.NewError(proto.ErrMethodNotFound, "method not supported: "+msg.Method, nil))
	}
}

func (s *Server) handleToolsCall(ctx context.Context, msg *proto.Message) *proto.Message {
	var p proto.CallToolParams
	if err := json.Unmarshal(msg.Params, &p); err != nil {
		return s.err(msg.ID, proto.NewError(proto.ErrInvalidParams, "invalid tools/call params", err.Error()))
	}
	result, rpcErr := s.sup.Call(ctx, p.Name, p.Arguments)
	if rpcErr != nil {
		return s.err(msg.ID, rpcErr)
	}
	return s.raw(msg.ID, result)
}

func (s *Server) ok(id *proto.ID, v any) *proto.Message {
	b, err := json.Marshal(v)
	if err != nil {
		return s.err(id, proto.NewError(proto.ErrInternal, "marshal result", err.Error()))
	}
	return s.raw(id, b)
}

func (s *Server) raw(id *proto.ID, result json.RawMessage) *proto.Message {
	if id == nil {
		return nil
	}
	return &proto.Message{JSONRPC: proto.Version, ID: id, Result: result}
}

func (s *Server) err(id *proto.ID, e *proto.RPCError) *proto.Message {
	if id == nil {
		return nil
	}
	if e != nil {
		e.Message = security.RedactString(e.Message)
		if len(e.Data) > 0 {
			e.Data = security.RedactJSON(e.Data)
		}
	}
	return &proto.Message{JSONRPC: proto.Version, ID: id, Error: e}
}

// handleRankTools is 1mcp.in's custom MCP extension for semantic tool surfacing.
// Clients pass {query, k}; we return top-k full Tool entries by relevance.
// Phase 8+ publishes this through the 1mcp.in SDK so non-VS Code clients can opt in.
func (s *Server) handleRankTools(msg *proto.Message) *proto.Message {
	var p struct {
		Query string `json:"query"`
		K     int    `json:"k"`
	}
	if err := json.Unmarshal(msg.Params, &p); err != nil {
		return s.err(msg.ID, proto.NewError(proto.ErrInvalidParams, "invalid params", err.Error()))
	}
	if p.K <= 0 {
		p.K = 5
	}
	return s.ok(msg.ID, struct {
		Tools []proto.Tool `json:"tools"`
	}{Tools: s.sup.RankTools(p.Query, p.K)})
}
