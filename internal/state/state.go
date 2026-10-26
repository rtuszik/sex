// Package state remembers the path each session was created for; Rex does not store one.
package state

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Record struct {
	Path string `json:"path"`
}

type State struct {
	Sessions map[string]Record `json:"sessions"`
}

func DefaultPath() (string, error) {
	dir := os.Getenv("XDG_STATE_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(dir, "sex", "state.json"), nil
}

type Store struct{ path string }

func NewStore(path string) *Store { return &Store{path: path} }

func (s *Store) Load() (State, error) {
	st := State{Sessions: map[string]Record{}}
	b, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return st, nil
	}
	if err != nil {
		return st, err
	}
	// Corrupt state is not fatal: paths fall back to the terminal cwd.
	if err := json.Unmarshal(b, &st); err != nil {
		return State{Sessions: map[string]Record{}}, nil
	}
	if st.Sessions == nil {
		st.Sessions = map[string]Record{}
	}
	return st, nil
}

// Save writes atomically and drops sessions that no longer exist.
func (s *Store) Save(st State, live map[string]bool) error {
	for id := range st.Sessions {
		if live != nil && !live[id] {
			delete(st.Sessions, id)
		}
	}
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".state-*.json")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(b); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), s.path)
}
