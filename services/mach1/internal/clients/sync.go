// Package clients provides continuous sync daemon for maintaining 1mcp configuration.
//
// The SyncDaemon monitors configured AI clients and automatically re-injects
// mach1 configuration if it has been removed or modified by the client or user.
package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"
)

// SyncConfig holds the configuration for the sync daemon.
type SyncConfig struct {
	// Interval between sync checks (default: 30s)
	Interval time.Duration
	// Logger for sync operations (optional)
	Logger *slog.Logger
	// AutoRepair re-injects config if mach1 entry is missing (default: true)
	AutoRepair bool
	// VerifyRules also checks and repairs rule files (default: true)
	VerifyRules bool
}

// DefaultSyncConfig returns sensible defaults.
func DefaultSyncConfig() *SyncConfig {
	return &SyncConfig{
		Interval:    30 * time.Second,
		AutoRepair:  true,
		VerifyRules: true,
	}
}

// SyncDaemon continuously monitors and maintains 1mcp client configurations.
type SyncDaemon struct {
	config    *SyncConfig
	clients   []Kind
	entry     ServerEntry
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	mu        sync.RWMutex
	running   bool
	lastSync  map[Kind]time.Time
	repairLog map[Kind][]SyncEvent
}

// SyncEvent records a sync/repair action.
type SyncEvent struct {
	Time      time.Time
	Client    Kind
	Action    string // "repaired_config", "repaired_rules", "verified"
	Path      string
	Error     error
}

// NewSyncDaemon creates a new sync daemon for the specified clients.
func NewSyncDaemon(clients []Kind, entry ServerEntry, config *SyncConfig) *SyncDaemon {
	if config == nil {
		config = DefaultSyncConfig()
	}
	if config.Interval == 0 {
		config.Interval = 30 * time.Second
	}

	return &SyncDaemon{
		config:    config,
		clients:   clients,
		entry:     entry,
		lastSync:  make(map[Kind]time.Time),
		repairLog: make(map[Kind][]SyncEvent),
	}
}

// Start begins the sync daemon.
// It performs an immediate verification of all clients, then starts polling.
func (d *SyncDaemon) Start(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.running {
		return fmt.Errorf("sync daemon already running")
	}

	d.ctx, d.cancel = context.WithCancel(ctx)
	d.running = true

	// Perform immediate verification
	d.config.Logger.Info("sync daemon starting", "clients", len(d.clients))
	for _, client := range d.clients {
		d.wg.Add(1)
		go d.syncLoop(client)
	}

	return nil
}

// Stop halts the sync daemon and waits for goroutines to finish.
func (d *SyncDaemon) Stop() error {
	d.mu.Lock()
	if !d.running {
		d.mu.Unlock()
		return nil
	}
	d.running = false
	d.cancel()
	d.mu.Unlock()

	d.wg.Wait()
	d.config.Logger.Info("sync daemon stopped")
	return nil
}

// IsRunning reports whether the daemon is active.
func (d *SyncDaemon) IsRunning() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.running
}

// GetRepairLog returns the history of repair events for a client.
func (d *SyncDaemon) GetRepairLog(client Kind) []SyncEvent {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.repairLog[client]
}

// syncLoop runs the periodic sync for a single client.
func (d *SyncDaemon) syncLoop(client Kind) {
	defer d.wg.Done()

	// Perform initial sync immediately
	d.verifyAndRepair(client)

	ticker := time.NewTicker(d.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			d.verifyAndRepair(client)
		}
	}
}

// verifyAndRepair checks if mach1 config is present and repairs if needed.
func (d *SyncDaemon) verifyAndRepair(client Kind) {
	start := time.Now()

	// Check config
	needsRepair, err := d.needsConfigRepair(client)
	if err != nil {
		d.logEvent(client, SyncEvent{
			Time:   start,
			Client: client,
			Action: "verify_error",
			Error:  err,
		})
		return
	}

	if needsRepair && d.config.AutoRepair {
		path, err := d.repairConfig(client)
		d.logEvent(client, SyncEvent{
			Time:   start,
			Client: client,
			Action: "repaired_config",
			Path:   path,
			Error:  err,
		})
	} else {
		d.logEvent(client, SyncEvent{
			Time:   start,
			Client: client,
			Action: "verified",
		})
	}

	// Check rules if enabled
	if d.config.VerifyRules {
		d.verifyAndRepairRules(client)
	}

	d.mu.Lock()
	d.lastSync[client] = time.Now()
	d.mu.Unlock()
}

