# Phase 1: Foundation - Research

**Researched:** 2026-02-09
**Domain:** Go CLI framework (Cobra), project structure, terminal output, version injection
**Confidence:** HIGH

## Summary

This phase establishes the Go project skeleton: module initialization, Cobra CLI with subcommand routing, version injection via ldflags + `debug.ReadBuildInfo`, shell completions, NO_COLOR/TTY detection, and backwards compatibility hints for old bash-era flags.

The standard stack is well-established and heavily documented: Cobra v1.10.x for CLI, `fatih/color` for terminal coloring (with NO_COLOR built in), `mattn/go-isatty` for TTY detection (pulled in transitively by fatih/color), and Go's built-in `runtime/debug` + ldflags for version injection. The Go 1.21+ minimum is more than satisfied by the installed Go 1.25.6.

For the interactive root menu (Claude's Discretion item), this research recommends a minimal custom implementation using `fatih/color` + raw stdin reads for Phase 1, deferring to Bubble Tea in Phase 5 when the full TUI is built. This avoids pulling in a heavy TUI framework for a simple 6-item menu.

**Primary recommendation:** Use cobra v1.10.x + fatih/color. Keep Phase 1 focused on the CLI skeleton -- stub all RunE functions to print "not implemented yet" and return nil. Wire version, completion, and backwards compat hints fully.

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/spf13/cobra` | v1.10.2 | CLI framework, subcommand routing, flag parsing, help generation | Used by kubectl, docker, gh, hugo. 173K+ projects. Published Dec 2025. |
| `github.com/spf13/pflag` | v1.0.6+ | POSIX flag parsing (cobra dependency) | Pulled in by cobra, provides short+long flag forms (`-v/--verbose`) |
| `github.com/fatih/color` | v1.18.0 | Terminal color output with NO_COLOR support | 26K+ importers, built-in NO_COLOR + TTY detection, handles Windows |
| `github.com/mattn/go-isatty` | v0.0.20+ | TTY detection | Transitive dependency of fatih/color, 5.4K importers |

### Supporting (Phase 1 only)

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `runtime/debug` | stdlib | Build info (vcs.revision, vcs.time) | Fallback when ldflags not provided (e.g., `go install`) |
| `log/slog` | stdlib (Go 1.21+) | Structured logging | Internal logging in later phases, but set up the foundation now |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `fatih/color` | `gookit/color` | gookit has more features (256/RGB) but fatih/color is simpler, more widely used, and sufficient. fatih/color is already a cobra ecosystem staple. |
| `fatih/color` | `muesli/termenv` + `charmbracelet/lipgloss` | Full Charm stack. Overkill for Phase 1. Will be pulled in via bubbletea in Phase 5 anyway. |
| Manual ldflags | `carlmjohnson/versioninfo` | Convenience wrapper. Unnecessary -- 15 lines of code replaces it. |
| Custom TTY check | `golang.org/x/term` | More features (terminal size, raw mode). Not needed until Phase 5 TUI. |

**Installation:**
```bash
go mod init github.com/pxpxltd/ssu
go get github.com/spf13/cobra@v1.10.2
go get github.com/fatih/color@v1.18.0
```

Note: `go-isatty` and `pflag` are pulled in transitively. No need to explicitly `go get` them.

## Architecture Patterns

### Recommended Project Structure

```
ssu/
├── cmd/
│   └── ssu/
│       └── main.go              # Entry point: build vars, root.Execute()
├── internal/
│   ├── cli/                     # Cobra command definitions
│   │   ├── root.go              # Root command, PersistentPreRunE, global flags
│   │   ├── root_test.go
│   │   ├── status.go            # Status subcommand stub
│   │   ├── update.go            # Update subcommand stub
│   │   ├── push.go              # Push subcommand stub
│   │   ├── rollback.go          # Rollback subcommand stub
│   │   ├── backup.go            # Backup subcommand stub
│   │   ├── version.go           # Version subcommand (fully implemented)
│   │   ├── version_test.go
│   │   ├── completion.go        # Shell completion subcommand (fully implemented)
│   │   └── completion_test.go
│   ├── cli/compat/              # Backwards compatibility detection
│   │   ├── compat.go            # Old flag -> new subcommand hint mapper
│   │   └── compat_test.go
│   ├── cli/output/              # Terminal output utilities
│   │   ├── color.go             # Color definitions, NO_COLOR, TTY detection
│   │   ├── color_test.go
│   │   ├── symbols.go           # Unicode symbols (checkmark, cross, arrow, etc.)
│   │   └── printer.go           # Formatted output helpers (Success, Error, Warn, Info)
│   ├── git/                     # (empty in Phase 1 -- Phase 2)
│   ├── engine/                  # (empty in Phase 1 -- Phase 3)
│   ├── config/                  # (empty in Phase 1 -- Phase 4)
│   ├── backup/                  # (empty in Phase 1 -- Phase 4)
│   └── tui/                     # (empty in Phase 1 -- Phase 5)
├── legacy/
│   └── ssu                      # Original bash script (moved from root)
├── go.mod
├── go.sum
├── Makefile                     # Build with ldflags, test, lint
├── CLAUDE.md                    # (existing)
├── README.md                    # (existing)
├── LICENSE                      # (existing)
└── install.sh                   # (existing)
```

### Pattern 1: Cobra Command Organization (One File Per Command)

**What:** Each subcommand lives in its own file under `internal/cli/`. The file defines a `New<Command>Cmd()` function that returns a `*cobra.Command`. The root command adds them all.

**When to use:** Always for cobra CLIs with more than 2 subcommands.

**Example:**
```go
// internal/cli/status.go
package cli

import "github.com/spf13/cobra"

func NewStatusCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "status",
        Short: "Show submodule status",
        Long:  `Display the status of all submodules including branch, commits behind, and modification state.`,
        Example: `  ssu status
  ssu status --json`,
        RunE: func(cmd *cobra.Command, args []string) error {
            // TODO: wire to engine in Phase 5
            cmd.Println("status: not implemented yet")
            return nil
        },
    }
    return cmd
}
```

```go
// internal/cli/root.go
package cli

