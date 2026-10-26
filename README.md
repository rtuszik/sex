# sex

A session manager for [Rex](https://www.superlogical.com/rex/), built on
[sesh](https://github.com/joshmedeski/sesh) and [zoxide](https://github.com/ajeetdsouza/zoxide).

sex lists your running Rex sessions together with the sessions in your
`sesh.toml` and your zoxide directories. Pick one with fzf, and sex switches
the Rex app to it. If the session does not exist, sex creates it first, with the
startup command and windows from your sesh config.

sesh stays the source of truth: sex reads `sesh list --json` for config and
zoxide entries, so your existing `sesh.toml` works unchanged.

> Rex is in beta and its API still changes. sex is tested against Rex
> 0.0.1792787366 and sesh 2.33.

## Requirements

- Rex (macOS app or Linux server)
- [sesh](https://github.com/joshmedeski/sesh), [zoxide](https://github.com/ajeetdsouza/zoxide), [fzf](https://github.com/junegunn/fzf), git
- **Remote Control** turned on in Rex Settings > Rex Server, so the CLI can switch
  the app to another session. Key bindings from the Rex plugin can switch
  without it, except for the picker.

Run `sex doctor` to check all of these.

## Install

```sh
go install github.com/rtuszik/sex/cmd/sex@latest
```

Or download a binary from the releases page.

## Set up Rex

Save the plugin next to your Rex config and load it:

```sh
sex plugin > ~/.config/rex/sex.lua
```

```lua
-- ~/.config/rex/init.lua
require("sex").setup()
rex.bind("ctrl+b>shift+t", "sex_pick") -- fzf picker
rex.bind("ctrl+b>shift+l", "sex_last") -- previous session
```

```sh
rex config reload
```

Both actions also appear in the command palette.

`setup` takes these options:

| Option | Default | Description |
| --- | --- | --- |
| `bin` | first `sex` found in Homebrew, `~/go/bin`, `~/.local/bin`, or the mise shims | Path to `sex`. Rex runs commands without your shell's PATH additions, so set it when sex is installed elsewhere. |
| `host` | `"layer"` | Where the picker opens: `"layer"` (floating popup), `"split"`, or `"tab"` |
| `bounds` | `{ 0.1, 0.1, 0.8, 0.8 }` | Popup `{ x, y, w, h }` as fractions of the window |
| `clip_fix` | `true` | Work around Rex's layer clipping, see below |
| `direction` | `"horizontal"` | Split direction for `host = "split"` |
| `fzf_args` | `{}` | Extra fzf options |

Current Rex builds have two bugs in floating layers, and sex works around both:

- Rex clips a layer at its width and height measured from the window's corner,
  not from the layer's corner. sex makes the layer larger by its offset and
  gives fzf a right and bottom margin of the clipped size, so the visible popup
  matches `bounds`. Set `clip_fix = false` when Rex fixes the bug.
- A layer's default background is transparent, so `sex pick` asks the terminal
  for its background color and paints fzf with it. Pass your own
  `--color bg:...` in `fzf_args` to override it.

## Usage

| Command | Description |
| --- | --- |
| `sex list` | List sessions. Filter with `-r` (Rex), `-c` (config), `-z` (zoxide). Add `-i` for icons, `-H` to hide the current session, `-j` for JSON. |
| `sex connect NAME` | Switch to a session, creating it when needed. `-b` creates without switching. `-c CMD` replaces the startup command. |
| `sex pick` | Open fzf with the list, preview, and key bindings, then connect. Options after `--` go to fzf. |
| `sex last` | Switch to the session you were in before. `--print` prints its ID instead. |
| `sex preview NAME` | Print a session's screen, or run the preview command for a directory. |
| `sex kill NAME` | Kill a running Rex session. |
| `sex root` | Print the git root name of the current session. |
| `sex doctor` | Check Rex, Remote Control, sesh, zoxide, and fzf. |
| `sex plugin` | Print the Rex Lua plugin. |

`--client` selects the Rex app to switch when more than one is connected. It
defaults to `$REX_CLIENT`, then to the only connected app.

### Picker keys

| Key | Action |
| --- | --- |
| `ctrl-a` | All sources |
| `ctrl-t` | Rex sessions |
| `ctrl-g` | sesh config sessions |
| `ctrl-x` | zoxide directories |
| `ctrl-d` | Kill the selected Rex session |
| `ctrl-/` | Show or hide the preview (hidden at start) |
| `tab` / `shift-tab` | Down / up |

### Your own fzf command

```sh
sex connect "$(sex list --icons | fzf --ansi --preview 'sex preview {}')"
```

### zsh widget

```zsh
function sex-sessions() {
  sex pick -- --height 40% --reverse
  zle reset-prompt
}
zle -N sex-sessions
bindkey '\es' sex-sessions
```

## How it works

- **Sources.** Rex sessions come from the Rex server. Config sessions and zoxide
  directories come from `sesh list -c -z --json`, with sesh's wildcards, aliases,
  and `default_session` already applied.
- **Duplicates.** A config session is hidden while a Rex session with the same
  name runs. A zoxide directory is hidden while a Rex session runs in it.
- **Names.** Directories are named the way sesh names them: git repository,
  `dir_length`, `git_dir_length`, and `name_substitution` from `sesh.toml`.
  Rex labels allow `.` and `:`, so sex keeps them.
- **Windows.** `windows` from your sesh config become Rex tabs. The first tab
  runs the startup command and stays focused.
- **Paths.** Rex does not store a session's directory, so sex records it in
  `$XDG_STATE_HOME/sex/state.json`. For sessions sex did not create, it uses the
  working directory of the visible terminal.
- **Last session.** Rex records when the app last sized each terminal. sex uses
  that time to order sessions by when you last looked at them, including
  sidebar clicks. This is not a documented Rex API, so treat it as best effort.

## Differences from sesh

- tmuxinator, tmuxp, `sesh window`, and `sesh worktree` are not supported.
- `[tui]`, `cache`, and `tmux_command` have no effect.

## License

MIT. `internal/namer` is derived from sesh, see
[THIRD_PARTY_LICENSES.md](THIRD_PARTY_LICENSES.md).
