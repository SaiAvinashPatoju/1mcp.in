// onemcpctl is the OneMCP control CLI. It manages the same SQLite registry
// and secrets store that centralmcpd reads at launch, so any operation here
// is immediately reflected on the next router restart.
//
// Commands:
//
//	onemcpctl catalog list [--catalog FILE]
//	onemcpctl install <id>   [--catalog FILE]
//	onemcpctl uninstall <id>
//	onemcpctl list
//	onemcpctl enable <id> | disable <id>
//	onemcpctl env set <id> NAME=VALUE
//	onemcpctl env list <id>
//	onemcpctl env unset <id> NAME
//	onemcpctl start [--transport http|stdio]
//	onemcpctl connect <client>    # client = vscode|cursor|claude
//	onemcpctl disconnect <client>
//	onemcpctl doctor
//
// All commands exit 0 on success, non-zero on error, with human-readable
// stderr output. JSON output mode is reserved for Phase 8 (hub IPC).
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/onemcp/central-mcp/internal/catalog"
	"github.com/onemcp/central-mcp/internal/clients"
	"github.com/onemcp/central-mcp/internal/install"
	"github.com/onemcp/central-mcp/internal/manifest"
	"github.com/onemcp/central-mcp/internal/paths"
	"github.com/onemcp/central-mcp/internal/registry"
	"github.com/onemcp/central-mcp/internal/secrets"
)

const defaultCatalogRel = "packages/registry-index/index.json"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd, args := os.Args[1], os.Args[2:]
	var err error
	switch cmd {
	case "catalog":
		err = runCatalog(args)
	case "install":
		err = runInstall(args)
	case "tools":
		err = runTools(args)
	case "uninstall":
		err = runUninstall(args)
	case "list":
		err = runList(args)
	case "enable":
		err = runSetEnabled(args, true)
	case "disable":
		err = runSetEnabled(args, false)
	case "env":
		err = runEnv(args)
	case "start":
		err = runStart(args)
	case "connect":
		err = runConnect(args)
	case "disconnect":
		err = runDisconnect(args)
	case "doctor":
		err = runDoctor(args)
	case "-h", "--help", "help":
		usage()
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `onemcpctl - OneMCP control CLI

  catalog list [--catalog FILE]
  install <id> [--catalog FILE]
	tools pending
	tools approve <mcp-id> <tool-name>
  uninstall <id>
  list
  enable <id> | disable <id>
  env set <id> NAME=VALUE
  env unset <id> NAME
  env list <id>
	start [--transport http|stdio]
  connect <vscode|cursor|claude>
  disconnect <vscode|cursor|claude>
  doctor

Data dir:        $ONEMCP_HOME or %APPDATA%/OneMcp
Default catalog: ./packages/registry-index/index.json (or --catalog)
`)
}

func runStart(args []string) error {
	transport := "http"
	listen := "127.0.0.1:3000"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--transport":
			if i+1 >= len(args) {
				return errors.New("--transport requires stdio or http")
			}
			transport = args[i+1]
			i++
		case "--listen":
			if i+1 >= len(args) {
				return errors.New("--listen requires an address")
			}
			listen = args[i+1]
			i++
		default:
			return fmt.Errorf("unknown start flag: %s", args[i])
		}
	}
	exe, err := centralBinaryPath()
	if err != nil {
		return err
	}
	dbPath, err := paths.RegistryDB()
	if err != nil {
		return err
	}
	cmdArgs := []string{"--db", dbPath, "--transport", transport}
	if transport == "http" {
		cmdArgs = append(cmdArgs, "--listen", listen)
	}
	cmd := exec.Command(exe, cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	fmt.Printf("starting 1mcp router (%s)\n", transport)
	return cmd.Run()
}

// ----- catalog ---------------------------------------------------------------

