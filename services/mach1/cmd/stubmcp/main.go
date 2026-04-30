// stubmcp is a minimal in-tree MCP server used for end-to-end testing of
// mach1 without external runtimes (no npm, no python, no docker).
//
// It implements:
//   - initialize          Ã¢â€ â€™ returns server info
//   - tools/list          Ã¢â€ â€™ returns one "echo" tool
//   - tools/call(echo)    Ã¢â€ â€™ returns the arguments verbatim wrapped in a content block
//   - ping                Ã¢â€ â€™ empty result
//
// All other methods return method-not-found.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/proto"
)

func main() {
	r := bufio.NewScanner(os.Stdin)
	r.Buffer(make([]byte, 0, 64<<10), 8<<20)
	w := bufio.NewWriter(os.Stdout)
	for r.Scan() {
		var msg proto.Message
		if err := json.Unmarshal(r.Bytes(), &msg); err != nil {
			continue
		}
		if msg.IsRequest() {
			handle(&msg, w)
			_ = w.Flush()
		}
		// notifications and responses ignored
	}
}

func handle(msg *proto.Message, w *bufio.Writer) {
	resp := proto.Message{JSONRPC: proto.Version, ID: msg.ID}
	switch msg.Method {
	case "initialize":
		resp.Result = jsonMust(proto.InitializeResult{
			ProtocolVersion: proto.ProtocolVersion,
			ServerInfo:      proto.Implementation{Name: "stubmcp", Version: "0.1.0"},
			Capabilities:    proto.ServerCapabilities{Tools: &proto.ToolsCapability{}},
		})
	case "tools/list":
		resp.Result = jsonMust(proto.ListToolsResult{Tools: []proto.Tool{{
			Name:        "echo",
			Description: "Echoes its arguments back to the caller.",
			InputSchema: json.RawMessage(`{"type":"object","additionalProperties":true}`),
		}}})
	case "tools/call":
		var p proto.CallToolParams
		_ = json.Unmarshal(msg.Params, &p)
		text := string(p.Arguments)
		if text == "" {
			text = "{}"
		}
		resp.Result = jsonMust(proto.CallToolResult{
			Content: []proto.ToolContent{{Type: "text", Text: text}},
		})
	case "ping":
		resp.Result = json.RawMessage(`{}`)
	default:
		resp.Result = nil
		resp.Error = proto.NewError(proto.ErrMethodNotFound, "stub: unsupported method "+msg.Method, nil)
	}
	b, _ := json.Marshal(&resp)
	_, _ = w.Write(b)
	_, _ = w.Write([]byte{'\n'})
}

func jsonMust(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		fmt.Fprintln(os.Stderr, "stubmcp marshal:", err)
		os.Exit(1)
	}
	return b
}
