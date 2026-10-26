package main

import (
	"os"

	"github.com/rtuszik/sex/internal/cli"
)

var version = "dev"

func main() {
	if err := cli.NewRoot(version).Execute(); err != nil {
		os.Exit(1)
	}
}
