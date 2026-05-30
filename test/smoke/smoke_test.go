package smoke

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestLocalRepositorySmoke(t *testing.T) {
	tests := []struct {
		name string
		repo string
	}{
		{name: "configured local repository", repo: os.Getenv("HEXCHECK_SMOKE_REPO")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.repo == "" {
				t.Skip("HEXCHECK_SMOKE_REPO is not set")
			}
			root := projectRoot(t)
			bin := filepath.Join(t.TempDir(), "hexcheck")
			build := exec.Command("go", "build", "-o", bin, "./cmd/hexcheck")
			build.Dir = root
			if output, err := build.CombinedOutput(); err != nil {
				t.Fatalf("build hexcheck: %v\n%s", err, output)
			}

			cmd := exec.Command(bin, "-hexcheck.config", filepath.Join(root, "examples", "hexcheck.yaml"), "-hexcheck.root", tt.repo, "./...")
			cmd.Dir = tt.repo
			output, err := cmd.CombinedOutput()
			if err == nil {
				return
			}
			if exit, ok := err.(*exec.ExitError); ok && exit.ExitCode() == 3 && len(output) > 0 {
				return
			}
			t.Fatalf("hexcheck smoke command failed: %v\n%s", err, output)
		})
	}
}

func projectRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}
