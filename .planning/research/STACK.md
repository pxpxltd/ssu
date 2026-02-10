# Technology Stack

**Project:** SSU (Smart Submodule Updater) - Go Rewrite
**Researched:** 2026-02-09
**Overall Confidence:** HIGH (library choices), MEDIUM (exact versions — verify with `go list -m @latest`)

## Recommended Stack

### Go Version

| Technology | Version | Purpose | Confidence |
|------------|---------|---------|------------|
| Go | 1.23+ | Language runtime | MEDIUM (verify latest at go.dev/dl/) |

### CLI Framework

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| cobra | v1.8+ | Subcommand routing | De facto standard — kubectl, gh, hugo, docker CLI. Handles subcommands, flags, help generation, shell completion. | HIGH |

**DO NOT USE:** urfave/cli (smaller ecosystem), kong (unconventional), flag (no subcommand support)

### TUI Framework

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| bubbletea | v1.2+ | Interactive TUI | Elm architecture (Model-Update-View). The Go TUI framework. | HIGH |
| bubbles | v0.20+ | Pre-built TUI components | Spinner, table, list, textinput, viewport, progress bar | HIGH |
| lipgloss | v1.0+ | Terminal styling | CSS-like styling. Colors, borders, padding. Adaptive color support. | HIGH |

**DO NOT USE:** tview/tcell (lower level, overkill), pterm (not a TUI framework), raw ANSI codes

### Configuration

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| viper | v1.19+ | Config file management | YAML/JSON/TOML, multi-source merging, cobra integration | HIGH |

Config loading order: Defaults < `~/.ssu/config.yaml` < `.ssu.yaml` < env vars < CLI flags

### Git Operations

| Technology | Purpose | Why | Confidence |
|------------|---------|-----|------------|
| `os/exec` (stdlib) | Shell out to git | **Most important architectural decision.** Matches bash behavior exactly. 100% git compatibility. No abstraction leaks. | HIGH |

**DO NOT USE:**
- `go-git` — Incomplete submodule support, complex transport setup, wrong abstraction for wrapping git workflows. Adds ~10MB to binary.
- `git2go` (libgit2) — CGo dependency breaks cross-compilation.

### Concurrency

| Technology | Purpose | Why | Confidence |
|------------|---------|-----|------------|
| `golang.org/x/sync/errgroup` | Parallel fetch with error propagation | `SetLimit()` for bounded concurrency. Context cancellation for timeouts. | HIGH |
| `context` (stdlib) | Cancellation and timeouts | Prevents hung git processes (bash version has this bug). | HIGH |

### Logging

| Technology | Purpose | Why | Confidence |
|------------|---------|-----|------------|
| `log/slog` (stdlib) | Structured logging | Go 1.21+, leveled, composable handlers | HIGH |
| `lumberjack` v2 | Log rotation | Plugs into any io.Writer. Max size, max backups, max age. | HIGH |

**DO NOT USE:** logrus (maintenance mode), zap/zerolog (overkill for CLI)

### Testing

| Technology | Purpose | Why | Confidence |
|------------|---------|-----|------------|
| `testing` (stdlib) | Test framework | Built-in, no framework needed | HIGH |
| `testify` v1.9+ | Assertions and mocking | assert/require packages, mock for interfaces | HIGH |
| `testscript` | Integration testing | Script-based CLI testing. Runs actual binary, checks stdout/stderr/exit codes. | HIGH |

### Build and Distribution

| Technology | Purpose | Why | Confidence |
|------------|---------|-----|------------|
| GoReleaser v2+ | Release automation | Cross-platform builds, GitHub releases, Homebrew formula, checksums, version injection | HIGH |
| golangci-lint | Linting | Meta-linter (50+ linters). Standard for Go projects. | HIGH |
| GitHub Actions | CI/CD | Build, test, lint on push. Trigger goreleaser on tag. | HIGH |

## Complete Dependency List

### Runtime
```
github.com/spf13/cobra
github.com/spf13/viper
github.com/charmbracelet/bubbletea
github.com/charmbracelet/bubbles
github.com/charmbracelet/lipgloss
golang.org/x/sync
gopkg.in/natefinished/lumberjack.v2
```

### Dev/Test
```
github.com/stretchr/testify
github.com/rogpeppe/go-internal  (testscript)
```

## Alternatives Summary

| Category | Recommended | Why Not Alternatives |
|----------|-------------|---------------------|
| CLI | cobra | urfave/cli: smaller ecosystem; kong: unconventional |
| TUI | bubbletea | tview: lower-level; pterm: not interactive |
| Config | viper | koanf: less cobra integration |
| Git ops | os/exec | go-git: incomplete submodules; git2go: CGo breaks cross-compile |
| Logging | slog | logrus: maintenance mode; zap: overkill |
| Testing | testify | ginkgo: unnecessary BDD complexity |

## Project Structure

```
ssu/
├── cmd/ssu/main.go          # Entry point, version injection
├── internal/
│   ├── cmd/                  # Cobra command definitions
│   ├── engine/               # Business logic (scanner, updater, pusher, branch detection)
│   ├── git/                  # Git operations (interface + exec implementation + mock)
│   ├── tui/                  # Bubbletea models (selector, table, styles)
│   ├── config/               # Config management (viper, defaults, merge)
│   ├── backup/               # Backup/restore (JSON, atomic writes)
│   ├── model/                # Shared data types (SubmoduleInfo, Status, Results)
│   └── log/                  # Logging (slog + lumberjack)
├── .goreleaser.yaml
├── Makefile
├── go.mod / go.sum
├── README.md
└── LICENSE
```

## Roadmap Implications

1. Phase 1: Foundation — go mod init, cobra root, project structure, CI
2. Phase 2: Git wrapper package — core business logic, everything depends on it
3. TUI comes after git operations work — start with non-interactive output
4. Config (viper) wired up early so all commands can access settings
5. Distribution (goreleaser) is final — no point automating releases until tool works

---
*Stack research: 2026-02-09*
