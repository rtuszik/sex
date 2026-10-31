package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rtuszik/sex/internal/app"
	"github.com/rtuszik/sex/internal/app/apptest"
	"github.com/rtuszik/sex/internal/rex"
	"github.com/rtuszik/sex/internal/sesh"
	"github.com/rtuszik/sex/internal/state"
	"github.com/rtuszik/sex/plugin"
)

const appID = "client:app-0001"

type deps struct {
	rex       *apptest.Rex
	sesh      *apptest.Sesh
	namer     *apptest.Namer
	reloads   int
	reloadErr error
}

func run(t *testing.T, d *deps, args ...string) (string, string, error) {
	t.Helper()
	g := &globals{build: func(stderr io.Writer) (*app.App, error) {
		return &app.App{
			Rex: d.rex, Sesh: d.sesh, Namer: d.namer,
			Zoxide: &apptest.Zoxide{}, Store: &apptest.Store{State: state.State{Sessions: map[string]state.Record{}}},
			Shell: &apptest.Shell{Output: "dir listing\n"},
			Home:  "/home/me", Cwd: "/home/me", Getenv: func(string) string { return "" }, Stderr: stderr,
		}, nil
	}, reload: func(context.Context) error {
		d.reloads++
		return d.reloadErr
	}}
	root := newRoot("test", g)
	var out, errOut bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errOut)
	root.SetArgs(args)
	err := root.ExecuteContext(context.Background())
	return out.String(), errOut.String(), err
}

func newDeps() *deps {
	return &deps{
		rex: &apptest.Rex{
			Apps: []rex.AppClient{{ID: appID, Label: "Mac"}},
			Snaps: []rex.Snapshot{
				{ID: "session:a", Label: "api", BlockID: "block:a"},
				{ID: "session:b", Label: "web", BlockID: "block:b"},
			},
			Screen: "screen\n",
		},
		sesh:  &apptest.Sesh{Entries: []sesh.Entry{{Src: "config", Name: "dotfiles", Path: "/home/me/.config"}}},
		namer: &apptest.Namer{Roots: map[string]string{"/home/me": "me"}},
	}
}

func TestListCommand(t *testing.T) {
	d := newDeps()
	out, _, err := run(t, d, "list")
	if err != nil {
		t.Fatal(err)
	}
	if out != "api\nweb\ndotfiles\n" {
		t.Fatalf("out %q", out)
	}

	out, _, err = run(t, d, "list", "-t", "-i", "-n")
	if err != nil {
		t.Fatal(err)
	}
	if out != " api\n web\n" {
		t.Fatalf("out %q", out)
	}

	out, _, err = run(t, d, "list", "-c", "-j")
	if err != nil {
		t.Fatal(err)
	}
	var items []map[string]any
	if err := json.Unmarshal([]byte(out), &items); err != nil || len(items) != 1 || items[0]["name"] != "dotfiles" {
		t.Fatalf("out %q, err %v", out, err)
	}

	d.sesh.Entries = nil
	out, _, err = run(t, d, "list", "-z", "-j")
	if err != nil || strings.TrimSpace(out) != "[]" {
		t.Fatalf("out %q, err %v", out, err)
	}

	d.rex.SnapErr = errors.New("down")
	if _, _, err := run(t, d, "list", "-r"); err == nil {
		t.Fatal("want error")
	}
}

func TestConnectLastKillPreviewRoot(t *testing.T) {
	d := newDeps()
	if _, _, err := run(t, d, "connect", "", "web"); err != nil {
		t.Fatal(err)
	}
	if strings.Join(d.rex.Selected, ",") != appID+"|session:b" {
		t.Fatalf("selected %v", d.rex.Selected)
	}

	d.rex.CreateRes = rex.Created{SessionID: "session:new", WindowIDs: []string{"window:1"}}
	if _, _, err := run(t, d, "connect", "-b", "-c", "htop", "dotfiles"); err != nil {
		t.Fatal(err)
	}
	if d.rex.Created[0].Windows[0].Input != "htop\n" || len(d.rex.Selected) != 1 {
		t.Fatalf("created %+v selected %v", d.rex.Created, d.rex.Selected)
	}

	d = newDeps()
	if _, _, err := run(t, d, "last"); err == nil {
		t.Fatal("want no previous session without shown times")
	}
	d.rex.Snaps[0].ShownAt = d.rex.Snaps[0].ShownAt.AddDate(2026, 0, 0)
	d.rex.Snaps[1].ShownAt = d.rex.Snaps[0].ShownAt.AddDate(0, 0, -1)
	out, _, err := run(t, d, "last", "--print")
	if err != nil || out != "session:b\n" {
		t.Fatalf("out %q, err %v", out, err)
	}
	if _, _, err := run(t, d, "last"); err != nil || d.rex.Selected[0] != appID+"|session:b" {
		t.Fatalf("err %v selected %v", err, d.rex.Selected)
	}
	d.rex.Apps = nil
	if _, _, err := run(t, d, "last", "-p"); err == nil {
		t.Fatal("want client error")
	}

	d = newDeps()
	if _, _, err := run(t, d, "kill", "api"); err != nil || d.rex.Destroyed[0] != "session:a" {
		t.Fatalf("err %v destroyed %v", err, d.rex.Destroyed)
	}

	out, _, err = run(t, d, "preview", "web")
	if err != nil || out != "screen\n" {
		t.Fatalf("out %q, err %v", out, err)
	}

	out, _, err = run(t, d, "root")
	if err != nil || out != "me\n" {
		t.Fatalf("out %q, err %v", out, err)
	}
	d.namer.Roots = nil
	if _, _, err := run(t, d, "root"); err == nil {
		t.Fatal("want root error")
	}
}

