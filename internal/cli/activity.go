package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/rtuszik/sex/internal/app"
	"github.com/spf13/cobra"
)

func newActivityCmd(g *globals) *cobra.Command {
	var active, asJSON, rows bool
	cmd := &cobra.Command{
		Use:     "activity",
		Aliases: []string{"a"},
		Short:   "List terminals whose programs report a status, most urgent first",
		Long: "List terminals whose programs report a status (OSC 7501), the same data as Rex's Activity view.\n" +
			"Blocked programs come first, then errors, working, done, and idle.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a, err := g.app(cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			items, err := a.Activity(cmd.Context(), !active)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if asJSON {
				if items == nil {
					items = []app.ActivityItem{}
				}
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				return enc.Encode(items)
			}
			for _, it := range items {
				row := app.FormatActivity(it, rows)
				if !rows {
					_, row, _ = strings.Cut(row, "\t")
				}
				if _, err := fmt.Fprintln(out, row); err != nil {
					return err
				}
			}
			return nil
		},
	}
	f := cmd.Flags()
	f.BoolVarP(&active, "active", "A", false, "hide idle programs")
	f.BoolVarP(&asJSON, "json", "j", false, "print JSON")
	f.BoolVar(&rows, "rows", false, "print picker rows")
	_ = f.MarkHidden("rows")
	cmd.AddCommand(newActivityPickCmd(g), newActivityPreviewCmd(g))
	return cmd
}

func newActivityPickCmd(g *globals) *cobra.Command {
	var active bool
	cmd := &cobra.Command{
		Use:   "pick [-- FZF_OPTIONS...]",
		Short: "Pick a terminal from the activity list with fzf and switch to it",
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := g.app(cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			self, err := os.Executable()
			if err != nil {
				return err
			}
			items, err := a.Activity(cmd.Context(), !active)
			if err != nil {
				return err
			}
			var rows strings.Builder
			for _, it := range items {
				rows.WriteString(app.FormatActivity(it, true) + "\n")
			}
			fzf := exec.CommandContext(cmd.Context(), "fzf", activityFzfArgs(self, g.client, active, bgColor(), args)...)
			fzf.Stdin = strings.NewReader(rows.String())
			choice, err := runFzf(fzf)
			if err != nil || choice == "" {
				return err
			}
			return a.FocusActivity(cmd.Context(), g.client, choice)
		},
	}
	cmd.Flags().BoolVarP(&active, "active", "A", false, "hide idle programs")
	return cmd
}

func newActivityPreviewCmd(g *globals) *cobra.Command {
	return &cobra.Command{
		Use:    "preview BLOCK",
		Short:  "Print the screen of a terminal from the activity list",
		Args:   cobra.ExactArgs(1),
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := g.app(cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			return a.PreviewActivity(cmd.Context(), args[0], cmd.OutOrStdout())
		},
	}
}

func activityFzfArgs(self, client string, active bool, bg string, extra []string) []string {
	sex := shellQuote(self)
	if client != "" {
		sex += " --client " + shellQuote(client)
	}
	list := sex + " activity --rows"
	if active {
		list += " --active"
	}
	args := append(fzfStyle(" activity ", "^r refresh  ^/ preview", bg),
		"--delimiter", "\t",
		"--with-nth", "2..",
		"--bind", "ctrl-r:reload("+list+")",
		"--preview", sex+" activity preview {1}",
	)
	return append(args, extra...)
}
