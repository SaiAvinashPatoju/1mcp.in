package transport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/onemcp/central-mcp/internal/framing"
	"github.com/onemcp/central-mcp/internal/observability"
	"github.com/onemcp/central-mcp/internal/proto"
	"github.com/onemcp/central-mcp/internal/router"
)

type HTTPOptions struct {
	Addr      string
	AuthToken string
	Logger    *slog.Logger
	Metrics   *observability.Metrics
}

func ServeHTTP(ctx context.Context, r *router.Server, opts HTTPOptions) error {
	if opts.Addr == "" {
		opts.Addr = "127.0.0.1:3000"
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	if opts.Metrics != nil {
		mux.Handle("GET /metrics", opts.Metrics.Handler())
	}
	mux.HandleFunc("POST /mcp", func(w http.ResponseWriter, req *http.Request) {
		if opts.AuthToken != "" && strings.TrimPrefix(req.Header.Get("Authorization"), "Bearer ") != opts.AuthToken {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing or invalid bearer token"})
			return
		}
		if req.Header.Get("Mcp-Protocol-Version") == "" {
			w.Header().Set("Mcp-Protocol-Version", proto.ProtocolVersion)
		}
		if req.ContentLength > framing.MaxMessageBytes {
			writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "message too large"})
			return
		}
		defer req.Body.Close()
		var msg proto.Message
		dec := json.NewDecoder(http.MaxBytesReader(w, req.Body, framing.MaxMessageBytes))
		if err := dec.Decode(&msg); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json-rpc message"})
			return
		}
		resp := r.Handle(req.Context(), &msg)
		if resp == nil {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		writeJSON(w, http.StatusOK, resp)
	})

	srv := &http.Server{
		Addr:              opts.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()
	opts.Logger.Info("streamable HTTP transport ready", "addr", opts.Addr, "endpoint", "/mcp")
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("http transport: %w", err)
	}
	return nil
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