import "github.com/spf13/cobra"

func NewRootCmd(version, commit, date string) *cobra.Command {
    root := &cobra.Command{
        Use:   "ssu",
        Short: "Smart Submodule Updater",
        Long:  `SSU intelligently manages git submodules with smart branch detection, automatic backups, and conflict handling.`,
        SilenceUsage:  true,
        SilenceErrors: true,
    }

    // Global persistent flags
    root.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
    root.PersistentFlags().BoolP("dry-run", "n", false, "Preview changes without modifying anything")
    root.PersistentFlags().BoolP("auto", "a", false, "Automatic mode (no prompts, for CI/CD)")
    root.PersistentFlags().IntP("jobs", "j", 8, "Number of parallel fetch jobs")

    // Add subcommands
    root.AddCommand(
        NewStatusCmd(),
        NewUpdateCmd(),
        NewPushCmd(),
        NewRollbackCmd(),
        NewBackupCmd(),
        NewVersionCmd(version, commit, date),
        NewCompletionCmd(),
    )

    return root
}
```

### Pattern 2: Version Injection via ldflags + debug.ReadBuildInfo Fallback

**What:** Declare version variables in `main.go` with defaults. Build with ldflags to inject values. Fall back to `debug.ReadBuildInfo` for `go install` builds.

**When to use:** Always. This is the standard Go pattern.

**Example:**
```go
// cmd/ssu/main.go
package main

import (
    "fmt"
    "os"
    "runtime/debug"

    "github.com/pxpxltd/ssu/internal/cli"
)

// Set via ldflags at build time:
//   go build -ldflags "-X main.version=1.2.0 -X main.commit=abc123 -X main.date=2026-02-09"
var (
    version = "dev"
    commit  = "unknown"
    date    = "unknown"
)

