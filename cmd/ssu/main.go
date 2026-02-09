package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/pxpxltd/ssu/internal/cli"
)

// Set via ldflags at build time:
//
//	go build -ldflags "-X main.version=1.2.0 -X main.commit=abc123 -X main.date=2026-02-09"
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	// Fall back to VCS info embedded by Go toolchain when ldflags are not set
	// (e.g., installed via `go install`).
	if version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok {
			for _, s := range info.Settings {
				switch s.Key {
				case "vcs.revision":
					if len(s.Value) >= 7 {
						commit = s.Value[:7]
					}
				case "vcs.time":
					date = s.Value
				case "vcs.modified":
					if s.Value == "true" {
						commit += "-dirty"
					}
				}
			}
		}
	}

	root := cli.NewRootCmd(version, commit, date)
	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
