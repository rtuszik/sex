package execx

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type Cmd struct {
	Name  string
	Args  []string
	Dir   string
	Stdin []byte
}

type Runner interface {
	Run(ctx context.Context, cmd Cmd) ([]byte, error)
}

type RunnerFunc func(ctx context.Context, cmd Cmd) ([]byte, error)

func (f RunnerFunc) Run(ctx context.Context, cmd Cmd) ([]byte, error) {
	return f(ctx, cmd)
}

type Error struct {
	Cmd    Cmd
	Stderr string
	Err    error
}

func (e *Error) Error() string {
	if e.Stderr != "" {
		return fmt.Sprintf("%s: %s", e.Cmd.Name, e.Stderr)
	}
	return fmt.Sprintf("%s: %v", e.Cmd.Name, e.Err)
}

func (e *Error) Unwrap() error { return e.Err }

type OS struct{}

// Run takes a fixed binary (rex, sesh, zoxide, git) and an argv; no shell parses it.
func (OS) Run(ctx context.Context, c Cmd) ([]byte, error) {
	cmd := exec.CommandContext(ctx, c.Name, c.Args...) // noopengrep: go.lang.security.audit.dangerous-exec-command.dangerous-exec-command
	cmd.Dir = c.Dir
	if c.Stdin != nil {
		cmd.Stdin = bytes.NewReader(c.Stdin)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return out, &Error{Cmd: c, Stderr: strings.TrimSpace(stderr.String()), Err: err}
	}
	return out, nil
}
