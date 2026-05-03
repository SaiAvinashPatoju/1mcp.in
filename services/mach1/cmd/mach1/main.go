// mach1 is the 1mcp.in central router. It speaks MCP over stdio to a
// single client (e.g. VS Code) and delegates to the supervisor, which owns
// the lifecycle of every installed-and-enabled child MCP (lazy start, idle
// shutdown, sandbox driver selection).
//
// stderr is for logs. stdout is reserved for the MCP wire protocol.
package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/catalog"
	"github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/install"
	"github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/manifest"
	"github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/metatools"
	"github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/observability"
	"github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/paths"
	"github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/registry"
	"github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/router"
	"github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/secrets"
	"github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/supervisor"
	transporthttp "github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/transport"
)

//go:embed catalog.json
var embeddedCatalogJSON []byte

// fileConfig is the dev-mode JSON config: a flat list of inline manifests
// useful for tests. Production loads from SQLite via the hub/CLI.
type fileConfig struct {
	MCPs []manifest.Manifest `json:"mcps"`
}

func main() {
	var (
		configPath  = flag.String("config", "", "path to dev JSON config (overrides --db when set)")
		dbPath      = flag.String("db", "", "path to registry SQLite db (default: paths.RegistryDB)")
		logLevel    = flag.String("log", "info", "log level: debug|info|warn|error")
		transport   = flag.String("transport", "stdio", "client transport: stdio|http")
		listenAddr  = flag.String("listen", "127.0.0.1:3000", "HTTP transport listen address")
		metricsAddr = flag.String("metrics-addr", "127.0.0.1:3031", "Prometheus metrics listen address; empty disables standalone metrics server")
		httpToken   = flag.String("http-token", os.Getenv("MACH1_HTTP_TOKEN"), "bearer token required for HTTP transport; defaults to MACH1_HTTP_TOKEN")
		catalogPath = flag.String("catalog", os.Getenv("MACH1_CATALOG"), "path to marketplace catalog JSON")
	)
	flag.Parse()

	logger := newLogger(*logLevel)
	slog.SetDefault(logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		logger.Info("signal received; shutting down")
		cancel()
	}()

	metrics := observability.NewMetrics()
	sup, db, sec, getManifest, err := buildSupervisor(ctx, *configPath, *dbPath, logger, metrics)
	if err != nil {
		logger.Error("build supervisor", "err", err)
		os.Exit(1)
	}
	defer sup.Close()

	// Warmup happens in parallel; bound the total wall time to keep startup
	// snappy even if one MCP is misbehaving.
	warmupCtx, warmupCancel := context.WithTimeout(ctx, 30*time.Second)
	sup.Start(warmupCtx)
	warmupCancel()

	// Load marketplace catalog.
	var catalogEntries []manifest.Manifest
	if *catalogPath != "" {
		if c, err := catalog.Load(*catalogPath); err == nil {
			catalogEntries = c
		} else {
			logger.Warn("catalog load failed", "path", *catalogPath, "err", err)
		}
	}
	if len(catalogEntries) == 0 {
		for _, p := range []string{"packages/registry-index/index.json", "../../packages/registry-index/index.json"} {
			if c, err := catalog.Load(p); err == nil {
				catalogEntries = c
				break
			}
		}
	}
	if len(catalogEntries) == 0 && len(embeddedCatalogJSON) > 2 {
		if c, err := catalog.LoadBytes(embeddedCatalogJSON); err == nil {
			catalogEntries = c
			logger.Info("loaded catalog from embedded data", "entries", len(catalogEntries))
		} else {
			logger.Warn("embedded catalog parse failed", "err", err)
		}
	}

	installer := &install.Installer{DB: db, Logger: logger}
	meta := metatools.New(sup, db, sec, installer, catalogEntries, getManifest, logger)

	if *metricsAddr != "" && *transport != "http" {
		go serveStandaloneMetrics(ctx, *metricsAddr, metrics, logger)
	}

	srv := router.New(os.Stdin, os.Stdout, sup, meta, logger)
	var runErr error
	switch *transport {
	case "stdio":
		runErr = srv.Run(ctx)
	case "http":
		runErr = transporthttp.ServeHTTP(ctx, srv, transporthttp.HTTPOptions{Addr: *listenAddr, AuthToken: *httpToken, Logger: logger, Metrics: metrics})
	default:
		logger.Error("invalid transport", "transport", *transport)
		os.Exit(2)
	}
	if runErr != nil && !errors.Is(runErr, context.Canceled) {
		logger.Error("router exited", "err", runErr)
		os.Exit(1)
	}
}

