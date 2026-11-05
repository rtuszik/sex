package lua

import (
	"strings"
	"testing"
)

func TestLoaderAsksBinaryForSource(t *testing.T) {
	if !strings.Contains(string(Loader), `"plugin", "--source"`) {
		t.Fatal("loader content missing")
	}
}

// An unset variable in the candidate table would be nil and end the search at once.
func TestLoaderCandidatesHaveNoEnvLookups(t *testing.T) {
	loader := string(Loader)
	start := strings.Index(loader, "local candidates = {")
	end := strings.Index(loader[start:], "\n}") + start
	if start < 0 || strings.Contains(loader[start:end], "getenv") {
		t.Fatal("candidates must be plain strings")
	}
}
