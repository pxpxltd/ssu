# Phase 4: Config + Safety - Research

**Researched:** 2026-02-09
**Domain:** YAML configuration layering, backup/rollback, structured logging
**Confidence:** HIGH

## Summary

Phase 4 implements three distinct subsystems: layered YAML configuration (Viper), JSON backup/rollback with atomic writes, and structured file logging (slog + lumberjack). All three are well-understood problems with mature Go libraries.

The critical version constraint is that the project uses Go 1.21 and Viper v1.20.x is the latest compatible line (v1.21.0 bumped to Go 1.23). Viper v1.20 dropped HCL/INI/properties but retains YAML, JSON, TOML, and dotenv -- YAML is all we need. slog is stdlib in Go 1.21 (its debut release). Lumberjack v2 has no Go version constraints and implements io.Writer for seamless slog integration.

For config layering, Viper's MergeConfigMap approach is the proven pattern: read global config first, then merge project config on top, then let env vars and flags override via AutomaticEnv and BindPFlags. The backup system is straightforward JSON marshal + atomic write (temp file + os.Rename), with parsing of bash-era format for migration. Logging uses slog.TextHandler with ReplaceAttr for the human-readable `[timestamp] [LEVEL] message` format, writing to lumberjack for rotation.

**Primary recommendation:** Use Viper v1.20.1, slog (stdlib), lumberjack v2, and hand-roll atomic writes (3 lines of code -- no library needed).

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| spf13/viper | v1.20.1 | Config loading, layering, env/flag binding | Cobra companion, battle-tested, supports MergeConfigMap for layered overrides |
| log/slog (stdlib) | Go 1.21 | Structured logging | Standard library, zero dependencies, Handler interface for customization |
| gopkg.in/natefinch/lumberjack.v2 | v2.0 (latest v2.2.1) | Log file rotation | De facto standard for Go log rotation, implements io.Writer |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| encoding/json (stdlib) | Go 1.21 | Backup JSON marshal/unmarshal | Backup file format |
| os (stdlib) | Go 1.21 | Atomic write (TempFile + Rename) | Backup file safety |
| path/filepath (stdlib) | Go 1.21 | Cross-platform path handling | ~/.ssu directory management |
| time (stdlib) | Go 1.21 | Timestamp formatting for backups/logs | Backup filenames, log format |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Viper | knadh/koanf | 313% smaller binary, no key lowercasing; but Viper is natural Cobra companion, already familiar ecosystem |
| Viper | Manual YAML + os.Getenv | Simpler, no dep; but loses flag binding, env prefix, type coercion, MergeConfigMap |
| lumberjack | Custom rotation | Could avoid dep; but rotation is tricky (race conditions, cleanup), lumberjack is 1 file |
| Hand-roll atomic write | natefinch/atomic or google/renameio | Abstracts platform differences; but SSU targets Linux/macOS only, os.Rename is atomic on both |

**Installation:**
```bash
go get github.com/spf13/viper@v1.20.1
go get gopkg.in/natefinch/lumberjack.v2
# slog, encoding/json, os, path/filepath are stdlib
```

**CRITICAL VERSION NOTE:** Viper v1.21.0+ requires Go 1.23. The project uses Go 1.21, so pin to v1.20.1. If the project upgrades to Go 1.23+ later, Viper can be updated.

## Architecture Patterns

### Recommended Project Structure
```
internal/
├── config/
│   ├── config.go          # Config struct, defaults, Load() function
│   ├── config_test.go     # Unit tests with temp dirs and env vars
│   └── source.go          # Source annotation tracking (for `ssu config show`)
├── backup/
│   ├── backup.go          # Backup struct, Create/List/Clean/Read functions
│   ├── rollback.go        # Rollback logic (restore SHA + branch)
│   ├── atomic.go          # AtomicWrite helper (temp file + rename)
│   ├── compat.go          # Bash-era format reader
│   └── backup_test.go     # Tests with temp dirs
├── logging/
│   ├── logging.go         # InitLogger, slog handler setup, lumberjack config
│   └── logging_test.go    # Tests with temp log files
```

