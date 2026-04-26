// onemcpe2e is the OneMCP end-to-end test driver. It spawns a centralmcpd
// process, drives it as an MCP client over stdio, and emits a markdown report
// of latency + correctness per registered MCP.
//
// Usage:
//
//	onemcpe2e --bin path/to/centralmcpd \
//	          --config services/central-mcp/config.example.json \
//	          --smoke  test/e2e/smokes.json \
//	          --out    e2e-report.md
//
// Exit codes:
//   0  every MCP responded to tools/list and every smoke call succeeded
//   1  one or more failures (details in the report)
//   2  bad input / cannot start centralmcpd
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/onemcp/central-mcp/internal/framing"
	"github.com/onemcp/central-mcp/internal/proto"
)

// Smoke is one tool-call to attempt as a smoke test for a given MCP.
type Smoke struct {
	MCPID string          `json:"mcpId"`   // matches manifest id
	Tool  string          `json:"tool"`    // bare tool name (no namespace prefix)
	Args  json.RawMessage `json:"args"`    // arbitrary JSON arguments
	Allow []string        `json:"allowOK"` // result substrings any of which mark success (optional)
}

func main() {
	var (
		binPath    = flag.String("bin", findDefaultBin(), "path to centralmcpd binary")
		configPath = flag.String("config", "", "passed through as --config")
		dbPath     = flag.String("db", "", "passed through as --db")
		smokePath  = flag.String("smoke", "", "JSON array of Smoke entries (optional)")
		outPath    = flag.String("out", "e2e-report.md", "markdown report output path")
		timeout    = flag.Duration("timeout", 60*time.Second, "overall test timeout")
	)
	flag.Parse()

	if *binPath == "" {
		die(2, "--bin required and centralmcpd not found in PATH")
	}
	if *configPath == "" && *dbPath == "" {
		die(2, "either --config or --db is required")
	}

	smokes, err := loadSmokes(*smokePath)
	if err != nil {
		die(2, fmt.Sprintf("load smokes: %v", err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	report, err := runE2E(ctx, *binPath, *configPath, *dbPath, smokes)
	if err != nil {
		die(2, err.Error())
	}
	if err := os.WriteFile(*outPath, []byte(report.Markdown()), 0o644); err != nil {
		die(2, fmt.Sprintf("write report: %v", err))
	}
	fmt.Printf("report written: %s\n", *outPath)
	if report.HasFailures() {
		os.Exit(1)
	}
}

type Report struct {
	Started     time.Time
	Init        Latency
	ToolsList   Latency
	ToolsTotal  int
	Tools       map[string][]string // mcpId -> tool names
	SmokeRuns   []SmokeRun
	OverallFail string
}

type Latency struct {
	OK       bool
	Duration time.Duration
	Err      string
}

type SmokeRun struct {
	MCPID    string
	Tool     string
	OK       bool
	Duration time.Duration
	Err      string
}

func runE2E(ctx context.Context, bin, configPath, dbPath string, smokes []Smoke) (*Report, error) {
	rep := &Report{Started: time.Now(), Tools: map[string][]string{}}

	args := []string{"--log", "warn"}
	if configPath != "" {
		args = append(args, "--config", configPath)
	}
	if dbPath != "" {
		args = append(args, "--db", dbPath)
	}

	cmd := exec.CommandContext(ctx, bin, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return rep, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return rep, err
	}
	cmd.Stderr = os.Stderr // surface router logs to the user
	if err := cmd.Start(); err != nil {
		return rep, fmt.Errorf("start centralmcpd: %w", err)
	}
	defer func() {
		_ = stdin.Close()
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	r := framing.NewReader(stdout)
	w := framing.NewWriter(stdin)

	// initialize
	rep.Init = timed(func() error {
		_, err := rpc(ctx, w, r, 1, "initialize", proto.InitializeParams{
			ProtocolVersion: proto.ProtocolVersion,
			ClientInfo:      proto.Implementation{Name: "onemcpe2e", Version: "0.1.0"},
			Capabilities:    json.RawMessage(`{}`),
		})
		return err
	})
	if !rep.Init.OK {
		rep.OverallFail = "initialize failed: " + rep.Init.Err
		return rep, nil
	}
	_ = sendNotify(w, "notifications/initialized", struct{}{})

	// tools/list
	var toolsResp *proto.Message
	rep.ToolsList = timed(func() error {
		var err error
		toolsResp, err = rpc(ctx, w, r, 2, "tools/list", struct{}{})
		return err
	})
	if !rep.ToolsList.OK {
		rep.OverallFail = "tools/list failed: " + rep.ToolsList.Err
		return rep, nil
	}
	var lr proto.ListToolsResult
	_ = json.Unmarshal(toolsResp.Result, &lr)
	for _, t := range lr.Tools {
		id, _, _ := strings.Cut(t.Name, "__")
		rep.Tools[id] = append(rep.Tools[id], t.Name)
		rep.ToolsTotal++
	}

	// smokes
	for i, sm := range smokes {
		run := SmokeRun{MCPID: sm.MCPID, Tool: sm.Tool}
		toolName := sm.MCPID + "__" + sm.Tool
		t0 := time.Now()
		resp, err := rpc(ctx, w, r, 100+i, "tools/call", proto.CallToolParams{
			Name:      toolName,
			Arguments: sm.Args,
		})
		run.Duration = time.Since(t0)
		switch {
		case err != nil:
			run.Err = err.Error()
		case len(sm.Allow) > 0 && !containsAny(string(resp.Result), sm.Allow):
			run.Err = "result did not match allowOK substrings"
		default:
			run.OK = true
		}
		rep.SmokeRuns = append(rep.SmokeRuns, run)
	}
	return rep, nil
}

// rpc sends one request and waits for the matching response. Other inbound
// messages (notifications, mismatched ids) are ignored — the central router
// does not currently send unsolicited messages, so this is safe in MVP.
func rpc(ctx context.Context, w *framing.Writer, r *framing.Reader, id int, method string, params any) (*proto.Message, error) {
	rpcID := proto.NewStringID(fmt.Sprintf("e2e-%d", id))
	rawParams, _ := json.Marshal(params)
	if err := w.Write(&proto.Message{
		JSONRPC: proto.Version,
		ID:      &rpcID,
		Method:  method,
		Params:  rawParams,
	}); err != nil {
		return nil, err
	}
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		raw, err := r.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil, errors.New("centralmcpd closed stdout unexpectedly")
			}
			return nil, err
		}
		var msg proto.Message
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}
		if !msg.IsResponse() {
			continue
		}
		if msg.ID == nil || msg.ID.String() != rpcID.String() {
			continue
		}
		if msg.Error != nil {
			return &msg, fmt.Errorf("rpc error: %s", msg.Error.Message)
		}
		return &msg, nil
	}
}

