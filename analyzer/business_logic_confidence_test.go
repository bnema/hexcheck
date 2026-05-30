package analyzer

import (
	"testing"

	"github.com/bnema/hexcheck/config"
)

func TestEffectiveBusinessLogicMinConfidence(t *testing.T) {
	tests := []struct {
		name string
		mode string
		min  string
		want string
	}{
		{name: "audit uses configured minimum", mode: "audit", min: "low", want: "low"},
		{name: "ci promotes low to high", mode: "ci", min: "low", want: "high"},
		{name: "ci promotes medium to high", mode: "ci", min: "medium", want: "high"},
		{name: "ci keeps high", mode: "ci", min: "high", want: "high"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{Heuristics: config.Heuristics{BusinessLogicMode: tt.mode, BusinessLogicMinConfidence: tt.min}}
			if got := effectiveBusinessLogicMinConfidence(cfg); got != tt.want {
				t.Fatalf("effectiveBusinessLogicMinConfidence() = %q, want %q", got, tt.want)
			}
		})
	}
}
