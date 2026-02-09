# Architecture Patterns

**Domain:** Go CLI tool with TUI for git submodule management
**Researched:** 2026-02-09
**Confidence:** HIGH (patterns from cobra, bubbletea, gh, lazygit are well-established)

## Overall Pattern: Layered CLI with Separated TUI

```
                    +-------------------+
                    |   CLI Entry       |   cmd/ssu/main.go
                    |   (cobra root)    |
                    +--------+----------+
                             |
              +--------------+--------------+
              |              |              |
         +----+----+   +----+----+   +-----+-----+
         | status  |   | update  |   | push/etc  |   internal/cmd/*.go
         | command |   | command |   | commands  |
         +----+----+   +----+----+   +-----+-----+
              |              |              |
              +--------------+--------------+
                             |
                    +--------+----------+
                    |   Core Engine     |   internal/engine/
                    |   (business logic)|
                    +--------+----------+
                             |
              +--------------+--------------+
              |              |              |
         +----+----+   +----+----+   +-----+-----+
         |   Git   |   |  Config |   |  Backup   |   internal/git/
         | Service |   | Service |   | Service   |   internal/config/
         +---------+   +---------+   +-----------+   internal/backup/

              TUI Layer (orthogonal)
         +-----------------------------+
         |  bubbletea Models           |   internal/tui/
         |  (selector, table, progress)|
         +-----------------------------+
```

## Component Boundaries

| Component | Package | Responsibility | Depends On |
|-----------|---------|----------------|------------|
| CLI Entry | `cmd/ssu/` | Binary entry point, cobra root | Command handlers |
| Commands | `internal/cmd/` | Parse flags, wire deps, invoke engine | Engine, Config, TUI |
| Engine | `internal/engine/` | Scan/update/push orchestration | Git Service, Backup |
| Git Service | `internal/git/` | All git operations behind interface | `os/exec` |
| Config | `internal/config/` | Load/merge YAML configs | Filesystem |
| Backup | `internal/backup/` | Create/restore JSON backups | Filesystem |
| TUI | `internal/tui/` | bubbletea models for interaction | Engine data (model types only) |
| Models | `internal/model/` | Shared data types | None |

## Data Flow

### Update Workflow

1. `main.go` → `cobra.Execute()`
2. `cmd/update.go` → Load config, create GitService, create Engine
3. `engine.Scan(ctx)` → List submodules → Parallel fetch (errgroup) → Detect status → Returns `[]SubmoduleInfo`
4. If interactive: Pass to TUI selector → Returns selected paths
5. If auto: Select all pending
6. `engine.Update(ctx, selectedPaths)` → Create backup → Merge each → Handle conflicts → Returns `UpdateResult`
7. Command renders summary

**Key insight:** Data flows DOWN (commands → engine → services). Results flow UP. TUI is a side-channel — receives data, returns selections, never calls git.

## Recommended Project Layout

```
ssu/
├── cmd/ssu/main.go              # Entry point, version injection
├── internal/
│   ├── cmd/                     # Cobra command definitions
│   │   ├── root.go              # Root command, global flags, version
│   │   ├── status.go            # ssu status
│   │   ├── update.go            # ssu update
│   │   ├── push.go              # ssu push
│   │   ├── rollback.go          # ssu rollback
│   │   └── compat.go            # Backwards compat hints
│   ├── engine/                  # Business logic (the core)
│   │   ├── scanner.go           # Scan: list, fetch, analyze
│   │   ├── updater.go           # Update: merge, conflict handling
│   │   ├── pusher.go            # Push: push ahead submodules
│   │   ├── branch.go            # Smart branch detection
│   │   └── engine.go            # Engine struct, constructor
│   ├── git/                     # Git abstraction
│   │   ├── service.go           # GitService interface
│   │   ├── exec.go              # Real implementation (os/exec)
│   │   └── mock.go              # Mock for testing
│   ├── config/                  # Configuration
│   │   ├── config.go            # Config struct, defaults, merge
│   │   └── loader.go            # Load from files + env
│   ├── backup/                  # Backup/restore
│   │   ├── backup.go            # Create/restore/list
│   │   └── format.go            # JSON format (compat with bash)
│   ├── tui/                     # TUI layer
│   │   ├── selector.go          # Multi-select with checkboxes
│   │   ├── table.go             # Status table
│   │   └── styles.go            # lipgloss styles
│   ├── model/                   # Shared types
│   │   ├── submodule.go         # SubmoduleInfo, Status enum
│   │   └── result.go            # UpdateResult, PushResult
│   └── log/                     # Logging
│       └── logger.go            # File + stderr logger
├── go.mod / go.sum
├── .goreleaser.yaml
├── Makefile
└── README.md
```