func main() {
    // Fall back to VCS info embedded by Go toolchain
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
```

### Pattern 3: Backwards Compatibility Hint via os.Args Pre-Check

**What:** Before cobra parses arguments, scan `os.Args` for old bash-era flags (`--status`, `--push`, `--rollback`, `--dry-run` as standalone). Print a friendly hint suggesting the new subcommand syntax and exit.

**When to use:** Specifically for migrating from the bash `ssu` to the Go `ssu`.

**Why not use cobra's FParseErrWhitelist:** Cobra would treat `--status` as an unknown flag and show generic help. We want a friendly, specific migration message. Pre-checking `os.Args` is simpler and gives full control over the message.

**Example:**
```go
// internal/cli/compat/compat.go
package compat

import (
    "fmt"
    "os"
    "strings"
)

// OldFlagHints maps old bash-era flags to their new subcommand equivalents.
var OldFlagHints = map[string]string{
    "--status":   "ssu status",
    "--push":     "ssu push",
    "--rollback": "ssu rollback",
    "--auto":     "ssu update --auto  (or: ssu push --auto)",
    "--dry-run":  "ssu update --dry-run  (or: ssu status)",
}

// CheckOldFlags inspects os.Args for old-style flags used as the first argument
// (i.e., where the user meant it as a command, not a flag to a subcommand).
// Returns true if an old flag was detected and a hint was printed.
func CheckOldFlags(args []string) bool {
    if len(args) < 2 {
        return false
    }
    first := args[1]
    if hint, ok := OldFlagHints[first]; ok {
        fmt.Fprintf(os.Stderr, "Hint: Did you mean `%s`?\n", hint)
        fmt.Fprintf(os.Stderr, "Run `ssu help` for the new command syntax.\n")
        return true
    }
    return false
}
```

Called in `main.go` before `root.Execute()`:
```go
if compat.CheckOldFlags(os.Args) {
    os.Exit(1)
}
```

### Pattern 4: NO_COLOR + TTY Detection Output Layer

**What:** Centralize color/output decisions in a single package. Check NO_COLOR, TERM=dumb, and TTY status once at startup. Expose color-aware print functions.

**Example:**
```go
// internal/cli/output/color.go
package output

import (
    "os"

    "github.com/fatih/color"
)

// Colors - pre-configured color objects
var (
    Success = color.New(color.FgGreen)
    Error   = color.New(color.FgRed)
    Warning = color.New(color.FgYellow)
    Info    = color.New(color.FgCyan)
    Muted   = color.New(color.FgHiBlack)   // Gray for secondary text
    Bold    = color.New(color.Bold)

    // Status-specific colors (matching bash version semantics)
    Pending  = color.New(color.FgGreen)     // Submodule needs update
    Current  = color.New(color.FgCyan)      // Up to date
    Modified = color.New(color.FgYellow)    // Has local changes
    Ahead    = color.New(color.FgMagenta)   // Unpushed commits
    Conflict = color.New(color.FgRed)       // Merge conflict
)

// IsColorEnabled returns whether color output is active.
// fatih/color handles NO_COLOR, TERM=dumb, and non-TTY automatically.
func IsColorEnabled() bool {
    return !color.NoColor
}

// IsTTY returns whether stdout is connected to a terminal.
func IsTTY() bool {
    return !color.NoColor
}

// ForceDisableColor explicitly turns off all color output.
func ForceDisableColor() {
    color.NoColor = true
}
```

### Pattern 5: Shell Completion Subcommand

**What:** A `completion` subcommand that generates shell scripts for bash, zsh, fish, and powershell.

**Example:**
```go
// internal/cli/completion.go
package cli

import (
    "os"

    "github.com/spf13/cobra"
)

func NewCompletionCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "completion [bash|zsh|fish|powershell]",
        Short: "Generate shell completion script",
        Long: `Generate a completion script for the specified shell.

To load completions:

Bash:
  $ source <(ssu completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ ssu completion bash > /etc/bash_completion.d/ssu
  # macOS:
  $ ssu completion bash > $(brew --prefix)/etc/bash_completion.d/ssu

Zsh:
  $ ssu completion zsh > "${fpath[1]}/_ssu"
  # You may need to restart your shell or run: compinit

Fish:
  $ ssu completion fish | source

  # To load completions for each session, execute once:
  $ ssu completion fish > ~/.config/fish/completions/ssu.fish

PowerShell:
  $ ssu completion powershell | Out-String | Invoke-Expression
`,
        DisableFlagsInUseLine: true,
        ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
        Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
        RunE: func(cmd *cobra.Command, args []string) error {
            switch args[0] {
            case "bash":
                return cmd.Root().GenBashCompletionV2(os.Stdout, true)
            case "zsh":
                return cmd.Root().GenZshCompletion(os.Stdout)
            case "fish":
                return cmd.Root().GenFishCompletion(os.Stdout, true)
            case "powershell":
                return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
            }
            return nil
        },
    }
    return cmd
}
```

### Anti-Patterns to Avoid

- **Putting business logic in command RunE functions:** Commands should be thin wrappers that parse flags and delegate to engine/service packages. In Phase 1, stubs are fine; in later phases, RunE calls engine methods.
- **Using cobra.Command.Run instead of RunE:** Always use `RunE` (error-returning variant). It enables proper error propagation and exit code control.
- **Printing to `fmt.Println` directly in commands:** Use `cmd.Println()` or the output package. This ensures output goes through cobra's configured writers, making testing possible.
- **Global mutable state for flags:** Pass flag values through function parameters or a config struct, not package-level variables. Exception: `color.NoColor` is designed to be a global toggle.
- **Creating empty `internal/` subdirectories with no .go files:** Go ignores empty directories. Either add a `doc.go` placeholder or omit until the phase that needs them. Recommendation: just omit empty package directories. Create them in their respective phases.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Flag parsing with short+long forms | Custom arg parser | `cobra` + `pflag` | POSIX compliance, help generation, completion |
| Terminal color with NO_COLOR | Raw ANSI escape codes | `fatih/color` | Handles NO_COLOR, TERM=dumb, non-TTY, Windows |
| TTY detection | `os.Stat` on stdout | `fatih/color` (uses `go-isatty` internally) | Cross-platform (Linux, macOS, Windows, Cygwin) |
| Shell completion scripts | Manual bash/zsh/fish scripts | `cobra.GenBashCompletionV2()` etc. | Maintained upstream, handles edge cases |
| Version string formatting | String concatenation | Structured version info struct | Consistent output, testable |
| Build info extraction | Only ldflags | ldflags + `debug.ReadBuildInfo` fallback | Works with both `go build` and `go install` |

**Key insight:** The Cobra ecosystem provides nearly everything Phase 1 needs out of the box. The only custom code is the backwards-compatibility hint checker and the output color wrapper.

## Common Pitfalls

### Pitfall 1: Cobra SilenceUsage and SilenceErrors

**What goes wrong:** By default, cobra prints usage text on every error, cluttering output. It also prints the error itself, which can cause duplicate error messages if you also print the error in main.
**Why it happens:** Cobra's defaults are designed for simple CLIs where usage on error is helpful.
**How to avoid:** Set `SilenceUsage: true` and `SilenceErrors: true` on the root command. Handle error display yourself in `main.go`.
**Warning signs:** Error messages appear twice, or help text shows on validation errors.

### Pitfall 2: ldflags Package Path Must Be Fully Qualified

**What goes wrong:** `-X version=1.0` fails silently. The variable stays at its default value.
**Why it happens:** The `-X` flag requires the full package path: `-X main.version=1.0` or `-X github.com/pxpxltd/ssu/internal/pkg.version=1.0`.
**How to avoid:** Always use the full path. Keep version variables in `main` package for simplicity (shortest path). Verify with `go version -m ./ssu` after building.
**Warning signs:** Version shows "dev" in a tagged release build.

### Pitfall 3: Completion Output Must Be Clean

**What goes wrong:** Shell completion breaks silently -- tab produces garbage or errors.
**Why it happens:** Any `fmt.Println` or logging that writes to stdout during completion generation will corrupt the completion script.
**How to avoid:** The completion command should write ONLY the completion script to stdout. Use `SilenceUsage: true`. Don't use `cobra.OnInitialize` for anything that prints to stdout.
**Warning signs:** `source <(ssu completion bash)` produces shell errors.

### Pitfall 4: PersistentPreRunE Inheritance

**What goes wrong:** Root command's `PersistentPreRunE` (e.g., git context check) runs before `version` and `completion` commands, causing "not a git repository" errors when running `ssu version` outside a repo.
**Why it happens:** `PersistentPreRunE` is inherited by ALL child commands.
**How to avoid:** Either (a) check the command name in PersistentPreRunE and skip validation for `version`/`completion`, or (b) don't put the git context check in PersistentPreRunE -- put it in a shared helper that only git-dependent commands call.
**Warning signs:** `ssu version` fails outside a git repo.

### Pitfall 5: NO_COLOR Must Be Empty-String Aware

**What goes wrong:** Program ignores `NO_COLOR=` (set but empty) or responds to it incorrectly.
**Why it happens:** The NO_COLOR spec says: "present and not an empty string". `NO_COLOR=` (empty) should NOT disable color. `NO_COLOR=1` should.
**How to avoid:** `fatih/color` handles this correctly -- it checks `os.Getenv("NO_COLOR") != ""`. Don't add custom NO_COLOR logic that would conflict.
**Warning signs:** Colors disabled when NO_COLOR is not set, or colors enabled when NO_COLOR=1.

### Pitfall 6: Exit Codes Lost in Cobra

**What goes wrong:** Program always exits 0 even on errors.
**Why it happens:** `root.Execute()` returns an error, but if you don't check it and call `os.Exit()`, the process exits 0.
**How to avoid:** In `main()`, check the error from `root.Execute()` and exit with the appropriate code. Define exit code constants.
**Warning signs:** CI pipelines don't detect failures.

## Code Examples

### Exit Code Constants

```go
// internal/cli/exitcodes.go
package cli

