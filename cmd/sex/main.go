package main

import (
	"os"
	"runtime/debug"

	"github.com/rtuszik/sex/internal/cli"
)

var version = "dev"

func main() {
	if err := cli.NewRoot(resolveVersion(version, debug.ReadBuildInfo)).Execute(); err != nil {
		os.Exit(1)
	}
}

// resolveVersion falls back to the module version that `go install pkg@version` records.
func resolveVersion(v string, readBuildInfo func() (*debug.BuildInfo, bool)) string {
	if v != "dev" {
		return v
	}
	if info, ok := readBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return v
}