func sendNotify(w *framing.Writer, method string, params any) error {
	rawParams, _ := json.Marshal(params)
	return w.Write(&proto.Message{JSONRPC: proto.Version, Method: method, Params: rawParams})
}

func timed(fn func() error) Latency {
	t0 := time.Now()
	err := fn()
	l := Latency{Duration: time.Since(t0)}
	if err != nil {
		l.Err = err.Error()
	} else {
		l.OK = true
	}
	return l
}

func containsAny(s string, needles []string) bool {
	for _, n := range needles {
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
}

func loadSmokes(path string) ([]Smoke, error) {
	if path == "" {
		return nil, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	// Tolerate UTF-8 BOM. PowerShell's Set-Content -Encoding UTF8 emits one
	// by default and many editors do too.
	b = bytes.TrimPrefix(b, []byte{0xEF, 0xBB, 0xBF})
	var out []Smoke
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func findDefaultBin() string {
	for _, p := range []string{"bin/centralmcpd.exe", "bin/centralmcpd"} {
		if _, err := os.Stat(p); err == nil {
			abs, _ := filepath.Abs(p)
			return abs
		}
	}
	if p, err := exec.LookPath("centralmcpd"); err == nil {
		return p
	}
	return ""
}

func die(code int, msg string) {
	fmt.Fprintln(os.Stderr, "e2e:", msg)
	os.Exit(code)
}

// Markdown renders the report as a human-readable markdown document.
func (r *Report) Markdown() string {
	var b strings.Builder
	fmt.Fprintf(&b, "# OneMCP E2E Report\n\n")
	fmt.Fprintf(&b, "_started %s_\n\n", r.Started.Format(time.RFC3339))

	fmt.Fprintf(&b, "## Handshake\n\n")
	fmt.Fprintf(&b, "| Step | OK | Latency | Error |\n|---|---|---|---|\n")
	fmt.Fprintf(&b, "| initialize | %v | %s | %s |\n", r.Init.OK, r.Init.Duration.Round(time.Millisecond), r.Init.Err)
	fmt.Fprintf(&b, "| tools/list | %v | %s | %s |\n\n", r.ToolsList.OK, r.ToolsList.Duration.Round(time.Millisecond), r.ToolsList.Err)

	if r.OverallFail != "" {
		fmt.Fprintf(&b, "**FAILED:** %s\n", r.OverallFail)
		return b.String()
	}

	fmt.Fprintf(&b, "## MCPs (%d tools across %d MCPs)\n\n", r.ToolsTotal, len(r.Tools))
	fmt.Fprintf(&b, "| MCP | # tools | sample tool |\n|---|---|---|\n")
	for id, tools := range r.Tools {
		sample := ""
		if len(tools) > 0 {
			sample = tools[0]
		}
		fmt.Fprintf(&b, "| %s | %d | `%s` |\n", id, len(tools), sample)
	}
	fmt.Fprintln(&b)

	if len(r.SmokeRuns) > 0 {
		fmt.Fprintf(&b, "## Smoke calls\n\n")
		fmt.Fprintf(&b, "| MCP | Tool | OK | Latency | Error |\n|---|---|---|---|---|\n")
		for _, s := range r.SmokeRuns {
			fmt.Fprintf(&b, "| %s | %s | %v | %s | %s |\n", s.MCPID, s.Tool, s.OK, s.Duration.Round(time.Millisecond), s.Err)
		}
	}
	return b.String()
}

// HasFailures reports whether any step or smoke call failed.
func (r *Report) HasFailures() bool {
	if r.OverallFail != "" || !r.Init.OK || !r.ToolsList.OK {
		return true
	}
	for _, s := range r.SmokeRuns {
		if !s.OK {
			return true
		}
	}
	return false
}
