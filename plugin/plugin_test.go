package plugin

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSource(t *testing.T) {
	got := Source("v1.2.3", "/Users/a b/\"x\"\\bin\n\x01é")
	want := `local SEX_VERSION, SEX_BIN, SEX_RUN = "v1.2.3", "/Users/a b/\"x\"\\bin\010\001é", "#`
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

func runStart(t *testing.T, home, bin string) (string, error) {
	t.Helper()
	cmd := exec.Command("/bin/sh", "-c", runScript, "sex", bin, "pick", "--", "a b")
	cmd.Env = []string{"HOME=" + home, "PATH=/usr/bin:/bin"}
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func fakeSex(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\nprintf '%s|' \"$0\" \"$@\"\ncommand -v fzf || :\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestRunScriptUsesGivenBinary(t *testing.T) {
	home := t.TempDir()
	bin := filepath.Join(home, "elsewhere", "sex")
	fakeSex(t, bin)
	out, err := runStart(t, home, bin)
	if err != nil || !strings.HasPrefix(out, bin+"|pick|--|a b|") {
		t.Fatalf("out %q, err %v", out, err)
	}
}

// A remote host lacks the plugin host's path; sex and fzf live in mise shims there.
func TestRunScriptFindsSexOnOtherHost(t *testing.T) {
	home := t.TempDir()
	shims := filepath.Join(home, ".local", "share", "mise", "shims")
	fakeSex(t, filepath.Join(shims, "sex"))
	fakeSex(t, filepath.Join(shims, "fzf"))
	out, err := runStart(t, home, "/Users/nobody/bin/sex")
	if err != nil || !strings.HasPrefix(out, filepath.Join(shims, "sex")+"|pick|--|a b|/") || !strings.HasSuffix(out, "/fzf\n") {
		t.Fatalf("out %q, err %v", out, err)
	}
}

func TestRunScriptReportsMissingSex(t *testing.T) {
	out, err := runStart(t, t.TempDir(), "/Users/nobody/bin/sex")
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 127 || !strings.Contains(out, "sex: not found on") {
		t.Fatalf("out %q, err %v", out, err)
	}
}
