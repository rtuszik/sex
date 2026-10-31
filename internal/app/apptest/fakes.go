// Package apptest provides in-memory fakes for app dependencies.
package apptest

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/rtuszik/sex/internal/rex"
	"github.com/rtuszik/sex/internal/sesh"
	"github.com/rtuszik/sex/internal/state"
)

type Rex struct {
	Snaps        []rex.Snapshot
	SnapErr      error
	SnapClient   []string
	Apps         []rex.AppClient
	AppsErr      error
	Owner        string
	OwnerErr     error
	OwnerQueries []string
	Created      []rex.CreateOpts
	CreateRes    rex.Created
	CreateErr    error
	Focused      []string
	FocusErr     error
	Selected     []string
	SelectErr    error
	Destroyed    []string
	DestroyErr   error
	Screen       string
	CaptureErr   error
	Captured     []string
}

func (r *Rex) Snapshot(_ context.Context, clientID string) ([]rex.Snapshot, error) {
	r.SnapClient = append(r.SnapClient, clientID)
	return r.Snaps, r.SnapErr
}

func (r *Rex) AppClients(context.Context) ([]rex.AppClient, error) { return r.Apps, r.AppsErr }

func (r *Rex) BlockOwner(_ context.Context, sessionID, blockID string) (string, error) {
	r.OwnerQueries = append(r.OwnerQueries, sessionID+"|"+blockID)
	return r.Owner, r.OwnerErr
}

func (r *Rex) Create(_ context.Context, opts rex.CreateOpts) (rex.Created, error) {
	r.Created = append(r.Created, opts)
	return r.CreateRes, r.CreateErr
}

func (r *Rex) FocusWindow(_ context.Context, sessionID, windowID string) error {
	r.Focused = append(r.Focused, sessionID+"|"+windowID)
	return r.FocusErr
}

func (r *Rex) Select(_ context.Context, clientID, sessionID string) error {
	r.Selected = append(r.Selected, clientID+"|"+sessionID)
	return r.SelectErr
}

func (r *Rex) Destroy(_ context.Context, sessionID string) error {
	r.Destroyed = append(r.Destroyed, sessionID)
	return r.DestroyErr
}

func (r *Rex) Capture(_ context.Context, sessionID, blockID string) (string, error) {
	r.Captured = append(r.Captured, sessionID+"|"+blockID)
	return r.Screen, r.CaptureErr
}

// Sesh returns the entries whose source matches the query.
type Sesh struct {
	Entries []sesh.Entry
	Err     error
	mu      sync.Mutex
	calls   []string
}

func (s *Sesh) List(_ context.Context, q sesh.Query) ([]sesh.Entry, error) {
	s.mu.Lock()
	s.calls = append(s.calls, fmt.Sprintf("config=%t zoxide=%t live=%t", q.Config, q.Zoxide, q.Live))
	s.mu.Unlock()
	if s.Err != nil {
		return nil, s.Err
	}
	var out []sesh.Entry
	for _, e := range s.Entries {
		if (q.Config && e.Src == "config") || (q.Zoxide && e.Src == "zoxide") {
			out = append(out, e)
		}
	}
	return out, nil
}

func (s *Sesh) Calls() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.calls...)
}

type Namer struct {
	Names map[string]string
	Roots map[string]string
}

func (n *Namer) Name(_ context.Context, path string) (string, error) {
	if name, ok := n.Names[path]; ok {
		return name, nil
	}
	return "", errors.New("no name for " + path)
}

func (n *Namer) RootName(_ context.Context, path string) (string, error) {
	if name, ok := n.Roots[path]; ok {
		return name, nil
	}
	return "", errors.New("no root for " + path)
}

type Zoxide struct {
	Added   []string
	AddErr  error
	Queries map[string]string
}

func (z *Zoxide) Add(_ context.Context, path string) error {
	z.Added = append(z.Added, path)
	return z.AddErr
}

func (z *Zoxide) Query(_ context.Context, q string) (string, error) {
	if p, ok := z.Queries[q]; ok {
		return p, nil
	}
	return "", errors.New("no match")
}

type Store struct {
	State   state.State
	LoadErr error
	SaveErr error
	Saved   *state.State
	Live    map[string]bool
}

func (s *Store) Load() (state.State, error) {
	st := state.State{Sessions: map[string]state.Record{}}
	for k, v := range s.State.Sessions {
		st.Sessions[k] = v
	}
	return st, s.LoadErr
}

func (s *Store) Save(st state.State, live map[string]bool) error {
	s.Saved = &st
	s.Live = live
	return s.SaveErr
}

type Shell struct {
	Dir     string
	Command string
	Output  string
	Err     error
}

func (s *Shell) Run(_ context.Context, dir, command string, w io.Writer) error {
	s.Dir, s.Command = dir, command
	_, _ = io.WriteString(w, s.Output)
	return s.Err
}
