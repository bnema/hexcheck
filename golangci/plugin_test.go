package golangci

import "testing"

func TestNew(t *testing.T) {
	tests := []struct {
		name         string
		settings     any
		wantErr      bool
		wantAnalyzer string
	}{
		{
			name:         "nil settings",
			settings:     nil,
			wantAnalyzer: "hexcheck",
		},
		{
			name:         "config settings",
			settings:     map[string]any{"config": ".hexcheck.yaml", "module": "example.com/project"},
			wantAnalyzer: "hexcheck",
		},
		{
			name:     "invalid settings type",
			settings: "bad",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := New(tt.settings)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			analyzers, err := p.BuildAnalyzers()
			if err != nil {
				t.Fatal(err)
			}
			if len(analyzers) != 1 || analyzers[0].Name != tt.wantAnalyzer {
				t.Fatalf("unexpected analyzers: %#v", analyzers)
			}
			if p.GetLoadMode() == "" {
				t.Fatal("load mode is empty")
			}
		})
	}
}
