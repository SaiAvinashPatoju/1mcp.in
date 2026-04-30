// Package secrets stores per-MCP secret env vars outside the registry SQLite
// database, in a 0600-permissioned file under the 1mcp.in data dir.
//
// Values are encrypted at rest on Windows with DPAPI. Other platforms keep the
// same interface and permission hardening until their native keychain backends
// are added. Secret values must never be logged.
package secrets

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Store is goroutine-safe.
type Store struct {
	path string
	mu   sync.Mutex
	data map[string]map[string]string // mcpId -> envName -> value
}

// Open loads (or creates) the secrets file at path. Parent dirs are created
// with 0700 to mirror the file mode.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("mkdir secrets dir: %w", err)
	}
	s := &Store{path: path, data: map[string]map[string]string{}}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read secrets: %w", err)
	}
	if len(b) == 0 {
		return s, nil
	}
	plaintext, err := unprotectSecretBytes(b)
	if err != nil {
		return nil, fmt.Errorf("decrypt secrets: %w", err)
	}
	if err := json.Unmarshal(plaintext, &s.data); err != nil {
		return nil, fmt.Errorf("parse secrets: %w", err)
	}
	return s, nil
}

// Get returns a copy of the secret env map for mcpId (may be empty).
func (s *Store) Get(mcpID string) map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]string, len(s.data[mcpID]))
	for k, v := range s.data[mcpID] {
		out[k] = v
	}
	return out
}

// Set stores a single secret env var. Empty value deletes the key.
func (s *Store) Set(mcpID, name, value string) error {
	if mcpID == "" || name == "" {
		return errors.New("secrets: empty mcpID or name")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[mcpID]; !ok {
		s.data[mcpID] = map[string]string{}
	}
	if value == "" {
		delete(s.data[mcpID], name)
		if len(s.data[mcpID]) == 0 {
			delete(s.data, mcpID)
		}
	} else {
		s.data[mcpID][name] = value
	}
	return s.flushLocked()
}

// DeleteAll removes every secret for mcpID. Used on uninstall.
func (s *Store) DeleteAll(mcpID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, mcpID)
	return s.flushLocked()
}

// Names returns the list of stored secret names for mcpID (values withheld).
func (s *Store) Names(mcpID string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	m := s.data[mcpID]
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func (s *Store) flushLocked() error {
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	sealed, err := protectSecretBytes(b)
	if err != nil {
		return fmt.Errorf("encrypt secrets: %w", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, sealed, 0o600); err != nil {
		return fmt.Errorf("write secrets: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("rename secrets: %w", err)
	}
	return nil
}
