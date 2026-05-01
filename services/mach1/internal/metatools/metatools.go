// Package metatools exposes 11 mach1 meta-tools that let an AI client manage
// MCPs directly through the standard MCP tools/call interface.
package metatools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/composition"
	"github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/envdetect"
	"github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/install"
	"github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/manifest"
	"github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/proto"
	"github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/registry"
	"github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/secrets"
	"github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/supervisor"
)

// Handler is one meta-tool implementation.
type Handler interface {
	Schema() proto.Tool
	Handle(ctx context.Context, args json.RawMessage) (json.RawMessage, *proto.RPCError)
}

// Registry holds all meta-tool handlers.
type Registry struct {
	handlers map[string]Handler
}

// New builds the meta-tools registry with the required dependencies.
func New(sup *supervisor.Supervisor, db *registry.DB, sec *secrets.Store, installer *install.Installer, catalog []manifest.Manifest, getManifest func(id string) (*manifest.Manifest, error), logger *slog.Logger) *Registry {
	if logger == nil {
		logger = slog.Default()
	}

	detector := envdetect.New(catalog)

	reg := &Registry{}

	// Caller used by composition engine (routes to sup or meta).
	caller := func(ctx context.Context, toolName string, args json.RawMessage) (json.RawMessage, *proto.RPCError) {
		if strings.HasPrefix(toolName, "mach1_") {
			return reg.Handle(ctx, toolName, args)
		}
		return sup.Call(ctx, toolName, args)
	}
	engine := composition.New(caller, logger)

	metaToolsFn := func() []proto.Tool { return reg.Tools() }

	reg.handlers = map[string]Handler{
		"mach1_list_tools":       &listToolsHandler{sup: sup, metaTools: metaToolsFn},
		"mach1_browse_discover":  &browseDiscoverHandler{catalog: catalog, db: db, sec: sec},
		"mach1_install_mcp":      &installMCPHandler{catalog: catalog, installer: installer, detector: detector},
		"mach1_install_batch":    &installBatchHandler{catalog: catalog, installer: installer, detector: detector},
		"mach1_config_env":       &configEnvHandler{db: db, sec: sec, getManifest: getManifest, sup: sup},
		"mach1_start_mcp":        &startMCPHandler{db: db, sup: sup},
		"mach1_close_mcp":        &closeMCPHandler{db: db, sup: sup},
		"mach1_check_enabled":    &checkEnabledHandler{db: db, sup: sup, catalog: catalog},
		"mach1_semantic_search":  &semanticSearchHandler{sup: sup},
		"mach1_route":            &routeHandler{sup: sup, reg: reg},
		"mach1_compose":          &composeHandler{engine: engine},
	}
	return reg
}

// Tools returns the schema for every meta-tool.
func (r *Registry) Tools() []proto.Tool {
	out := make([]proto.Tool, 0, len(r.handlers))
	for _, h := range r.handlers {
		out = append(out, h.Schema())
	}
	return out
}

// Handle dispatches to the named meta-tool.
func (r *Registry) Handle(ctx context.Context, name string, args json.RawMessage) (json.RawMessage, *proto.RPCError) {
	h, ok := r.handlers[name]
	if !ok {
		return nil, proto.NewError(proto.ErrMethodNotFound, "unknown meta-tool: "+name, nil)
	}
	return h.Handle(ctx, args)
}

// ----------------------------------------------------------------------
// 1. mach1_list_tools
// ----------------------------------------------------------------------

type listToolsHandler struct {
	sup       *supervisor.Supervisor
	metaTools func() []proto.Tool
}

func (h *listToolsHandler) Schema() proto.Tool {
	return proto.Tool{
		Name:        "mach1_list_tools",
		Description: "List all tools from all enabled MCPs plus meta-tools.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"mcp":{"type":"string","description":"Optional filter by MCP id"}}}`),
	}
}

func (h *listToolsHandler) Handle(ctx context.Context, args json.RawMessage) (json.RawMessage, *proto.RPCError) {
	var p struct {
		MCP string `json:"mcp"`
	}
	_ = json.Unmarshal(args, &p)

	tools := h.sup.Tools()
	tools = append(tools, h.metaTools()...)

	type info struct {
		Name        string                  `json:"name"`
		Description string                  `json:"description"`
		MCP         string                  `json:"mcp"`
		Annotations *proto.ToolAnnotations `json:"annotations,omitempty"`
	}
	var out []info
	for _, t := range tools {
		if p.MCP != "" {
			prefix, _, _ := strings.Cut(t.Name, supervisor.NamespaceSep)
			if prefix != p.MCP && t.Name != p.MCP {
				continue
			}
		}
		mcp := ""
		if strings.HasPrefix(t.Name, "mach1_") {
			mcp = "mach1"
		} else {
			mcp, _, _ = strings.Cut(t.Name, supervisor.NamespaceSep)
		}
		out = append(out, info{
			Name:        t.Name,
			Description: t.Description,
			MCP:         mcp,
			Annotations: t.Annotations,
		})
	}
	b, _ := json.Marshal(map[string]any{"tools": out})
	return b, nil
}

