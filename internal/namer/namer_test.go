package namer

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/rtuszik/sex/internal/execx"
	"github.com/rtuszik/sex/internal/sesh"
)

type fakeGit struct {
	worktrees map[string]string
	toplevels map[string]string
}

func (g fakeGit) WorktreeList(_ context.Context, path string) (string, error) {
	if out, ok := g.worktrees[path]; ok {
		return out, nil
	}
	return "", errors.New("not a git repository")
}

func (g fakeGit) ShowTopLevel(_ context.Context, path string) (string, error) {
	if out, ok := g.toplevels[path]; ok {
		return out, nil
	}
	return "", errors.New("not a git repository")
}

func newNamer(git fakeGit, cfg sesh.Config) *Namer {
	n := New(git, cfg, "/home/me")
	n.evalSymlinks = func(p string) (string, error) { return p, nil }
	return n
}

const mainRepo = "worktree /home/me/src/sesh\nHEAD abc\nbranch refs/heads/main\n\nworktree /home/me/src/sesh-feat\nHEAD def\n"

func TestName(t *testing.T) {
	git := fakeGit{
		worktrees: map[string]string{
			"/home/me/src/sesh":          mainRepo,
			"/home/me/src/sesh/cmd":      mainRepo,
			"/home/me/src/sesh-feat":     mainRepo,
			"/home/me/src/bare/main":     "worktree /home/me/src/bare/.bare\nbare\n\nworktree /home/me/src/bare/main\n",
			"/home/me/src/gitbare/x":     "worktree /home/me/src/gitbare/.git\nbare\n",
			"/home/me/src/odd":           "worktree /home/me/src/odd/repo.git\nbare\n",
			"/home/me/src/github.com/me": "worktree /home/me/src/github.com/me\n",
		},
		toplevels: map[string]string{"/home/me/src/sesh-feat": "/home/me/src/sesh-feat"},
	}
	tests := []struct {
		name string
		cfg  sesh.Config
		path string
		want string
	}{
		{"repo root", sesh.Config{}, "/home/me/src/sesh", "sesh"},
		{"nested dir", sesh.Config{}, "/home/me/src/sesh/cmd", "sesh/cmd"},
		{"linked worktree", sesh.Config{}, "/home/me/src/sesh-feat", "sesh-feat"},
		{"worktree root mode", sesh.Config{GitNamerUseWorktreeRoot: true}, "/home/me/src/sesh-feat", "sesh-feat"},
		{"bare .bare", sesh.Config{}, "/home/me/src/bare/main", "bare/main"},
		{"bare .git", sesh.Config{}, "/home/me/src/gitbare/x", "gitbare/x"},
		{"bare other", sesh.Config{}, "/home/me/src/odd", "repo.git/home/me/src/odd"},
		{"git dir length", sesh.Config{GitDirLength: 2}, "/home/me/src/github.com/me", "github.com/me"},
		{"dir", sesh.Config{}, "/home/me/notes", "notes"},
		{"dir length", sesh.Config{DirLength: 2}, "/home/me/notes", "me/notes"},
		{"worktree root mode outside git", sesh.Config{GitNamerUseWorktreeRoot: true}, "/home/me/notes", "notes"},
		{"literal substitution", sesh.Config{NameSubstitutions: []sesh.NameSubstitution{{Find: "~/src/", Replace: ""}}}, "/home/me/src/sesh", "sesh"},
		{"regex substitution", sesh.Config{NameSubstitutions: []sesh.NameSubstitution{{Find: `.*/(\w+)$`, Replace: "ws-$1", Regex: true}}}, "/home/me/notes", "ws-notes"},
		{"no-op substitution", sesh.Config{NameSubstitutions: []sesh.NameSubstitution{{Find: "zzz", Replace: "y"}}}, "/home/me/notes", "notes"},
		{"blank substitution falls through", sesh.Config{NameSubstitutions: []sesh.NameSubstitution{{Find: "~/notes", Replace: "  "}}}, "/home/me/notes", "notes"},
		{"home itself", sesh.Config{NameSubstitutions: []sesh.NameSubstitution{{Find: "~", Replace: "home"}}}, "/home/me", "home"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newNamer(git, tt.cfg).Name(context.Background(), tt.path)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRootName(t *testing.T) {
	git := fakeGit{
		worktrees: map[string]string{
			"/home/me/src/sesh/cmd":  mainRepo,
			"/home/me/src/sesh-feat": mainRepo,
			"/home/me/src/nested/x":  "worktree /home/me/src/nested\n",
		},
		toplevels: map[string]string{
			"/home/me/src/sesh/cmd":  "/home/me/src/sesh",
			"/home/me/src/sesh-feat": "/home/me/src/sesh-feat",
		},
	}
	tests := []struct {
		cfg  sesh.Config
		path string
		want string
	}{
		{sesh.Config{}, "/home/me/src/sesh/cmd", "sesh"},
		{sesh.Config{}, "/home/me/src/sesh-feat", "sesh-feat"},
		{sesh.Config{}, "/home/me/src/nested/x", "nested"},
		{sesh.Config{GitNamerUseWorktreeRoot: true}, "/home/me/src/sesh/cmd", "sesh"},
		{sesh.Config{GitNamerUseWorktreeRoot: true}, "/home/me/plain", "plain"},
		{sesh.Config{}, "/home/me/plain", "plain"},
	}
	for _, tt := range tests {
		got, err := newNamer(git, tt.cfg).RootName(context.Background(), tt.path)
		if err != nil {
			t.Fatal(err)
		}
		if got != tt.want {
			t.Fatalf("%s: got %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestNameErrors(t *testing.T) {
	n := newNamer(fakeGit{}, sesh.Config{})
	n.evalSymlinks = func(string) (string, error) { return "", errors.New("no such file") }
	if _, err := n.Name(context.Background(), "/x"); err == nil {
		t.Fatal("want symlink error")
	}
	n = newNamer(fakeGit{}, sesh.Config{})
	if _, err := n.Name(context.Background(), ""); err == nil {
		t.Fatal("want error for empty path")
	}
	n = New(fakeGit{}, sesh.Config{}, "/home/me")
	if _, err := n.Name(context.Background(), "/definitely/missing/path"); err == nil {
		t.Fatal("want error from real EvalSymlinks")
	}
}

func TestValidNameTruncates(t *testing.T) {
	long := strings.Repeat("é", maxLabel+10)
	if got := validName("  " + long + "  "); len([]rune(got)) != maxLabel {
		t.Fatalf("len %d", len([]rune(got)))
	}
	if got := validName(" a.b:c "); got != "a.b:c" {
		t.Fatalf("got %q", got)
	}
}

func TestLastNComponents(t *testing.T) {
	tests := []struct {
		path string
		n    int
		want string
	}{
		{"", 2, ""},
		{"/a/b/c", 1, "c"},
		{"/a/b/c", 2, "b/c"},
		{"/a/b/c", 5, "a/b/c"},
		{"/", 2, "/"},
		{"a", 3, "a"},
	}
	for _, tt := range tests {
		if got := LastNComponents(tt.path, tt.n); got != tt.want {
			t.Fatalf("%q,%d: got %q, want %q", tt.path, tt.n, got, tt.want)
		}
	}
}

func TestApplySubstitutions(t *testing.T) {
	rules := []sesh.NameSubstitution{
		{Find: ""},
		{Find: "[bad", Regex: true},
		{Find: "a", Replace: "b"},
		{Find: `b(\d)`, Replace: "c$1", Regex: true},
	}
	if got := ApplySubstitutions("a1 a2", rules); got != "c1 c2" {
		t.Fatalf("got %q", got)
	}
}

func TestShortenHome(t *testing.T) {
	n := New(fakeGit{}, sesh.Config{}, "")
	if got := n.shortenHome("/x"); got != "/x" {
		t.Fatalf("got %q", got)
	}
	n = New(fakeGit{}, sesh.Config{}, "/home/me")
	if got := n.shortenHome("/home/meow"); got != "/home/meow" {
		t.Fatalf("got %q", got)
	}
}

func TestExecGit(t *testing.T) {
	var got []string
	run := execx.RunnerFunc(func(_ context.Context, cmd execx.Cmd) ([]byte, error) {
		got = append(got, cmd.Name+" "+strings.Join(cmd.Args, " "))
		return []byte(" out \n"), nil
	})
	g := ExecGit{Run: run}
	if out, _ := g.WorktreeList(context.Background(), "/r"); out != "out" {
		t.Fatalf("out %q", out)
	}
	if out, _ := g.ShowTopLevel(context.Background(), "/r"); out != "out" {
		t.Fatalf("out %q", out)
	}
	want := "git -C /r worktree list --porcelain,git -C /r rev-parse --show-toplevel"
	if strings.Join(got, ",") != want {
		t.Fatalf("got %v", got)
	}
}
