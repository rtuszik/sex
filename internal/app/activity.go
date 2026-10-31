package app

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/rtuszik/sex/internal/rex"
)

// stateRank orders program states by how much they need the user; unknown states count as idle.
var stateRank = map[string]int{"blocked": 0, "error": 1, "working": 2, "done": 3, "idle": 4}

type ActivityItem struct {
	SessionID string    `json:"session_id"`
	Session   string    `json:"session"`
	WindowID  string    `json:"window_id"`
	Tab       string    `json:"tab"`
	BlockID   string    `json:"block_id"`
	State     string    `json:"state"`
	Kind      string    `json:"kind,omitempty"`
	App       string    `json:"app,omitempty"`
	Message   string    `json:"message,omitempty"`
	Progress  *int      `json:"progress,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

func rank(state string) int {
	if r, ok := stateRank[state]; ok {
		return r
	}
	return stateRank["idle"]
}

var defaultTabRe = regexp.MustCompile(`^Window [0-9a-f]{8}$`)

// Activity lists terminals with a program status, most urgent first, like Rex's Activity view.
func (a *App) Activity(ctx context.Context, includeIdle bool) ([]ActivityItem, error) {
	entries, err := a.Rex.Activity(ctx)
	if err != nil {
		return nil, err
	}
	var items []ActivityItem
	for _, e := range entries {
		it, ok := summarize(e)
		if !ok || (!includeIdle && rank(it.State) == stateRank["idle"]) {
			continue
		}
		items = append(items, it)
	}
	sort.SliceStable(items, func(i, j int) bool {
		if ri, rj := rank(items[i].State), rank(items[j].State); ri != rj {
			return ri < rj
		}
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})
	return items, nil
}

// summarize reduces a terminal's records to its most urgent one, the newest on ties.
func summarize(e rex.Activity) (ActivityItem, bool) {
	var best *rex.StatusRecord
	for i := range e.Records {
		r := &e.Records[i]
		if best == nil || rank(r.State) < rank(best.State) ||
			(rank(r.State) == rank(best.State) && r.UpdatedAt.After(best.UpdatedAt)) {
			best = r
		}
	}
	if best == nil {
		return ActivityItem{}, false
	}
	tab := e.Window
	if tab == "" || defaultTabRe.MatchString(tab) {
		tab = fmt.Sprintf("tab %d", e.WindowPos)
	}
	app := best.EffectiveApp
	if app == "" {
		app = best.App
	}
	msg := best.Msg
	if msg == "" {
		msg = best.Title
	}
	state := best.State
	if _, ok := stateRank[state]; !ok {
		state = "idle"
	}
	return ActivityItem{
		SessionID: e.SessionID, Session: e.Session, WindowID: e.WindowID, Tab: tab, BlockID: e.BlockID,
		State: state, Kind: best.Kind, App: app, Message: msg, Progress: best.Progress, UpdatedAt: best.UpdatedAt,
	}, true
}

var stateGlyphs = map[string]struct {
	glyph string
	color int
}{
	"blocked": {"●", 33},
	"error":   {"●", 31},
	"working": {"●", 34},
	"done":    {"●", 32},
	"idle":    {"○", 90},
}

// FormatActivity renders one picker row: block ID, a tab, then the visible text.
func FormatActivity(it ActivityItem, color bool) string {
	g := stateGlyphs[it.State]
	state := it.State
	if it.Kind != "" {
		state += " (" + it.Kind + ")"
	}
	if it.Progress != nil {
		state += fmt.Sprintf(" %d%%", *it.Progress)
	}
	head := g.glyph + " " + state
	if color {
		head = fmt.Sprintf("\x1b[%dm%s\x1b[39m", g.color, head)
	}
	parts := []string{head, it.Session + " › " + it.Tab}
	if it.App != "" {
		parts = append(parts, it.App)
	}
	if it.Message != "" {
		parts = append(parts, strings.ReplaceAll(it.Message, "\t", " "))
	}
	return it.BlockID + "\t" + strings.Join(parts, "  ")
}

// FocusActivity shows a terminal: it becomes the focused pane of its tab, and the app switches to it.
func (a *App) FocusActivity(ctx context.Context, client, blockID string) error {
	blockID, _, _ = strings.Cut(strings.TrimSpace(blockID), "\t")
	items, err := a.Activity(ctx, true)
	if err != nil {
		return err
	}
	for _, it := range items {
		if it.BlockID != blockID {
			continue
		}
		clientID, err := a.ResolveClient(ctx, client)
		if err != nil {
			return err
		}
		if err := a.Rex.FocusBlock(ctx, it.SessionID, it.BlockID); err != nil {
			return err
		}
		return explain(a.Rex.SelectWindow(ctx, clientID, it.SessionID, it.WindowID))
	}
	return fmt.Errorf("%q no longer reports a status", blockID)
}

// PreviewActivity prints the terminal's screen.
func (a *App) PreviewActivity(ctx context.Context, blockID string, w io.Writer) error {
	blockID, _, _ = strings.Cut(strings.TrimSpace(blockID), "\t")
	items, err := a.Activity(ctx, true)
	if err != nil {
		return err
	}
	for _, it := range items {
		if it.BlockID == blockID {
			screen, err := a.Rex.Capture(ctx, it.SessionID, it.BlockID)
			if err != nil {
				return err
			}
			_, err = io.WriteString(w, tail(screen, a.previewLines()))
			return err
		}
	}
	return fmt.Errorf("%q no longer reports a status", blockID)
}
