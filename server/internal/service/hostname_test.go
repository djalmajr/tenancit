package service

import "testing"

func TestCanonicalHostname(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "lowercase and trailing dot", in: "  App.Example.COM. ", want: "app.example.com"},
		{name: "single label", in: "LOCALHOST", want: "localhost"},
		{name: "empty", in: " . ", want: ""},
		{name: "non ascii", in: "tést.example", want: ""},
		{name: "empty label", in: "bad..example", want: ""},
		{name: "leading hyphen", in: "-bad.example", want: ""},
		{name: "trailing hyphen", in: "bad-.example", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CanonicalHostname(tt.in); got != tt.want {
				t.Fatalf("CanonicalHostname(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
