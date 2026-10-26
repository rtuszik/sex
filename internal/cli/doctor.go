package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/rtuszik/sex/internal/execx"
	"github.com/rtuszik/sex/internal/rex"
	"github.com/rtuszik/sex/internal/sesh"
	"github.com/spf13/cobra"
)

const probeAction = "sex.doctor.probe"

var errChecksFailed = errors.New("some checks failed")

type checker struct {
	w      io.Writer
	failed bool
}

func (c *checker) ok(format string, args ...any) {
	_, _ = fmt.Fprintf(c.w, "ok    "+format+"\n", args...)
}

func (c *checker) warn(format string, args ...any) {
	_, _ = fmt.Fprintf(c.w, "warn  "+format+"\n", args...)
}

func (c *checker) fail(format string, args ...any) {
	c.failed = true
	_, _ = fmt.Fprintf(c.w, "FAIL  "+format+"\n", args...)
}

func newDoctorCmd(g *globals) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check Rex, Remote Control, sesh, zoxide, and fzf",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c := &checker{w: cmd.OutOrStdout()}
			runDoctor(cmd.Context(), c, g, rex.New(execx.OS{}, rex.FindBin()), exec.LookPath)
			if c.failed {
				return errChecksFailed
			}
			return nil
		},
	}
}

type rexDoctor interface {
	Sessions(ctx context.Context) ([]rex.Session, error)
	AppClients(ctx context.Context) ([]rex.AppClient, error)
	Perform(ctx context.Context, clientID, action string, args any) error
}

func runDoctor(ctx context.Context, c *checker, g *globals, r rexDoctor, lookPath func(string) (string, error)) {
	if sessions, err := r.Sessions(ctx); err != nil {
		c.fail("rex server: %v", err)
	} else {
		c.ok("rex server: %d sessions (cli %s)", len(sessions), rex.FindBin())
	}

	apps, err := r.AppClients(ctx)
	switch {
	case err != nil:
		c.fail("rex apps: %v", err)
	case len(apps) == 0:
		c.warn("rex apps: none connected, sex can create sessions but not switch to them")
	}
	for _, app := range apps {
		// An unknown action is answered with not_supported only when the app accepts actions.
		err := r.Perform(ctx, app.ID, probeAction, nil)
		switch {
		case rex.IsRemoteControlOff(err):
			c.fail("%s: Remote Control is off, turn it on in Rex Settings > Rex Server", app.Label)
		case err == nil || rex.IsCode(err, "not_supported"):
			c.ok("%s: Remote Control is on (%s)", app.Label, app.ID)
		default:
			c.warn("%s: Remote Control unknown: %v", app.Label, err)
		}
	}

	for _, bin := range []string{"sesh", "zoxide", "fzf", "git"} {
		path, err := lookPath(bin)
		switch {
		case err != nil && bin == "sesh":
			c.fail("sesh: not found, sex reads config and zoxide entries from it")
		case err != nil:
			c.warn("%s: not found", bin)
		default:
			c.ok("%s: %s", bin, path)
		}
	}

	cfgPath, err := sesh.ConfigPath(g.seshConfig)
	if err != nil {
		c.fail("sesh config: %v", err)
		return
	}
	if _, err := os.Stat(cfgPath); err != nil {
		c.warn("sesh config: %s not found, using defaults", cfgPath)
		return
	}
	if _, err := sesh.LoadConfig(cfgPath); err != nil {
		c.fail("sesh config: %v", strings.TrimSpace(err.Error()))
		return
	}
	c.ok("sesh config: %s", cfgPath)
}
