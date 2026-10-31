package plugin

import (
	"strings"
	"testing"
)

func TestSource(t *testing.T) {
	got := Source("v1.2.3", "/Users/a b/\"x\"\\bin\n\x01é")
	want := `local SEX_VERSION, SEX_BIN = "v1.2.3", "/Users/a b/\"x\"\\bin\010\001é"` + "\n"
	if !strings.HasPrefix(got, want) {
		t.Fatalf("got %q", got[:strings.Index(got, "\n")+1])
	}
	if !strings.Contains(got, `name = "sex_version"`) || !strings.Contains(string(Loader), `"plugin", "--source"`) {
		t.Fatal("plugin or loader content missing")
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
