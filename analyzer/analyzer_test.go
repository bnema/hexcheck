package analyzer

import (
	"testing"

	"github.com/bnema/hexcheck/config"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
	}{
		{
			name: "boundary imports and heuristics",
			patterns: []string{
				"example.com/project/internal/domain",
				"example.com/project/internal/application/usecase",
				"example.com/project/internal/infrastructure/http",
			},
		},
		{
			name: "type leaks",
			patterns: []string{
				"example.com/project/internal/application/port",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analysistest.Run(t, analysistest.TestData(), New(Options{Config: testConfig(), ModulePath: "example.com/project"}), tt.patterns...)
		})
	}
}

func testConfig() *config.Config {
	cfg := &config.Config{
		Version: 1,
		Components: map[string]config.Component{
			"domain":  {Role: config.RoleCore, Paths: []string{"internal/domain/**"}},
			"usecase": {Role: config.RoleUsecase, Paths: []string{"internal/application/usecase/**"}},
			"ports":   {Role: config.RolePorts, Paths: []string{"internal/application/port/**"}},
			"http":    {Role: config.RoleAdapter, Paths: []string{"internal/infrastructure/http/**"}},
			"sql":     {Role: config.RoleAdapter, Paths: []string{"internal/infrastructure/sql/**"}},
		},
		Rules: config.DefaultRuleSeverities(),
		ExternalTypes: config.ExternalTypes{
			FrameworkPackages: []string{"example.com/framework"},
		},
		Allow: []config.Allow{
			{Rule: "no-infra-imports-in-usecase", Path: "internal/application/usecase/*_test.go", Reason: "test architecture rule has a more specific diagnostic"},
		},
	}
	if err := cfg.Validate(); err != nil {
		panic(err)
	}
	return cfg
}