func TestBuildErrorsPropagate(t *testing.T) {
	boom := errors.New("config broken")
	for _, args := range [][]string{{"list"}, {"connect", "x"}, {"last"}, {"preview", "x"}, {"kill", "x"}, {"root"}, {"pick"}, {"activity"}, {"activity", "pick"}, {"activity", "preview", "x"}} {
		g := &globals{build: func(io.Writer) (*app.App, error) { return nil, boom }}
		root := newRoot("test", g)
		root.SetOut(io.Discard)
		root.SetErr(io.Discard)
		root.SetArgs(args)
		if err := root.Execute(); !errors.Is(err, boom) {
			t.Fatalf("%v: err = %v", args, err)
		}
	}
}

func TestPluginCommand(t *testing.T) {
	out, _, err := run(t, newDeps(), "plugin")
	if err != nil || out != string(plugin.Loader) {
		t.Fatalf("err %v, out %.80q", err, out)
	}
	out, _, err = run(t, newDeps(), "plugin", "--source")
	if err != nil || !strings.HasPrefix(out, `local SEX_VERSION, SEX_BIN = "test", "/`) || !strings.Contains(out, `name = "sex_pick"`) {
		t.Fatalf("err %v, out %.120q", err, out)
	}
}

func TestPluginInstall(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lua", "sex.lua")
	out, _, err := run(t, newDeps(), "plugin", "install", "--path", path)
	if err != nil || !strings.Contains(out, "wrote "+path) {
		t.Fatalf("out %q, err %v", out, err)
	}
	if got, _ := os.ReadFile(path); string(got) != string(plugin.Loader) {
		t.Fatal("loader not written")
	}
	if _, _, err := run(t, newDeps(), "plugin", "install", "--path", path); err == nil || !strings.Contains(err.Error(), "--force") {
		t.Fatalf("err = %v", err)
	}
	if _, _, err := run(t, newDeps(), "plugin", "install", "--path", path, "--force"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := run(t, newDeps(), "plugin", "install", "--path", filepath.Join(path, "x.lua")); err == nil {
		t.Fatal("want error writing below a file")
	}

	t.Setenv("REX_CONFIG", "/etc/rex/main.lua")
	if got, _ := defaultPluginPath(); got != "/etc/rex/sex.lua" {
		t.Fatalf("got %s", got)
	}
	t.Setenv("REX_CONFIG", "")
	t.Setenv("XDG_CONFIG_HOME", dir)
	if got, _ := defaultPluginPath(); got != filepath.Join(dir, "rex", "sex.lua") {
		t.Fatalf("got %s", got)
	}
	out, _, err = run(t, newDeps(), "plugin", "install")
	if err != nil || !strings.Contains(out, filepath.Join(dir, "rex", "sex.lua")) {
		t.Fatalf("out %q, err %v", out, err)
	}
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "/home/me")
	if got, _ := defaultPluginPath(); got != "/home/me/.config/rex/sex.lua" {
		t.Fatalf("got %s", got)
	}
}

func TestReloadOnUpgrade(t *testing.T) {
	d := newDeps()
	if _, _, err := run(t, d, "--plugin-version", "test", "list", "-r"); err != nil || d.reloads != 0 {
		t.Fatalf("same version must not reload: reloads %d, err %v", d.reloads, err)
	}
	if _, _, err := run(t, d, "list", "-r"); err != nil || d.reloads != 0 {
		t.Fatalf("no plugin version must not reload: reloads %d, err %v", d.reloads, err)
	}
	if _, _, err := run(t, d, "--plugin-version", "v0.0.1", "list", "-r"); err != nil || d.reloads != 1 {
		t.Fatalf("older plugin must reload once: reloads %d, err %v", d.reloads, err)
	}
	d.reloadErr = errors.New("refused")
	_, stderr, err := run(t, d, "--plugin-version", "v0.0.1", "list", "-r")
	if err != nil || !strings.Contains(stderr, "reloading the Rex config after an upgrade: refused") {
		t.Fatalf("a failed reload only warns: stderr %q, err %v", stderr, err)
	}
}