// ----------------------------------------------------------------------
// 2. mach1_browse_discover
// ----------------------------------------------------------------------

type browseDiscoverHandler struct {
	catalog []manifest.Manifest
	db      *registry.DB
	sec     *secrets.Store
}

func (h *browseDiscoverHandler) Schema() proto.Tool {
	return proto.Tool{
		Name:        "mach1_browse_discover",
		Description: "Search the marketplace catalog.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"},"category":{"type":"string"},"runtime":{"type":"string"}}}`),
	}
}

func (h *browseDiscoverHandler) Handle(ctx context.Context, args json.RawMessage) (json.RawMessage, *proto.RPCError) {
	var p struct {
		Query    string `json:"query"`
		Category string `json:"category"`
		Runtime  string `json:"runtime"`
	}
	_ = json.Unmarshal(args, &p)

	type entryOut struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Installed   bool   `json:"installed"`
		EnvComplete bool   `json:"env_complete"`
		Category    string `json:"category"`
	}
	var out []entryOut
	for i := range h.catalog {
		m := &h.catalog[i]
		if p.Query != "" && !strings.Contains(strings.ToLower(m.Name+" "+m.Description), strings.ToLower(p.Query)) {
			continue
		}
		if p.Runtime != "" && m.Runtime != p.Runtime {
			continue
		}
		// Category is best-effort from tags.
		cat := ""
		if len(m.Tags) > 0 {
			cat = m.Tags[0]
		}
		if p.Category != "" && !strings.EqualFold(cat, p.Category) {
			continue
		}
		installed := false
		envComplete := false
		if h.db != nil {
			if e, _, err := h.db.Get(ctx, m.ID); err == nil {
				installed = true
				envComplete = true
				for _, ev := range m.EnvSchema {
					if ev.Required {
						if _, ok := e.Env[ev.Name]; !ok {
							if h.sec != nil {
								if _, ok2 := h.sec.Get(m.ID)[ev.Name]; !ok2 {
									envComplete = false
									break
								}
							} else {
								envComplete = false
								break
							}
						}
					}
				}
			}
		}
		out = append(out, entryOut{
			ID:          m.ID,
			Name:        m.Name,
			Description: m.Description,
			Installed:   installed,
			EnvComplete: envComplete,
			Category:    cat,
		})
	}
	b, _ := json.Marshal(map[string]any{"entries": out})
	return b, nil
}

// ----------------------------------------------------------------------
// 3. mach1_install_mcp
// ----------------------------------------------------------------------

type installMCPHandler struct {
	catalog   []manifest.Manifest
	installer *install.Installer
	detector  *envdetect.Detector
}

func (h *installMCPHandler) Schema() proto.Tool {
	return proto.Tool{
		Name:        "mach1_install_mcp",
		Description: "Install one MCP from the catalog.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"id":{"type":"string"}},"required":["id"]}`),
	}
}

func (h *installMCPHandler) Handle(ctx context.Context, args json.RawMessage) (json.RawMessage, *proto.RPCError) {
	var p struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return nil, proto.NewError(proto.ErrInvalidParams, "invalid params", err.Error())
	}
	var m *manifest.Manifest
	for i := range h.catalog {
		if h.catalog[i].ID == p.ID {
			m = &h.catalog[i]
			break
		}
	}
	if m == nil {
		return nil, proto.NewError(proto.ErrInvalidParams, "unknown catalog id", p.ID)
	}
	if h.installer == nil {
		return nil, proto.NewError(proto.ErrInternal, "installer not available", nil)
	}
	res, err := h.installer.Install(ctx, m)
	if err != nil {
		return nil, proto.NewError(proto.ErrInternal, "install failed: "+err.Error(), nil)
	}
	// Run env detection.
	envStatus := map[string]any{"configured": []string{}, "missing": []string{}}
	if h.detector != nil {
		report := h.detector.Report(m.ID, os.Environ())
		configured := make([]string, 0, len(report.Configured))
		for _, c := range report.Configured {
			configured = append(configured, c.Var)
		}
		missing := make([]string, 0, len(report.Missing))
		for _, mi := range report.Missing {
			missing = append(missing, mi.Var)
		}
		envStatus["configured"] = configured
		envStatus["missing"] = missing
	}
	b, _ := json.Marshal(map[string]any{
		"id":         res.ID,
		"installed":  true,
		"reinstalled": res.Already,
		"env_status": envStatus,
	})
	return b, nil
}

