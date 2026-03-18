// Package main is the entry point for the preflight binary.
package main

import (
	"os"

	"github.com/GyeongHoKim/preflight/internal/cli"
)

// version, commit, and buildDate are injected at build time via -ldflags.
var (
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
)

func main() {
	os.Exit(cli.Execute(version, commit, buildDate))
}
