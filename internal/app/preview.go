package app

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/rtuszik/sex/internal/sesh"
)

const defaultPreviewLines = 40

// Preview prints the screen of a running session, or runs the directory's preview command.
func (a *App) Preview(ctx context.Context, raw string, w io.Writer) error {
	name := StripIcon(raw)
	running, err := a.sessions(ctx, "")
	if err != nil {
		a.warnf("listing Rex sessions: %v", err)
	}
	if it, ok := findRunning(running, name); ok {
		if it.blockID == "" {
			return fmt.Errorf("%q has no terminal to preview", it.Name)
		}
		screen, err := a.Rex.Capture(ctx, it.SessionID, it.blockID)
		if err != nil {
			return err
		}
		_, err = io.WriteString(w, tail(screen, a.previewLines()))
		return err
	}

	path, command := "", ""
	entries, err := a.Sesh.List(ctx, sesh.Query{Config: true, Zoxide: true})
	if err != nil {
		a.warnf("%v", err)
	}
	if e := findEntry(entries, name, a.expand(name)); e != nil {
		path, command = e.Path, e.PreviewCommand
	} else if p := a.expand(name); isDir(p) {
		path = p
	} else {
		return fmt.Errorf("%q: %w", name, ErrNotFound)
	}
	if command == "" {
		command = "ls -la {}"
	}
	return a.Shell.Run(ctx, path, strings.ReplaceAll(command, "{}", shellQuote(path)), w)
}

func (a *App) previewLines() int {
	if n, err := strconv.Atoi(a.env("FZF_PREVIEW_LINES")); err == nil && n > 0 {
		return n
	}
	return defaultPreviewLines
}

func tail(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n") + "\n"
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