// ----------------------------------------------------------------------
// 4. mach1_install_batch
// ----------------------------------------------------------------------

type installBatchHandler struct {
	catalog   []manifest.Manifest
	installer *install.Installer
	detector  *envdetect.Detector
}

func (h *installBatchHandler) Schema() proto.Tool {
	return proto.Tool{
		Name:        "mach1_install_batch",
		Description: "Install multiple MCPs from the catalog.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"ids":{"type":"array","items":{"type":"string"}}},"required":["ids"]}`),
	}
}

func (h *installBatchHandler) Handle(ctx context.Context, args json.RawMessage) (json.RawMessage, *proto.RPCError) {
	var p struct {
		IDs []string `json:"ids"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return nil, proto.NewError(proto.ErrInvalidParams, "invalid params", err.Error())
	}
	var results []map[string]any
	for _, id := range p.IDs {
		single, rpcErr := (&installMCPHandler{catalog: h.catalog, installer: h.installer, detector: h.detector}).Handle(ctx, json.RawMessage(`{"id":"`+id+`"}`))
		if rpcErr != nil {
			results = append(results, map[string]any{
				"id":        id,
				"installed": false,
				"error":     rpcErr.Message,
			})
			continue
		}
		var singleMap map[string]any
		_ = json.Unmarshal(single, &singleMap)
		results = append(results, singleMap)
	}
	b, _ := json.Marshal(map[string]any{"results": results})
	return b, nil
}

// ----------------------------------------------------------------------
// 5. mach1_config_env
// ----------------------------------------------------------------------

type configEnvHandler struct {
	db          *registry.DB
	sec         *secrets.Store
	getManifest func(id string) (*manifest.Manifest, error)
	sup         *supervisor.Supervisor
}

func (h *configEnvHandler) Schema() proto.Tool {
	return proto.Tool{
		Name:        "mach1_config_env",
		Description: "Set env vars for an MCP. Aliases are resolved to canonical names.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"mcp_id":{"type":"string"},"vars":{"type":"object"}},"required":["mcp_id","vars"]}`),
	}
}

func (h *configEnvHandler) Handle(ctx context.Context, args json.RawMessage) (json.RawMessage, *proto.RPCError) {
	var p struct {
		McpID string            `json:"mcp_id"`
		Vars  map[string]string `json:"vars"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return nil, proto.NewError(proto.ErrInvalidParams, "invalid params", err.Error())
	}
	m, err := h.getManifest(p.McpID)
	if err != nil {
		return nil, proto.NewError(proto.ErrInvalidParams, "cannot load manifest", err.Error())
	}
	aliases := m.ResolveEnvKeys()
	resolved := map[string]string{}
	var set []string
	var errs []string
	for k, v := range p.Vars {
		canonical := k
		if c, ok := aliases[k]; ok {
			canonical = c
		}
		resolved[canonical] = v
		set = append(set, canonical)
	}
	// Split secrets from plain env.
	isSecret := map[string]bool{}
	for _, ev := range m.EnvSchema {
		if ev.Secret {
			isSecret[ev.Name] = true
		}
	}
	plainEnv := map[string]string{}
	for k, v := range resolved {
		if isSecret[k] {
			if h.sec != nil {
				if err := h.sec.Set(p.McpID, k, v); err != nil {
					errs = append(errs, fmt.Sprintf("secret %s: %v", k, err))
				}
			} else {
				errs = append(errs, fmt.Sprintf("secret store not available for %s", k))
			}
		} else {
			plainEnv[k] = v
		}
	}
	if h.db != nil && len(plainEnv) > 0 {
		// Merge with existing env.
		e, _, err := h.db.Get(ctx, p.McpID)
		if err == nil {
			for k, v := range e.Env {
				if _, ok := plainEnv[k]; !ok {
					plainEnv[k] = v
				}
			}
		}
		if err := h.db.SetEnv(ctx, p.McpID, plainEnv); err != nil {
			errs = append(errs, fmt.Sprintf("set env: %v", err))
		}
	}
	if h.sup != nil && len(errs) == 0 {
		if err := h.sup.SetEnv(ctx, p.McpID, plainEnv); err != nil {
			errs = append(errs, fmt.Sprintf("restart: %v", err))
		}
	}
	b, _ := json.Marshal(map[string]any{"set": set, "errors": errs})
	return b, nil
}

// ----------------------------------------------------------------------
// 6. mach1_start_mcp
// ----------------------------------------------------------------------

type startMCPHandler struct {
	db *registry.DB
	sup *supervisor.Supervisor
}

func (h *startMCPHandler) Schema() proto.Tool {
	return proto.Tool{
		Name:        "mach1_start_mcp",
		Description: "Enable and warmup an MCP.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"id":{"type":"string"}},"required":["id"]}`),
	}
}