func runCatalog(args []string) error {
	if len(args) < 1 || args[0] != "list" {
		return errors.New("usage: onemcpctl catalog list [--catalog FILE]")
	}
	path := pickCatalogFlag(args[1:])
	entries, err := catalog.Load(path)
	if err != nil && len(entries) == 0 {
		return err
	}
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tNAME\tVERSION\tRUNTIME\tDESCRIPTION")
	for _, m := range entries {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", m.ID, m.Name, m.Version, m.Runtime, truncate(m.Description, 60))
	}
	return tw.Flush()
}

// ----- install / uninstall ---------------------------------------------------

func runInstall(args []string) error {
	if len(args) < 1 {
		return errors.New("usage: onemcpctl install <id>")
	}
	id := args[0]
	catPath := pickCatalogFlag(args[1:])
	entries, _ := catalog.Load(catPath)
	m := catalog.Find(entries, id)
	if m == nil {
		return fmt.Errorf("id %q not found in catalog %s", id, catPath)
	}

	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	inst := &install.Installer{DB: db}
	res, err := inst.Install(context.Background(), m)
	if err != nil {
		return err
	}
	verb := "installed"
	if res.Already {
		verb = "reinstalled"
	}
	fmt.Printf("%s %s (%s) in %s\n", verb, m.ID, m.Version, res.Duration.Round(time.Millisecond))
	if res.Verification != "" {
		fmt.Printf("verified %s catalog digest: %s\n", res.Verification, m.SHA256)
	}
	if res.Warning != "" {
		fmt.Println("warning:", res.Warning)
	}
	if needs := requiredEnvMissing(m); len(needs) > 0 {
		fmt.Printf("\nRequired env not yet set: %s\n", strings.Join(needs, ", "))
		fmt.Printf("  onemcpctl env set %s %s=...\n", m.ID, needs[0])
	}
	return nil
}

func runTools(args []string) error {
	if len(args) < 1 {
		return errors.New("usage: onemcpctl tools pending | tools approve <mcp-id> <tool-name>")
	}
	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()
	switch args[0] {
	case "pending":
		pending, err := db.ListPendingToolReviews(context.Background())
		if err != nil {
			return err
		}
		if len(pending) == 0 {
			fmt.Println("no tool definition changes pending review")
			return nil
		}
		tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "MCP\tTOOL\tAPPROVED_HASH\tCURRENT_HASH")
		for _, r := range pending {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", r.MCPID, r.ToolName, r.ApprovedHash, r.CurrentHash)
		}
		return tw.Flush()
	case "approve":
		if len(args) < 3 {
			return errors.New("usage: onemcpctl tools approve <mcp-id> <tool-name>")
		}
		if err := db.ApproveToolDefinition(context.Background(), args[1], args[2]); err != nil {
			return err
		}
		fmt.Printf("approved tool definition %s__%s\n", args[1], args[2])
		return nil
	default:
		return fmt.Errorf("unknown tools subcommand: %s", args[0])
	}
}

func runUninstall(args []string) error {
	if len(args) < 1 {
		return errors.New("usage: onemcpctl uninstall <id>")
	}
	id := args[0]
	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()
	inst := &install.Installer{DB: db}
	if err := inst.Uninstall(context.Background(), id); err != nil {
		return err
	}
	if s, err := openSecrets(); err == nil {
		_ = s.DeleteAll(id)
	}
	fmt.Println("uninstalled", id)
	return nil
}

// ----- list / enable / disable -----------------------------------------------

func runList(_ []string) error {
	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()
	entries, err := db.ListAll(context.Background())
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		fmt.Println("no MCPs installed. try: onemcpctl install <id>")
		return nil
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].ID < entries[j].ID })
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tVERSION\tRUNTIME\tENABLED\tCOMMAND")
	for _, e := range entries {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%v\t%s %s\n", e.ID, e.Version, e.Runtime, e.Enabled, e.Command, strings.Join(e.Args, " "))
	}
	return tw.Flush()
}

