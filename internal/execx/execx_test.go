package execx

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
)

func TestOSRun(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	out, err := OS{}.Run(ctx, Cmd{Name: "sh", Args: []string{"-c", "pwd; cat"}, Dir: dir, Stdin: []byte("in")})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(string(out), "in") || !strings.Contains(string(out), dir[len(dir)-8:]) {
		t.Fatalf("out %q", out)
	}

	_, err = OS{}.Run(ctx, Cmd{Name: "sh", Args: []string{"-c", "echo bad >&2; exit 3"}})
	var execErr *Error
	if !errors.As(err, &execErr) || execErr.Error() != "sh: bad" {
		t.Fatalf("err = %v", err)
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 3 {
		t.Fatalf("unwrap = %v", errors.Unwrap(err))
	}

	_, err = OS{}.Run(ctx, Cmd{Name: "sh", Args: []string{"-c", "exit 1"}})
	if err == nil || !strings.HasPrefix(err.Error(), "sh: exit status 1") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunnerFunc(t *testing.T) {
	f := RunnerFunc(func(_ context.Context, c Cmd) ([]byte, error) { return []byte(c.Name), nil })
	out, _ := f.Run(context.Background(), Cmd{Name: "x"})
	if string(out) != "x" {
		t.Fatalf("out %q", out)
	}
}
