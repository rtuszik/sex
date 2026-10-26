package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rtuszik/sex/internal/app"
	"github.com/rtuszik/sex/plugin"
	"github.com/spf13/cobra"
)

func newListCmd(g *globals) *cobra.Command {
	var opts app.ListOptions
	var tmux, icons, noColor, asJSON bool
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "l"},
		Short:   "List Rex sessions, sesh config sessions, and zoxide directories",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a, err := g.app(cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			opts.Rex = opts.Rex || tmux
			opts.Client = g.client
			items, err := a.List(cmd.Context(), opts)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if asJSON {
				if items == nil {
					items = []app.Item{}
				}
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				return enc.Encode(items)
			}
			for _, it := range items {
				if _, err := fmt.Fprintln(out, app.Format(it, icons, !noColor)); err != nil {
					return err
				}
			}
			return nil
		},
	}
	f := cmd.Flags()
	f.BoolVarP(&opts.Rex, "rex", "r", false, "show running Rex sessions")
	f.BoolVarP(&tmux, "tmux", "t", false, "alias for --rex")
	f.BoolVarP(&opts.Config, "config", "c", false, "show sesh config sessions")
	f.BoolVarP(&opts.Zoxide, "zoxide", "z", false, "show zoxide directories")
	f.BoolVarP(&opts.HideCurrent, "hide-current", "H", false, "hide the session on screen")
	f.BoolVarP(&icons, "icons", "i", false, "show source icons")
	f.BoolVarP(&noColor, "no-color", "n", false, "show icons without color")
	f.BoolVarP(&asJSON, "json", "j", false, "print JSON")
	_ = f.MarkHidden("tmux")
	return cmd
}

func newConnectCmd(g *globals) *cobra.Command {
	var opts app.ConnectOptions
	cmd := &cobra.Command{
		Use:     "connect NAME",
		Aliases: []string{"cn"},
		Short:   "Switch to a session, creating it from sesh config, zoxide, or a directory",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := g.app(cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			opts.Client = g.client
			_, err = a.Connect(cmd.Context(), strings.Join(args, " "), opts)
			return err
		},
	}
	cmd.Flags().StringVarP(&opts.Command, "command", "c", "", "run COMMAND instead of the startup command")
	cmd.Flags().BoolVarP(&opts.Background, "background", "b", false, "create the session without switching to it")
	return cmd
}

func newLastCmd(g *globals) *cobra.Command {
	var print bool
	cmd := &cobra.Command{
		Use:   "last",
		Short: "Switch to the session shown before the current one",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a, err := g.app(cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			if print {
				_, id, err := a.Last(cmd.Context(), g.client)
				if err != nil {
					return err
				}
				_, err = fmt.Fprintln(cmd.OutOrStdout(), id)
				return err
			}
			_, err = a.SwitchLast(cmd.Context(), g.client)
			return err
		},
	}
	cmd.Flags().BoolVarP(&print, "print", "p", false, "print the session ID instead of switching")
	return cmd
}

func newPreviewCmd(g *globals) *cobra.Command {
	return &cobra.Command{
		Use:   "preview NAME",
		Short: "Print a session's screen or a directory preview",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := g.app(cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			return a.Preview(cmd.Context(), strings.Join(args, " "), cmd.OutOrStdout())
		},
	}
}

func newKillCmd(g *globals) *cobra.Command {
	return &cobra.Command{
		Use:   "kill NAME",
		Short: "Kill a running Rex session",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := g.app(cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			_, err = a.Kill(cmd.Context(), strings.Join(args, " "))
			return err
		},
	}
}

func newRootCmd(g *globals) *cobra.Command {
	return &cobra.Command{
		Use:   "root",
		Short: "Print the git root name of the current session",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a, err := g.app(cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			name, err := a.Root(cmd.Context(), g.client)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), name)
			return err
		},
	}
}

func newPluginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "plugin",
		Short: "Print the Rex plugin (save as ~/.config/rex/sex.lua)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := cmd.OutOrStdout().Write(plugin.Lua)
			return err
		},
	}
}
