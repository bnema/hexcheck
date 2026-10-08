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
		{"internal/domain/**", "internal/domain", true},
		{"cmd/**", "cmd/hexcheck/main.go", true},
		{"cmd/**", "internal/cmd/main.go", false},
		{"internal/*/port/**", "internal/application/port/user.go", true},
		{"internal/*/port/**", "internal/application/usecase/user.go", false},
		{"internal/foo_test.go", "internal/foo_test.go", true},
		{"**/mocks/**", "mocks/m.go", true},
		{"**/mocks/**", "internal/a/b/mocks/m.go", true},
		{"**/mocks/**", "internal/mocksy/m.go", false},
		{"internal/**/repo.go", "internal/repo.go", true},
		{"internal/**/repo.go", "internal/a/b/repo.go", true},
		{"internal/**/repo.go", "internal/a/b/other.go", false},
		{"**/*_gen.go", "a/b/x_gen.go", true},
		{"internal/[ab]*/**", "internal/adapters/x.go", true},
		{"internal/[ab]*/**", "internal/core/x.go", false},
		{"internal/[/**", "internal/x.go", false},
		{"./internal/**/", "internal/x.go", true},
		{`internal\domain\**`, `internal\domain\user.go`, true},
		{"", "", true},
		{"", "x", false},
		{"internal", "internal/x.go", false},
	}
	for _, tt := range tests {
		t.Run(tt.pattern+"|"+tt.name, func(t *testing.T) {
			if got := Match(tt.pattern, tt.name); got != tt.want {
				t.Fatalf("Match(%q, %q) = %v, want %v", tt.pattern, tt.name, got, tt.want)
			}
		})
	}
}

func TestSpecificity(t *testing.T) {
	tests := []struct {
		pattern string
		want    int
	}{
		{"internal/domain/**", len("internal/domain/")},
		{"internal/domain/user.go", len("internal/domain/user.go")},
		{"**/mocks/**", 0},
		{"./cmd/[ab]", len("cmd/")},
	}
	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			if got := Specificity(tt.pattern); got != tt.want {
				t.Fatalf("Specificity(%q) = %d, want %d", tt.pattern, got, tt.want)
			}
		})
	}
}
