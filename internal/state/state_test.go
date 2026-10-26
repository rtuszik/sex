package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRoundTripAndPrune(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sex", "state.json")
	s := NewStore(path)

	st, err := s.Load()
	if err != nil || len(st.Sessions) != 0 {
		t.Fatalf("st %+v, err %v", st, err)
	}
	st.Sessions["session:a"] = Record{Path: "/a"}
	st.Sessions["session:gone"] = Record{Path: "/gone"}
	if err := s.Save(st, map[string]bool{"session:a": true}); err != nil {
		t.Fatal(err)
	}
	st, err = s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Sessions) != 1 || st.Sessions["session:a"].Path != "/a" {
		t.Fatalf("st %+v", st)
	}

	if err := s.Save(State{Sessions: map[string]Record{"session:b": {Path: "/b"}}}, nil); err != nil {
		t.Fatal(err)
	}
	if st, _ = s.Load(); st.Sessions["session:b"].Path != "/b" {
		t.Fatalf("nil live map must keep everything: %+v", st)
	}
}

func TestLoadCorruptOrEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	for _, content := range []string{"{not json", "{}"} {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		st, err := NewStore(path).Load()
		if err != nil || st.Sessions == nil {
			t.Fatalf("%q: st %+v, err %v", content, st, err)
		}
	}
	if _, err := NewStore(dir).Load(); err == nil {
		t.Fatal("want read error for a directory")
	}
}

func TestSaveErrors(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := NewStore(filepath.Join(file, "state.json")).Save(State{Sessions: map[string]Record{}}, nil); err == nil {
		t.Fatal("want mkdir error")
	}
}

func TestDefaultPath(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/xdg")
	if got, _ := DefaultPath(); got != "/xdg/sex/state.json" {
		t.Fatalf("got %s", got)
	}
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("HOME", "/home/me")
	if got, _ := DefaultPath(); got != "/home/me/.local/state/sex/state.json" {
		t.Fatalf("got %s", got)
	}
}
