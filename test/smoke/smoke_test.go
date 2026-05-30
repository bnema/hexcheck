package smoke

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestLocalRepositorySmoke(t *testing.T) {
	repo := os.Getenv("HEXCHECK_SMOKE_REPO")
	if repo == "" {
		t.Skip("HEXCHECK_SMOKE_REPO is not set")
	}
	root := projectRoot(t)
	cmd := exec.Command("go", "run", "./cmd/hexcheck", "-hexcheck.config", filepath.Join(root, "examples", "hexcheck.yaml"), "-hexcheck.root", repo, "./...")
	cmd.Dir = repo
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("hexcheck smoke command failed: %v\n%s", err, output)
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
