package rex

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

//go:embed snapshot.lua
var snapshotScript []byte

type Snapshot struct {
	ID            string    `json:"session_id"`
	Label         string    `json:"label"`
	FirstWindowID string    `json:"first_window_id"`
	BlockID       string    `json:"block_id"`
	Cwd           string    `json:"cwd"`
	ShownAt       time.Time `json:"shown_at"`
}

// Snapshot reads every session in one rex call. ShownAt is the last time
// clientID sized the session's visible terminal, a proxy for when it was on screen.
func (c *Client) Snapshot(ctx context.Context, clientID string) ([]Snapshot, error) {
	args, err := json.Marshal(map[string]string{"client": clientID})
	if err != nil {
		return nil, err
	}
	out, err := c.exec(ctx, snapshotScript, "do", "-", "--args", string(args))
	if err != nil {
		return nil, err
	}
	var res struct {
		Sessions []Snapshot `json:"sessions"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		return nil, fmt.Errorf("rex: decoding snapshot: %w", err)
	}
	return res.Sessions, nil
}

// ByRecent sorts sessions most recently shown first, keeping server order for ties.
func ByRecent(sessions []Snapshot) []Snapshot {
	sorted := append([]Snapshot(nil), sessions...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].ShownAt.After(sorted[j].ShownAt)
	})
	return sorted
}
