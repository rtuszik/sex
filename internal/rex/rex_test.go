package rex

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rtuszik/sex/internal/execx"
)

type call struct {
	args  []string
	stdin map[string]any
}

// fake answers each rex invocation with the next canned output.
type fake struct {
	calls   []call
	outputs []string
	errs    []error
}

func (f *fake) Run(_ context.Context, cmd execx.Cmd) ([]byte, error) {
	c := call{args: cmd.Args}
	if len(cmd.Stdin) > 0 && cmd.Stdin[0] == '{' {
		_ = json.Unmarshal(cmd.Stdin, &c.stdin)
	}
	f.calls = append(f.calls, c)
	i := len(f.calls) - 1
	var out string
	var err error
	if i < len(f.outputs) {
		out = f.outputs[i]
	}
	if i < len(f.errs) {
		err = f.errs[i]
	}
	return []byte(out), err
}

func newClient(outputs ...string) (*Client, *fake) {
	f := &fake{outputs: outputs}
	return New(f, "rex"), f
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestSessions(t *testing.T) {
	c, f := newClient(`{"sessions":[{"session_id":"session:1","label":"api"}]}`)
	got, err := c.Sessions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "session:1" || got[0].Label != "api" {
		t.Fatalf("got %+v", got)
	}
	if strings.Join(f.calls[0].args, " ") != "--autostart=false api call session.list -" {
		t.Fatalf("args %v", f.calls[0].args)
	}
}

func TestAppClientsFiltersApps(t *testing.T) {
	c, _ := newClient(`{"clients":[{"client_id":"client:cli"},{"client_id":"client:x","info":{"kind":"cli"}},{"client_id":"client:app","info":{"kind":"app","name":"Rex","label":"Mac"}}]}`)
	got, err := c.AppClients(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != (AppClient{ID: "client:app", Name: "Rex", Label: "Mac"}) {
		t.Fatalf("got %+v", got)
	}
}

func TestCreateRequest(t *testing.T) {
	c, f := newClient(`{"session_id":"session:new","initial_windows":[{"window_id":"window:1"},{"window_id":"window:2"}]}`)
	got, err := c.Create(context.Background(), CreateOpts{Label: "api", Windows: []Window{
		{Cwd: "/src/api", Input: "nvim\n"},
		{Label: "git", Cwd: "/src/api"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if got.SessionID != "session:new" || strings.Join(got.WindowIDs, ",") != "window:1,window:2" {
		t.Fatalf("got %+v", got)
	}
	want := `{"initial_windows":[{"layout":{"block":{"flavor":"com.superlogical.terminal.shell","options":{"cwd":"/src/api","initial_input":"nvim\n"}}}},{"layout":{"block":{"flavor":"com.superlogical.terminal.shell","options":{"cwd":"/src/api"}}},"window_label":"git"}],"label":"api"}`
	if gotReq := mustJSON(t, f.calls[0].stdin); gotReq != want {
		t.Fatalf("request\n got %s\nwant %s", gotReq, want)
	}
}

func TestSimpleCalls(t *testing.T) {
	ctx := context.Background()
	c, f := newClient("{}", "{}", "{}", "{}")
	if err := c.FocusWindow(ctx, "session:1", "window:1"); err != nil {
		t.Fatal(err)
	}
	if err := c.Destroy(ctx, "session:1"); err != nil {
		t.Fatal(err)
	}
	if err := c.Select(ctx, "client:1", "session:1"); err != nil {
		t.Fatal(err)
	}
	if err := c.Perform(ctx, "client:1", "noop", nil); err != nil {
		t.Fatal(err)
	}
	wants := []string{
		`api call session.focus_window - {"session_id":"session:1","window_id":"window:1"}`,
		`api call session.destroy - {"session_id":"session:1"}`,
		`api call client.perform - {"action":"session.select","args":{"session_id":"session:1"},"client_id":"client:1"}`,
		`api call client.perform - {"action":"noop","client_id":"client:1"}`,
	}
	for i, want := range wants {
		got := strings.Join(f.calls[i].args, " ") + " " + mustJSON(t, f.calls[i].stdin)
		if got != want {
			t.Fatalf("call %d\n got %s\nwant %s", i, got, want)
		}
	}
}

func TestCapture(t *testing.T) {
	c, f := newClient("screen\n")
	got, err := c.Capture(context.Background(), "session:1", "block:1")
	if err != nil {
		t.Fatal(err)
	}
	if got != "screen\n" || strings.Join(f.calls[0].args, " ") != "--autostart=false capture -s session:1 -b block:1 --format vt --trim" {
		t.Fatalf("got %q, args %v", got, f.calls[0].args)
	}
}

func TestSnapshot(t *testing.T) {
	c, f := newClient(`{"sessions":[{"session_id":"session:1","label":"a","block_id":"block:1","first_window_id":"window:1","cwd":"/a","shown_at":"2026-10-25T21:18:11.424871+01:00"},{"session_id":"session:2","label":"b"}]}`)
	got, err := c.Snapshot(context.Background(), "client:1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Cwd != "/a" || got[0].ShownAt.IsZero() || !got[1].ShownAt.IsZero() {
		t.Fatalf("got %+v", got)
	}
	if strings.Join(f.calls[0].args, " ") != `--autostart=false do - --args {"client":"client:1"}` {
		t.Fatalf("args %v", f.calls[0].args)
	}
}

func TestByRecent(t *testing.T) {
	now := time.Now()
	in := []Snapshot{{ID: "old", ShownAt: now.Add(-time.Hour)}, {ID: "none"}, {ID: "new", ShownAt: now}, {ID: "none2"}}
	got := ByRecent(in)
	var ids []string
	for _, s := range got {
		ids = append(ids, s.ID)
	}
	if strings.Join(ids, ",") != "new,old,none,none2" || in[0].ID != "old" {
		t.Fatalf("got %v", ids)
	}
}

func TestErrors(t *testing.T) {
	ctx := context.Background()
	refused := &execx.Error{Stderr: `Error: stream response "control-1" failed: Rex.app is not accepting actions. Turn on Remote Control in its Settings under Rex Server. (status=403 code=refused)`}
	f := &fake{errs: []error{refused}}
	c := New(f, "rex")
	err := c.Select(ctx, "client:1", "session:1")
	if !IsRemoteControlOff(err) {
		t.Fatalf("err = %v, want refused", err)
	}
	if want := "rex: Rex.app is not accepting actions. Turn on Remote Control in its Settings under Rex Server. (status=403 code=refused)"; err.Error() != want {
		t.Fatalf("message %q", err.Error())
	}

	plain := errors.New("exec: not found")
	c = New(&fake{errs: []error{plain}}, "rex")
	if _, err := c.Sessions(ctx); !errors.Is(err, plain) || IsCode(err, "refused") {
		t.Fatalf("err = %v", err)
	}

	c = New(&fake{errs: []error{&execx.Error{Stderr: "Error: boom"}}}, "rex")
	if _, err := c.Sessions(ctx); err == nil || err.Error() != "rex: boom" || !IsCode(err, "") {
		t.Fatalf("err = %v", err)
	}

	for _, out := range []string{"not json"} {
		c, _ = newClient(out, out, out, out)
		if _, err := c.Sessions(ctx); err == nil {
			t.Fatal("want decode error")
		}
		if _, err := c.Snapshot(ctx, ""); err == nil {
			t.Fatal("want decode error")
		}
	}
	c = New(&fake{errs: []error{plain, plain, plain, plain}}, "rex")
	if _, err := c.Snapshot(ctx, ""); err == nil {
		t.Fatal("want error")
	}
	if _, err := c.AppClients(ctx); err == nil {
		t.Fatal("want error")
	}
	if _, err := c.Create(ctx, CreateOpts{}); err == nil {
		t.Fatal("want error")
	}
	if err := c.Call(ctx, "x", func() {}, nil); err == nil {
		t.Fatal("want marshal error")
	}
}

func TestFindBin(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "rex")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("REX_BIN_DIR", dir)
	if got := FindBin(); got != bin {
		t.Fatalf("got %s, want %s", got, bin)
	}
	t.Setenv("REX_BIN_DIR", "")
	t.Setenv("PATH", dir)
	if got := FindBin(); got != bin {
		t.Fatalf("got %s, want %s", got, bin)
	}
	t.Setenv("PATH", "")
	t.Setenv("HOME", t.TempDir())
	if got := FindBin(); got != "rex" {
		t.Fatalf("got %s, want rex", got)
	}
}

func TestBlockOwner(t *testing.T) {
	c, f := newClient(`{"columns":80,"owner":"client:app"}`)
	got, err := c.BlockOwner(context.Background(), "session:1", "block:1")
	if err != nil || got != "client:app" {
		t.Fatalf("got %q, err %v", got, err)
	}
	want := `--autostart=false api call block.method - {"args":{},"block_id":"block:1","method":"size","session_id":"session:1"}`
	if got := strings.Join(f.calls[0].args, " ") + " " + mustJSON(t, f.calls[0].stdin); got != want {
		t.Fatalf("call\n got %s\nwant %s", got, want)
	}
	c = New(&fake{errs: []error{errors.New("down")}}, "rex")
	if _, err := c.BlockOwner(context.Background(), "s", "b"); err == nil {
		t.Fatal("want error")
	}
}

func TestMutatingCallsMayAutostart(t *testing.T) {
	c, f := newClient("{}")
	if err := c.Destroy(context.Background(), "session:1"); err != nil {
		t.Fatal(err)
	}
	if f.calls[0].args[0] == "--autostart=false" {
		t.Fatal("destroy must not disable autostart")
	}
}

func TestIsUnreachable(t *testing.T) {
	err := exec.Command("sh", "-c", "exit 4").Run()
	wrapped := wrap(&execx.Error{Stderr: "Error: no Rex server is running at /x.sock, and autostart is disabled", Err: err})
	if !IsUnreachable(wrapped) {
		t.Fatalf("want unreachable: %#v", wrapped)
	}
	other := wrap(&execx.Error{Stderr: "Error: boom", Err: exec.Command("sh", "-c", "exit 1").Run()})
	if IsUnreachable(other) || IsUnreachable(errors.New("plain")) {
		t.Fatal("only exit status 4 is unreachable")
	}
}
