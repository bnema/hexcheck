package analyzer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsTestDoubleName(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{name: "fakeUserRepository", want: true},
		{name: "FakeUserRepository", want: true},
		{name: "stubClock", want: true},
		{name: "StubClock", want: true},
		{name: "user_fake", want: true},
		{name: "fake", want: true},
		{name: "stub", want: true},
		{name: "StubbornHandler", want: false},
		{name: "stubbornHandler", want: false},
		{name: "FakeoutService", want: false},
		{name: "fakeoutService", want: false},
		{name: "outfake", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isTestDoubleName(tt.name); got != tt.want {
				t.Fatalf("isTestDoubleName(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestDeclaredTypes(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   []string
	}{
		{
			name: "plain and generic declarations",
			source: `package mocks

type MockCache[K comparable, V any] struct{}
type (
	MockUserRepository struct{}
	helper string
)
`,
			want: []string{"MockCache", "MockUserRepository", "helper"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "mocks.go")
			if err := os.WriteFile(path, []byte(tt.source), 0o600); err != nil {
				t.Fatal(err)
			}
			got := declaredTypes(path)
			if len(got) != len(tt.want) {
				t.Fatalf("declaredTypes() = %#v, want %#v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("declaredTypes() = %#v, want %#v", got, tt.want)
				}
			}
		})
	}
}
