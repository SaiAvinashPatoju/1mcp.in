// Package proto defines JSON-RPC 2.0 envelopes and the slice of MCP messages
// the router needs for Phase 1 (initialize, tools/list, tools/call,
// notifications/initialized). Kept dependency-free on purpose.
package proto

import (
	"encoding/json"
	"errors"
)

// JSON-RPC 2.0 ----------------------------------------------------------------

const Version = "2.0"

// ID is a JSON-RPC id. Spec allows string, number, or null. We carry the raw
// bytes so we can echo it back byte-for-byte without losing numeric type.
type ID struct{ raw json.RawMessage }

func (i ID) MarshalJSON() ([]byte, error) {
	if len(i.raw) == 0 {
		return []byte("null"), nil
	}
	return i.raw, nil
}

func (i *ID) UnmarshalJSON(b []byte) error {
	i.raw = append(i.raw[:0], b...)
	return nil
}

func (i ID) IsNull() bool { return len(i.raw) == 0 || string(i.raw) == "null" }
func (i ID) String() string {
	if i.IsNull() {
		return "<null>"
	}
	return string(i.raw)
}

// NewStringID is a helper for synthetic ids on outbound requests to upstreams.
func NewStringID(s string) ID {
	b, _ := json.Marshal(s)
	return ID{raw: b}
}

// Message is the union envelope. Exactly one of {Method present} (request or
// notification) or {Result/Error present} (response) is meaningful per message.
type Message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *ID             `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

func (m *Message) IsRequest() bool      { return m.Method != "" && m.ID != nil }
func (m *Message) IsNotification() bool { return m.Method != "" && m.ID == nil }
func (m *Message) IsResponse() bool     { return m.Method == "" && m.ID != nil }

type RPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e *RPCError) Error() string { return e.Message }

// Standard error codes.
const (
	ErrParse          = -32700
	ErrInvalidRequest = -32600
	ErrMethodNotFound = -32601
	ErrInvalidParams  = -32602
	ErrInternal       = -32603
)

// NewError builds an RPCError, optionally embedding data.
func NewError(code int, msg string, data any) *RPCError {
	e := &RPCError{Code: code, Message: msg}
	if data != nil {
		if b, err := json.Marshal(data); err == nil {
			e.Data = b
		}
	}
	return e
}

// MCP slice ------------------------------------------------------------------

// ProtocolVersion is the MCP protocol revision the router speaks.
// Pinned per ADR-0001; bump deliberately.
const ProtocolVersion = "2024-11-05"

type InitializeParams struct {
	ProtocolVersion string          `json:"protocolVersion"`
	Capabilities    json.RawMessage `json:"capabilities,omitempty"`
	ClientInfo      Implementation  `json:"clientInfo"`
}

type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      Implementation     `json:"serverInfo"`
}

type Implementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ServerCapabilities struct {
	Tools *ToolsCapability `json:"tools,omitempty"`
}

type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"inputSchema"`
	Annotations *ToolAnnotations `json:"annotations,omitempty"`
}

type ToolAnnotations struct {
	ReadOnly    bool `json:"readOnly,omitempty"`
	Destructive bool `json:"destructive,omitempty"`
	Idempotent  bool `json:"idempotent,omitempty"`
}

type ListToolsResult struct {
	Tools []Tool `json:"tools"`
}

type CallToolParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// CallToolResult is intentionally opaque (passed through verbatim from the
// upstream child); we only construct it on local error paths.
type CallToolResult struct {
	Content []ToolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

type ToolContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

var ErrShortRead = errors.New("proto: short read")
