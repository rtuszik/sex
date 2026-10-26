package sesh

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rtuszik/sex/internal/execx"
)

func TestListArgs(t *testing.T) {
	tests := []struct {
		configPath string
		q          Query
		want       string
	}{
		{"", Query{Config: true, Zoxide: true}, "list -j -c -z"},
		{"", Query{Config: true, Live: true}, "list -j --no-cache -c"},
		{"/x/sesh.toml", Query{Zoxide: true, Live: true}, "-C /x/sesh.toml list -j --no-cache -z"},
	}
	for _, tt := range tests {
		var got string
		run := execx.RunnerFunc(func(_ context.Context, cmd execx.Cmd) ([]byte, error) {
			got = cmd.Name + " " + strings.Join(cmd.Args, " ")
			return []byte(`[{"Src":"zoxide","Name":"~/a","Path":"/a","Score":2.5,"WindowConfigs":[{"Name":"git","StartupScript":"lazygit","Path":"/a"}]}]`), nil
		})
		entries, err := New(run, "", tt.configPath).List(context.Background(), tt.q)
		if err != nil {
			t.Fatal(err)
		}
		if got != "sesh "+tt.want {
			t.Fatalf("got %q, want %q", got, "sesh "+tt.want)
		}
		if len(entries) != 1 || entries[0].Score != 2.5 || entries[0].WindowConfigs[0].StartupScript != "lazygit" {
			t.Fatalf("entries %+v", entries)
		}
	}
}

func TestListNothingRequested(t *testing.T) {
	run := execx.RunnerFunc(func(context.Context, execx.Cmd) ([]byte, error) {
		t.Fatal("sesh must not run")
		return nil, nil
	})
	entries, err := New(run, "sesh", "").List(context.Background(), Query{Live: true})
	if err != nil || entries != nil {
		t.Fatalf("entries %v, err %v", entries, err)
	}
}

func TestListErrors(t *testing.T) {
	fail := execx.RunnerFunc(func(context.Context, execx.Cmd) ([]byte, error) { return nil, errors.New("boom") })
	if _, err := New(fail, "", "").List(context.Background(), Query{Config: true, Zoxide: true}); err == nil || !strings.Contains(err.Error(), "sesh list: boom") {
		t.Fatalf("err = %v", err)
	}
	bad := execx.RunnerFunc(func(context.Context, execx.Cmd) ([]byte, error) { return []byte("nope"), nil })
	if _, err := New(bad, "", "").List(context.Background(), Query{Config: true, Zoxide: true}); err == nil {
		t.Fatal("want decode error")
	}
}

func TestConfigPath(t *testing.T) {
	if got, _ := ConfigPath("/explicit.toml"); got != "/explicit.toml" {
		t.Fatalf("got %s", got)
	}
	t.Setenv("XDG_CONFIG_HOME", "/xdg")
	if got, _ := ConfigPath(""); got != "/xdg/sesh/sesh.toml" {
		t.Fatalf("got %s", got)
	}
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "/home/me")
	if got, _ := ConfigPath(""); got != "/home/me/.config/sesh/sesh.toml" {
		t.Fatalf("got %s", got)
	}
}

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadConfig(filepath.Join(dir, "missing.toml"))
	if err != nil || cfg.DirLength != 1 {
		t.Fatalf("cfg %+v, err %v", cfg, err)
	}

	path := filepath.Join(dir, "sesh.toml")
	content := `
blacklist = ["scratch"]
dir_length = 2
git_dir_length = 3
git_namer_use_worktree_root = true
[default_session]
startup_command = "nvim"
[[name_substitution]]
find = "~/c/"
replace = ""
[[name_substitution]]
find = "x(.*)"
replace = "$1"
regex = true
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err = LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DirLength != 2 || cfg.GitDirLength != 3 || !cfg.GitNamerUseWorktreeRoot || cfg.Blacklist[0] != "scratch" ||
		len(cfg.NameSubstitutions) != 2 || !cfg.NameSubstitutions[1].Regex {
		t.Fatalf("cfg %+v", cfg)
	}

	if err := os.WriteFile(path, []byte("dir_length = 0"), 0o644); err != nil {
		t.Fatal(err)
	}
	if cfg, _ = LoadConfig(path); cfg.DirLength != 1 {
		t.Fatalf("dir_length %d", cfg.DirLength)
	}

	if err := os.WriteFile(path, []byte("dir_length = ["), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(path); err == nil {
		t.Fatal("want parse error")
	}
	if _, err := LoadConfig(dir); err == nil {
		t.Fatal("want read error for a directory")
	}
}
