package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/rtuszik/sex/internal/rex"
	"github.com/rtuszik/sex/internal/sesh"
	"github.com/rtuszik/sex/internal/state"
)

type ConnectOptions struct {
	Client     string
	Command    string
	Background bool
}

var ErrNotFound = errors.New("no session, config, directory, or zoxide match")

// Resolve maps a picker row or name to a running session or a session to create,
// in sesh's order: running, config, zoxide, directory, zoxide query.
func (a *App) Resolve(ctx context.Context, raw string, running []Item) (Item, error) {
	name := StripIcon(raw)
	if name == "" {
		return Item{}, errors.New("empty session name")
	}
	if it, ok := findRunning(running, name); ok {
		return it, nil
	}
	entries, err := a.Sesh.List(ctx, sesh.Query{Config: true, Zoxide: true, Live: true})
	if err != nil {
		a.warnf("%v", err)
	}
	if e := findEntry(entries, name, a.expand(name)); e != nil {
		return a.fromEntry(ctx, e, running)
	}
	if path := a.expand(name); isDir(path) {
		return a.fromPath(ctx, path, running)
	}
	if path, err := a.Zoxide.Query(ctx, name); err == nil && path != "" {
		return a.fromPath(ctx, path, running)
	}
	return Item{}, fmt.Errorf("%q: %w", name, ErrNotFound)
}

func findRunning(running []Item, name string) (Item, bool) {
	for _, it := range running {
		if it.SessionID == name {
			return it, true
		}
	}
	for _, it := range running {
		if strings.EqualFold(it.Name, name) {
			return it, true
		}
	}
	return Item{}, false
}

func findEntry(entries []sesh.Entry, name, path string) *sesh.Entry {
	for i := range entries {
		e := &entries[i]
		if e.Name == name || (e.Alias != "" && strings.EqualFold(e.Alias, name)) {
			return e
		}
	}
	for i := range entries {
		if samePath(entries[i].Path, path) {
			return &entries[i]
		}
	}
	return nil
}

func (a *App) fromEntry(ctx context.Context, e *sesh.Entry, running []Item) (Item, error) {
	if e.Src != SourceConfig {
		return a.fromPath(ctx, e.Path, running)
	}
	if it, ok := findRunning(running, e.Name); ok {
		return it, nil
	}
	return Item{Source: e.Src, Name: e.Name, Path: e.Path, entry: e}, nil
}

