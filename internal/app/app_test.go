package app

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rtuszik/sex/internal/app/apptest"
	"github.com/rtuszik/sex/internal/rex"
	"github.com/rtuszik/sex/internal/sesh"
	"github.com/rtuszik/sex/internal/state"
)

const appID = "client:app-0001"

var t0 = time.Date(2026, 10, 25, 21, 0, 0, 0, time.UTC)

type fixture struct {
	app    *App
	rex    *apptest.Rex
	sesh   *apptest.Sesh
	namer  *apptest.Namer
	zoxide *apptest.Zoxide
	store  *apptest.Store
	shell  *apptest.Shell
	stderr *bytes.Buffer
	env    map[string]string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{
		rex: &apptest.Rex{
			Apps: []rex.AppClient{{ID: appID, Label: "MacBook"}},
			Snaps: []rex.Snapshot{
				{ID: "session:a", Label: "api", Cwd: "/src/api", BlockID: "block:a", FirstWindowID: "window:a", ShownAt: t0.Add(-time.Minute)},
				{ID: "session:b", Label: "web", Cwd: "/src/web", BlockID: "block:b", ShownAt: t0},
				{ID: "session:c", Label: "notes", Cwd: "/notes"},
			},
		},
		sesh: &apptest.Sesh{Entries: []sesh.Entry{
			{Src: "config", Name: "web", Path: "/src/web"},
			{Src: "config", Name: "dotfiles", Path: "/home/me/.config", Alias: "dot", StartupCommand: "nvim {}", PreviewCommand: "bat {}",
				WindowConfigs: []sesh.WindowConfig{{Name: "git", StartupScript: "lazygit"}, {Name: "logs", Path: "~/logs"}}},
			{Src: "zoxide", Name: "~/src/api", Path: "/src/api", Score: 10},
			{Src: "zoxide", Name: "~/src/cli", Path: "/home/me/src/cli", Score: 5},
		}},
		namer:  &apptest.Namer{Names: map[string]string{"/home/me/src/cli": "cli"}, Roots: map[string]string{}},
		zoxide: &apptest.Zoxide{Queries: map[string]string{}},
		store:  &apptest.Store{State: state.State{Sessions: map[string]state.Record{}}},
		shell:  &apptest.Shell{},
		stderr: &bytes.Buffer{},
		env:    map[string]string{},
	}
	f.app = &App{
		Rex: f.rex, Sesh: f.sesh, Namer: f.namer, Zoxide: f.zoxide, Store: f.store, Shell: f.shell,
		Home: "/home/me", Cwd: "/home/me/work",
		Getenv: func(k string) string { return f.env[k] },
		Stderr: f.stderr,
	}
	return f
}

func names(items []Item) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.Source+":"+it.Name)
	}
	return out
}

