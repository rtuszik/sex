package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/rtuszik/sex/internal/app"
	"github.com/rtuszik/sex/internal/termbg"
	"github.com/spf13/cobra"
)

func newPickCmd(g *globals) *cobra.Command {
	return &cobra.Command{
		Use:   "pick [-- FZF_OPTIONS...]",
		Short: "Pick a session with fzf and connect to it",
		Long:  "Pick a session with fzf and connect to it. Options after -- go to fzf, after the defaults.",
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := g.app(cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			self, err := os.Executable()
			if err != nil {
				return err
			}
			items, err := a.List(cmd.Context(), app.ListOptions{Client: g.client})
			if err != nil {
				return err
			}
			var rows strings.Builder
			for _, it := range items {
				rows.WriteString(app.Format(it, true, true) + "\n")
			}
			// Rex floating layers draw the default background as transparent; paint it explicitly.
			bg := ""
			if os.Getenv("REX_BLOCK") != "" {
				bg, _ = termbg.Query()
			}
			fzf := exec.CommandContext(cmd.Context(), "fzf", fzfArgs(self, g.client, bg, args)...)
			fzf.Stdin = strings.NewReader(rows.String())
			fzf.Stderr = os.Stderr
			out, err := fzf.Output()
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) && (exitErr.ExitCode() == 1 || exitErr.ExitCode() == 130) {
				return nil
			}
			if err != nil {
				return err
			}
			choice := strings.TrimSpace(string(out))
			if choice == "" {
				return nil
			}
			_, err = a.Connect(cmd.Context(), choice, app.ConnectOptions{Client: g.client})
			return err
		},
	}
}

func fzfArgs(self, client, bg string, extra []string) []string {
	sex := shellQuote(self)
	if client != "" {
		sex += " --client " + shellQuote(client)
	}
	list := sex + " list --icons"
	args := []string{
		"--no-sort", "--ansi",
		"--layout", "reverse",
		"--style", "full:rounded",
		"--input-label", " sex ",
		"--prompt", "⚡  ",
		"--pointer", "▌",
		"--info", "inline-right",
		"--footer", "^a all  ^t rex  ^g config  ^x zoxide  ^d kill  ^/ preview",
		"--color", fzfColors(bg),
		"--bind", "tab:down,btab:up",
		"--bind", "ctrl-/:toggle-preview",
		"--bind", "ctrl-a:change-prompt(⚡  )+reload(" + list + ")",
		"--bind", "ctrl-t:change-prompt(  )+reload(" + list + " -r)",
		"--bind", "ctrl-g:change-prompt(⚙️  )+reload(" + list + " -c)",
		"--bind", "ctrl-x:change-prompt(📁  )+reload(" + list + " -z)",
		"--bind", "ctrl-d:execute(" + sex + " kill {})+change-prompt(⚡  )+reload(" + list + ")",
		"--preview", sex + " preview {}",
		"--preview-window", "right:55%:hidden",
	}
	return append(args, extra...)
}

// fzfColors uses the 16 theme colors so the picker follows any Rex theme.
// With a known background it also sets every background fzf would leave transparent.
func fzfColors(bg string) string {
	colors := "base16,border:8,label:4:bold,prompt:4,pointer:5,marker:5,spinner:5,hl:3,hl+:3:bold,info:8,header:8,footer:8"
	if bg == "" {
		return colors
	}
	return colors + ",bg:" + bg + ",gutter:" + bg + ",bg+:" + shade(bg)
}

// shade lightens a dark color or darkens a light one, for the current line.
func shade(hex string) string {
	var r, g, b int
	if _, err := fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b); err != nil {
		return hex
	}
	target, amount := 255.0, 0.12
	if 0.299*float64(r)+0.587*float64(g)+0.114*float64(b) > 128 {
		target, amount = 0, 0.08
	}
	mix := func(c int) int { return int(float64(c) + (target-float64(c))*amount + 0.5) }
	return fmt.Sprintf("#%02x%02x%02x", mix(r), mix(g), mix(b))
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
