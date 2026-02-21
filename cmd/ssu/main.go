package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime/debug"
	"strings"

	"github.com/pxpxltd/ssu/internal/cli"
	"github.com/pxpxltd/ssu/internal/cli/compat"
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
	// Fall back to git describe / VCS info when ldflags are not set
	// (e.g., plain `go build` or `go run` during development).
	if version == "dev" {
		// Try git describe first (same as Makefile).
		if out, err := exec.Command("git", "describe", "--tags", "--always", "--dirty").Output(); err == nil {
			if v := strings.TrimSpace(string(out)); v != "" {
				version = v
			}
		}

		// Fill in commit and date from Go's embedded VCS info.
		if info, ok := debug.ReadBuildInfo(); ok {
			// Use module version if installed via `go install @version`.
			if version == "dev" && info.Main.Version != "" && info.Main.Version != "(devel)" {
				version = info.Main.Version
			}
			for _, s := range info.Settings {
				switch s.Key {
				case "vcs.revision":
					if len(s.Value) >= 7 {
						commit = s.Value[:7]
					}
				case "vcs.time":
					date = s.Value
				case "vcs.modified":
					if s.Value == "true" && !strings.HasSuffix(commit, "-dirty") {
						commit += "-dirty"
					}
				}
			}
		}
	}

	// Check for old-style bash flags and print migration hints.
	if compat.CheckOldFlags(os.Args, os.Stderr) {
		os.Exit(cli.ExitError)
	}

	root := cli.NewRootCmd(version, commit, date)
	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(cli.ExitError)
	}
}