// fromPath names a directory and reuses a running session with that name or path.
func (a *App) fromPath(ctx context.Context, path string, running []Item) (Item, error) {
	path = a.expand(path)
	name, err := a.Namer.Name(ctx, path)
	if err != nil {
		return Item{}, err
	}
	if it, ok := findRunning(running, name); ok {
		return it, nil
	}
	for _, it := range running {
		if samePath(it.Path, path) {
			return it, nil
		}
	}
	return Item{Source: SourceZoxide, Name: name, Path: path}, nil
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// Connect switches to a session, creating it first when needed. It returns the session ID.
func (a *App) Connect(ctx context.Context, raw string, opts ConnectOptions) (string, error) {
	clientID, clientErr := a.ResolveClient(ctx, opts.Client)
	if clientErr != nil && !opts.Background {
		return "", clientErr
	}
	running, err := a.sessions(ctx, clientID)
	if err != nil {
		return "", err
	}
	it, err := a.Resolve(ctx, raw, running)
	if err != nil {
		return "", err
	}
	sessionID := it.SessionID
	if sessionID == "" {
		if sessionID, err = a.create(ctx, it, opts.Command, running); err != nil {
			return "", err
		}
	}
	if opts.Background {
		return sessionID, nil
	}
	if err := a.Rex.Select(ctx, clientID, sessionID); err != nil {
		return sessionID, explain(err)
	}
	return sessionID, nil
}

func explain(err error) error {
	if rex.IsRemoteControlOff(err) {
		return fmt.Errorf("%w\nTurn on Remote Control in Rex Settings > Rex Server, or switch from a key binding (see sex plugin)", err)
	}
	return err
}

func (a *App) create(ctx context.Context, it Item, command string, running []Item) (string, error) {
	if it.entry == nil && it.Path != "" {
		it.entry = a.enrich(ctx, it.Path)
	}
	main := rex.Window{Cwd: it.Path, Input: inputLine(startupCommand(it, command))}
	windows := []rex.Window{main}
	if it.entry != nil {
		for _, w := range it.entry.WindowConfigs {
			cwd := w.Path
			if cwd == "" {
				cwd = it.Path
			}
			windows = append(windows, rex.Window{Label: w.Name, Cwd: a.expand(cwd), Input: inputLine(w.StartupScript)})
		}
	}
	created, err := a.Rex.Create(ctx, rex.CreateOpts{Label: it.Name, Windows: windows})
	if err != nil {
		return "", err
	}
	if len(created.WindowIDs) > 1 {
		if err := a.Rex.FocusWindow(ctx, created.SessionID, created.WindowIDs[0]); err != nil {
			a.warnf("focusing first window: %v", err)
		}
	}
	a.remember(created.SessionID, it.Path, running)
	if it.Path != "" {
		if err := a.Zoxide.Add(ctx, it.Path); err != nil {
			a.warnf("zoxide add: %v", err)
		}
	}
	return created.SessionID, nil
}

// enrich asks sesh for the wildcard and default_session settings of a plain directory.
func (a *App) enrich(ctx context.Context, path string) *sesh.Entry {
	if err := a.Zoxide.Add(ctx, path); err != nil {
		return nil
	}
	entries, err := a.Sesh.List(ctx, sesh.Query{Zoxide: true, Live: true})
	if err != nil {
		return nil
	}
	for i := range entries {
		if samePath(entries[i].Path, path) {
			return &entries[i]
		}
	}
	return nil
}

func startupCommand(it Item, override string) string {
	if override != "" {
		return override
	}
	if it.entry == nil || it.entry.DisableStartupCommand {
		return ""
	}
	return strings.ReplaceAll(it.entry.StartupCommand, "{}", shellQuote(it.Path))
}

func inputLine(cmd string) string {
	if strings.TrimSpace(cmd) == "" {
		return ""
	}
	return cmd + "\n"
}

func (a *App) remember(sessionID, path string, running []Item) {
	st, err := a.Store.Load()
	if err != nil {
		a.warnf("reading state: %v", err)
		return
	}
	live := map[string]bool{sessionID: true}
	for _, it := range running {
		live[it.SessionID] = true
	}
	st.Sessions[sessionID] = state.Record{Path: path}
	if err := a.Store.Save(st, live); err != nil {
		a.warnf("writing state: %v", err)
	}
}

// Last returns the session shown before the current one.
func (a *App) Last(ctx context.Context, client string) (string, string, error) {
	clientID, err := a.ResolveClient(ctx, client)
	if err != nil {
		return "", "", err
	}
	running, err := a.sessions(ctx, clientID)
	if err != nil {
		return "", "", err
	}
	if len(running) < 2 || running[1].ShownAt == nil {
		return clientID, "", errors.New("no previous session")
	}
	return clientID, running[1].SessionID, nil
}

func (a *App) SwitchLast(ctx context.Context, client string) (string, error) {
	clientID, sessionID, err := a.Last(ctx, client)
	if err != nil {
		return "", err
	}
	return sessionID, explain(a.Rex.Select(ctx, clientID, sessionID))
}

func (a *App) Kill(ctx context.Context, raw string) (string, error) {
	running, err := a.sessions(ctx, "")
	if err != nil {
		return "", err
	}
	it, ok := findRunning(running, StripIcon(raw))
	if !ok {
		return "", fmt.Errorf("%q is not a running Rex session", StripIcon(raw))
	}
	return it.Name, a.Rex.Destroy(ctx, it.SessionID)
}

// Root names the git root of the current session, for filtering the picker.
func (a *App) Root(ctx context.Context, client string) (string, error) {
	clientID, _ := a.ResolveClient(ctx, client)
	path := a.Cwd
	if running, err := a.sessions(ctx, clientID); err == nil {
		for _, it := range running {
			if it.Current && it.Path != "" {
				path = it.Path
			}
		}
	}
	return a.Namer.RootName(ctx, path)
}
