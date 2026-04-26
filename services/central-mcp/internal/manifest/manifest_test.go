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
