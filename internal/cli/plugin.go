package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rtuszik/sex/internal/execx"
	"github.com/rtuszik/sex/internal/rex"
	"github.com/rtuszik/sex/plugin"
	"github.com/spf13/cobra"
)

func newPluginCmd(version string) *cobra.Command {
	var source bool
	cmd := &cobra.Command{
		Use:   "plugin",
		Short: "Print the Rex plugin loader",
		Long: "Print the Rex plugin loader. Save it where Rex's require finds it, for example as\n" +
			"sex.lua next to init.lua. On each config load it asks this binary for the plugin,\n" +
			"so upgrading sex needs no new copy. --source prints the plugin itself.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !source {
				_, err := cmd.OutOrStdout().Write(plugin.Loader)
				return err
			}
			exe, err := executable()
			if err != nil {
				return err
			}
			_, err = fmt.Fprint(cmd.OutOrStdout(), plugin.Source(version, exe))
			return err
		},
	}
	cmd.Flags().BoolVar(&source, "source", false, "print the plugin code instead of the loader")
	cmd.AddCommand(newPluginInstallCmd())
	return cmd
}

// executable resolves shims and symlinks, so the plugin calls the real binary.
func executable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		return real, nil
	}
	return exe, nil
}

func newPluginInstallCmd() *cobra.Command {
	var path string
	var force bool
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Write the plugin loader next to the Rex config",
		Long: "Write the plugin loader. The default path is sex.lua in the directory of $REX_CONFIG,\n" +
			"or $XDG_CONFIG_HOME/rex, or ~/.config/rex.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if path == "" {
				p, err := defaultPluginPath()
				if err != nil {
					return err
				}
				path = p
			}
			if _, err := os.Stat(path); err == nil && !force {
				return fmt.Errorf("%s exists; pass --force to replace it", path)
			}
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(path, plugin.Loader, 0o644); err != nil {
				return err
			}
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "wrote %s\nload it with require(\"sex\").setup() and run rex config reload\n", path)
			return err
		},
	}
	cmd.Flags().StringVar(&path, "path", "", "file to write")
	cmd.Flags().BoolVar(&force, "force", false, "replace an existing file")
	return cmd
}

func defaultPluginPath() (string, error) {
	if c := os.Getenv("REX_CONFIG"); c != "" {
		return filepath.Join(filepath.Dir(c), "sex.lua"), nil
	}
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "rex", "sex.lua"), nil
}

// reloadOnUpgrade reloads the Rex config when the plugin that started sex is older than sex.
func (g *globals) reloadOnUpgrade(ctx context.Context, version string) error {
	if g.pluginVersion == "" || g.pluginVersion == version {
		return nil
	}
	if g.reload != nil {
		return g.reload(ctx)
	}
	return rex.New(execx.OS{}, rex.FindBin()).Call(ctx, "server.config_reload", nil, nil)
}
