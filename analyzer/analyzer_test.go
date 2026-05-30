package analyzer

import (
	"testing"

	"github.com/bnema/hexcheck/config"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzerBoundaryImports(t *testing.T) {
	cfg := testConfig()
	analysistest.Run(t, analysistest.TestData(), New(Options{Config: cfg, ModulePath: "example.com/project"}),
		"example.com/project/internal/domain",
		"example.com/project/internal/application/usecase",
		"example.com/project/internal/infrastructure/http",
	)
}

func TestAnalyzerTypeLeaks(t *testing.T) {
	cfg := testConfig()
	analysistest.Run(t, analysistest.TestData(), New(Options{Config: cfg, ModulePath: "example.com/project"}),
		"example.com/project/internal/application/port",
	)
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
	}
	if err := cfg.Validate(); err != nil {
		panic(err)
	}
	return cfg
}