### Pattern 1: Viper Config Loading with Layered Merge
**What:** Load defaults, global config, project config, env vars, and CLI flags in priority order.
**When to use:** Application startup (PersistentPreRunE on root command).

```go
// internal/config/config.go

type Config struct {
    Git      GitConfig      `mapstructure:"git"`
    Branches BranchConfig   `mapstructure:"branches"`
    Backup   BackupConfig   `mapstructure:"backup"`
    Log      LogConfig      `mapstructure:"log"`
}

type GitConfig struct {
    ParallelJobs int      `mapstructure:"parallel_jobs"`
    Skip         []string `mapstructure:"skip"`
    FailFast     bool     `mapstructure:"fail_fast"`
}

type BranchConfig struct {
    Priority []string `mapstructure:"priority"`
    Override string   `mapstructure:"override"`
}

type BackupConfig struct {
    Enabled    bool `mapstructure:"enabled"`
    MaxBackups int  `mapstructure:"max_backups"`
}

type LogConfig struct {
    MaxSizeMB  int `mapstructure:"max_size_mb"`
    MaxBackups int `mapstructure:"max_backups"`
}

func Load(projectRoot string) (*Config, error) {
    v := viper.New()

    // 1. Defaults
    v.SetDefault("git.parallel_jobs", 8)
    v.SetDefault("git.fail_fast", false)
    v.SetDefault("branches.priority", []string{"develop", "master", "main"})
    v.SetDefault("backup.enabled", true)
    v.SetDefault("backup.max_backups", 10)
    v.SetDefault("log.max_size_mb", 10)
    v.SetDefault("log.max_backups", 5)

    // 2. Global config: ~/.ssu/config.yaml
    home, _ := os.UserHomeDir()
    globalPath := filepath.Join(home, ".ssu", "config.yaml")
    if _, err := os.Stat(globalPath); err == nil {
        v.SetConfigFile(globalPath)
        if err := v.ReadInConfig(); err != nil {
            return nil, fmt.Errorf("reading global config: %w", err)
        }
    }

    // 3. Project config: .ssu.yaml (merge on top)
    projectPath := filepath.Join(projectRoot, ".ssu.yaml")
    if _, err := os.Stat(projectPath); err == nil {
        pv := viper.New()
        pv.SetConfigFile(projectPath)
        if err := pv.ReadInConfig(); err != nil {
            return nil, fmt.Errorf("reading project config: %w", err)
        }
        v.MergeConfigMap(pv.AllSettings())
    }

    // 4. Env vars: SSU_ prefix
    v.SetEnvPrefix("SSU")
    v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
    v.AutomaticEnv()

    // 5. Legacy env vars (unprefixed, silent support)
    if val := os.Getenv("PARALLEL_JOBS"); val != "" {
        if v.GetString("git.parallel_jobs") == "" || !v.IsSet("git.parallel_jobs") {
            v.Set("git.parallel_jobs", val)
        }
    }

    var cfg Config
    if err := v.Unmarshal(&cfg); err != nil {
        return nil, fmt.Errorf("parsing config: %w", err)
    }
    return &cfg, nil
}
```

### Pattern 2: Atomic File Write
**What:** Write data to a temp file in the same directory, then rename atomically.
**When to use:** Backup file creation (prevents corruption on crash/power loss).

```go
// internal/backup/atomic.go

func AtomicWrite(path string, data []byte, perm os.FileMode) error {
    dir := filepath.Dir(path)

    // Create temp file in same directory (same filesystem = atomic rename)
    tmp, err := os.CreateTemp(dir, ".ssu-backup-*.tmp")
    if err != nil {
        return fmt.Errorf("creating temp file: %w", err)
    }
    tmpPath := tmp.Name()

    // Clean up temp file on any error
    defer func() {
        if tmpPath != "" {
            os.Remove(tmpPath)
        }
    }()

    if _, err := tmp.Write(data); err != nil {
        tmp.Close()
        return fmt.Errorf("writing temp file: %w", err)
    }

    // Sync to disk before rename (prevents 0-length file on crash)
    if err := tmp.Sync(); err != nil {
        tmp.Close()
        return fmt.Errorf("syncing temp file: %w", err)
    }

    if err := tmp.Close(); err != nil {
        return fmt.Errorf("closing temp file: %w", err)
    }

    // Atomic rename (POSIX guarantees atomicity on same filesystem)
    if err := os.Rename(tmpPath, path); err != nil {
        return fmt.Errorf("renaming temp to target: %w", err)
    }

    tmpPath = "" // Prevent deferred cleanup
    return nil
}
```