Use `internal/` exclusively — SSU is a CLI tool, not a library.

## Key Patterns

### 1. Interface-Based Git Abstraction (Most Critical)

```go
type Service interface {
    ListSubmodules(ctx context.Context, root string) ([]string, error)
    FetchAll(ctx context.Context, path string) error
    CurrentBranch(ctx context.Context, path string) (string, error)
    RemoteBranches(ctx context.Context, path string) ([]string, error)
    CommitsBehind(ctx context.Context, path, branch string) (int, error)
    HasLocalChanges(ctx context.Context, path string) (bool, error)
    Merge(ctx context.Context, path, ref string) error
    Push(ctx context.Context, path string) error
    Stash(ctx context.Context, path string) error
    StashPop(ctx context.Context, path string) error
    // ... etc
}
```

Real impl uses os/exec. Mock impl for unit tests.

### 2. Engine as Orchestrator

```go
type Engine struct {
    git    git.Service
    config *config.Config
    logger *log.Logger
    root   string
}

func (e *Engine) Scan(ctx context.Context) (*model.ScanResult, error)
func (e *Engine) Update(ctx context.Context, paths []string) (*model.UpdateResult, error)
func (e *Engine) Push(ctx context.Context, paths []string) (*model.PushResult, error)
```

### 3. TUI Receives Data, Returns Selections

```go
// TUI never calls git directly
selector := tui.NewSelector("Select submodules", items)
p := tea.NewProgram(selector)
result, _ := p.Run()
selected := result.(tui.SelectorModel).Selections()
```

### 4. Concurrent Fetch with errgroup

```go
g, ctx := errgroup.WithContext(ctx)
g.SetLimit(config.ParallelJobs)
for _, path := range paths {
    path := path
    g.Go(func() error {
        return e.git.FetchAll(ctx, path)
    })
}
g.Wait()
```

### 5. Status as Typed Enum

```go
type SubmoduleStatus int
const (
    StatusCurrent SubmoduleStatus = iota
    StatusPending
    StatusModified
    StatusAhead
    StatusConflict
    StatusMissing
    StatusSkipped
)
```

### 6. Config Layering

Defaults < `~/.ssu/config.yaml` < `.ssu.yaml` < env vars < CLI flags

### 7. Backwards Compatibility

```go
func registerCompatHints(root *cobra.Command) {
    hints := map[string]string{
        "status": "ssu status", "push": "ssu push",
        "rollback": "ssu rollback <file>",
    }
    // Register hidden flags, detect in PersistentPreRunE, print migration hint
}
```

## Anti-Patterns to Avoid

| Anti-Pattern | Why Bad | Instead |
|-------------|---------|---------|
| Using go-git | Incomplete submodules, wrong abstraction | os/exec behind interface |
| Global state | Bash's biggest flaw; Go rewrite should fix this | Dependency injection |
| TUI calling git | Blocks UI, untestable, breaks non-interactive | TUI receives data, returns selections |
| One giant command handler | Reproduces bash main() problem | Thin commands, logic in engine |
| String error returns | Can't distinguish failure types | Typed error types |
| Over-engineering | Enterprise patterns for a 950-line bash rewrite | ~1500-2500 LOC target, concrete types first |

## Build Order

```
Phase 1: Foundation (no dependencies)
  ├── internal/model/     (pure data types)
  ├── internal/config/    (yaml + filesystem)
  └── internal/log/       (filesystem)

Phase 2: Git Layer (depends on model)
  └── internal/git/       (interface + exec + mock)

Phase 3: Engine (depends on git, model, config)
  └── internal/engine/    (scanner, updater, pusher, branch detection)

Phase 4: TUI (depends on model only)
  └── internal/tui/       (selector, table, styles)

Phase 5: Commands (depends on everything)
  ├── internal/cmd/       (cobra commands)
  └── cmd/ssu/main.go     (entry point)

Phase 6: Distribution
  └── .goreleaser.yaml, Makefile
```

**Critical path:** model → git interface → engine → commands. TUI is parallel track.

## Testability Strategy

| Boundary | Real | Test |
|----------|------|------|
| Git operations | os/exec to git CLI | MockService in-memory |
| Filesystem (backup, config) | os package | t.TempDir() |
| TUI | tea.Program | teatest package |
| Output | os.Stdout | bytes.Buffer |

---
*Architecture research: 2026-02-09*