func buildSupervisor(ctx context.Context, configPath, dbFlag string, logger *slog.Logger, metrics *observability.Metrics) (*supervisor.Supervisor, *registry.DB, *secrets.Store, func(id string) (*manifest.Manifest, error), error) {
	if configPath != "" {
		return buildFromConfig(configPath, logger, metrics)
	}
	dbPath := dbFlag
	if dbPath == "" {
		p, err := paths.RegistryDB()
		if err != nil {
			return nil, nil, nil, nil, err
		}
		dbPath = p
	}
	return buildFromDB(ctx, dbPath, logger, metrics)
}

func buildFromConfig(path string, logger *slog.Logger, metrics *observability.Metrics) (*supervisor.Supervisor, *registry.DB, *secrets.Store, func(id string) (*manifest.Manifest, error), error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("read config: %w", err)
	}
	var cfg fileConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, nil, nil, nil, fmt.Errorf("parse config: %w", err)
	}
	manifests := map[string]*manifest.Manifest{}
	entries := make([]registry.Entry, 0, len(cfg.MCPs))
	for i := range cfg.MCPs {
		m := &cfg.MCPs[i]
		if err := m.Validate(); err != nil {
			return nil, nil, nil, nil, fmt.Errorf("config[%s]: %w", m.ID, err)
		}
		manifests[m.ID] = m
		entries = append(entries, registry.Entry{
			ID:      m.ID,
			Name:    m.Name,
			Version: m.Version,
			Enabled: true,
			Runtime: m.Runtime,
			Command: pickCommand(m),
			Args:    m.Entrypoint.Args,
			Cwd:     m.Entrypoint.Cwd,
		})
	}
	getManifest := func(id string) (*manifest.Manifest, error) {
		if m, ok := manifests[id]; ok {
			return m, nil
		}
		return nil, fmt.Errorf("manifest for %s not found", id)
	}
	sup, err := supervisor.New(entries, getManifest, nil, supervisor.Options{Logger: logger, Metrics: metrics})
	return sup, nil, nil, getManifest, err
}

func buildFromDB(ctx context.Context, dbPath string, logger *slog.Logger, metrics *observability.Metrics) (*supervisor.Supervisor, *registry.DB, *secrets.Store, func(id string) (*manifest.Manifest, error), error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, nil, nil, nil, fmt.Errorf("mkdir registry dir: %w", err)
	}
	db, err := registry.Open(dbPath)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	// DB stays open for the process lifetime; supervisor copies what it needs
	// at construction so we don't hold the handle in the hot path.
	entries, err := db.ListEnabled(ctx)
	if err != nil {
		_ = db.Close()
		return nil, nil, nil, nil, err
	}

	secPath, err := paths.SecretsFile()
	if err != nil {
		_ = db.Close()
		return nil, nil, nil, nil, err
	}
	sec, err := secrets.Open(secPath)
	if err != nil {
		_ = db.Close()
		return nil, nil, nil, nil, err
	}

	getManifest := func(id string) (*manifest.Manifest, error) {
		_, manifestJSON, err := db.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		return manifest.Parse(manifestJSON)
	}
	sup, err := supervisor.New(entries, getManifest, sec, supervisor.Options{Logger: logger, Registry: db, Metrics: metrics})
	return sup, db, sec, getManifest, err
}

func serveStandaloneMetrics(ctx context.Context, addr string, metrics *observability.Metrics, logger *slog.Logger) {
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", metrics.Handler())
	// Try the requested address first; if it's in use, fall back to a
	// kernel-assigned port so we don't block startup.
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Warn("metrics address in use; falling back to random port", "requested", addr, "error", err)
		listener, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			logger.Warn("metrics endpoint stopped", "err", err)
			return
		}
	}
	logger.Info("metrics endpoint ready", "addr", listener.Addr().String(), "path", "/metrics")
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()
	if err := srv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Warn("metrics endpoint stopped", "err", err)
	}
}

func pickCommand(m *manifest.Manifest) string {
	if m.Runtime == "docker" {
		return m.Entrypoint.Image
	}
	return m.Entrypoint.Command
}

func newLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: lvl}))
}