func runSetEnabled(args []string, enabled bool) error {
	if len(args) < 1 {
		op := "disable"
		if enabled {
			op = "enable"
		}
		return fmt.Errorf("usage: onemcpctl %s <id>", op)
	}
	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()
	if err := db.SetEnabled(context.Background(), args[0], enabled); err != nil {
		return err
	}
	state := "disabled"
	if enabled {
		state = "enabled"
	}
	fmt.Printf("%s %s\n", state, args[0])
	return nil
}

// ----- env -------------------------------------------------------------------

func runEnv(args []string) error {
	if len(args) < 1 {
		return errors.New("usage: onemcpctl env set|unset|list ...")
	}
	switch args[0] {
	case "set":
		return envSet(args[1:])
	case "unset":
		return envUnset(args[1:])
	case "list":
		return envList(args[1:])
	}
	return fmt.Errorf("unknown env subcommand: %s", args[0])
}

func envSet(args []string) error {
	if len(args) < 2 {
		return errors.New("usage: onemcpctl env set <id> NAME=VALUE")
	}
	id := args[0]
	name, value, ok := strings.Cut(args[1], "=")
	if !ok || name == "" {
		return errors.New("expected NAME=VALUE")
	}

	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()
	_, manifestJSON, err := db.Get(context.Background(), id)
	if err != nil {
		return fmt.Errorf("get %s: %w", id, err)
	}
	m, err := manifest.Parse(manifestJSON)
	if err != nil {
		return fmt.Errorf("decode stored manifest: %w", err)
	}
	isSecret := false
	known := false
	for _, e := range m.EnvSchema {
		if e.Name == name {
			known = true
			isSecret = e.Secret
			break
		}
	}
	if !known {
		fmt.Fprintf(os.Stderr, "note: %s is not declared in the manifest envSchema; storing as non-secret.\n", name)
	}

	if isSecret {
		s, err := openSecrets()
		if err != nil {
			return err
		}
		if err := s.Set(id, name, value); err != nil {
			return err
		}
		fmt.Printf("stored secret %s for %s (%s)\n", name, id, redact(value))
		return nil
	}

	entry, _, err := db.Get(context.Background(), id)
	if err != nil {
		return err
	}
	if entry.Env == nil {
		entry.Env = map[string]string{}
	}
	entry.Env[name] = value
	if err := db.SetEnv(context.Background(), id, entry.Env); err != nil {
		return err
	}
	fmt.Printf("set %s=%s for %s\n", name, value, id)
	return nil
}

func envUnset(args []string) error {
	if len(args) < 2 {
		return errors.New("usage: onemcpctl env unset <id> NAME")
	}
	id, name := args[0], args[1]
	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()
	entry, _, err := db.Get(context.Background(), id)
	if err == nil && entry.Env != nil {
		if _, ok := entry.Env[name]; ok {
			delete(entry.Env, name)
			if err := db.SetEnv(context.Background(), id, entry.Env); err != nil {
				return err
			}
		}
	}
	if s, err := openSecrets(); err == nil {
		_ = s.Set(id, name, "")
	}
	fmt.Printf("unset %s for %s\n", name, id)
	return nil
}

func envList(args []string) error {
	if len(args) < 1 {
		return errors.New("usage: onemcpctl env list <id>")
	}
	id := args[0]
	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()
	entry, _, err := db.Get(context.Background(), id)
	if err != nil {
		return err
	}
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tKIND\tVALUE")
	for _, name := range sortedKeys(entry.Env) {
		fmt.Fprintf(tw, "%s\tplain\t%s\n", name, entry.Env[name])
	}
	if s, err := openSecrets(); err == nil {
		for _, n := range s.Names(id) {
			fmt.Fprintf(tw, "%s\tsecret\t%s\n", n, "(set)")
		}
	}
	return tw.Flush()
}

// ----- connect / disconnect --------------------------------------------------

