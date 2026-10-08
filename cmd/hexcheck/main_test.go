package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bnema/hexcheck/config"
)

// writeModule creates a tiny module with one core->adapter import (error rule)
// and one adapter->adapter import (warn rule).
func writeModule(t *testing.T, rules string) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"go.mod":                     "module example.com/m\n\ngo 1.25\n",
		"internal/domain/d.go":       "package domain\n\nimport _ \"example.com/m/internal/adapters/db\"\n",
		"internal/adapters/db/db.go": "package db\n",
		"internal/adapters/web/w.go": "package web\n\nimport _ \"example.com/m/internal/adapters/db\"\n",
		".hexcheck.yaml": `version: 1
components:
  core: { role: core, paths: [internal/domain/**] }
  db: { role: adapter, paths: [internal/adapters/db/**] }
  web: { role: adapter, paths: [internal/adapters/web/**] }
` + rules,
	}
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestRunExitCodes(t *testing.T) {
	tests := []struct {
		name     string
		rules    string
		args     []string
		wantCode int
		wantOut  []string
	}{
		{
			name:     "error rule fails",
			args:     nil,
			wantCode: exitFindings,
			wantOut:  []string{"[error] no-adapter-imports-in-core", "[warn] no-adapter-to-adapter-imports"},
		},
		{
			name:     "warn only passes by default",
			rules:    "rules:\n  no-adapter-imports-in-core: warn\n",
			wantCode: exitOK,
			wantOut:  []string{"[warn] no-adapter-imports-in-core"},
		},
		{
			name:     "warn fails with -fail-on=warn",
			rules:    "rules:\n  no-adapter-imports-in-core: warn\n",
			args:     []string{"-fail-on", "warn"},
			wantCode: exitFindings,
		},
		{
			name:     "invalid config fails",
			rules:    "rules:\n  not-a-rule: error\n",
			wantCode: exitFailure,
			wantOut:  []string{"unknown rule"},
		},
		{
			name:     "invalid fail-on",
			args:     []string{"-fail-on", "info"},
			wantCode: exitUsage,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := writeModule(t, tt.rules)
			var stdout, stderr bytes.Buffer
			args := append([]string{"-root", dir}, tt.args...)
			got := run(append(args, "./..."), &stdout, &stderr)
			if got != tt.wantCode {
				t.Fatalf("exit = %d, want %d\nstderr:\n%s", got, tt.wantCode, stderr.String())
			}
			for _, want := range tt.wantOut {
				if !strings.Contains(stderr.String(), want) {
					t.Fatalf("stderr missing %q:\n%s", want, stderr.String())
				}
			}
		})
	}
}

func TestRunJSON(t *testing.T) {
	dir := writeModule(t, "")
	var stdout, stderr bytes.Buffer
	if got := run([]string{"-root", dir, "-json", "./..."}, &stdout, &stderr); got != exitFindings {
		t.Fatalf("exit = %d, want %d\nstderr:\n%s", got, exitFindings, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"category": "no-adapter-imports-in-core"`) {
		t.Fatalf("JSON output missing category:\n%s", stdout.String())
	}
}

func TestInit(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{"internal/domain", "internal/adapters", "cmd/app"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	var stdout, stderr bytes.Buffer
	if got := run([]string{"init", dir}, &stdout, &stderr); got != exitOK {
		t.Fatalf("init exit = %d\n%s", got, stderr.String())
	}
	cfg, err := config.Load(filepath.Join(dir, ".hexcheck.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for name, role := range map[string]config.Role{"core": config.RoleCore, "adapters": config.RoleAdapter, "entrypoints": config.RoleEntrypoint} {
		if cfg.Components[name].Role != role {
			t.Fatalf("component %s role = %q, want %q", name, cfg.Components[name].Role, role)
		}
	}
	if _, ok := cfg.Components["usecases"]; ok {
		t.Fatal("usecases should not be generated without a matching directory")
	}

	if got := run([]string{"init", dir}, &stdout, &stderr); got != exitFailure {
		t.Fatalf("second init exit = %d, want %d", got, exitFailure)
	}
	if got := run([]string{"init", "-force", dir}, &stdout, &stderr); got != exitOK {
		t.Fatalf("init -force exit = %d\n%s", got, stderr.String())
	}
}
