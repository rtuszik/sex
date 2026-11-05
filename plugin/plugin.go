// Package plugin embeds the Rex Lua plugin.
package plugin

import (
	_ "embed"
	"fmt"
	"strings"
)

//go:embed sex.lua
var source string

//go:embed run.sh
var runScript string

// Source returns plugin with the binary's version, path, and start script filled in.
func Source(version, bin string) string {
	return fmt.Sprintf("local SEX_VERSION, SEX_BIN, SEX_RUN = %s, %s, %s\n", luaQuote(version), luaQuote(bin), luaQuote(runScript)) + source
}

// luaQuote writes a Lua 5.1 string literal; Lua 5.1 has no \x escapes.
func luaQuote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '"' || c == '\\':
			b.WriteByte('\\')
			b.WriteByte(c)
		case c < 0x20 || c == 0x7f:
			fmt.Fprintf(&b, "\\%03d", c)
		default:
			b.WriteByte(c)
		}
	}
	b.WriteByte('"')
	return b.String()
}
