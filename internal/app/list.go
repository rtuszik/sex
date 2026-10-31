package app

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/rtuszik/sex/internal/sesh"
)

type ListOptions struct {
	Rex         bool
	Config      bool
	Zoxide      bool
	HideCurrent bool
	Client      string
}

func (a *App) List(ctx context.Context, opts ListOptions) ([]Item, error) {
	var items []Item
	err := a.Stream(ctx, opts, func(batch []Item) error {
		items = append(items, batch...)
		return nil
	})
	return items, err
}

// Stream emits Rex sessions as soon as they are read, then the config and zoxide rows.
func (a *App) Stream(ctx context.Context, opts ListOptions, emit func([]Item) error) error {
	all := !opts.Rex && !opts.Config && !opts.Zoxide
	wantRex, wantConfig, wantZoxide := all || opts.Rex, all || opts.Config, all || opts.Zoxide

	type seshResult struct {
		entries []sesh.Entry
		err     error
	}
	seshDone := make(chan seshResult, 1)
	go func() {
		entries, err := a.Sesh.List(ctx, sesh.Query{Config: wantConfig, Zoxide: wantZoxide})
		seshDone <- seshResult{entries, err}
	}()

	clientID, err := a.ResolveClient(ctx, opts.Client)
	if err != nil && opts.Client != "" {
		return err
	}
	// Rex sessions are always read so config and zoxide rows can be deduplicated against them.
	rexItems, err := a.sessions(ctx, clientID)
	if err != nil {
		if wantRex && !wantConfig && !wantZoxide {
			return err
		}
		a.warnf("listing Rex sessions: %v", err)
	}

	blacklist := compileBlacklist(a.Config.Blacklist)
	var shown []Item
	names := map[string]bool{}
	var paths []string
	for _, it := range rexItems {
		names[strings.ToLower(it.Name)] = true
		paths = append(paths, it.Path)
		if !wantRex || (opts.HideCurrent && it.Current) || isBlacklisted(blacklist, it.Name) {
			continue
		}
		shown = append(shown, it)
	}
	if len(shown) > 0 {
		if err := emit(shown); err != nil {
			return err
		}
	}

	res := <-seshDone
	if res.err != nil {
		if !wantRex {
			return res.err
		}
		a.warnf("%v", res.err)
	}
	var rest []Item
	for i := range res.entries {
		e := &res.entries[i]
		if e.Src == SourceConfig && names[strings.ToLower(e.Name)] {
			continue
		}
		if e.Src == SourceZoxide && containsPath(paths, e.Path) {
			continue
		}
		rest = append(rest, Item{Source: e.Src, Name: e.Name, Path: e.Path, Score: e.Score, Icon: e.Icon, entry: e})
	}
	if len(rest) > 0 {
		return emit(rest)
	}
	return nil
}

func containsPath(paths []string, p string) bool {
	for _, q := range paths {
		if samePath(q, p) {
			return true
		}
	}
	return false
}

func compileBlacklist(patterns []string) []*regexp.Regexp {
	var res []*regexp.Regexp
	for _, p := range patterns {
		if re, err := regexp.Compile(p); err == nil {
			res = append(res, re)
		}
	}
	return res
}

func isBlacklisted(blacklist []*regexp.Regexp, name string) bool {
	for _, re := range blacklist {
		if re.MatchString(name) {
			return true
		}
	}
	return false
}

type glyph struct {
	icon  string
	color int
}

var glyphs = map[string]glyph{
	SourceRex:    {icon: "", color: 34},
	SourceConfig: {icon: "", color: 90},
	SourceZoxide: {icon: "", color: 36},
}

// Format renders one picker row; StripIcon undoes the icon prefix.
func Format(it Item, icons, color bool) string {
	if !icons {
		return it.Name
	}
	g, ok := glyphs[it.Source]
	if !ok {
		return it.Name
	}
	icon := g.icon
	if it.Icon != "" {
		icon = it.Icon
	}
	if color {
		icon = fmt.Sprintf("\x1b[%dm%s\x1b[39m", g.color, icon)
	}
	return icon + " " + it.Name
}

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// StripIcon drops ANSI codes and a leading token without letters or digits, such as a glyph.
func StripIcon(s string) string {
	s = strings.TrimSpace(ansiRe.ReplaceAllString(s, ""))
	first, rest, ok := strings.Cut(s, " ")
	if !ok || strings.IndexFunc(first, func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }) >= 0 {
		return s
	}
	return strings.TrimSpace(rest)
}
