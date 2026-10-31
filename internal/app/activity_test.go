package app

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/rtuszik/sex/internal/rex"
)

func activities() []rex.Activity {
	p := 40
	return []rex.Activity{
		{SessionID: "session:a", Session: "api", WindowID: "window:a", Window: "Window 2b8b7b0c", WindowPos: 1, BlockID: "block:idle",
			Records: []rex.StatusRecord{{App: "claude-code", EffectiveApp: "claude-code", State: "idle", UpdatedAt: t0}}},
		{SessionID: "session:b", Session: "web", WindowID: "window:b", Window: "build", WindowPos: 2, BlockID: "block:busy",
			Records: []rex.StatusRecord{
				{App: "cargo", State: "working", Msg: "compiling", Progress: &p, UpdatedAt: t0},
				{ID: "test", EffectiveApp: "cargo", State: "blocked", Kind: "permission", Title: "run tests?", UpdatedAt: t0.Add(-time.Minute)},
			}},
		{SessionID: "session:c", Session: "notes", WindowID: "window:c", Window: "", WindowPos: 3, BlockID: "block:done",
			Records: []rex.StatusRecord{{App: "codex", State: "done", UpdatedAt: t0.Add(time.Minute)}}},
		{SessionID: "session:d", Session: "odd", WindowID: "window:d", WindowPos: 1, BlockID: "block:odd",
			Records: []rex.StatusRecord{{App: "x", State: "dancing", UpdatedAt: t0.Add(time.Hour)}}},
		{SessionID: "session:e", BlockID: "block:empty"},
	}
}

func TestActivityOrderAndSummary(t *testing.T) {
	f := newFixture(t)
	f.rex.Activities = activities()
	items, err := f.app.Activity(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, it := range items {
		got = append(got, it.BlockID+":"+it.State)
	}
	// Most urgent first; unknown states count as idle; newer idle first.
	assertSlice(t, got, []string{"block:busy:blocked", "block:done:done", "block:odd:idle", "block:idle:idle"})
	busy := items[0]
	assertEqual(t, busy.App, "cargo")
	assertEqual(t, busy.Message, "run tests?")
	assertEqual(t, busy.Kind, "permission")
	assertEqual(t, busy.Tab, "build")
	assertEqual(t, items[1].Tab, "tab 3")
	assertEqual(t, items[3].Tab, "tab 1")

	items, err = f.app.Activity(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	assertEqual(t, len(items), 2)

	f.rex.ActivityErr = errors.New("down")
	if _, err := f.app.Activity(context.Background(), true); err == nil {
		t.Fatal("want error")
	}
}

func TestFormatActivity(t *testing.T) {
	p := 40
	it := ActivityItem{BlockID: "block:1", State: "working", Session: "web", Tab: "build", App: "cargo", Message: "a\tb", Progress: &p}
	assertEqual(t, FormatActivity(it, false), "block:1\t● working 40%  web › build  cargo  a b")
	it = ActivityItem{BlockID: "block:2", State: "blocked", Kind: "question", Session: "s", Tab: "tab 1"}
	assertEqual(t, FormatActivity(it, false), "block:2\t● blocked (question)  s › tab 1")
	if !strings.Contains(FormatActivity(it, true), "\x1b[33m") {
		t.Fatal("blocked should be yellow")
	}
}

func TestFocusActivity(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	f.rex.Activities = activities()
	if err := f.app.FocusActivity(ctx, "", "block:busy\t● blocked  web › build"); err != nil {
		t.Fatal(err)
	}
	assertSlice(t, f.rex.FocusedBlocks, []string{"session:b|block:busy"})
	assertSlice(t, f.rex.Selected, []string{appID + "|session:b|window:b"})

	if err := f.app.FocusActivity(ctx, "", "block:gone"); err == nil {
		t.Fatal("want error for a terminal without status")
	}
	f.rex.SelectErr = &rex.APIError{Code: "refused", Msg: "rex: refused"}
	if err := f.app.FocusActivity(ctx, "", "block:busy"); err == nil || !strings.Contains(err.Error(), "Remote Control") {
		t.Fatalf("err = %v", err)
	}
	f.rex.FocusBlockErr = errors.New("focus failed")
	if err := f.app.FocusActivity(ctx, "", "block:busy"); err == nil || err.Error() != "focus failed" {
		t.Fatalf("err = %v", err)
	}
	f.rex.Apps = nil
	if err := f.app.FocusActivity(ctx, "", "block:busy"); !errors.Is(err, ErrNoApp) {
		t.Fatalf("err = %v", err)
	}
	f.rex.ActivityErr = errors.New("down")
	if err := f.app.FocusActivity(ctx, "", "block:busy"); err == nil {
		t.Fatal("want error")
	}
}

func TestPreviewActivity(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	f.rex.Activities = activities()
	f.rex.Screen = "screen\n"
	var out bytes.Buffer
	if err := f.app.PreviewActivity(ctx, "block:done", &out); err != nil {
		t.Fatal(err)
	}
	assertEqual(t, out.String(), "screen\n")
	assertSlice(t, f.rex.Captured, []string{"session:c|block:done"})
	if err := f.app.PreviewActivity(ctx, "block:gone", &out); err == nil {
		t.Fatal("want error")
	}
	f.rex.CaptureErr = errors.New("capture failed")
	if err := f.app.PreviewActivity(ctx, "block:done", &out); err == nil {
		t.Fatal("want error")
	}
	f.rex.ActivityErr = errors.New("down")
	if err := f.app.PreviewActivity(ctx, "block:done", &out); err == nil {
		t.Fatal("want error")
	}
}
