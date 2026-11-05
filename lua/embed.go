// Package lua embeds the Rex plugin loader, the entry point for plugin managers.
package lua

import _ "embed"

//go:embed sex.lua
var Loader []byte
