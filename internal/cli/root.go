package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/rtuszik/sex/internal/app"
	"github.com/rtuszik/sex/internal/execx"
	"github.com/rtuszik/sex/internal/namer"
	"github.com/rtuszik/sex/internal/rex"
	"github.com/rtuszik/sex/internal/sesh"
	"github.com/rtuszik/sex/internal/state"
	"github.com/rtuszik/sex/internal/zoxide"
	"github.com/spf13/cobra"
)

type globals struct {
	seshConfig    string
	client        string
	pluginVersion string
	build         func(stderr io.Writer) (*app.App, error)
	reload        func(ctx context.Context) error
}

func NewRoot(version string) *cobra.Command {
	return newRoot(version, &globals{})
}

func newRoot(version string, g *globals) *cobra.Command {
	root := &cobra.Command{
		Use:           "sex",
		Short:         "Session manager for Rex, powered by sesh and zoxide",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: false,
		PersistentPreRun: func(cmd *cobra.Command, _ []string) {
			if err := g.reloadOnUpgrade(cmd.Context(), version); err != nil {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "sex: reloading the Rex config after an upgrade: %v\n", err)
			}
		},
	}
	root.PersistentFlags().StringVar(&g.seshConfig, "sesh-config", "", "path to sesh.toml")
	root.PersistentFlags().StringVar(&g.client, "client", "", "Rex app client to switch, as ID or label (default: $REX_CLIENT or the only app)")
	root.PersistentFlags().StringVar(&g.pluginVersion, "plugin-version", "", "version of the Rex plugin that started sex")
	_ = root.PersistentFlags().MarkHidden("plugin-version")
	root.AddCommand(
		newListCmd(g),
		newConnectCmd(g),
		newLastCmd(g),
		newPreviewCmd(g),
		newKillCmd(g),
		newRootCmd(g),
		newPickCmd(g),
		newActivityCmd(g),
		newDoctorCmd(g, version),
		newPluginCmd(version),
	)
	return root
}

func (g *globals) app(stderr io.Writer) (*app.App, error) {
	if g.build != nil {
		return g.build(stderr)
	}
	run := execx.OS{}
	cfgPath, err := sesh.ConfigPath(g.seshConfig)
	if err != nil {
		return nil, err
	}
	cfg, err := sesh.LoadConfig(cfgPath)
	if err != nil {
		return nil, err
	}
	statePath, err := state.DefaultPath()
	if err != nil {
		return nil, err
	}
	home, _ := os.UserHomeDir()
	cwd, _ := os.Getwd()
	return &app.App{
		Rex:    rex.New(run, rex.FindBin()),
		Sesh:   sesh.New(run, "", g.seshConfig),
		Namer:  namer.New(namer.ExecGit{Run: run}, cfg, home),
		Zoxide: zoxide.New(run),
		Store:  state.NewStore(statePath),
		Shell:  shell{},
		Config: cfg,
		Home:   home,
		Cwd:    cwd,
		Getenv: os.Getenv,
		Stderr: stderr,
	}, nil
}

type shell struct{}

// Run executes the preview_command from the user's own sesh.toml; the substituted path is shell-quoted.
func (shell) Run(ctx context.Context, dir, command string, w io.Writer) error {
	cmd := exec.CommandContext(ctx, "sh", "-c", command) // noopengrep: go.lang.security.audit.dangerous-exec-command.dangerous-exec-command
	cmd.Dir = dir
	cmd.Stdout = w
	cmd.Stderr = w
	return cmd.Run()
}