func assertEqual[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func assertSlice(t *testing.T, got, want []string) {
	t.Helper()
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestResolveClient(t *testing.T) {
	ctx := context.Background()
	two := []rex.AppClient{{ID: "client:aaaa1111", Label: "Mac"}, {ID: "client:bbbb2222", Label: "iPad"}}
	tests := []struct {
		name     string
		explicit string
		env      string
		apps     []rex.AppClient
		appsErr  error
		block    bool
		owner    string
		ownerErr error
		want     string
		wantErr  string
	}{
		{name: "explicit id", explicit: "client:x", want: "client:x"},
		{name: "env id", env: "client:y", want: "client:y"},
		{name: "only app", apps: two[:1], want: "client:aaaa1111"},
		{name: "label", explicit: "ipad", apps: two, want: "client:bbbb2222"},
		{name: "id suffix", explicit: "2222", apps: two, want: "client:bbbb2222"},
		{name: "no match", explicit: "tv", apps: two, wantErr: "no Rex app matches"},
		{name: "no apps", wantErr: ErrNoApp.Error()},
		{name: "several", apps: two, wantErr: "several Rex apps"},
		{name: "several, terminal owner", apps: two, block: true, owner: "client:bbbb2222", want: "client:bbbb2222"},
		{name: "several, owner not an app", apps: two, block: true, owner: "client:web", wantErr: "several Rex apps"},
		{name: "several, owner error", apps: two, block: true, ownerErr: errors.New("gone"), wantErr: "several Rex apps"},
		{name: "rex error", appsErr: errors.New("down"), wantErr: "down"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			f.rex.Apps, f.rex.AppsErr = tt.apps, tt.appsErr
			f.env["REX_CLIENT"] = tt.env
			f.rex.Owner, f.rex.OwnerErr = tt.owner, tt.ownerErr
			if tt.block {
				f.env["REX_SESSION"], f.env["REX_BLOCK"] = "session:me", "block:me"
			}
			got, err := f.app.ResolveClient(ctx, tt.explicit)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			assertEqual(t, got, tt.want)
		})
	}
}

func TestListAllSourcesDeduplicated(t *testing.T) {
	f := newFixture(t)
	f.store.State.Sessions["session:c"] = state.Record{Path: "/home/me/notes"}
	items, err := f.app.List(context.Background(), ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	// web config hidden by name, ~/src/api zoxide hidden by path.
	assertSlice(t, names(items), []string{"rex:web", "rex:api", "rex:notes", "config:dotfiles", "zoxide:~/src/cli"})
	assertEqual(t, items[0].Current, true)
	assertEqual(t, items[2].Path, "/home/me/notes")
	assertEqual(t, items[2].ShownAt == nil, true)
	assertSlice(t, f.rex.SnapClient, []string{appID})
	assertSlice(t, f.sesh.Calls(), []string{"config=true zoxide=true live=false"})
}

func TestListSourceFlags(t *testing.T) {
	tests := []struct {
		opts ListOptions
		want []string
	}{
		{ListOptions{Rex: true}, []string{"rex:web", "rex:api", "rex:notes"}},
		{ListOptions{Config: true}, []string{"config:dotfiles"}},
		{ListOptions{Zoxide: true}, []string{"zoxide:~/src/cli"}},
		{ListOptions{Rex: true, HideCurrent: true}, []string{"rex:api", "rex:notes"}},
	}
	for _, tt := range tests {
		f := newFixture(t)
		items, err := f.app.List(context.Background(), tt.opts)
		if err != nil {
			t.Fatal(err)
		}
		assertSlice(t, names(items), tt.want)
	}
}

func TestListCurrentFallsBackToEnv(t *testing.T) {
	f := newFixture(t)
	for i := range f.rex.Snaps {
		f.rex.Snaps[i].ShownAt = time.Time{}
	}
	f.env["REX_SESSION"] = "session:c"
	items, err := f.app.List(context.Background(), ListOptions{Rex: true, HideCurrent: true})
	if err != nil {
		t.Fatal(err)
	}
	assertSlice(t, names(items), []string{"rex:api", "rex:web"})
}

func TestListBlacklist(t *testing.T) {
	f := newFixture(t)
	f.app.Config.Blacklist = []string{"^no", "[invalid"}
	items, err := f.app.List(context.Background(), ListOptions{Rex: true})
	if err != nil {
		t.Fatal(err)
	}
	assertSlice(t, names(items), []string{"rex:web", "rex:api"})
}

func TestListDegradesWhenOneSourceFails(t *testing.T) {
	f := newFixture(t)
	f.rex.SnapErr = errors.New("rex down")
	items, err := f.app.List(context.Background(), ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	assertSlice(t, names(items), []string{"config:web", "config:dotfiles", "zoxide:~/src/api", "zoxide:~/src/cli"})
	if !strings.Contains(f.stderr.String(), "rex down") {
		t.Fatalf("missing warning: %q", f.stderr.String())
	}

	f = newFixture(t)
	f.sesh.Err = errors.New("sesh missing")
	items, err = f.app.List(context.Background(), ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	assertEqual(t, len(items), 3)
}

func TestListFailsWhenOnlySourceFails(t *testing.T) {
	f := newFixture(t)
	f.rex.SnapErr = errors.New("rex down")
	if _, err := f.app.List(context.Background(), ListOptions{Rex: true}); err == nil {
		t.Fatal("want error")
	}
	f = newFixture(t)
	f.sesh.Err = errors.New("sesh missing")
	if _, err := f.app.List(context.Background(), ListOptions{Zoxide: true}); err == nil {
		t.Fatal("want error")
	}
	f = newFixture(t)
	if _, err := f.app.List(context.Background(), ListOptions{Client: "tv"}); err == nil {
		t.Fatal("want error for unknown explicit client")
	}
}

func TestFormatAndStripIcon(t *testing.T) {
	it := Item{Source: SourceRex, Name: "api"}
	assertEqual(t, Format(it, false, true), "api")
	assertEqual(t, Format(it, true, false), " api")
	colored := Format(it, true, true)
	assertEqual(t, StripIcon(colored), "api")
	assertEqual(t, colored, "\x1b[34m\x1b[39m \x1b[32mapi\x1b[0m")
	it.Current = true
	assertEqual(t, Format(it, true, true), "\x1b[34m\x1b[39m \x1b[1;32mapi\x1b[0m")
	assertEqual(t, StripIcon(Format(it, true, true)), "api")
	assertEqual(t, Format(Item{Source: SourceConfig, Name: "cfg"}, true, true), "\x1b[90m\x1b[39m cfg")
	assertEqual(t, Format(Item{Source: "other", Name: "x"}, true, true), "x")
	assertEqual(t, Format(Item{Source: SourceConfig, Name: "x", Icon: "★"}, true, false), "★ x")

	for in, want := range map[string]string{
		" api":                   "api",
		"★ my notes":             "my notes",
		"nvim config":            "nvim config",
		"~/src/cli":              "~/src/cli",
		"  ":                     "",
		"\x1b[36m\x1b[39m ~/x y": "~/x y",
	} {
		assertEqual(t, StripIcon(in), want)
	}
}

func TestConnectRunningSession(t *testing.T) {
	f := newFixture(t)
	id, err := f.app.Connect(context.Background(), " API", ConnectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	assertEqual(t, id, "session:a")
	assertSlice(t, f.rex.Selected, []string{appID + "|session:a"})
	assertEqual(t, len(f.rex.Created), 0)
	assertEqual(t, len(f.sesh.Calls()), 0)
}

func TestConnectBySessionID(t *testing.T) {
	f := newFixture(t)
	id, err := f.app.Connect(context.Background(), "session:c", ConnectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	assertEqual(t, id, "session:c")
}

func TestConnectCreatesConfigSession(t *testing.T) {
	f := newFixture(t)
	f.rex.CreateRes = rex.Created{SessionID: "session:new", WindowIDs: []string{"window:1", "window:2", "window:3"}}
	id, err := f.app.Connect(context.Background(), "dot", ConnectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	assertEqual(t, id, "session:new")
	want := rex.CreateOpts{Label: "dotfiles", Windows: []rex.Window{
		{Cwd: "/home/me/.config", Input: "nvim '/home/me/.config'\n"},
		{Label: "git", Cwd: "/home/me/.config", Input: "lazygit\n"},
		{Label: "logs", Cwd: "/home/me/logs"},
	}}
	if len(f.rex.Created) != 1 || !equalCreate(f.rex.Created[0], want) {
		t.Fatalf("created %+v, want %+v", f.rex.Created, want)
	}
	assertSlice(t, f.sesh.Calls(), []string{"config=true zoxide=true live=true"})
	assertSlice(t, f.rex.Focused, []string{"session:new|window:1"})
	assertSlice(t, f.rex.Selected, []string{appID + "|session:new"})
	assertSlice(t, f.zoxide.Added, []string{"/home/me/.config"})
	assertEqual(t, f.store.Saved.Sessions["session:new"].Path, "/home/me/.config")
	assertEqual(t, f.store.Live["session:a"] && f.store.Live["session:new"], true)
}

func equalCreate(a, b rex.CreateOpts) bool {
	if a.Label != b.Label || len(a.Windows) != len(b.Windows) {
		return false
	}
	for i := range a.Windows {
		if a.Windows[i] != b.Windows[i] {
			return false
		}
	}
	return true
}

func TestConnectCommandOverridesStartup(t *testing.T) {
	f := newFixture(t)
	f.rex.CreateRes = rex.Created{SessionID: "session:new", WindowIDs: []string{"window:1"}}
	if _, err := f.app.Connect(context.Background(), "dotfiles", ConnectOptions{Command: "htop"}); err != nil {
		t.Fatal(err)
	}
	assertEqual(t, f.rex.Created[0].Windows[0].Input, "htop\n")
}

func TestConnectDisabledStartup(t *testing.T) {
	f := newFixture(t)
	f.sesh.Entries[1].DisableStartupCommand = true
	f.rex.CreateRes = rex.Created{SessionID: "session:new", WindowIDs: []string{"window:1"}}
	if _, err := f.app.Connect(context.Background(), "dotfiles", ConnectOptions{}); err != nil {
		t.Fatal(err)
	}
	assertEqual(t, f.rex.Created[0].Windows[0].Input, "")
	assertEqual(t, len(f.rex.Focused), 0)
}

func TestConnectZoxideEntryNamedByNamer(t *testing.T) {
	f := newFixture(t)
	f.rex.CreateRes = rex.Created{SessionID: "session:new", WindowIDs: []string{"window:1"}}
	if _, err := f.app.Connect(context.Background(), " ~/src/cli", ConnectOptions{}); err != nil {
		t.Fatal(err)
	}
	assertEqual(t, f.rex.Created[0].Label, "cli")
	assertEqual(t, f.rex.Created[0].Windows[0].Cwd, "/home/me/src/cli")
}

func TestConnectPathReusesRunningSession(t *testing.T) {
	f := newFixture(t)
	f.namer.Names["/src/web"] = "web-renamed"
	id, err := f.app.Connect(context.Background(), "/src/web", ConnectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	// Config "web" matches the path first and maps to the running session by name.
	assertEqual(t, id, "session:b")

	f = newFixture(t)
	f.sesh.Entries = nil
	dir := t.TempDir()
	f.namer.Names[dir] = "api"
	id, err = f.app.Connect(context.Background(), dir, ConnectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	assertEqual(t, id, "session:a")

	f = newFixture(t)
	f.sesh.Entries = nil
	f.rex.Snaps[0].Cwd = dir
	f.namer.Names[dir] = "fresh"
	id, err = f.app.Connect(context.Background(), dir, ConnectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	assertEqual(t, id, "session:a")
}

func TestConnectDirectoryEnrichedBySesh(t *testing.T) {
	f := newFixture(t)
	dir := t.TempDir()
	f.namer.Names[dir] = "proj"
	f.rex.CreateRes = rex.Created{SessionID: "session:new", WindowIDs: []string{"window:1"}}
	// sesh only lists the directory after zoxide add, with its wildcard startup.
	f.sesh.Entries = append(f.sesh.Entries, sesh.Entry{Src: "zoxide", Name: "~/proj", Path: dir, StartupCommand: "make dev"})
	f.sesh.Entries[len(f.sesh.Entries)-1].Path = dir
	if _, err := f.app.Connect(context.Background(), dir, ConnectOptions{}); err != nil {
		t.Fatal(err)
	}
	assertEqual(t, f.rex.Created[0].Windows[0].Input, "make dev\n")
}

func TestConnectRelativeDirectory(t *testing.T) {
	f := newFixture(t)
	f.sesh.Entries = nil
	base := t.TempDir()
	if err := os.Mkdir(filepath.Join(base, "rel"), 0o755); err != nil {
		t.Fatal(err)
	}
	f.app.Cwd = base
	f.namer.Names[filepath.Join(base, "rel")] = "rel"
	f.rex.CreateRes = rex.Created{SessionID: "session:new", WindowIDs: []string{"window:1"}}
	if _, err := f.app.Connect(context.Background(), "rel", ConnectOptions{}); err != nil {
		t.Fatal(err)
	}
	assertEqual(t, f.rex.Created[0].Windows[0].Cwd, filepath.Join(base, "rel"))
	assertSlice(t, f.zoxide.Added, []string{filepath.Join(base, "rel"), filepath.Join(base, "rel")})
}

func TestConnectZoxideQuery(t *testing.T) {
	f := newFixture(t)
	f.sesh.Entries = nil
	f.zoxide.Queries["cl"] = "/home/me/src/cli"
	f.rex.CreateRes = rex.Created{SessionID: "session:new", WindowIDs: []string{"window:1"}}
	if _, err := f.app.Connect(context.Background(), "cl", ConnectOptions{}); err != nil {
		t.Fatal(err)
	}
	assertEqual(t, f.rex.Created[0].Label, "cli")
}

func TestConnectErrors(t *testing.T) {
	ctx := context.Background()

	f := newFixture(t)
	if _, err := f.app.Connect(ctx, "nothing-here", ConnectOptions{}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}

	f = newFixture(t)
	if _, err := f.app.Connect(ctx, "  ", ConnectOptions{}); err == nil {
		t.Fatal("want error for empty name")
	}

	f = newFixture(t)
	f.rex.Apps = nil
	if _, err := f.app.Connect(ctx, "api", ConnectOptions{}); !errors.Is(err, ErrNoApp) {
		t.Fatalf("err = %v, want ErrNoApp", err)
	}

	f = newFixture(t)
	f.rex.SnapErr = errors.New("rex down")
	if _, err := f.app.Connect(ctx, "api", ConnectOptions{}); err == nil {
		t.Fatal("want snapshot error")
	}

	f = newFixture(t)
	f.rex.CreateErr = errors.New("create failed")
	if _, err := f.app.Connect(ctx, "dotfiles", ConnectOptions{}); err == nil || !strings.Contains(err.Error(), "create failed") {
		t.Fatalf("err = %v", err)
	}

	f = newFixture(t)
	f.namer.Names = map[string]string{}
	if _, err := f.app.Connect(ctx, " ~/src/cli", ConnectOptions{}); err == nil {
		t.Fatal("want namer error")
	}

	f = newFixture(t)
	f.rex.SelectErr = &rex.APIError{Code: "refused", Msg: "rex: refused"}
	_, err := f.app.Connect(ctx, "api", ConnectOptions{})
	if err == nil || !strings.Contains(err.Error(), "Remote Control") {
		t.Fatalf("err = %v, want Remote Control hint", err)
	}

	f = newFixture(t)
	f.rex.SelectErr = errors.New("other")
	if _, err := f.app.Connect(ctx, "api", ConnectOptions{}); err == nil || err.Error() != "other" {
		t.Fatalf("err = %v", err)
	}
}

func TestConnectBackgroundWithoutApp(t *testing.T) {
	f := newFixture(t)
	f.rex.Apps = nil
	f.rex.CreateRes = rex.Created{SessionID: "session:new", WindowIDs: []string{"window:1"}}
	id, err := f.app.Connect(context.Background(), "dotfiles", ConnectOptions{Background: true})
	if err != nil {
		t.Fatal(err)
	}
	assertEqual(t, id, "session:new")
	assertEqual(t, len(f.rex.Selected), 0)
}

func TestConnectWarnsOnSideEffectFailures(t *testing.T) {
	f := newFixture(t)
	f.rex.CreateRes = rex.Created{SessionID: "session:new", WindowIDs: []string{"window:1", "window:2"}}
	f.rex.FocusErr = errors.New("focus failed")
	f.zoxide.AddErr = errors.New("zoxide failed")
	f.store.SaveErr = errors.New("save failed")
	if _, err := f.app.Connect(context.Background(), "dotfiles", ConnectOptions{}); err != nil {
		t.Fatal(err)
	}
	for _, msg := range []string{"focus failed", "zoxide failed", "save failed"} {
		if !strings.Contains(f.stderr.String(), msg) {
			t.Fatalf("stderr %q missing %q", f.stderr.String(), msg)
		}
	}

	f = newFixture(t)
	f.rex.CreateRes = rex.Created{SessionID: "session:new", WindowIDs: []string{"window:1"}}
	f.store.LoadErr = errors.New("load failed")
	if _, err := f.app.Connect(context.Background(), "dotfiles", ConnectOptions{}); err != nil {
		t.Fatal(err)
	}
	assertEqual(t, f.store.Saved == nil, true)
}

func TestLast(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	id, err := f.app.SwitchLast(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	assertEqual(t, id, "session:a")
	assertSlice(t, f.rex.Selected, []string{appID + "|session:a"})

	f = newFixture(t)
	f.rex.Snaps[0].ShownAt = time.Time{}
	if _, err := f.app.SwitchLast(ctx, ""); err == nil {
		t.Fatal("want no previous session")
	}

	f = newFixture(t)
	f.rex.Apps = nil
	if _, _, err := f.app.Last(ctx, ""); err == nil {
		t.Fatal("want client error")
	}

	f = newFixture(t)
	f.rex.SnapErr = errors.New("down")
	if _, _, err := f.app.Last(ctx, ""); err == nil {
		t.Fatal("want snapshot error")
	}
}

func TestKill(t *testing.T) {
	f := newFixture(t)
	name, err := f.app.Kill(context.Background(), " notes")
	if err != nil {
		t.Fatal(err)
	}
	assertEqual(t, name, "notes")
	assertSlice(t, f.rex.Destroyed, []string{"session:c"})

	if _, err := f.app.Kill(context.Background(), "dotfiles"); err == nil {
		t.Fatal("want error for a session that is not running")
	}
	f.rex.SnapErr = errors.New("down")
	if _, err := f.app.Kill(context.Background(), "api"); err == nil {
		t.Fatal("want snapshot error")
	}
}

func TestPreviewRunningSession(t *testing.T) {
	f := newFixture(t)
	f.env["FZF_PREVIEW_LINES"] = "2"
	f.rex.Screen = "one\ntwo\nthree\n\n"
	var out bytes.Buffer
	if err := f.app.Preview(context.Background(), " api", &out); err != nil {
		t.Fatal(err)
	}
	assertEqual(t, out.String(), "two\nthree\n")
	assertSlice(t, f.rex.Captured, []string{"session:a|block:a"})

	f.rex.Snaps[0].BlockID = ""
	if err := f.app.Preview(context.Background(), "api", &out); err == nil {
		t.Fatal("want error without block")
	}
	f.rex.Snaps[0].BlockID = "block:a"
	f.rex.CaptureErr = errors.New("capture failed")
	if err := f.app.Preview(context.Background(), "api", &out); err == nil {
		t.Fatal("want capture error")
	}
}

func TestPreviewEntryAndDirectory(t *testing.T) {
	f := newFixture(t)
	f.shell.Output = "preview!"
	var out bytes.Buffer
	if err := f.app.Preview(context.Background(), " dotfiles", &out); err != nil {
		t.Fatal(err)
	}
	assertEqual(t, f.shell.Command, "bat '/home/me/.config'")
	assertEqual(t, f.shell.Dir, "/home/me/.config")
	assertEqual(t, out.String(), "preview!")

	dir := t.TempDir()
	f.rex.SnapErr = errors.New("down")
	if err := f.app.Preview(context.Background(), dir, &out); err != nil {
		t.Fatal(err)
	}
	assertEqual(t, f.shell.Command, "ls -la "+shellQuote(dir))

	if err := f.app.Preview(context.Background(), "nothing-here", &out); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v", err)
	}
}

func TestRoot(t *testing.T) {
	f := newFixture(t)
	f.namer.Roots["/src/web"] = "web-root"
	got, err := f.app.Root(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	assertEqual(t, got, "web-root")

	f.rex.SnapErr = errors.New("down")
	f.namer.Roots["/home/me/work"] = "work"
	got, err = f.app.Root(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	assertEqual(t, got, "work")
}

func TestExpandAndHelpers(t *testing.T) {
	f := newFixture(t)
	assertEqual(t, f.app.expand("~"), "/home/me")
	assertEqual(t, f.app.expand("~/x"), "/home/me/x")
	assertEqual(t, f.app.expand("rel"), "/home/me/work/rel")
	assertEqual(t, f.app.expand("/a/../b"), "/b")
	assertEqual(t, samePath("/a/", "/a"), true)
	assertEqual(t, samePath("", ""), false)
	assertEqual(t, inputLine("  "), "")
	assertEqual(t, tail("a\nb\n", 5), "a\nb\n")
	assertEqual(t, shellQuote("it's"), `'it'\''s'`)
	f.env["FZF_PREVIEW_LINES"] = "x"
	assertEqual(t, f.app.previewLines(), defaultPreviewLines)
	var nilEnv App
	assertEqual(t, nilEnv.env("X"), "")
	nilEnv.warnf("dropped")
}

func TestResolveClientAsksOwnerOfCurrentTerminal(t *testing.T) {
	f := newFixture(t)
	f.rex.Apps = append(f.rex.Apps, rex.AppClient{ID: "client:phone", Label: "iPhone"})
	f.rex.Owner = "client:phone"
	f.env["REX_SESSION"], f.env["REX_BLOCK"] = "session:me", "block:me"
	got, err := f.app.ResolveClient(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	assertEqual(t, got, "client:phone")
	assertSlice(t, f.rex.OwnerQueries, []string{"session:me|block:me"})
}

func TestConnectBackgroundWithoutServer(t *testing.T) {
	f := newFixture(t)
	f.rex.Apps = nil
	f.rex.SnapErr = &rex.APIError{Msg: "rex: no Rex server is running", Exit: 4}
	f.rex.CreateRes = rex.Created{SessionID: "session:new", WindowIDs: []string{"window:1"}}
	id, err := f.app.Connect(context.Background(), "dotfiles", ConnectOptions{Background: true})
	if err != nil {
		t.Fatal(err)
	}
	assertEqual(t, id, "session:new")

	f = newFixture(t)
	f.rex.SnapErr = &rex.APIError{Msg: "rex: no Rex server is running", Exit: 4}
	if _, err := f.app.Connect(context.Background(), "dotfiles", ConnectOptions{}); err == nil {
		t.Fatal("a switching connect needs a running server")
	}
}

func TestStreamEmitsRexSessionsFirst(t *testing.T) {
	f := newFixture(t)
	var batches [][]string
	err := f.app.Stream(context.Background(), ListOptions{}, func(batch []Item) error {
		batches = append(batches, names(batch))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	assertEqual(t, len(batches), 2)
	assertSlice(t, batches[0], []string{"rex:web", "rex:api", "rex:notes"})
	assertSlice(t, batches[1], []string{"config:dotfiles", "zoxide:~/src/cli"})

	stop := errors.New("reader gone")
	err = f.app.Stream(context.Background(), ListOptions{}, func([]Item) error { return stop })
	if !errors.Is(err, stop) {
		t.Fatalf("err = %v, want the emit error", err)
	}

	f.rex.Snaps = nil
	calls := 0
	if err := f.app.Stream(context.Background(), ListOptions{Rex: true}, func([]Item) error { calls++; return nil }); err != nil || calls != 0 {
		t.Fatalf("empty source must not emit: calls %d, err %v", calls, err)
	}
}