// needsConfigRepair checks if the mach1 entry is missing from client config.
func (d *SyncDaemon) needsConfigRepair(client Kind) (bool, error) {
	path, key, err := configPath(client)
	if err != nil {
		return false, err
	}

	// Check if config file exists
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			// No config file - needs creation
			return true, nil
		}
		return false, err
	}

	// Read and parse config
	var root map[string]any
	switch client {
	case OpenCode:
		root, err = readJSONCObject(path)
	default:
		root, err = readJSONObject(path)
	}
	if err != nil {
		return false, err
	}

	servers, _ := root[key].(map[string]any)
	if servers == nil {
		return true, nil // No servers section - needs repair
	}

	// Check for mach1 entry
	if _, ok := servers[EntryName]; !ok {
		return true, nil // mach1 entry missing - needs repair
	}

	return false, nil
}

// repairConfig re-injects the mach1 entry into the client config.
func (d *SyncDaemon) repairConfig(client Kind) (string, error) {
	switch client {
	case OpenCode:
		path, _, err := ConnectTakeoverOpenCode(d.entry)
		return path, err
	case Codex:
		path, _, err := ConnectTakeoverCodex(d.entry)
		return path, err
	default:
		path, _, err := ConnectTakeover(client, d.entry)
		return path, err
	}
}

// verifyAndRepairRules checks and repairs rule files if needed.
func (d *SyncDaemon) verifyAndRepairRules(client Kind) {
	// Find project root
	projectDir, err := FindProjectRoot("")
	if err != nil {
		return // No project found - skip rules check
	}

	// Check if rule file exists and has directive
	hasDirective, err := d.checkRulesDirective(client, projectDir)
	if err != nil || hasDirective {
		return // Either error or already has directive
	}

	// Inject rules
	result, err := InjectRules(client, projectDir)
	d.logEvent(client, SyncEvent{
		Time:   time.Now(),
		Client: client,
		Action: "repaired_rules",
		Path:   result.Path,
		Error:  err,
	})
}

// checkRulesDirective checks if the rule file has the 1MCP directive.
func (d *SyncDaemon) checkRulesDirective(client Kind, projectDir string) (bool, error) {
	path, err := FindRuleFile(client, projectDir)
	if err != nil {
		return false, err
	}
	if path == "" {
		return false, nil // No rule file - will be created if needed
	}

	return HasDirective(path)
}

// logEvent records a sync event in the repair log.
func (d *SyncDaemon) logEvent(client Kind, event SyncEvent) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.repairLog[client] = append(d.repairLog[client], event)

	// Keep only last 100 events per client
	if len(d.repairLog[client]) > 100 {
		d.repairLog[client] = d.repairLog[client][len(d.repairLog[client])-100:]
	}

	// Log to configured logger
	if d.config.Logger != nil {
		if event.Error != nil {
			d.config.Logger.Warn("sync event",
				"client", client,
				"action", event.Action,
				"path", event.Path,
				"error", event.Error,
			)
		} else if event.Action == "repaired_config" || event.Action == "repaired_rules" {
			d.config.Logger.Info("sync event",
				"client", client,
				"action", event.Action,
				"path", event.Path,
			)
		}
	}
}

// ---------------------------------------------------------------------------
// Client-specific sync status helpers
// ---------------------------------------------------------------------------

// SyncStatus describes the current sync state for a client.
type SyncStatus struct {
	Client     Kind
	LastSync   time.Time
	RepairCount int
	Healthy    bool
}

// GetStatus returns the current sync status for all monitored clients.
func (d *SyncDaemon) GetStatus() []SyncStatus {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var statuses []SyncStatus
	for _, client := range d.clients {
		lastSync := d.lastSync[client]
		repairs := 0
		for _, event := range d.repairLog[client] {
			if event.Action == "repaired_config" || event.Action == "repaired_rules" {
				repairs++
			}
		}

		// Healthy if synced within 2x interval
		healthy := time.Since(lastSync) < d.config.Interval*2

		statuses = append(statuses, SyncStatus{
			Client:      client,
			LastSync:    lastSync,
			RepairCount: repairs,
			Healthy:     healthy,
		})
	}

	return statuses
}

// MarshalJSON implements json.Marshaler for SyncEvent.
func (e SyncEvent) MarshalJSON() ([]byte, error) {
	type Alias SyncEvent
	return json.Marshal(&struct {
		Time   string `json:"time"`
		Error  string `json:"error,omitempty"`
		*Alias
	}{
		Time:   e.Time.Format(time.RFC3339),
		Error:  errorString(e.Error),
		Alias:  (*Alias)(&e),
	})
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