func (h *startMCPHandler) Handle(ctx context.Context, args json.RawMessage) (json.RawMessage, *proto.RPCError) {
	var p struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return nil, proto.NewError(proto.ErrInvalidParams, "invalid params", err.Error())
	}
	if h.db != nil {
		if err := h.db.SetEnabled(ctx, p.ID, true); err != nil {
			return nil, proto.NewError(proto.ErrInternal, "enable failed: "+err.Error(), nil)
		}
	}
	if h.sup != nil {
		if err := h.sup.Enable(ctx, p.ID); err != nil {
			return nil, proto.NewError(proto.ErrInternal, "start failed: "+err.Error(), nil)
		}
	}
	b, _ := json.Marshal(map[string]any{"id": p.ID, "status": "started"})
	return b, nil
}

// ----------------------------------------------------------------------
// 7. mach1_close_mcp
// ----------------------------------------------------------------------

type closeMCPHandler struct {
	db  *registry.DB
	sup *supervisor.Supervisor
}

func (h *closeMCPHandler) Schema() proto.Tool {
	return proto.Tool{
		Name:        "mach1_close_mcp",
		Description: "Disable an MCP and shut down its process.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"id":{"type":"string"}},"required":["id"]}`),
	}
}

func (h *closeMCPHandler) Handle(ctx context.Context, args json.RawMessage) (json.RawMessage, *proto.RPCError) {
	var p struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return nil, proto.NewError(proto.ErrInvalidParams, "invalid params", err.Error())
	}
	if h.db != nil {
		if err := h.db.SetEnabled(ctx, p.ID, false); err != nil {
			return nil, proto.NewError(proto.ErrInternal, "disable failed: "+err.Error(), nil)
		}
	}
	if h.sup != nil {
		if err := h.sup.Disable(ctx, p.ID); err != nil {
			return nil, proto.NewError(proto.ErrInternal, "close failed: "+err.Error(), nil)
		}
	}
	b, _ := json.Marshal(map[string]any{"id": p.ID, "status": "closed"})
	return b, nil
}

// ----------------------------------------------------------------------
// 8. mach1_check_enabled
// ----------------------------------------------------------------------

type checkEnabledHandler struct {
	db      *registry.DB
	sup     *supervisor.Supervisor
	catalog []manifest.Manifest
}

func (h *checkEnabledHandler) Schema() proto.Tool {
	return proto.Tool{
		Name:        "mach1_check_enabled",
		Description: "List all MCPs with their installation and runtime status.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
	}
}

func (h *checkEnabledHandler) Handle(ctx context.Context, args json.RawMessage) (json.RawMessage, *proto.RPCError) {
	type mcpOut struct {
		ID           string `json:"id"`
		Name         string `json:"name"`
		Installed    bool   `json:"installed"`
		Enabled      bool   `json:"enabled"`
		HealthStatus string `json:"health_status"`
		EnvComplete  bool   `json:"env_complete"`
		ToolCount    int    `json:"tool_count"`
	}
	out := []mcpOut{}
	seen := map[string]bool{}

	if h.db != nil {
		entries, err := h.db.ListAll(ctx)
		if err == nil {
			for _, e := range entries {
				seen[e.ID] = true
				st := mcpOut{
					ID:        e.ID,
					Name:      e.Name,
					Installed: true,
					Enabled:   e.Enabled,
				}
				if h.sup != nil {
					if status, err := h.sup.GetMCPStatus(e.ID); err == nil {
						st.Enabled = status.Enabled
						st.EnvComplete = status.EnvComplete
						st.ToolCount = status.ToolCount
						if status.Healthy {
							st.HealthStatus = "healthy"
						} else if status.Running {
							st.HealthStatus = "degraded"
						} else {
							st.HealthStatus = "stopped"
						}
					}
				}
				out = append(out, st)
			}
		}
	}
	for i := range h.catalog {
		m := &h.catalog[i]
		if seen[m.ID] {
			continue
		}
		out = append(out, mcpOut{
			ID:           m.ID,
			Name:         m.Name,
			Installed:    false,
			Enabled:      false,
			HealthStatus: "not_installed",
			EnvComplete:  false,
			ToolCount:    0,
		})
	}
	b, _ := json.Marshal(map[string]any{"mcps": out})
	return b, nil
}

