package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
		check   func(t *testing.T, cfg *Config)
	}{
		{
			name: "valid config",
			yaml: `version: 1
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
`,
			check: func(t *testing.T, cfg *Config) {
				t.Helper()
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
			},
		},
		{
			name: "invalid role",
			yaml: `version: 1
components:
  bad:
    paths: [internal/**]
    role: wat
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, ".hexcheck.yaml")
			if err := os.WriteFile(path, []byte(tt.yaml), 0o600); err != nil {
				t.Fatal(err)
			}
			cfg, err := Load(path)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if tt.check != nil {
				tt.check(t, cfg)
			}
		})
	}
}

func TestLoadRejectsExplicitZeroVersion(t *testing.T) {
	tests := []struct {
		name string
		yaml string
	}{
		{name: "zero", yaml: "version: 0\n"},
		{name: "positive zero", yaml: "version: +0\n"},
		{name: "double zero", yaml: "version: 00\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, ".hexcheck.yaml")
			data := []byte(tt.yaml + `components:
  adapters:
    paths: [internal/infrastructure/**]
    role: adapter
`)
			if err := os.WriteFile(path, data, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(path); err == nil {
				t.Fatal("expected explicit zero version to be rejected")
			}
		})
	}
}

func TestLoadHonorsExplicitExcludeTestFilesFalse(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".hexcheck.yaml")
	data := []byte(`version: 1
components:
  adapters:
    paths: [internal/infrastructure/**]
    role: adapter
heuristics:
  excludeTestFiles: false
`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Heuristics.ExcludeTestFiles == nil || *cfg.Heuristics.ExcludeTestFiles {
		t.Fatal("excludeTestFiles=false was not preserved")
	}
	if !filepath.IsAbs(cfg.Root) {
		t.Fatalf("Root = %q, want absolute path", cfg.Root)
	}
}

func TestValidateRejectsInvalidHeuristicKnobs(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{name: "negative threshold", mutate: func(c *Config) { c.Heuristics.BusinessLogicThreshold = intPtr(-1) }},
		{name: "negative strong", mutate: func(c *Config) { c.Heuristics.BusinessLogicMinStrongSignals = intPtr(-1) }},
		{name: "negative weak", mutate: func(c *Config) { c.Heuristics.BusinessLogicMinWeakSignals = intPtr(-1) }},
		{name: "zero max nodes", mutate: func(c *Config) { c.Heuristics.BusinessLogicMaxFunctionNodes = intPtr(0) }},
		{name: "zero max diagnostics", mutate: func(c *Config) { c.Heuristics.BusinessLogicMaxDiagnosticsPerPackage = intPtr(0) }},
		{name: "invalid business logic mode", mutate: func(c *Config) { c.Heuristics.BusinessLogicMode = "loud" }},
		{name: "invalid business logic min confidence", mutate: func(c *Config) { c.Heuristics.BusinessLogicMinConfidence = "sure" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Version: 1, Components: map[string]Component{"adapters": {Role: RoleAdapter, Paths: []string{"internal/**"}}}}
			cfg.applyDefaults()
			tt.mutate(cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestComponentForPathPrecedence(t *testing.T) {
	cfg := &Config{Components: map[string]Component{
		"all":     {Role: RoleCore, Paths: []string{"internal/**"}},
		"adapter": {Role: RoleAdapter, Paths: []string{"internal/infrastructure/**"}},
		"ignore":  {Role: RoleIgnore, Paths: []string{"internal/infrastructure/generated/**"}},
	}}
	cfg.applyDefaults()

	tests := []struct {
		name string
		path string
		want Role
	}{
		{name: "specific adapter beats broad core", path: "internal/infrastructure/postgres/repo.go", want: RoleAdapter},
		{name: "ignore beats adapter", path: "internal/infrastructure/generated/foo.go", want: RoleIgnore},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match, ok := cfg.ComponentForPath(tt.path)
			if !ok || match.Role != tt.want {
				t.Fatalf("match = %#v, %v; want role %q", match, ok, tt.want)
			}
		})
	}
}