type fakePlugin struct {
	info rex.PluginInfo
	err  error
}

func (f fakePlugin) PluginInfo(context.Context) (rex.PluginInfo, error) { return f.info, f.err }

func TestCheckPlugin(t *testing.T) {
	tests := []struct {
		r    fakePlugin
		exe  string
		want string
	}{
		{fakePlugin{err: errors.New("no action")}, "/bin/sex", "warn  rex plugin: not loaded"},
		{fakePlugin{info: rex.PluginInfo{Error: "sex: /old/sex plugin --source failed"}}, "/bin/sex", "FAIL  rex plugin: sex: /old/sex plugin --source failed"},
		{fakePlugin{info: rex.PluginInfo{Version: "v0.1.0", Bin: "/bin/sex"}}, "/bin/sex", "warn  rex plugin: loaded version v0.1.0, this sex is v0.2.0"},
		{fakePlugin{info: rex.PluginInfo{Version: "v0.2.0", Bin: "/old/sex"}}, "/bin/sex", "warn  rex plugin: runs /old/sex, but this sex is /bin/sex"},
		{fakePlugin{info: rex.PluginInfo{Version: "v0.2.0", Bin: "/bin/sex"}}, "/bin/sex", "ok    rex plugin: v0.2.0 (/bin/sex)"},
	}
	for _, tt := range tests {
		var out bytes.Buffer
		checkPlugin(context.Background(), &checker{w: &out}, tt.r, "v0.2.0", tt.exe)
		if !strings.HasPrefix(out.String(), tt.want) {
			t.Fatalf("got %q, want prefix %q", out.String(), tt.want)
		}
	}
}

func TestRealAppBuilds(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "sesh.toml")
	if err := os.WriteFile(cfg, []byte("dir_length = 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	a, err := (&globals{seshConfig: cfg}).app(io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if a.Config.DirLength != 2 {
		t.Fatalf("config %+v", a.Config)
	}
	if err := os.WriteFile(cfg, []byte("dir_length = ["), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := (&globals{seshConfig: cfg}).app(io.Discard); err == nil {
		t.Fatal("want config error")
	}
}

func TestShellRun(t *testing.T) {
	var out bytes.Buffer
	dir := t.TempDir()
	if err := (shell{}).Run(context.Background(), dir, "echo hi; echo err >&2", &out); err != nil {
		t.Fatal(err)
	}
	if out.String() != "hi\nerr\n" {
		t.Fatalf("out %q", out.String())
	}
}

func TestFzfArgs(t *testing.T) {
	args := fzfArgs("/usr/bin/s ex", "client:1", "#161616", []string{"--height", "40%"})
	joined := strings.Join(args, "\n")
	for _, want := range []string{
		`ctrl-t:change-prompt(  )+reload('/usr/bin/s ex' --client 'client:1' list --icons -r)`,
		`ctrl-d:execute('/usr/bin/s ex' --client 'client:1' kill {})`,
		`'/usr/bin/s ex' --client 'client:1' preview {}`,
		"--preview-window\nright:55%:hidden",
		"ctrl-/:toggle-preview",
		",bg:#161616,gutter:#161616,bg+:#323232",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in\n%s", want, joined)
		}
	}
	if args[len(args)-2] != "--height" {
		t.Fatalf("extra args must come last: %v", args[len(args)-2:])
	}
	plain := strings.Join(fzfArgs("sex", "", "", nil), " ")
	if strings.Contains(plain, "--client") || strings.Contains(plain, "bg:") {
		t.Fatalf("no --client or background without values: %s", plain)
	}
}

func TestShade(t *testing.T) {
	for in, want := range map[string]string{
		"#161616": "#323232",
		"#ffffff": "#ebebeb",
		"#000000": "#1f1f1f",
		"oops":    "oops",
	} {
		if got := shade(in); got != want {
			t.Fatalf("shade(%s) = %s, want %s", in, got, want)
		}
	}
}

type fakeDoctorRex struct {
	sessionsErr error
	apps        []rex.AppClient
	appsErr     error
	perform     map[string]error
}

func (f fakeDoctorRex) Sessions(context.Context) ([]rex.Session, error) {
	return []rex.Session{{ID: "session:a"}}, f.sessionsErr
}

func (f fakeDoctorRex) AppClients(context.Context) ([]rex.AppClient, error) { return f.apps, f.appsErr }

func (f fakeDoctorRex) Perform(_ context.Context, clientID, _ string, _ any) error {
	return f.perform[clientID]
}

