// Package plugin embeds the Rex Lua plugin.
package plugin

import _ "embed"

//go:embed sex.lua
var Lua []byte
