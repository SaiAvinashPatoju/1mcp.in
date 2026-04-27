package observability

import (
	"strings"
	"testing"
	"time"
)

func TestMetricsRender(t *testing.T) {
	m := NewMetrics()
	m.Record(ToolCall{MCPID: "github", ToolName: "list_prs", Success: true, Duration: 2 * time.Millisecond})
	m.Record(ToolCall{MCPID: "github", ToolName: "list_prs", Success: false, Duration: time.Millisecond})
	out := m.Render()
	for _, want := range []string{"onemcp_tool_calls_total", "onemcp_tool_call_errors_total", "github", "list_prs"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %s", want, out)
		}
	}
}
