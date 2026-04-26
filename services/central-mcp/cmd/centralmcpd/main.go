// centralmcpd is the OneMCP central router. It speaks MCP over stdio to a
// single client (e.g. VS Code) and delegates to the supervisor, which owns
// the lifecycle of every installed-and-enabled child MCP (lazy start, idle
// shutdown, sandbox driver selection).
//
// stderr is for logs. stdout is reserved for the MCP wire protocol.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/onemcp/central-mcp/internal/manifest"
	"github.com/onemcp/central-mcp/internal/paths"
	"github.com/onemcp/central-mcp/internal/registry"
	"github.com/onemcp/central-mcp/internal/router"
	"github.com/onemcp/central-mcp/internal/secrets"
	"github.com/onemcp/central-mcp/internal/supervisor"
)

// fileConfig is the dev-mode JSON config: a flat list of inline manifests
// useful for tests. Production loads from SQLite via the hub/CLI.
type fileConfig struct {
	MCPs []manifest.Manifest `json:"mcps"`
}

func main() {
	var (
		configPath = flag.String("config", "", "path to dev JSON config (overrides --db when set)")
		dbPath     = flag.String("db", "", "path to registry SQLite db (default: paths.RegistryDB)")
		logLevel   = flag.String("log", "info", "log level: debug|info|warn|error")
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

	sup, err := buildSupervisor(ctx, *configPath, *dbPath, logger)
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

	srv := router.New(os.Stdin, os.Stdout, sup, logger)
	if err := srv.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("router exited", "err", err)
		os.Exit(1)
	}
}

func buildSupervisor(ctx context.Context, configPath, dbFlag string, logger *slog.Logger) (*supervisor.Supervisor, error) {
	if configPath != "" {
		return buildFromConfig(configPath, logger)
	}
	dbPath := dbFlag
	if dbPath == "" {
		p, err := paths.RegistryDB()
		if err != nil {
			return nil, err
		}
		dbPath = p
	}
	return buildFromDB(ctx, dbPath, logger)
}

func buildFromConfig(path string, logger *slog.Logger) (*supervisor.Supervisor, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg fileConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	manifests := map[string]*manifest.Manifest{}
	entries := make([]registry.Entry, 0, len(cfg.MCPs))
	for i := range cfg.MCPs {
		m := &cfg.MCPs[i]
		if err := m.Validate(); err != nil {
			return nil, fmt.Errorf("config[%s]: %w", m.ID, err)
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
	return supervisor.New(entries, func(id string) (*manifest.Manifest, error) {
		if m, ok := manifests[id]; ok {
			return m, nil
		}
		return nil, fmt.Errorf("manifest for %s not found", id)
	}, nil, supervisor.Options{Logger: logger})
}

func buildFromDB(ctx context.Context, dbPath string, logger *slog.Logger) (*supervisor.Supervisor, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("mkdir registry dir: %w", err)
	}
	db, err := registry.Open(dbPath)
	if err != nil {
		return nil, err
	}
	// DB stays open for the process lifetime; supervisor copies what it needs
	// at construction so we don't hold the handle in the hot path.
	entries, err := db.ListEnabled(ctx)
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	secPath, err := paths.SecretsFile()
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	sec, err := secrets.Open(secPath)
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	getManifest := func(id string) (*manifest.Manifest, error) {
		_, manifestJSON, err := db.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		return manifest.Parse(manifestJSON)
	}
	return supervisor.New(entries, getManifest, sec, supervisor.Options{Logger: logger})
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