### Pattern 3: slog with Custom Format + Lumberjack Rotation
**What:** Human-readable log format matching bash behavior, with size-based rotation.
**When to use:** Application startup, before any operations.

```go
// internal/logging/logging.go

func InitLogger(logDir string, verbose bool, maxSizeMB, maxBackups int) (*slog.Logger, error) {
    if err := os.MkdirAll(logDir, 0o755); err != nil {
        return nil, fmt.Errorf("creating log directory: %w", err)
    }

    logFile := filepath.Join(logDir, "submodule-update.log")

    writer := &lumberjack.Logger{
        Filename:   logFile,
        MaxSize:    maxSizeMB, // megabytes
        MaxBackups: maxBackups,
        LocalTime:  true,
    }

    // Custom format: [2024-01-15 10:30:00] [INFO] message
    handler := slog.NewTextHandler(writer, &slog.HandlerOptions{
        Level: slog.LevelInfo,
        ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
            if a.Key == slog.TimeKey {
                if t, ok := a.Value.Any().(time.Time); ok {
                    a.Value = slog.StringValue(t.Format("2006-01-02 15:04:05"))
                }
            }
            return a
        },
    })

    logger := slog.New(handler)
    return logger, nil
}
```

**Note on log format:** slog.TextHandler outputs `time=... level=... msg=...` format. To match the exact bash format `[2024-01-15 10:30:00] [INFO] message`, a custom Handler implementation is needed (about 30 lines). The TextHandler with ReplaceAttr gets close but keeps key=value format. Recommendation: implement a thin custom handler that wraps the formatting logic.

### Pattern 4: Cobra PersistentPreRunE for Config Integration
**What:** Load config and bind CLI flags in the Cobra hook that runs before any command.
**When to use:** Root command setup.

```go
// In root.go PersistentPreRunE:
root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
    // Load layered config
    cfg, err := config.Load(projectRoot)
    if err != nil {
        return err
    }

    // CLI flags override config (highest priority)
    if cmd.Flags().Changed("jobs") {
        cfg.Git.ParallelJobs, _ = cmd.Flags().GetInt("jobs")
    }
    if cmd.Flags().Changed("verbose") {
        // verbose affects log level
    }

    // Store config in context or command annotations for subcommands
    cmd.SetContext(config.WithConfig(cmd.Context(), cfg))
    return nil
}
```

### Anti-Patterns to Avoid
- **Global Viper instance:** Use `viper.New()` for testability. The global `viper.Get*()` functions are convenient but make testing painful.
- **BindPFlags in init():** Binding must happen after cobra parses flags, not during init. Use PersistentPreRunE.
- **Viper for simple reads:** Don't use Viper to re-read config at runtime. Load once at startup, pass the Config struct down.
- **MergeInConfig for layered files:** MergeInConfig uses search paths which makes it unpredictable. Use SetConfigFile + ReadInConfig for global, then MergeConfigMap for project override.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Config file parsing + env overlay | YAML parser + os.Getenv | Viper | Handles type coercion, nested keys, env prefix, flag binding, merge |
| Log file rotation | Rename + size check on each write | lumberjack | Thread-safe, handles cleanup, compression, timestamp naming |
| Structured logging | fmt.Fprintf with timestamps | slog | Leveled, context-aware, Handler interface for multiple outputs |
| Config struct binding | Manual map traversal | Viper Unmarshal + mapstructure tags | Handles nested structs, type conversion, default values |