const (
    ExitSuccess  = 0  // Operation completed successfully
    ExitError    = 1  // General error (invalid args, git failure, etc.)
    ExitConflict = 2  // Merge conflict detected
)
```

### Makefile with ldflags

```makefile
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS  = -s -w \
           -X main.version=$(VERSION) \
           -X main.commit=$(COMMIT) \
           -X main.date=$(DATE)

.PHONY: build test lint clean

build:
	go build -ldflags "$(LDFLAGS)" -o ssu ./cmd/ssu

test:
	go test ./...

lint:
	golangci-lint run

clean:
	rm -f ssu
```

### Version Command

```go
// internal/cli/version.go
package cli

import (
    "fmt"
    "runtime"

    "github.com/spf13/cobra"
)

func NewVersionCmd(version, commit, date string) *cobra.Command {
    cmd := &cobra.Command{
        Use:   "version",
        Short: "Print version information",
        Args:  cobra.NoArgs,
        Run: func(cmd *cobra.Command, args []string) {
            fmt.Fprintf(cmd.OutOrStdout(), "ssu %s\n", version)
            fmt.Fprintf(cmd.OutOrStdout(), "  commit: %s\n", commit)
            fmt.Fprintf(cmd.OutOrStdout(), "  built:  %s\n", date)
            fmt.Fprintf(cmd.OutOrStdout(), "  go:     %s\n", runtime.Version())
        },
    }
    return cmd
}
```

### Testing Cobra Commands

```go
// internal/cli/version_test.go
package cli