// ----------------------------------------------------------------------
// 9. mach1_semantic_search
// ----------------------------------------------------------------------

type semanticSearchHandler struct {
	sup *supervisor.Supervisor
}

func (h *semanticSearchHandler) Schema() proto.Tool {
	return proto.Tool{
		Name:        "mach1_semantic_search",
		Description: "Find the right tool by natural-language description.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"},"k":{"type":"number"}},"required":["query"]}`),
	}
}

func (h *semanticSearchHandler) Handle(ctx context.Context, args json.RawMessage) (json.RawMessage, *proto.RPCError) {
	var p struct {
		Query string `json:"query"`
		K     int    `json:"k"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return nil, proto.NewError(proto.ErrInvalidParams, "invalid params", err.Error())
	}
	if p.K <= 0 {
		p.K = 5
	}
	results := h.sup.RankToolsWithScores(p.Query, p.K)
	type item struct {
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Score       float64 `json:"score"`
	}
	var out []item
	for _, r := range results {
		out = append(out, item{
			Name:        r.Tool.Name,
			Description: r.Tool.Description,
			Score:       r.Score,
		})
	}
	b, _ := json.Marshal(map[string]any{"tools": out})
	return b, nil
}

// ----------------------------------------------------------------------
// 10. mach1_route
// ----------------------------------------------------------------------

type routeHandler struct {
	sup *supervisor.Supervisor
	reg *Registry
}

func (h *routeHandler) Schema() proto.Tool {
	return proto.Tool{
		Name:        "mach1_route",
		Description: "Search and execute the best matching tool for a query.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"},"args":{"type":"object"}},"required":["query"]}`),
	}
}

func (h *routeHandler) Handle(ctx context.Context, args json.RawMessage) (json.RawMessage, *proto.RPCError) {
	var p struct {
		Query string          `json:"query"`
		Args  json.RawMessage `json:"args"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return nil, proto.NewError(proto.ErrInvalidParams, "invalid params", err.Error())
	}
	results := h.sup.RankToolsWithScores(p.Query, 1)
	if len(results) == 0 {
		return nil, proto.NewError(proto.ErrMethodNotFound, "no matching tool found", nil)
	}
	toolName := results[0].Tool.Name
	if p.Args == nil {
		p.Args = json.RawMessage(`{}`)
	}
	var result json.RawMessage
	var rpcErr *proto.RPCError
	if strings.HasPrefix(toolName, "mach1_") {
		result, rpcErr = h.reg.Handle(ctx, toolName, p.Args)
	} else {
		result, rpcErr = h.sup.Call(ctx, toolName, p.Args)
	}
	if rpcErr != nil {
		return nil, rpcErr
	}
	b, _ := json.Marshal(map[string]any{"tool": toolName, "result": json.RawMessage(result)})
	return b, nil
}

// ----------------------------------------------------------------------
// 11. mach1_compose
// ----------------------------------------------------------------------

type composeHandler struct {
	engine *composition.Engine
}

func (h *composeHandler) Schema() proto.Tool {
	return proto.Tool{
		Name:        "mach1_compose",
		Description: "Execute a sequence of tool calls, stopping on first error.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"steps":{"type":"array","items":{"type":"object","properties":{"tool":{"type":"string"},"args":{"type":"object"}}}}},"required":["steps"]}`),
	}
}

func (h *composeHandler) Handle(ctx context.Context, args json.RawMessage) (json.RawMessage, *proto.RPCError) {
	var p struct {
		Steps []struct {
			Tool string          `json:"tool"`
			Args json.RawMessage `json:"args"`
		} `json:"steps"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return nil, proto.NewError(proto.ErrInvalidParams, "invalid params", err.Error())
	}
	steps := make([]composition.Step, 0, len(p.Steps))
	for _, s := range p.Steps {
		if s.Args == nil {
			s.Args = json.RawMessage(`{}`)
		}
		steps = append(steps, composition.Step{Tool: s.Tool, Args: s.Args})
	}
	res, err := h.engine.Run(ctx, steps)
	if err != nil {
		return nil, proto.NewError(proto.ErrInternal, "compose failed: "+err.Error(), nil)
	}
	b, _ := json.Marshal(res)
	return b, nil
}