**Key insight:** The temptation is to avoid Viper's dependency weight. But Viper's MergeConfigMap, env binding, and Cobra flag integration save hundreds of lines of error-prone glue code. The dependency cost is justified for a CLI tool.

## Common Pitfalls

### Pitfall 1: Viper Key Lowercasing
**What goes wrong:** Viper lowercases all config keys internally. If your YAML has `ParallelJobs`, Viper stores it as `paralleljobs`.
**Why it happens:** Viper normalizes keys for case-insensitive lookup.
**How to avoid:** Use lowercase snake_case in YAML (`parallel_jobs`) and matching `mapstructure` tags. Never mix cases.
**Warning signs:** Config values are empty despite being in the file.

### Pitfall 2: MergeConfigMap Replaces Slices (Does Not Append)
**What goes wrong:** If global config has `branches.priority: [develop, master]` and project config has `branches.priority: [staging]`, the merged result is `[staging]` -- not `[develop, master, staging]`.
**Why it happens:** Viper's MergeConfigMap does a shallow merge for slices.
**How to avoid:** This is actually the desired behavior for SSU (project config overrides, not appends). Document this in user-facing config docs. If you ever need append semantics, do it manually after Unmarshal.
**Warning signs:** Users report "my global skip list disappeared".

### Pitfall 3: Viper BindPFlag Default vs Changed
**What goes wrong:** If you BindPFlag before checking if the flag was explicitly set, Viper uses the pflag default as if the user set it, overriding config file values.
**Why it happens:** Viper can't distinguish between "flag set by user" and "flag has default".
**How to avoid:** Use `cmd.Flags().Changed("flagname")` to check if the user explicitly set the flag. Only apply flag values when Changed() returns true.
**Warning signs:** Config file values are always overridden by flag defaults.

### Pitfall 4: Atomic Write on Different Filesystem
**What goes wrong:** os.Rename fails with "invalid cross-device link" if temp file is on a different mount point than the target.
**Why it happens:** os.CreateTemp defaults to /tmp which may be a different filesystem (tmpfs).
**How to avoid:** Always create temp files in the same directory as the target file using `os.CreateTemp(filepath.Dir(targetPath), ...)`.
**Warning signs:** Backup creation fails on some systems but not others.

### Pitfall 5: slog TextHandler Format vs Bash-Era Format
**What goes wrong:** slog.TextHandler outputs `time=2024-01-15T10:30:00 level=INFO msg="Updated module"` but the bash version outputs `[2024-01-15 10:30:00] [INFO] Updated module`.
**Why it happens:** TextHandler uses key=value format by default. ReplaceAttr can change values but not the overall output structure.
**How to avoid:** Write a custom slog.Handler (about 30-40 lines) that formats output in the bash-compatible bracket format. The Handler interface is simple: Enabled, Handle, WithAttrs, WithGroup.
**Warning signs:** Log files look different between bash and Go versions.

### Pitfall 6: Backup Directory Auto-Creation Race
**What goes wrong:** Two concurrent SSU invocations both try to create ~/.ssu/project/backups/ and one fails.
**Why it happens:** os.MkdirAll is not atomic.
**How to avoid:** os.MkdirAll is idempotent -- if the directory already exists, it returns nil. This is a non-issue in practice. Just call MkdirAll and check err.
**Warning signs:** None -- this is a theoretical concern that os.MkdirAll already handles.

### Pitfall 7: Legacy Env Var Priority
**What goes wrong:** Both `PARALLEL_JOBS` (legacy) and `SSU_GIT_PARALLEL_JOBS` (canonical) are set, and the wrong one wins.
**Why it happens:** No clear priority between legacy and canonical env vars.
**How to avoid:** Canonical `SSU_*` vars always win. Legacy vars only apply if the canonical equivalent is not set. Check canonical first, fall back to legacy.
**Warning signs:** Users migrating from bash get unexpected values.

## Code Examples

