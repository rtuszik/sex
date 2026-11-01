# Starts sex on the host that runs the picker, which can be a remote Rex server. $1 is the plugin host's sex.
b=$1
shift
PATH="/opt/homebrew/bin:/usr/local/bin:$HOME/go/bin:$HOME/.local/bin:$HOME/.nix-profile/bin:/etc/profiles/per-user/$USER/bin:/run/current-system/sw/bin:$HOME/.local/share/mise/shims:$HOME/.asdf/shims:$PATH"
export PATH
if [ ! -x "$b" ]; then
  b=$(command -v sex) || {
    echo "sex: not found on $(uname -n); install it or add it to PATH" >&2
    exit 127
  }
fi
exec "$b" "$@"
