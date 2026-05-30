package glob

import "testing"

func TestMatch(t *testing.T) {
	tests := []struct {
		pattern string
		name    string
		want    bool
	}{
		{"internal/domain/**", "internal/domain/user.go", true},
		{"internal/domain/**", "internal/domain/entity/user.go", true},
		{"cmd/**", "cmd/hexcheck/main.go", true},
		{"cmd/**", "internal/cmd/main.go", false},
		{"internal/*/port/**", "internal/application/port/user.go", true},
		{"internal/*/port/**", "internal/application/usecase/user.go", false},
		{"internal/foo_test.go", "internal/foo_test.go", true},
	}
	for _, tt := range tests {
		if got := Match(tt.pattern, tt.name); got != tt.want {
			t.Fatalf("Match(%q, %q) = %v, want %v", tt.pattern, tt.name, got, tt.want)
		}
	}
}