### Config YAML Structure (Recommended Key Names)
```yaml
# ~/.ssu/config.yaml or .ssu.yaml
git:
  parallel_jobs: 8
  skip:
    - "vendor/legacy"
    - "plugins/deprecated"
  fail_fast: false

branches:
  priority:
    - develop
    - master
    - main
  # override: "staging"  # Equivalent to --branch flag

backup:
  enabled: true
  max_backups: 10

log:
  max_size_mb: 10
  max_backups: 5
```

### Bash-Era Backup JSON Format (Must Read)
```json
{
  "timestamp": "2024-01-15T10:30:00+00:00",
  "submodules": {
    "plugins/module1": {"sha": "abc123def456", "branch": "develop"},
    "plugins/module2": {"sha": "789012abc345", "branch": "main"}
  }
}
```
Filename pattern: `.submodule-backup-YYYYMMDD-HHMMSS.json`
Location (bash era): `~/.ssu/<project>/`
Location (Go era): `~/.ssu/<project>/backups/`

### New Go-Era Backup JSON Format (Must Write)
```json
{
  "version": 2,
  "timestamp": "2024-01-15T10:30:00+00:00",
  "submodules": {
    "plugins/module1": {"sha": "abc123def456", "branch": "develop"},
    "plugins/module2": {"sha": "789012abc345", "branch": "main"}
  }
}
```
Filename pattern: `backup-YYYYMMDD-HHMMSS.json` (drop the dot prefix -- hidden files are confusing)
The `"version": 2` field distinguishes Go-era from bash-era backups. The reader checks for both formats.

### Source Annotation for `ssu config show`
```go
type AnnotatedValue struct {
    Value  any
    Source string // "default", "global (~/.ssu/config.yaml)", "project (.ssu.yaml)", "env (SSU_GIT_PARALLEL_JOBS)", "flag (--jobs)"
}

// Track which layer set each value by recording sources during Load()
```

### Custom slog Handler for Bash-Compatible Format
```go
type BracketHandler struct {
    w     io.Writer
    level slog.Leveler
    mu    sync.Mutex
}

func (h *BracketHandler) Enabled(_ context.Context, level slog.Level) bool {
    return level >= h.level.Level()
}

func (h *BracketHandler) Handle(_ context.Context, r slog.Record) error {
    h.mu.Lock()
    defer h.mu.Unlock()

    ts := r.Time.Format("2006-01-02 15:04:05")
    level := strings.ToUpper(r.Level.String())

    // [2024-01-15 10:30:00] [INFO] message
    _, err := fmt.Fprintf(h.w, "[%s] [%s] %s\n", ts, level, r.Message)
    return err
}

func (h *BracketHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
    return h // SSU logs are simple messages, no structured attrs needed for file logs
}

func (h *BracketHandler) WithGroup(name string) slog.Handler {
    return h // No group support needed
}
```

### Backup Clean with Suffix Detection
```go
func ParseKeepArg(arg string) (mode string, value int, err error) {
    if strings.HasSuffix(arg, "d") {
        // Time-based: "7d" = 7 days
        days, err := strconv.Atoi(strings.TrimSuffix(arg, "d"))
        if err != nil || days <= 0 {
            return "", 0, fmt.Errorf("invalid duration: %s", arg)
        }
        return "time", days, nil
    }
    // Count-based: "5" = keep 5 most recent
    count, err := strconv.Atoi(arg)
    if err != nil || count <= 0 {
        return "", 0, fmt.Errorf("invalid count: %s", arg)
    }
    return "count", count, nil
}
```

