package analyzer

import "testing"

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
