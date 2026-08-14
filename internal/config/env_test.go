package config

import (
	"strings"
	"testing"
)

func TestParseDotEnv(t *testing.T) {
	in := `
# comment
GEMINI_API_KEY=abc123
QUOTED="hello world"
SPACED = padded
EMPTY=
=novalue
noequals
`
	got := ParseDotEnv(strings.NewReader(in))
	want := map[string]string{
		"GEMINI_API_KEY": "abc123",
		"QUOTED":         "hello world",
		"SPACED":         "padded",
		"EMPTY":          "",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d vars, want %d: %v", len(got), len(want), got)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
}
