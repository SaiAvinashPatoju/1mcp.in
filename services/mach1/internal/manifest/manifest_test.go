package manifest

import "testing"

func TestValidateOK(t *testing.T) {
	m := &Manifest{
		ID: "memory", Name: "Memory", Version: "1.0.0",
		Transport: "stdio", Runtime: "node",
		Entrypoint: Entrypoint{Command: "npx", Args: []string{"-y", "x"}},
	}
	if err := m.Validate(); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestValidateEmptyTransportDefaultsToStdio(t *testing.T) {
	m := &Manifest{
		ID: "memory", Name: "Memory", Version: "1.0.0",
		Transport: "", Runtime: "node",
		Entrypoint: Entrypoint{Command: "npx", Args: []string{"-y", "x"}},
	}
	if err := m.Validate(); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if m.Transport != "stdio" {
		t.Fatalf("expected transport to default to stdio, got %q", m.Transport)
	}
}

func TestValidateBadID(t *testing.T) {
	m := &Manifest{ID: "X", Name: "n", Version: "1.0.0", Transport: "stdio", Runtime: "node",
		Entrypoint: Entrypoint{Command: "x"}}
	if err := m.Validate(); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateDockerNeedsImage(t *testing.T) {
	m := &Manifest{ID: "ok", Name: "n", Version: "1.0.0", Transport: "stdio", Runtime: "docker",
		Entrypoint: Entrypoint{}}
	if err := m.Validate(); err == nil {
		t.Fatal("expected docker image error")
	}
}

func TestParseRejectsInvalid(t *testing.T) {
	_, err := Parse([]byte(`{"id":"ok","name":"n","version":"bad","transport":"stdio","runtime":"node","entrypoint":{"command":"x"}}`))
	if err == nil {
		t.Fatal("expected version error")
	}
}

func TestResolveEnvKeys(t *testing.T) {
	m := &Manifest{
		ID: "github", Name: "GitHub", Version: "1.0.0",
		Transport: "stdio", Runtime: "node",
		Entrypoint: Entrypoint{Command: "npx"},
		EnvSchema: []EnvVar{
			{Name: "GITHUB_PERSONAL_ACCESS_TOKEN", Secret: true, Required: true, Aliases: []string{"GITHUB_TOKEN", "GH_TOKEN"}},
		},
	}
	if err := m.Validate(); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	aliases := m.ResolveEnvKeys()
	if len(aliases) != 2 {
		t.Fatalf("expected 2 aliases, got %d", len(aliases))
	}
	if aliases["GITHUB_TOKEN"] != "GITHUB_PERSONAL_ACCESS_TOKEN" {
		t.Fatalf("expected GITHUB_TOKEN -> GITHUB_PERSONAL_ACCESS_TOKEN, got %s", aliases["GITHUB_TOKEN"])
	}
	if aliases["GH_TOKEN"] != "GITHUB_PERSONAL_ACCESS_TOKEN" {
		t.Fatalf("expected GH_TOKEN -> GITHUB_PERSONAL_ACCESS_TOKEN, got %s", aliases["GH_TOKEN"])
	}
}
