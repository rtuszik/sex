// Package namer derives session names from paths.
// Ported from sesh (https://github.com/joshmedeski/sesh), MIT License, Copyright (c) 2023 Josh Medeski.
package namer

import (
	"context"
	"fmt"
	pathpkg "path"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/rtuszik/sex/internal/execx"
	"github.com/rtuszik/sex/internal/sesh"
)

const maxLabel = 128

type Git interface {
	WorktreeList(ctx context.Context, path string) (string, error)
	ShowTopLevel(ctx context.Context, path string) (string, error)
}

type ExecGit struct{ Run execx.Runner }

func (g ExecGit) WorktreeList(ctx context.Context, path string) (string, error) {
	out, err := g.Run.Run(ctx, execx.Cmd{Name: "git", Args: []string{"-C", path, "worktree", "list", "--porcelain"}})
	return strings.TrimSpace(string(out)), err
}

func (g ExecGit) ShowTopLevel(ctx context.Context, path string) (string, error) {
	out, err := g.Run.Run(ctx, execx.Cmd{Name: "git", Args: []string{"-C", path, "rev-parse", "--show-toplevel"}})
	return strings.TrimSpace(string(out)), err
}

type Namer struct {
	git          Git
	cfg          sesh.Config
	home         string
	evalSymlinks func(string) (string, error)
}

func New(git Git, cfg sesh.Config, home string) *Namer {
	return &Namer{git: git, cfg: cfg, home: home, evalSymlinks: filepath.EvalSymlinks}
}

type strategy func(*Namer, context.Context, string) string

func (n *Namer) Name(ctx context.Context, path string) (string, error) {
	return n.run(ctx, path, []strategy{(*Namer).substitution, (*Namer).gitName, (*Namer).dirName})
}

func (n *Namer) RootName(ctx context.Context, path string) (string, error) {
	return n.run(ctx, path, []strategy{(*Namer).substitution, (*Namer).gitRootName, (*Namer).dirName})
}

func (n *Namer) run(ctx context.Context, path string, strategies []strategy) (string, error) {
	resolved, err := n.evalSymlinks(path)
	if err != nil {
		return "", err
	}
	for _, s := range strategies {
		if name := validName(s(n, ctx, resolved)); name != "" {
			return name, nil
		}
	}
	return "", fmt.Errorf("could not determine name from path: %s", path)
}

// validName keeps characters tmux rejects; Rex labels allow them.
func validName(name string) string {
	name = strings.TrimSpace(name)
	if utf8.RuneCountInString(name) > maxLabel {
		name = string([]rune(name)[:maxLabel])
	}
	return name
}

func (n *Namer) shortenHome(path string) string {
	if n.home == "" {
		return path
	}
	if path == n.home {
		return "~"
	}
	if rest, ok := strings.CutPrefix(path, n.home+"/"); ok {
		return "~/" + rest
	}
	return path
}

func (n *Namer) substitution(_ context.Context, path string) string {
	if len(n.cfg.NameSubstitutions) == 0 {
		return ""
	}
	short := filepath.ToSlash(n.shortenHome(path))
	name := ApplySubstitutions(short, n.cfg.NameSubstitutions)
	if name == short {
		return ""
	}
	return name
}

// ApplySubstitutions feeds each rule the previous rule's output; invalid regexes are skipped.
func ApplySubstitutions(name string, rules []sesh.NameSubstitution) string {
	for _, rule := range rules {
		if rule.Find == "" {
			continue
		}
		if rule.Regex {
			re, err := regexp.Compile(rule.Find)
			if err != nil {
				continue
			}
			name = re.ReplaceAllString(name, rule.Replace)
			continue
		}
		name = strings.ReplaceAll(name, rule.Find, rule.Replace)
	}
	return name
}

func (n *Namer) dirName(_ context.Context, path string) string {
	return LastNComponents(path, n.cfg.DirLength)
}

// LastNComponents returns the last n path components joined by "/".
func LastNComponents(path string, n int) string {
	if path == "" {
		return ""
	}
	clean := filepath.ToSlash(filepath.Clean(path))
	if n <= 1 {
		return pathpkg.Base(clean)
	}
	parts := make([]string, 0, n)
	current := clean
	for len(parts) < n && current != "/" && current != "." {
		base := pathpkg.Base(current)
		if base == "" || base == "." {
			break
		}
		parts = append(parts, base)
		parent := pathpkg.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	if len(parts) == 0 {
		return pathpkg.Base(clean)
	}
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	return strings.Join(parts, "/")
}

// gitRootPath returns the main working tree from `git worktree list --porcelain`.
func gitRootPath(out string) string {
	var path string
	var bare bool
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			if path != "" {
				break
			}
			continue
		}
		if p, ok := strings.CutPrefix(line, "worktree "); ok {
			path = p
			continue
		}
		if line == "bare" {
			bare = true
		}
	}
	if path == "" {
		return ""
	}
	if bare {
		if trimmed, ok := strings.CutSuffix(path, "/.bare"); ok {
			return trimmed
		}
		if trimmed, ok := strings.CutSuffix(path, "/.git"); ok {
			return trimmed
		}
	}
	return path
}

func (n *Namer) mainRoot(ctx context.Context, path string) string {
	out, err := n.git.WorktreeList(ctx, path)
	if err != nil {
		return ""
	}
	return gitRootPath(out)
}

func (n *Namer) anchor(ctx context.Context, path string) string {
	if n.cfg.GitNamerUseWorktreeRoot {
		top, err := n.git.ShowTopLevel(ctx, path)
		if err != nil {
			return ""
		}
		return top
	}
	return n.mainRoot(ctx, path)
}

func (n *Namer) repoName(anchor string) string {
	if n.cfg.GitDirLength > 1 {
		return LastNComponents(anchor, n.cfg.GitDirLength)
	}
	return pathpkg.Base(filepath.ToSlash(anchor))
}

func (n *Namer) gitName(ctx context.Context, path string) string {
	anchor := n.anchor(ctx, path)
	if anchor == "" {
		return ""
	}
	return n.repoName(anchor) + strings.TrimPrefix(filepath.ToSlash(path), anchor)
}

func (n *Namer) gitRootName(ctx context.Context, path string) string {
	if n.cfg.GitNamerUseWorktreeRoot {
		top, err := n.git.ShowTopLevel(ctx, path)
		if err != nil || top == "" {
			return ""
		}
		return n.repoName(top)
	}
	root := n.mainRoot(ctx, path)
	if root == "" {
		return ""
	}
	name := n.repoName(root)
	top, err := n.git.ShowTopLevel(ctx, path)
	if err != nil || top == "" {
		return name
	}
	return name + strings.TrimPrefix(top, root)
}