import (
    "bytes"
    "strings"
    "testing"
)

func TestVersionCmd(t *testing.T) {
    root := NewRootCmd("1.2.3", "abc1234", "2026-02-09")
    buf := new(bytes.Buffer)
    root.SetOut(buf)
    root.SetArgs([]string{"version"})

    err := root.Execute()
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    output := buf.String()
    if !strings.Contains(output, "1.2.3") {
        t.Errorf("expected version in output, got: %s", output)
    }
    if !strings.Contains(output, "abc1234") {
        t.Errorf("expected commit in output, got: %s", output)
    }
}
```

### Backwards Compat Hint Test

```go
// internal/cli/compat/compat_test.go
package compat

import "testing"

func TestCheckOldFlags(t *testing.T) {
    tests := []struct {
        args     []string
        detected bool
    }{
        {[]string{"ssu", "--status"}, true},
        {[]string{"ssu", "--push"}, true},
        {[]string{"ssu", "status"}, false},
        {[]string{"ssu"}, false},
        {[]string{"ssu", "update", "--dry-run"}, false}, // --dry-run as flag to subcommand is fine
    }

    for _, tt := range tests {
        got := CheckOldFlags(tt.args)
        if got != tt.detected {
            t.Errorf("CheckOldFlags(%v) = %v, want %v", tt.args, got, tt.detected)
        }
    }
}
```

### Interactive Root Menu (Minimal -- Phase 1)

For Phase 1, the root command's RunE should show a simple numbered menu when run interactively (TTY), or show help when piped. This keeps Phase 1 lightweight. The full Bubble Tea TUI replaces this in Phase 5.

```go
// internal/cli/root.go (RunE for root command)
RunE: func(cmd *cobra.Command, args []string) error {
    // If not a TTY, just show help
    if color.NoColor || !isatty.IsTerminal(os.Stdout.Fd()) {
        return cmd.Help()
    }
    // Simple interactive menu for TTY
    fmt.Println("What would you like to do?")
    fmt.Println()
    fmt.Println("  1) status    - Show submodule status")
    fmt.Println("  2) update    - Update submodules")
    fmt.Println("  3) push      - Push ahead submodules")
    fmt.Println("  4) rollback  - Rollback from backup")
    fmt.Println("  5) backup    - Manage backups")
    fmt.Println("  6) help      - Show full help")
    fmt.Println()
    fmt.Print("Choose [1-6]: ")
    // Read single character, dispatch to subcommand
    // (implementation detail -- ~20 lines of bufio.Scanner)
    return nil
},
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `GenBashCompletion()` | `GenBashCompletionV2()` | cobra v1.8+ | V2 uses new completion API with descriptions, V1 is legacy |
| ldflags only for version | ldflags + `debug.ReadBuildInfo` fallback | Go 1.18 (2022) | `go install` builds get VCS info automatically |
| `fatih/color` v1.7 (no NO_COLOR) | `fatih/color` v1.13+ | 2022 | Built-in NO_COLOR support added |
| `cobra.Command.Run` | `cobra.Command.RunE` | Always preferred | Error-returning variant is standard practice |
| `cobra.ExactArgs` alone | `cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs)` | cobra v1.8+ | Compose validators for stricter checking |

