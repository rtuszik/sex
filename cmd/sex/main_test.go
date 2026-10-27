package main

import (
	"runtime/debug"
	"testing"
)

func TestResolveVersion(t *testing.T) {
	info := func(v string, ok bool) func() (*debug.BuildInfo, bool) {
		return func() (*debug.BuildInfo, bool) {
			return &debug.BuildInfo{Main: debug.Module{Version: v}}, ok
		}
	}
	tests := []struct {
		name  string
		v     string
		build func() (*debug.BuildInfo, bool)
		want  string
	}{
		{"ldflags win", "v1.2.3", info("v9.9.9", true), "v1.2.3"},
		{"go install", "dev", info("v0.1.0", true), "v0.1.0"},
		{"pseudo version", "dev", info("v0.0.0-20261027202907-8ca6b08c0a1b", true), "v0.0.0-20261027202907-8ca6b08c0a1b"},
		{"local build", "dev", info("(devel)", true), "dev"},
		{"empty", "dev", info("", true), "dev"},
		{"no build info", "dev", info("v0.1.0", false), "dev"},
	}
	for _, tt := range tests {
		if got := resolveVersion(tt.v, tt.build); got != tt.want {
			t.Fatalf("%s: got %q, want %q", tt.name, got, tt.want)
		}
	}
}
