package analyzer

import (
	"testing"

	"github.com/bnema/hexcheck/config"
)

func TestResolveConfigDefaultsPartialInMemoryConfig(t *testing.T) {
	tests := []struct {
		name string
		cfg  *config.Config
	}{
		{
			name: "partial config",
			cfg: &config.Config{Version: 1, Components: map[string]config.Component{
				"adapters": {Role: config.RoleAdapter, Paths: []string{"internal/**"}},
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, _, err := resolveConfig(Options{Config: tt.cfg}, "", "")
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Heuristics.BusinessLogicThreshold == nil || cfg.Heuristics.BusinessLogicMaxFunctionNodes == nil {
				t.Fatal("heuristic defaults were not applied")
			}
			if tt.cfg.Heuristics.BusinessLogicThreshold != nil {
				t.Fatal("input config was mutated")
			}
		})
	}
}