**Deprecated/outdated:**
- `GenBashCompletion()` (v1): Still works but `GenBashCompletionV2()` is preferred
- `go-survey/survey`: No longer maintained, recommends bubbletea
- Custom `os.Getenv("NO_COLOR")` checks alongside fatih/color: Redundant, fatih/color handles it

## Open Questions

1. **Interactive root menu scope**
   - What we know: User wants an interactive menu, not just help text. Bubble Tea is the eventual TUI framework (Phase 5).
   - What's unclear: How polished should the Phase 1 menu be? A simple numbered menu is functional but not pretty.
   - Recommendation: Implement a simple numbered menu with color in Phase 1. In Phase 5, replace it with a Bubble Tea select list. This avoids introducing bubbletea as a dependency before it's needed.

2. **Git context check placement**
   - What we know: Commands like `status`, `update`, `push` need `.git` + `.gitmodules`. Commands like `version`, `completion`, `help` do not.
   - What's unclear: Whether to use `PersistentPreRunE` with exemptions or per-command `PreRunE`.
   - Recommendation: Use a shared `requireGitRepo()` helper function. Commands that need it call it in their own `PreRunE`. This is explicit and avoids the inheritance footgun.

3. **Flag --dry-run collision with compat hint**
   - What we know: `--dry-run` is both an old bash-era standalone flag AND a valid flag for `ssu update --dry-run`.
   - What's unclear: Need to distinguish `ssu --dry-run` (old style, should hint) from `ssu update --dry-run` (valid).
   - Recommendation: Only check `args[1]` (first argument after binary name). If it's `--dry-run` and there's no subcommand before it, hint. If it follows a known subcommand, let cobra handle it normally.

## Sources

### Primary (HIGH confidence)
- [spf13/cobra v1.10.2](https://pkg.go.dev/github.com/spf13/cobra) - API surface, Command struct, completion methods, version verified
- [fatih/color v1.18.0](https://pkg.go.dev/github.com/fatih/color) - NO_COLOR support, API, TTY detection behavior
- [mattn/go-isatty v0.0.20](https://pkg.go.dev/github.com/mattn/go-isatty) - TTY detection API, platform support
- [Go official module layout](https://go.dev/doc/modules/layout) - cmd/ + internal/ structure
- [NO_COLOR specification](https://no-color.org/) - Environment variable behavior specification
- [Cobra shell completion guide](https://cobra.dev/docs/how-to-guides/shell-completion/) - Completion subcommand pattern

### Secondary (MEDIUM confidence)
- [Go build info with debug.ReadBuildInfo and ldflags](https://www.piotrbelina.com/blog/go-build-info-debug-readbuildinfo-ldflags/) - Hybrid version injection pattern, verified against Go docs
- [Go project structure practices (2025)](https://www.glukhov.org/post/2025/12/go-project-structure/) - Community patterns, cross-referenced with official Go docs
- [Cobra enterprise guide](https://cobra.dev/docs/explanations/enterprise-guide/) - Flag deprecation patterns

### Tertiary (LOW confidence)
- Interactive menu approach (numbered menu vs. bubbletea) - Based on community patterns from web search, no single authoritative source

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All libraries verified via pkg.go.dev with exact versions
- Architecture: HIGH - cmd/internal pattern is official Go recommendation, cobra patterns from official docs
- Pitfalls: HIGH - Verified through official cobra docs and known issues
- Backwards compat approach: MEDIUM - Custom pattern, but implementation is straightforward
- Interactive menu: LOW - Claude's discretion area, multiple valid approaches

**Research date:** 2026-02-09
**Valid until:** 2026-04-09 (60 days -- cobra ecosystem is stable, slow-moving)
