package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/bnema/hexcheck/config"
)

// layoutCandidates lists common directory layouts per component, in output order.
var layoutCandidates = []struct {
	name  string
	role  config.Role
	paths []string
}{
	{"core", config.RoleCore, []string{"internal/domain", "internal/core", "domain", "core"}},
	{"usecases", config.RoleUsecase, []string{"internal/application/usecase", "internal/application/usecases", "internal/usecase", "internal/usecases", "internal/app", "usecase", "usecases"}},
	{"ports", config.RolePorts, []string{"internal/application/port", "internal/application/ports", "internal/port", "internal/ports", "internal/domain/repository", "ports"}},
	{"adapters", config.RoleAdapter, []string{"internal/infrastructure", "internal/adapters", "internal/adapter", "internal/infra", "adapters", "infrastructure"}},
	{"entrypoints", config.RoleEntrypoint, []string{"cmd", "internal/cli", "internal/http", "internal/api"}},
}

func runInit(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("hexcheck init", flag.ContinueOnError)
	fs.SetOutput(stderr)
	force := fs.Bool("force", false, "overwrite an existing .hexcheck.yaml")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitOK
		}
		return exitUsage
	}
	dir := "."
	if fs.NArg() > 0 {
		dir = fs.Arg(0)
	}
	target := filepath.Join(dir, ".hexcheck.yaml")
	if _, err := os.Stat(target); err == nil && !*force {
		fmt.Fprintf(stderr, "hexcheck: %s already exists (use -force to overwrite)\n", target)
		return exitFailure
	}

	content, detected := starterConfig(dir)
	if err := writeValidated(target, content); err != nil {
		fmt.Fprintf(stderr, "hexcheck: %v\n", err)
		return exitFailure
	}
	fmt.Fprintf(stdout, "wrote %s (%d component(s) detected); review the paths, then run: hexcheck ./...\n", target, detected)
	return exitOK
}

// writeValidated writes content to a temp file next to target, loads it as a
// config, and only then renames it over target.
func writeValidated(target, content string) error {
	tmp, err := os.CreateTemp(filepath.Dir(target), ".hexcheck-*.yaml")
	if err != nil {
		return err
	}
	// After a successful rename the temp file is gone, so this cleanup only
	// matters on the error paths.
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err := tmp.WriteString(content); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if _, err := config.Load(tmp.Name()); err != nil {
		return fmt.Errorf("generated config is invalid: %w", err)
	}
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), target)
}

// starterConfig builds a config from directories that exist under dir.
// Components with no detected directory keep a commented-out default path.
func starterConfig(dir string) (string, int) {
	var b strings.Builder
	b.WriteString("version: 1\ncomponents:\n")
	detected := 0
	for _, c := range layoutCandidates {
		var found []string
		for _, p := range c.paths {
			if info, err := os.Stat(filepath.Join(dir, p)); err == nil && info.IsDir() {
				found = append(found, p+"/**")
			}
		}
		if len(found) == 0 {
			fmt.Fprintf(&b, "  # %s:\n  #   role: %s\n  #   paths: [%s/**]\n", c.name, c.role, c.paths[0])
			continue
		}
		detected++
		fmt.Fprintf(&b, "  %s:\n    role: %s\n    paths:\n", c.name, c.role)
		for _, p := range found {
			fmt.Fprintf(&b, "      - %s\n", p)
		}
	}
	b.WriteString(`  generated:
    role: ignore
    paths:
      - '**/mocks/**'
      - '**/generated/**'
      - '**/*_templ.go'
      - '**/*_gen.go'

# Every rule is enabled with its default severity. Override here, e.g.:
# rules:
#   suspicious-business-logic-in-adapter: off

mocking:
  generatedMockPaths:
    - '**/mocks/**'
`)
	return b.String(), detected
}
