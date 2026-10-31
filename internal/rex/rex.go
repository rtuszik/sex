package rex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/rtuszik/sex/internal/execx"
)

const (
	TerminalFlavor = "com.superlogical.terminal.shell"
	macAppBin      = "Library/Application Support/com.superlogical.rex/RexServer/Versions/Current/Rex Server.app/Contents/MacOS/rex"
)

var (
	codeRe        = regexp.MustCompile(`\(status=\d+ code=([a-z_]+)\)\s*$`)
	streamNoiseRe = regexp.MustCompile(`^stream response "[^"]*" failed: `)
)

type APIError struct {
	Code string
	Msg  string
	Exit int
}

func (e *APIError) Error() string { return e.Msg }

func IsCode(err error, code string) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.Code == code
}

func IsRemoteControlOff(err error) bool { return IsCode(err, "refused") }

// IsUnreachable reports the CLI's exit status 4: no server answered.
func IsUnreachable(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.Exit == 4
}

type Session struct {
	ID    string `json:"session_id"`
	Label string `json:"label"`
}

type AppClient struct {
	ID    string
	Name  string
	Label string
}

type Window struct {
	Label string
	Cwd   string
	Input string
}

type CreateOpts struct {
	Label   string
	Windows []Window
}

type Created struct {
	SessionID string
	WindowIDs []string
}

type Client struct {
	run execx.Runner
	bin string
}

func New(run execx.Runner, bin string) *Client {
	return &Client{run: run, bin: bin}
}

// FindBin prefers the CLI that matches the running server.
func FindBin() string {
	if dir := os.Getenv("REX_BIN_DIR"); dir != "" {
		if p := filepath.Join(dir, "rex"); isFile(p) {
			return p
		}
	}
	if p, err := exec.LookPath("rex"); err == nil {
		return p
	}
	if home, err := os.UserHomeDir(); err == nil {
		if p := filepath.Join(home, macAppBin); isFile(p) {
			return p
		}
	}
	return "rex"
}

func isFile(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

// read runs a call that must not start a server: listing should not leave an empty one behind.
func (c *Client) read(ctx context.Context, stdin []byte, args ...string) ([]byte, error) {
	return c.exec(ctx, stdin, append([]string{"--autostart=false"}, args...)...)
}

func (c *Client) exec(ctx context.Context, stdin []byte, args ...string) ([]byte, error) {
	out, err := c.run.Run(ctx, execx.Cmd{Name: c.bin, Args: args, Stdin: stdin})
	if err != nil {
		return nil, wrap(err)
	}
	return out, nil
}

func wrap(err error) error {
	var execErr *execx.Error
	if !errors.As(err, &execErr) || execErr.Stderr == "" {
		return err
	}
	msg := strings.TrimPrefix(execErr.Stderr, "Error: ")
	msg = streamNoiseRe.ReplaceAllString(msg, "")
	apiErr := &APIError{Msg: "rex: " + msg}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		apiErr.Exit = exitErr.ExitCode()
	}
	if m := codeRe.FindStringSubmatch(msg); m != nil {
		apiErr.Code = m[1]
	}
	return apiErr
}

func (c *Client) Call(ctx context.Context, method string, args, out any) error {
	return c.call(ctx, c.exec, method, args, out)
}

func (c *Client) call(ctx context.Context, run func(context.Context, []byte, ...string) ([]byte, error), method string, args, out any) error {
	if args == nil {
		args = map[string]any{}
	}
	in, err := json.Marshal(args)
	if err != nil {
		return err
	}
	b, err := run(ctx, in, "api", "call", method, "-")
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(b, out); err != nil {
		return fmt.Errorf("rex: decoding %s result: %w", method, err)
	}
	return nil
}

func (c *Client) Sessions(ctx context.Context) ([]Session, error) {
	var res struct {
		Sessions []Session `json:"sessions"`
	}
	if err := c.call(ctx, c.read, "session.list", nil, &res); err != nil {
		return nil, err
	}
	return res.Sessions, nil
}

func (c *Client) AppClients(ctx context.Context) ([]AppClient, error) {
	var res struct {
		Clients []struct {
			ID   string `json:"client_id"`
			Info *struct {
				Kind  string `json:"kind"`
				Name  string `json:"name"`
				Label string `json:"label"`
			} `json:"info"`
		} `json:"clients"`
	}
	if err := c.call(ctx, c.read, "client.list", nil, &res); err != nil {
		return nil, err
	}
	var apps []AppClient
	for _, cl := range res.Clients {
		if cl.Info != nil && cl.Info.Kind == "app" {
			apps = append(apps, AppClient{ID: cl.ID, Name: cl.Info.Name, Label: cl.Info.Label})
		}
	}
	return apps, nil
}

func (c *Client) Create(ctx context.Context, opts CreateOpts) (Created, error) {
	type options struct {
		Cwd          string `json:"cwd,omitempty"`
		InitialInput string `json:"initial_input,omitempty"`
	}
	type block struct {
		Flavor  string  `json:"flavor"`
		Options options `json:"options"`
	}
	type layout struct {
		Block block `json:"block"`
	}
	type window struct {
		Label  string `json:"window_label,omitempty"`
		Layout layout `json:"layout"`
	}
	req := struct {
		Label   string   `json:"label"`
		Windows []window `json:"initial_windows"`
	}{Label: opts.Label}
	for _, w := range opts.Windows {
		req.Windows = append(req.Windows, window{
			Label:  w.Label,
			Layout: layout{Block: block{Flavor: TerminalFlavor, Options: options{Cwd: w.Cwd, InitialInput: w.Input}}},
		})
	}
	var res struct {
		SessionID string `json:"session_id"`
		Windows   []struct {
			WindowID string `json:"window_id"`
		} `json:"initial_windows"`
	}
	if err := c.Call(ctx, "session.create", req, &res); err != nil {
		return Created{}, err
	}
	created := Created{SessionID: res.SessionID}
	for _, w := range res.Windows {
		created.WindowIDs = append(created.WindowIDs, w.WindowID)
	}
	return created, nil
}

func (c *Client) FocusWindow(ctx context.Context, sessionID, windowID string) error {
	return c.Call(ctx, "session.focus_window", map[string]string{"session_id": sessionID, "window_id": windowID}, nil)
}

func (c *Client) Destroy(ctx context.Context, sessionID string) error {
	return c.Call(ctx, "session.destroy", map[string]string{"session_id": sessionID}, nil)
}

func (c *Client) Perform(ctx context.Context, clientID, action string, args any) error {
	req := map[string]any{"client_id": clientID, "action": action}
	if args != nil {
		req["args"] = args
	}
	return c.Call(ctx, "client.perform", req, nil)
}

func (c *Client) Select(ctx context.Context, clientID, sessionID string) error {
	return c.Perform(ctx, clientID, "session.select", map[string]string{"session_id": sessionID})
}

func (c *Client) Capture(ctx context.Context, sessionID, blockID string) (string, error) {
	out, err := c.read(ctx, nil, "capture", "-s", sessionID, "-b", blockID, "--format", "vt", "--trim")
	return string(out), err
}

// BlockOwner returns the client that controls the terminal's size, the app it is shown in.
func (c *Client) BlockOwner(ctx context.Context, sessionID, blockID string) (string, error) {
	var res struct {
		Owner string `json:"owner"`
	}
	args := map[string]any{"session_id": sessionID, "block_id": blockID, "method": "size", "args": map[string]any{}}
	if err := c.call(ctx, c.read, "block.method", args, &res); err != nil {
		return "", err
	}
	return res.Owner, nil
}
