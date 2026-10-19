package analyzer

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/bnema/hexcheck/config"

	"golang.org/x/tools/go/analysis/analysistest"
)

const businessLogicRule = "suspicious-business-logic-in-adapter"

func TestBusinessLogicGoldenCases(t *testing.T) {
	result := analysistest.Run(t, analysistest.TestData(), New(Options{Config: testConfig(), ModulePath: "example.com/project"}), "example.com/project/internal/infrastructure/http")

	// rulesByFile maps each file to the diagnostic categories reported in it.
	rulesByFile := map[string][]string{}
	for _, diagnostic := range result[0].Diagnostics {
		file := filepath.ToSlash(result[0].Pass.Fset.Position(diagnostic.Pos).Filename)
		rulesByFile[file] = append(rulesByFile[file], diagnostic.Category)
	}

	tests := []struct {
		file string
		want bool
	}{
		{file: "typed_policy.go", want: true},
		{file: "mixed_detection.go", want: true},
		{file: "probing.go", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			if got := hasRuleInFile(rulesByFile, tt.file, businessLogicRule); got != tt.want {
				t.Fatalf("business-logic diagnostic in %s = %v, want %v", tt.file, got, tt.want)
			}
		})
	}
}

func TestBusinessLogicCapGolden(t *testing.T) {
	cfg := testConfig()
	cfg.Components["capped"] = config.Component{Role: config.RoleAdapter, Paths: []string{"internal/infrastructure/capped/**"}}
	cfg.Heuristics.BusinessLogicMaxDiagnosticsPerPackage = testIntPtr(1)
	analysistest.Run(t, analysistest.TestData(), New(Options{Config: cfg, ModulePath: "example.com/project"}), "example.com/project/internal/infrastructure/capped")
}

func TestUsedIgnoreInTestPackageVariants(t *testing.T) {
	cfg := testConfig()
	cfg.Components["variants"] = config.Component{Role: config.RoleAdapter, Paths: []string{"internal/infrastructure/variants/**"}}
	results := analysistest.Run(t, analysistest.TestData(), New(Options{Config: cfg, ModulePath: "example.com/project"}), "example.com/project/internal/infrastructure/variants")
	plain, withTests := false, false
	for _, result := range results {
		if len(result.Diagnostics) != 0 {
			t.Fatalf("%s: unexpected diagnostics: %v", result.Pass.Pkg.Path(), result.Diagnostics)
		}
		hasTestFile := false
		for _, file := range result.Pass.Files {
			if strings.HasSuffix(result.Pass.Fset.Position(file.Pos()).Filename, "_test.go") {
				hasTestFile = true
			}
		}
		if hasTestFile {
			withTests = true
		} else {
			plain = true
		}
	}
	if !plain || !withTests {
		t.Fatalf("expected both ordinary and test package variants; plain=%v withTests=%v", plain, withTests)
	}
}

func hasRuleInFile(rulesByFile map[string][]string, fileSuffix, rule string) bool {
	for file, rules := range rulesByFile {
		if !strings.HasSuffix(file, fileSuffix) {
			continue
		}
		for _, got := range rules {
			if got == rule {
				return true
			}
		}
	}
	return false
}
