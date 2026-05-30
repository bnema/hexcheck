package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".hexcheck.yaml")
	data := []byte(`version: 1
components:
  core:
    paths: [internal/domain/**]
    role: core
  adapters:
    paths: [internal/infrastructure/**]
    role: adapter
rules:
  no-adapter-imports-in-core: error
allow:
  - rule: no-local-fakes-for-ports
    path: internal/foo_test.go
    reason: stateful fake
`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Version != 1 {
		t.Fatalf("Version = %d", cfg.Version)
	}
	if cfg.Components["core"].Role != RoleCore {
		t.Fatalf("core role = %q", cfg.Components["core"].Role)
	}
	if cfg.Severity("no-adapter-imports-in-core") != SeverityError {
		t.Fatalf("severity not parsed")
	}
	if !cfg.IsAllowed("no-local-fakes-for-ports", "internal/foo_test.go") {
		t.Fatalf("allow entry not matched")
	}
}

func TestComponentForPathPrecedence(t *testing.T) {
	cfg := &Config{Components: map[string]Component{
		"all":     {Role: RoleCore, Paths: []string{"internal/**"}},
		"adapter": {Role: RoleAdapter, Paths: []string{"internal/infrastructure/**"}},
		"ignore":  {Role: RoleIgnore, Paths: []string{"internal/infrastructure/generated/**"}},
	}}
	cfg.applyDefaults()

	match, ok := cfg.ComponentForPath("internal/infrastructure/postgres/repo.go")
	if !ok || match.Role != RoleAdapter {
		t.Fatalf("match = %#v, %v", match, ok)
	}
	match, ok = cfg.ComponentForPath("internal/infrastructure/generated/foo.go")
	if !ok || match.Role != RoleIgnore {
		t.Fatalf("ignore match = %#v, %v", match, ok)
	}
}

func TestValidateRejectsBadRole(t *testing.T) {
	cfg := &Config{Version: 1, Components: map[string]Component{
		"bad": {Role: "wat", Paths: []string{"internal/**"}},
	}}
	cfg.applyDefaults()
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}
