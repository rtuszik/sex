package zoxide

import (
	"context"
	"strings"

	"github.com/rtuszik/sex/internal/execx"
)

type Client struct {
	run execx.Runner
	bin string
}

func New(run execx.Runner) *Client { return &Client{run: run, bin: "zoxide"} }

func (c *Client) Add(ctx context.Context, path string) error {
	_, err := c.run.Run(ctx, execx.Cmd{Name: c.bin, Args: []string{"add", path}})
	return err
}

func (c *Client) Query(ctx context.Context, query string) (string, error) {
	out, err := c.run.Run(ctx, execx.Cmd{Name: c.bin, Args: []string{"query", query}})
	return strings.TrimSpace(string(out)), err
}
