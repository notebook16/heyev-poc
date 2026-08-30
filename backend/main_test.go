package main

import "testing"

func TestIsRepeatInput(t *testing.T) {
	cases := map[string]bool{
		"":         false,
		"lock":     false,
		"r":        true,
		"R":        true,
		"repeat":   true,
		"REPEAT":   true,
		"^r":       true,
		"^R":       true,
		"\x12":     true,
		"  \x12  ": true,
	}
	for in, want := range cases {
		if got := isRepeatInput(in); got != want {
			t.Fatalf("isRepeatInput(%q) = %v, want %v", in, got, want)
		}
	}
}
