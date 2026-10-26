package sesh

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
	"github.com/rtuszik/sex/internal/execx"
)

type WindowConfig struct {
	Name          string
	StartupScript string
	Path          string
}

// Entry mirrors one row of `sesh list --json`.
type Entry struct {
	Src                   string
	Name                  string
	Path                  string
	StartupCommand        string
	PreviewCommand        string
	DisableStartupCommand bool
	WindowConfigs         []WindowConfig
	Score                 float64
	Icon                  string
	Alias                 string
}

type Client struct {
	run        execx.Runner
	bin        string
	configPath string
}

func New(run execx.Runner, bin, configPath string) *Client {
	if bin == "" {
		bin = "sesh"
	}
	return &Client{run: run, bin: bin, configPath: configPath}
}

// Query selects sources. Live bypasses sesh's cache, which only applies when `cache = true`.
type Query struct {
	Config bool
	Zoxide bool
	Live   bool
}

func (c *Client) List(ctx context.Context, q Query) ([]Entry, error) {
	if !q.Config && !q.Zoxide {
		return nil, nil
	}
	// Short flags: `sesh -C` is the config path, `sesh list -c` the config source.
	var args []string
	if c.configPath != "" {
		args = append(args, "-C", c.configPath)
	}
	args = append(args, "list", "-j")
	if q.Live {
		args = append(args, "--no-cache")
	}
	if q.Config {
		args = append(args, "-c")
	}
	if q.Zoxide {
		args = append(args, "-z")
	}
	out, err := c.run.Run(ctx, execx.Cmd{Name: c.bin, Args: args})
	if err != nil {
		return nil, fmt.Errorf("sesh list: %w", err)
	}
	var entries []Entry
	if err := json.Unmarshal(out, &entries); err != nil {
		return nil, fmt.Errorf("sesh list: decoding json: %w", err)
	}
	return entries, nil
}

type NameSubstitution struct {
	Find    string `toml:"find"`
	Replace string `toml:"replace"`
	Regex   bool   `toml:"regex"`
}

// Config holds the sesh.toml keys sex reads itself; sesh resolves everything else.
type Config struct {
	Blacklist               []string           `toml:"blacklist"`
	DirLength               int                `toml:"dir_length"`
	GitDirLength            int                `toml:"git_dir_length"`
	GitNamerUseWorktreeRoot bool               `toml:"git_namer_use_worktree_root"`
	NameSubstitutions       []NameSubstitution `toml:"name_substitution"`
}

func ConfigPath(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "sesh", "sesh.toml"), nil
}

// LoadConfig returns defaults when the file does not exist, like sesh.
func LoadConfig(path string) (Config, error) {
	cfg := Config{DirLength: 1}
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if err := toml.Unmarshal(b, &cfg); err != nil {
		return cfg, fmt.Errorf("parsing %s: %w", path, err)
	}
	if cfg.DirLength < 1 {
		cfg.DirLength = 1
	}
	return cfg, nil
}
