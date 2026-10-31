# sex

sex is a session manager for the Rex terminal multiplexer. It lists running
Rex sessions together with the sessions defined in sesh.toml and the
directories known to zoxide, creates a session when the selected one does
not exist, and switches the Rex app to it. It is written in Go.

sex does not read sesh configuration on its own. It takes config and zoxide
entries from `sesh list --json`, so an existing sesh.toml works unchanged.

## Requirements

- Rex. Floating pickers need app build 1531 or later.
- sesh 2.33 or later, zoxide, fzf, and git.
- Go 1.26 or later, to build from source.

The Rex app must have Remote Control enabled (Settings, Rex Server) for the
command line to switch sessions. Run `sex doctor` to check the setup.

## Installation

Release binaries for macOS and Linux are published on GitHub. With mise:

```sh
mise use -g github:rtuszik/sex
```

From source:

```sh
go install github.com/rtuszik/sex/cmd/sex@latest
```

## Rex plugin

The plugin adds three actions to Rex: `sex_pick` opens the session picker in a
floating layer, `sex_activity` opens the activity picker, and `sex_last`
switches to the previously shown session.

`sex plugin` prints a small loader. Save it anywhere Rex's `require` finds
it; `sex plugin install` writes it to sex.lua in the directory of
`$REX_CONFIG`, `$XDG_CONFIG_HOME/rex`, or `~/.config/rex`, and accepts
`--path`. Load it from init.lua and bind the actions:

```lua
require("sex").setup()
rex.bind("ctrl+b>shift+t", "sex_pick")
rex.bind("ctrl+b>shift+l", "sex_last")
rex.bind("ctrl+b>shift+a", "sex_activity")
```

Then run `rex config reload`. On each configuration load the loader runs
`sex plugin --source`, so the plugin always matches the installed binary.
`sex plugin --source` prints the plugin itself, for those who prefer to keep
a copy.

`setup` accepts a table with these fields:

```text
bin        Path to sex. Default: $SEX_BIN, then the first sex found in
           /opt/homebrew/bin, /usr/local/bin, ~/go/bin, ~/.local/bin, the
           Nix profiles, and the mise and asdf shims. Rex does not run
           commands with the PATH of an interactive shell.
host       "layer" (default), "split", or "tab".
bounds     Layer position { x, y, w, h } as fractions of the window.
           Default: { 0.1, 0.1, 0.8, 0.8 }.
direction  "horizontal" (default) or "vertical", for host = "split".
fzf_args   Extra fzf options, appended after the defaults.
```

`sex_last` switches through the app and works without Remote Control.
`sex_pick` and `sex_activity` do not.

## Upgrading

Upgrade the binary with the tool that installed it, for example
`mise upgrade` or `go install github.com/rtuszik/sex/cmd/sex@latest`. The
plugin passes its version to every command it starts; when sex is newer, it
reloads the Rex configuration once, and the next action uses the new plugin.
`sex doctor` reports a plugin that does not match the binary, including a
second copy of sex that the plugin still runs.

## Usage

```text
sex list [-r] [-c] [-z] [-i] [-H] [-j]
    List Rex sessions (-r), sesh config sessions (-c), and zoxide
    directories (-z); all three by default. -i adds source icons, -H hides
    the session on screen, -j prints JSON.

sex connect [-b] [-c command] name
    Switch to the session called name. If it does not exist, create it
    from the matching config entry, zoxide entry, or directory. -b creates
    the session without switching. -c replaces the startup command.

sex pick [-- fzf options]
    Select a session with fzf and connect to it.

sex last [-p]
    Switch to the session shown before the current one. -p prints its ID.

sex activity [-A] [-j]
    List terminals whose programs report a status (OSC 7501), the data
    behind Rex's Activity view. Blocked programs come first, then errors,
    working, done, and idle. -A hides idle programs.

sex activity pick [-A] [-- fzf options]
    Select a terminal from that list with fzf. sex focuses it and switches
    the app to its session and tab.

sex preview name     Print a session's screen or a directory preview.
sex kill name        Destroy a running Rex session.
sex root             Print the git root name of the current session.
sex doctor           Check Rex, Remote Control, sesh, zoxide, and fzf.
sex plugin           Print the Rex plugin.
```

The global option `--client` selects the Rex app to switch when more than
one is connected. It defaults to `$REX_CLIENT`, then to the only app.

In the picker, ctrl-a, ctrl-t, ctrl-g, and ctrl-x show all sources, Rex
sessions, config sessions, and zoxide directories. ctrl-d destroys the
selected Rex session. ctrl-/ toggles the preview. In the activity picker,
ctrl-r reloads the list.

To use a different fzf command:

```sh
sex connect "$(sex list | fzf --preview 'sex preview {}')"
```

## Notes

Config sessions are hidden while a Rex session with the same name runs.
zoxide directories are hidden while a Rex session runs in them.

Directories are named as sesh names them. Characters that tmux rejects, such
as `.` and `:`, are kept.

The `windows` of a config session become Rex tabs. The first tab runs the
startup command and keeps focus.

Rex does not record a session's directory. sex stores it in
`$XDG_STATE_HOME/sex/state.json` and falls back to the working directory of
the visible terminal for sessions it did not create.

`sex last` orders sessions by the time the app last sized their visible
terminal. This includes switches made from the Rex sidebar. The field is not
a documented Rex interface.

Inside Rex, `sex pick` reads the terminal background color (OSC 11) and
passes it to fzf.

## sesh compatibility

sex reads config and zoxide entries through `sesh list --json`, so sesh
resolves these sesh.toml keys before sex sees them: `[[session]]` (name,
path, startup_command, preview_command, windows, disable_startup_command,
alias, icon), `[[window]]`, `[[wildcard]]`, `[default_session]`, `import`,
`blacklist`, and `sort_order`. sex reads `dir_length`, `git_dir_length`,
`git_namer_use_worktree_root`, and `name_substitution` itself, from the main
sesh.toml only, as sesh does.

These keys behave differently:

```text
cache        sex list, sex pick, and sex preview use the sesh cache when
             cache = true. sex connect always reads live data.
blacklist    Also applied to Rex sessions.
sort_order   Applies to config and zoxide entries. Rex sessions are always
             listed first; the position of "tmux" is ignored.
[frecency]   Used by sesh for the zoxide list. sex runs zoxide add and
             zoxide query directly, so other backends are not used when
             connecting.
alias        Resolved by sex connect. alias_auto_connect and the [tui]
             alias options belong to the sesh picker and have no effect.
[icons]      Not used. sex shows its own source icons, or the icon of the
             session or wildcard.
```

These keys have no effect: `tmux_command`, `terminal`, `[tui]`, `[browser]`,
`[[worktree]]`, and the tmuxinator and tmuxp fields of `[[session]]`.

The sesh commands window, worktree, clone, mkdir, and rename have no sex
equivalent. `sex pick` uses fzf instead of the sesh picker.

## Building

```sh
mise run build
mise run check
```

`check` runs golangci-lint, the tests, zizmor, and opengrep. Release history
is in CHANGELOG.md.

## Reporting bugs

Open an issue at <https://github.com/rtuszik/sex/issues>. Include the output
of `sex doctor` and `rex version`.

## License

sex is distributed under the MIT License. See LICENSE. The package
`internal/namer` is derived from sesh; its license is in
THIRD_PARTY_LICENSES.md.
