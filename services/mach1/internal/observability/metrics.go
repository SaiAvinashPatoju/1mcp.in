package observability

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

type ToolCall struct {
	MCPID    string
	ToolName string
	Success  bool
	Duration time.Duration
}

type Metrics struct {
	mu      sync.RWMutex
	calls   map[string]uint64
	errors  map[string]uint64
	latency map[string]time.Duration
	started time.Time
}

func NewMetrics() *Metrics {
	return &Metrics{
		calls:   map[string]uint64{},
		errors:  map[string]uint64{},
		latency: map[string]time.Duration{},
		started: time.Now(),
	}
}

func (m *Metrics) Record(call ToolCall) {
	if m == nil {
		return
	}
	key := call.MCPID + "\xff" + call.ToolName
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls[key]++
	m.latency[key] += call.Duration
	if !call.Success {
		m.errors[key]++
	}
}

func (m *Metrics) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		_, _ = w.Write([]byte(m.Render()))
	})
}

func (m *Metrics) Render() string {
	if m == nil {
		return ""
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	keys := make([]string, 0, len(m.calls))
	for key := range m.calls {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var b strings.Builder
	fmt.Fprintf(&b, "# HELP mach1_router_uptime_seconds Router process uptime.\n")
	fmt.Fprintf(&b, "# TYPE mach1_router_uptime_seconds gauge\n")
	fmt.Fprintf(&b, "mach1_router_uptime_seconds %.0f\n", time.Since(m.started).Seconds())
	fmt.Fprintf(&b, "# HELP mach1_tool_calls_total Tool calls routed by mach1.\n")
	fmt.Fprintf(&b, "# TYPE mach1_tool_calls_total counter\n")
	for _, key := range keys {
		mcpID, toolName := splitKey(key)
		fmt.Fprintf(&b, "mach1_tool_calls_total{mcp=%q,tool=%q} %d\n", mcpID, toolName, m.calls[key])
	}
	fmt.Fprintf(&b, "# HELP mach1_tool_call_errors_total Failed tool calls routed by mach1.\n")
	fmt.Fprintf(&b, "# TYPE mach1_tool_call_errors_total counter\n")
	for _, key := range keys {
		mcpID, toolName := splitKey(key)
		fmt.Fprintf(&b, "mach1_tool_call_errors_total{mcp=%q,tool=%q} %d\n", mcpID, toolName, m.errors[key])
	}
	fmt.Fprintf(&b, "# HELP mach1_tool_call_latency_seconds_sum Cumulative routed tool call latency.\n")
	fmt.Fprintf(&b, "# TYPE mach1_tool_call_latency_seconds_sum counter\n")
	for _, key := range keys {
		mcpID, toolName := splitKey(key)
		fmt.Fprintf(&b, "mach1_tool_call_latency_seconds_sum{mcp=%q,tool=%q} %.6f\n", mcpID, toolName, m.latency[key].Seconds())
	}
	return b.String()
}

func splitKey(key string) (string, string) {
	parts := strings.SplitN(key, "\xff", 2)
	if len(parts) != 2 {
		return key, ""
	}
	return parts[0], parts[1]
}