func TestDoctor(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "sesh.toml")
	if err := os.WriteFile(cfg, []byte("dir_length = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	found := func(bin string) (string, error) { return "/bin/" + bin, nil }
	r := fakeDoctorRex{
		apps: []rex.AppClient{{ID: "client:on", Label: "On"}, {ID: "client:off", Label: "Off"}, {ID: "client:odd", Label: "Odd"}},
		perform: map[string]error{
			"client:on":  &rex.APIError{Code: "not_supported"},
			"client:off": &rex.APIError{Code: "refused"},
			"client:odd": errors.New("timeout"),
		},
	}
	var out bytes.Buffer
	c := &checker{w: &out}
	runDoctor(context.Background(), c, &globals{seshConfig: cfg}, r, found)
	text := out.String()
	for _, want := range []string{"ok    rex server", "ok    On: Remote Control is on", "FAIL  Off: Remote Control is off", "warn  Odd: Remote Control unknown", "ok    sesh: /bin/sesh", "ok    sesh config"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in\n%s", want, text)
		}
	}
	if !c.failed {
		t.Fatal("want failed")
	}

	missing := func(bin string) (string, error) { return "", errors.New("not found") }
	out.Reset()
	c = &checker{w: &out}
	runDoctor(context.Background(), c, &globals{seshConfig: filepath.Join(t.TempDir(), "none.toml")},
		fakeDoctorRex{sessionsErr: errors.New("down"), appsErr: errors.New("down")}, missing)
	text = out.String()
	for _, want := range []string{"FAIL  rex server", "FAIL  rex apps", "FAIL  sesh: not found", "warn  fzf: not found", "warn  sesh config"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in\n%s", want, text)
		}
	}

	out.Reset()
	c = &checker{w: &out}
	runDoctor(context.Background(), c, &globals{seshConfig: cfg}, fakeDoctorRex{}, found)
	if !strings.Contains(out.String(), "warn  rex apps: none connected") || c.failed {
		t.Fatalf("out %s", out.String())
	}

	if err := os.WriteFile(cfg, []byte("dir_length = ["), 0o644); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	c = &checker{w: &out}
	runDoctor(context.Background(), c, &globals{seshConfig: cfg}, fakeDoctorRex{}, found)
	if !strings.Contains(out.String(), "FAIL  sesh config") {
		t.Fatalf("out %s", out.String())
	}
}

func TestActivityCommand(t *testing.T) {
	d := newDeps()
	d.rex.Activities = []rex.Activity{
		{SessionID: "session:a", Session: "api", WindowID: "window:a", Window: "build", BlockID: "block:a",
			Records: []rex.StatusRecord{{App: "codex", State: "blocked"}}},
		{SessionID: "session:b", Session: "web", WindowID: "window:b", Window: "main", BlockID: "block:b",
			Records: []rex.StatusRecord{{App: "claude", State: "idle"}}},
	}
	out, _, err := run(t, d, "activity")
	if err != nil || out != "● blocked  api › build  codex\n○ idle  web › main  claude\n" {
		t.Fatalf("out %q, err %v", out, err)
	}
	out, _, err = run(t, d, "activity", "-A", "--rows")
	if err != nil || !strings.HasPrefix(out, "block:a\t") || strings.Contains(out, "block:b") {
		t.Fatalf("out %q, err %v", out, err)
	}
	out, _, err = run(t, d, "activity", "-j")
	if err != nil || !strings.Contains(out, `"state": "blocked"`) {
		t.Fatalf("out %q, err %v", out, err)
	}
	d.rex.Screen = "screen\n"
	out, _, err = run(t, d, "activity", "preview", "block:a")
	if err != nil || out != "screen\n" {
		t.Fatalf("out %q, err %v", out, err)
	}
	d.rex.Activities = nil
	out, _, err = run(t, d, "activity", "-j")
	if err != nil || strings.TrimSpace(out) != "[]" {
		t.Fatalf("out %q, err %v", out, err)
	}
	d.rex.ActivityErr = errors.New("down")
	if _, _, err := run(t, d, "activity"); err == nil {
		t.Fatal("want error")
	}
}

func TestActivityFzfArgs(t *testing.T) {
	joined := strings.Join(activityFzfArgs("/bin/sex", "client:1", true, "", []string{"--x"}), "\n")
	for _, want := range []string{
		"--with-nth\n2..",
		"ctrl-r:reload('/bin/sex' --client 'client:1' activity --rows --active)",
		"'/bin/sex' --client 'client:1' activity preview {1}",
		" activity ",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in\n%s", want, joined)
		}
	}
	if !strings.HasSuffix(joined, "--x") {
		t.Fatal("extra args must come last")
	}
}
