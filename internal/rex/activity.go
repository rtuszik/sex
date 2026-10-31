package rex

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"time"
)

//go:embed activity.lua
var activityScript []byte

// StatusRecord is one OSC 7501 program status report.
type StatusRecord struct {
	ID           string    `json:"id"`
	App          string    `json:"app"`
	EffectiveApp string    `json:"effective_app"`
	State        string    `json:"state"`
	Kind         string    `json:"kind"`
	Title        string    `json:"title"`
	Msg          string    `json:"msg"`
	Progress     *int      `json:"progress"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Activity struct {
	SessionID string         `json:"session_id"`
	Session   string         `json:"session"`
	WindowID  string         `json:"window_id"`
	Window    string         `json:"window"`
	WindowPos int            `json:"window_index"`
	BlockID   string         `json:"block_id"`
	Records   []StatusRecord `json:"records"`
}

// Activity returns every terminal that reported a program status, in one rex call.
func (c *Client) Activity(ctx context.Context) ([]Activity, error) {
	out, err := c.read(ctx, activityScript, "do", "-")
	if err != nil {
		return nil, err
	}
	var res struct {
		Entries []Activity `json:"entries"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		return nil, fmt.Errorf("rex: decoding activity: %w", err)
	}
	return res.Entries, nil
}

// SelectWindow switches the app to a session and shows one of its tabs.
func (c *Client) SelectWindow(ctx context.Context, clientID, sessionID, windowID string) error {
	return c.Perform(ctx, clientID, "session.select", map[string]string{"session_id": sessionID, "window_id": windowID})
}

func (c *Client) FocusBlock(ctx context.Context, sessionID, blockID string) error {
	return c.Call(ctx, "session.focus_block", map[string]string{"session_id": sessionID, "block_id": blockID}, nil)
}

type PluginInfo struct {
	Version string `json:"version"`
	Bin     string `json:"bin"`
	Error   string `json:"error"`
}

// PluginInfo asks the loaded Rex plugin for its version and the sex binary it runs.
func (c *Client) PluginInfo(ctx context.Context) (PluginInfo, error) {
	var info PluginInfo
	out, err := c.read(ctx, nil, "do", "sex_version")
	if err != nil {
		return info, err
	}
	if err := json.Unmarshal(out, &info); err != nil {
		return info, fmt.Errorf("rex: decoding plugin info: %w", err)
	}
	return info, nil
}