func runConnect(args []string) error {
	if len(args) < 1 {
		return errors.New("usage: onemcpctl connect <vscode|cursor|claude>")
	}
	kind := clients.Kind(args[0])
	exe, err := centralBinaryPath()
	if err != nil {
		return err
	}
	dbPath, err := paths.RegistryDB()
	if err != nil {
		return err
	}
	entry := clients.ServerEntry{
		Command: exe,
		Args:    []string{"--db", dbPath},
	}
	path, err := clients.Connect(kind, entry)
	if err != nil {
		return err
	}
	fmt.Printf("registered onemcp in %s\n", path)
	return nil
}

func runDisconnect(args []string) error {
	if len(args) < 1 {
		return errors.New("usage: onemcpctl disconnect <vscode|cursor|claude>")
	}
	kind := clients.Kind(args[0])
	path, removed, err := clients.Disconnect(kind)
	if err != nil {
		return err
	}
	if removed {
		fmt.Printf("removed onemcp from %s\n", path)
	} else {
		fmt.Printf("no onemcp entry in %s\n", path)
	}
	return nil
}

// ----- doctor ----------------------------------------------------------------

func runDoctor(_ []string) error {
	checks := []struct {
		label string
		fn    func() (string, error)
	}{
		{"central-mcp binary", func() (string, error) { return centralBinaryPath() }},
		{"registry db", func() (string, error) {
			p, err := paths.RegistryDB()
			if err != nil {
				return "", err
			}
			if _, err := os.Stat(p); err == nil {
				return p + " (exists)", nil
			}
			return p + " (will be created)", nil
		}},
		{"node (npx)", func() (string, error) { return exec.LookPath("npx") }},
		{"python (uvx)", func() (string, error) { return exec.LookPath("uvx") }},
		{"docker", func() (string, error) { return exec.LookPath("docker") }},
	}
	for _, c := range checks {
		v, err := c.fn()
		if err != nil {
			fmt.Printf("[!] %-20s %v\n", c.label, err)
			continue
		}
		fmt.Printf("[ok] %-19s %s\n", c.label, v)
	}
	return nil
}

// ----- helpers ---------------------------------------------------------------

func openDB() (*registry.DB, error) {
	p, err := paths.RegistryDB()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return nil, err
	}
	return registry.Open(p)
}

func openSecrets() (*secrets.Store, error) {
	p, err := paths.SecretsFile()
	if err != nil {
		return nil, err
	}
	return secrets.Open(p)
}

func centralBinaryPath() (string, error) {
	// Prefer ONEMCP_BIN override (used by tests and packaged builds).
	if v := os.Getenv("ONEMCP_BIN"); v != "" {
		if _, err := os.Stat(v); err == nil {
			return v, nil
		}
	}
	// Look beside this binary first.
	self, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(self)
		for _, name := range []string{"centralmcpd", "centralmcpd.exe"} {
			p := filepath.Join(dir, name)
			if _, err := os.Stat(p); err == nil {
				return p, nil
			}
		}
	}
	if p, err := exec.LookPath("centralmcpd"); err == nil {
		return p, nil
	}
	return "", errors.New("centralmcpd binary not found; set ONEMCP_BIN or place it next to onemcpctl")
}

func pickCatalogFlag(args []string) string {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--catalog" {
			return args[i+1]
		}
	}
	if v := os.Getenv("ONEMCP_CATALOG"); v != "" {
		return v
	}
	// Walk up from cwd to find the repo's default catalog (dev mode).
	cwd, _ := os.Getwd()
	dir := cwd
	for i := 0; i < 6; i++ {
		p := filepath.Join(dir, defaultCatalogRel)
		if _, err := os.Stat(p); err == nil {
			return p
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return defaultCatalogRel
}

func requiredEnvMissing(m *manifest.Manifest) []string {
	var out []string
	for _, e := range m.EnvSchema {
		if e.Required {
			out = append(out, e.Name)
		}
	}
	return out
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func redact(s string) string {
	if len(s) <= 4 {
		return "***"
	}
	return s[:2] + "***" + s[len(s)-2:]
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