### Config Context Pattern (Pass Config to Subcommands)
```go
// internal/config/context.go
type ctxKey struct{}

func WithConfig(ctx context.Context, cfg *Config) context.Context {
    return context.WithValue(ctx, ctxKey{}, cfg)
}

func FromContext(ctx context.Context) *Config {
    cfg, _ := ctx.Value(ctxKey{}).(*Config)
    return cfg
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| encoding/json for config | Viper YAML with layering | Established | Proper config hierarchy, env var binding |
| log package | log/slog | Go 1.21 (Aug 2023) | Structured logging, Handler interface, levels |
| Custom log rotation | lumberjack v2 | Established | Thread-safe rotation, compression |
| viper global instance | viper.New() per use | Best practice | Testable, no global state |
| mitchellh/mapstructure | go-viper/mapstructure/v2 | Viper v1.20+ | Community fork, maintained |

**Deprecated/outdated:**
- `golang.org/x/exp/slog`: Use `log/slog` from stdlib (Go 1.21+). The exp version was the pre-release.
- `sagikazarmark/slog-shim`: Not needed with Go 1.21+ (shim was for Go <1.21 compatibility).
- Viper global functions (`viper.Get()`, `viper.Set()`): Use instance methods for testability.

## Open Questions

1. **Viper v1.20 MergeConfigMap with mapstructure v2**
   - What we know: Viper v1.20 switched from mitchellh/mapstructure to go-viper/mapstructure/v2. The `mapstructure` struct tags remain the same name.
   - What's unclear: Whether there are any subtle behavior differences in v2 for our use case (nested structs with slices).
   - Recommendation: Test thoroughly with the exact config struct. If issues arise, fallback is manual Viper.Get() calls.

2. **Verbose stderr output alongside file logging**
   - What we know: User wants `-v` flag to print debug output to stderr in real time, while logs always go to file.
   - What's unclear: Best way to fan out slog to two handlers (file + stderr) at different levels.
   - Recommendation: Use a simple multi-handler that wraps two slog.Handlers. The file handler always logs at INFO+, and the stderr handler only logs when verbose is enabled (at DEBUG level). This is about 20 lines of code.

3. **Rollback branch checkout reliability**
   - What we know: Rollback should restore both SHA and branch (not just SHA, to avoid detached HEAD).
   - What's unclear: What happens if the branch was deleted since the backup was created?
   - Recommendation: Try checkout branch first, then reset to SHA. If branch doesn't exist, create it from the SHA. Log a warning if the branch name doesn't match the backup.

## Sources

### Primary (HIGH confidence)
- [Viper v1.20.1 go.mod](https://github.com/spf13/viper/blob/v1.20.1/go.mod) - Go 1.21.0 minimum confirmed
- [Viper README](https://github.com/spf13/viper) - Config layering, env binding, flag binding, Unmarshal patterns
- [Viper pkg.go.dev](https://pkg.go.dev/github.com/spf13/viper) - MergeInConfig, MergeConfigMap, SetEnvPrefix, AutomaticEnv API
- [slog pkg.go.dev](https://pkg.go.dev/log/slog) - Handler interface, HandlerOptions, ReplaceAttr, TextHandler, Level constants
- [Go blog: slog](https://go.dev/blog/slog) - slog introduced in Go 1.21, August 2023
- [Lumberjack README](https://github.com/natefinch/lumberjack) - Logger struct, MaxSize/MaxBackups/MaxAge/Compress fields, io.Writer interface

### Secondary (MEDIUM confidence)
- [Carolyn Van Slyck: Sting of the Viper](https://carolynvanslyck.com/blog/2020/08/sting-of-the-viper/) - Cobra + Viper PersistentPreRunE integration pattern
- [Viper issue #181](https://github.com/spf13/viper/issues/181) - MergeConfigMap pattern for multiple config files
- [Viper issue #462](https://github.com/spf13/viper/issues/462) - MergeInConfig replaces slices (does not append)
- [Michael Stapelberg: Atomic writes](https://michael.stapelberg.ch/posts/2017-01-28-golang_atomically_writing/) - Temp file + fsync + rename pattern

### Tertiary (LOW confidence)
- [koanf comparison](https://github.com/knadh/koanf/wiki/Comparison-with-spf13-viper) - Viper vs koanf tradeoffs (not used, but noted)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Viper, slog, lumberjack are the Go ecosystem standards; versions verified against go.mod
- Architecture: HIGH - Patterns verified with official documentation and known working examples
- Pitfalls: HIGH - Key lowercasing, slice merge, BindPFlag default, and atomic rename issues are well-documented
- Backup format: HIGH - Bash-era format documented in project CLAUDE.md

**Research date:** 2026-02-09
**Valid until:** 2026-04-09 (90 days -- all libraries are stable, no major releases expected)
