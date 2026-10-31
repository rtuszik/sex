// Package app joins Rex sessions with sesh's config and zoxide entries.
package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/rtuszik/sex/internal/rex"
	"github.com/rtuszik/sex/internal/sesh"
	"github.com/rtuszik/sex/internal/state"
)

const (
	SourceRex    = "rex"
	SourceConfig = "config"
	SourceZoxide = "zoxide"
)

type Rex interface {
	Snapshot(ctx context.Context, clientID string) ([]rex.Snapshot, error)
	AppClients(ctx context.Context) ([]rex.AppClient, error)
	BlockOwner(ctx context.Context, sessionID, blockID string) (string, error)
	Create(ctx context.Context, opts rex.CreateOpts) (rex.Created, error)
	FocusWindow(ctx context.Context, sessionID, windowID string) error
	Select(ctx context.Context, clientID, sessionID string) error
	Destroy(ctx context.Context, sessionID string) error
	Capture(ctx context.Context, sessionID, blockID string) (string, error)
}

type Sesh interface {
	List(ctx context.Context, q sesh.Query) ([]sesh.Entry, error)
}

type Namer interface {
	Name(ctx context.Context, path string) (string, error)
	RootName(ctx context.Context, path string) (string, error)
}

type Zoxide interface {
	Add(ctx context.Context, path string) error
	Query(ctx context.Context, query string) (string, error)
}

type Store interface {
	Load() (state.State, error)
	Save(st state.State, live map[string]bool) error
}

type Shell interface {
	Run(ctx context.Context, dir, command string, w io.Writer) error
}

type App struct {
	Rex    Rex
	Sesh   Sesh
	Namer  Namer
	Zoxide Zoxide
	Store  Store
	Shell  Shell
	Config sesh.Config
	Home   string
	Cwd    string
	Getenv func(string) string
	Stderr io.Writer
}

type Item struct {
	Source    string     `json:"source"`
	Name      string     `json:"name"`
	Path      string     `json:"path,omitempty"`
	SessionID string     `json:"session_id,omitempty"`
	ShownAt   *time.Time `json:"shown_at,omitempty"`
	Current   bool       `json:"current,omitempty"`
	Score     float64    `json:"score,omitempty"`
	Icon      string     `json:"icon,omitempty"`

	entry         *sesh.Entry
	blockID       string
	firstWindowID string
}

func (a *App) env(key string) string {
	if a.Getenv == nil {
		return ""
	}
	return a.Getenv(key)
}

func (a *App) warnf(format string, args ...any) {
	if a.Stderr != nil {
		_, _ = fmt.Fprintf(a.Stderr, "sex: "+format+"\n", args...)
	}
}

var ErrNoApp = errors.New("no Rex app is connected")

// ResolveClient picks the app whose view sex switches: explicit, then REX_CLIENT, then the only app,
// then the app that shows the current terminal.
func (a *App) ResolveClient(ctx context.Context, explicit string) (string, error) {
	want := explicit
	if want == "" {
		want = a.env("REX_CLIENT")
	}
	if strings.HasPrefix(want, "client:") {
		return want, nil
	}
	apps, err := a.Rex.AppClients(ctx)
	if err != nil {
		return "", err
	}
	if want != "" {
		for _, c := range apps {
			if strings.EqualFold(c.Label, want) || (len(want) >= 4 && strings.HasSuffix(c.ID, want)) {
				return c.ID, nil
			}
		}
		return "", fmt.Errorf("no Rex app matches client %q", want)
	}
	switch len(apps) {
	case 0:
		return "", ErrNoApp
	case 1:
		return apps[0].ID, nil
	}
	if id := a.terminalOwner(ctx, apps); id != "" {
		return id, nil
	}
	names := make([]string, 0, len(apps))
	for _, c := range apps {
		names = append(names, fmt.Sprintf("%s (%s)", c.Label, c.ID))
	}
	return "", fmt.Errorf("several Rex apps are connected, pass --client: %s", strings.Join(names, ", "))
}

func (a *App) terminalOwner(ctx context.Context, apps []rex.AppClient) string {
	session, block := a.env("REX_SESSION"), a.env("REX_BLOCK")
	if session == "" || block == "" {
		return ""
	}
	owner, err := a.Rex.BlockOwner(ctx, session, block)
	if err != nil {
		return ""
	}
	for _, c := range apps {
		if c.ID == owner {
			return owner
		}
	}
	return ""
}

// sessions returns Rex sessions most recently shown first, with the on-screen one marked current.
func (a *App) sessions(ctx context.Context, clientID string) ([]Item, error) {
	snaps, err := a.Rex.Snapshot(ctx, clientID)
	if err != nil {
		return nil, err
	}
	st, err := a.Store.Load()
	if err != nil {
		a.warnf("reading state: %v", err)
	}
	snaps = rex.ByRecent(snaps)
	current := ""
	if len(snaps) > 0 && !snaps[0].ShownAt.IsZero() {
		current = snaps[0].ID
	} else {
		current = a.env("REX_SESSION")
	}
	items := make([]Item, 0, len(snaps))
	for _, s := range snaps {
		it := Item{
			Source:        SourceRex,
			Name:          s.Label,
			Path:          s.Cwd,
			SessionID:     s.ID,
			Current:       s.ID == current,
			blockID:       s.BlockID,
			firstWindowID: s.FirstWindowID,
		}
		if rec, ok := st.Sessions[s.ID]; ok && rec.Path != "" {
			it.Path = rec.Path
		}
		if !s.ShownAt.IsZero() {
			shown := s.ShownAt
			it.ShownAt = &shown
		}
		items = append(items, it)
	}
	return items, nil
}

func (a *App) expand(path string) string {
	if path == "~" {
		return a.Home
	}
	if rest, ok := strings.CutPrefix(path, "~/"); ok {
		return filepath.Join(a.Home, rest)
	}
	if !filepath.IsAbs(path) && a.Cwd != "" {
		return filepath.Join(a.Cwd, path)
	}
	return filepath.Clean(path)
}

func samePath(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return filepath.Clean(a) == filepath.Clean(b)
}
