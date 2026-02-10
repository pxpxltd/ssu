# Phase 5: Commands + TUI - Research

**Researched:** 2026-02-09
**Domain:** Bubbletea TUI framework, Cobra command integration, interactive CLI patterns
**Confidence:** HIGH

## Summary

Phase 5 wires the engine (Phase 3) and config (Phase 4) layers to user-facing commands through bubbletea's Elm Architecture TUI framework. The phase replaces the existing placeholder `RunE` stubs in `internal/cli/*.go` with real implementations that call the engine's `Scan`, `Update`, and `Push` methods, displaying results through bubbletea models for interactive mode and plain text/JSON for non-interactive mode.

The standard stack is bubbletea v1.3.0 + bubbles v0.20.0 + lipgloss v1.0.0. These versions all require Go 1.18 minimum and are compatible with the project's `go 1.21` directive. The latest versions (bubbles v0.21.1, lipgloss v1.1.0) require Go 1.23+ and would force bumping the project's go.mod directive -- avoid these. The lipgloss/table sub-package provides the status table renderer. Bubbles provides progress bar, viewport (for changelog pane), help, and key binding components. The multi-select TUI selector is custom-built (no pre-built bubbles component exists for this) following bubbletea's Model/Update/View pattern.

The architecture uses a clean separation: each cobra command's `RunE` checks `--auto`/`--json`/non-TTY to decide between launching a bubbletea Program or printing directly. The bubbletea models receive engine results as messages via `tea.Cmd` functions. Ctrl+C handling uses bubbletea's built-in `tea.InterruptMsg` with `context.WithCancel` wired through `tea.WithContext` to propagate cancellation to the engine layer.

**Primary recommendation:** Build a custom multi-select model (not the bubbles/list component) because the CONTEXT.md requires specific keybindings (/, ?, a/A, space, confirmation step) and a split-pane layout with changelog that no pre-built component provides. Use bubbles/progress for the fetch progress bar, bubbles/viewport for the changelog detail pane, bubbles/help for the ? keybinding overlay, and lipgloss/table for the status command output.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/charmbracelet/bubbletea` | v1.3.0 | TUI framework (Elm Architecture) | Most popular Go TUI framework, handles raw mode, input, rendering, alt-screen |
| `github.com/charmbracelet/bubbles` | v0.20.0 | Pre-built TUI components | Progress bar, viewport, help, key bindings -- complements bubbletea |
| `github.com/charmbracelet/lipgloss` | v1.0.0 | Terminal styling and layout | Style definitions, table rendering, JoinHorizontal for split pane |
| `encoding/json` (stdlib) | Go 1.21 | JSON output for `--json` flag | Marshals ScanResult for machine-readable output |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/charmbracelet/lipgloss/table` | v1.0.0 (sub-package) | Status table rendering | `ssu status` colorized table output |
| `github.com/charmbracelet/bubbles/progress` | v0.20.0 | Animated progress bar | Parallel fetch progress display |
| `github.com/charmbracelet/bubbles/viewport` | v0.20.0 | Scrollable content pane | Changelog detail pane (right side of split) |
| `github.com/charmbracelet/bubbles/help` | v0.20.0 | Keybinding help display | ? key overlay showing available keys |
| `github.com/charmbracelet/bubbles/key` | v0.20.0 | Keybinding definitions | All TUI key bindings with help text integration |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Custom multi-select | bubbles/list | list has filtering but not multi-select checkboxes, no split pane, no confirmation step -- would need heavy customization anyway |
| lipgloss/table | Custom printf formatting | Table package handles column width calculation, Unicode, borders -- not worth hand-rolling |
| bubbles/progress | Custom progress string | Progress component handles animation, spring physics, terminal width -- not trivial to replicate |
| bubbletea v1.3.0 | bubbletea v1.3.10 (latest) | v1.3.10 pulls in Go 1.24 transitively via dependencies; v1.3.0 is stable and Go 1.18 compatible |
| bubbles v0.20.0 | bubbles v0.21.x | v0.21.0 requires Go 1.23.0; v0.20.0 has all needed components and works with Go 1.21 |

**Installation:**
```bash
go get github.com/charmbracelet/bubbletea@v1.3.0
go get github.com/charmbracelet/bubbles@v0.20.0
go get github.com/charmbracelet/lipgloss@v1.0.0
```

**CRITICAL VERSION NOTE:** The latest bubbles (v0.21.1) and bubbletea (v1.3.10) require Go 1.23+/1.24+ through transitive dependencies. Since the project specifies `go 1.21` in go.mod, pin to the versions above. Running `go mod tidy` without version pins will auto-upgrade the go directive. If the project later bumps to Go 1.23+, the latest versions can be used.

## Architecture Patterns

### Recommended Project Structure
```
internal/
  cli/
    output/             # Existing: color, printer, symbols
    tui/                # NEW: all bubbletea models
      selector.go       # Multi-select model with checkboxes, filtering, split pane
      selector_keys.go  # KeyMap definitions for selector
      progress.go       # Fetch progress bar model
      confirm.go        # y/N confirmation prompt model
      styles.go         # Shared lipgloss styles for TUI
      tui.go            # Shared types: Item interface, common messages
    status.go           # REPLACE: ssu status implementation
    update.go           # REPLACE: ssu update implementation
    push.go             # REPLACE: ssu push implementation
    exec.go             # NEW: ssu exec implementation
    init.go             # NEW: ssu init wizard
    root.go             # UPDATE: replace placeholder menu with bubbletea launcher
```

### Pattern 1: Cobra RunE with Mode Branching

**What:** Each command's `RunE` function checks flags and TTY to decide between TUI mode, auto mode, and JSON mode. The engine call is the same in all modes -- only the output layer changes.

**When to use:** Every command that supports `--auto` or `--json`.

**Example:**
```go
// Source: Standard cobra+bubbletea integration pattern
func NewStatusCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "status",
        Short: "Show submodule status",
        RunE: func(cmd *cobra.Command, args []string) error {
            cfg := config.FromContext(cmd.Context())
            eng := engine.New(git.NewExecGit())

            // Scan is common to all modes
            result, err := eng.Scan(cmd.Context(), engine.ScanOpts{
                RootDir:     cfg.ProjectRoot,
                SkipList:    cfg.Git.Skip,
                Concurrency: cfg.Git.ParallelJobs,
                BranchOpts:  cfg.BranchDetectOpts(),
            })
            if err != nil {
                return err
            }

            // Mode branching
            jsonFlag, _ := cmd.Flags().GetBool("json")
            if jsonFlag {
                return printStatusJSON(cmd.OutOrStdout(), result)
            }
            return printStatusTable(cmd.OutOrStdout(), result)
        },
    }
    cmd.Flags().Bool("json", false, "Output status as JSON")
    return cmd
}
```

### Pattern 2: Bubbletea Model Lifecycle in Cobra

**What:** Launch a `tea.Program` from within a cobra `RunE` function. The model receives engine results through tea messages, not through shared state.

**When to use:** Commands that need interactive TUI (update, push, exec).

**Example:**
```go
// Source: bubbletea official docs + cobra integration pattern
func runUpdateInteractive(ctx context.Context, eng *engine.Engine, scanResult *engine.ScanResult, cfg *config.Config) error {
    // Create model with scan results
    m := tui.NewSelectorModel(scanResult, tui.SelectorOpts{
        Title:    "Select submodules to update",
        ShowOnly: engine.StatusPending, // Filter to pending submodules
    })

    // Create and run bubbletea program
    p := tea.NewProgram(m,
        tea.WithAltScreen(),     // Full screen mode
        tea.WithContext(ctx),    // Propagate cancellation
    )

    finalModel, err := p.Run()
    if err != nil {
        return fmt.Errorf("TUI error: %w", err)
    }

    // Extract selections from final model
    sel := finalModel.(tui.SelectorModel)
    if sel.Cancelled() {
        return nil
    }
    selected := sel.Selected()
    // ... proceed to update selected submodules
    return nil
}
```

### Pattern 3: Custom Multi-Select with Elm Architecture

**What:** A bubbletea Model that renders a list of items with checkboxes, cursor navigation, filtering, and a split-pane layout with a changelog viewport on the right.

**When to use:** For `ssu update` and `ssu push` interactive selection.

**Key state:**
```go
type SelectorModel struct {
    items       []SelectorItem
    cursor      int
    selected    map[int]bool
    filterInput string
    filtering   bool       // true when / search is active
    showHelp    bool       // toggled by ?
    showDetail  bool       // toggled by detail key
    viewport    viewport.Model  // changelog pane
    help        help.Model
    keys        SelectorKeyMap
    width       int
    height      int
    confirmed   bool
    cancelled   bool
}
```

**Update handler structure:**
```go
func (m SelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        if m.filtering {
            return m.handleFilterInput(msg)
        }
        switch {
        case key.Matches(msg, m.keys.Up):
            m.cursor = max(0, m.cursor-1)
        case key.Matches(msg, m.keys.Down):
            m.cursor = min(len(m.filteredItems())-1, m.cursor+1)
        case key.Matches(msg, m.keys.Toggle):
            m.toggleSelection(m.cursor)
        case key.Matches(msg, m.keys.All):
            m.selectAll()
        case key.Matches(msg, m.keys.None):
            m.deselectAll()
        case key.Matches(msg, m.keys.Confirm):
            // Transition to confirmation view
        case key.Matches(msg, m.keys.Filter):
            m.filtering = true
        case key.Matches(msg, m.keys.Help):
            m.showHelp = !m.showHelp
        case key.Matches(msg, m.keys.Detail):
            m.showDetail = !m.showDetail
        case key.Matches(msg, m.keys.Quit):
            m.cancelled = true
            return m, tea.Quit
        }
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
    }
    return m, nil
}
```

### Pattern 4: Split Pane Layout with lipgloss.JoinHorizontal

**What:** Render the selector list on the left and a viewport with changelog content on the right, using lipgloss.JoinHorizontal for side-by-side layout.

**When to use:** TUI selector View() method when detail pane is visible.

**Example:**
```go
// Source: lipgloss JoinHorizontal official API
func (m SelectorModel) View() string {
    if !m.showDetail {
        return m.renderList()
    }

    // Calculate widths
    listWidth := m.width / 2
    detailWidth := m.width - listWidth - 1 // -1 for separator

    leftPane := lipgloss.NewStyle().
        Width(listWidth).
        Render(m.renderList())

    separator := lipgloss.NewStyle().
        Foreground(lipgloss.Color("240")).
        Render(strings.Repeat("\u2502\n", m.height))

    rightPane := lipgloss.NewStyle().
        Width(detailWidth).
        Render(m.viewport.View())

    return lipgloss.JoinHorizontal(lipgloss.Top, leftPane, separator, rightPane)
}
```

### Pattern 5: Progress Bar for Parallel Fetch

**What:** A bubbletea Model that displays a progress bar during parallel fetch, receiving progress events from the engine via `tea.Cmd` messages. Uses `p.Send()` from a goroutine to push progress events.

**When to use:** During scan/fetch phase of update and push commands.

**Example:**
```go
// Source: bubbletea progress-animated example pattern
type ProgressModel struct {
    bar     progress.Model
    total   int
    done    int
    current string  // Currently fetching submodule name
    results []string
    err     error
}

type fetchProgressMsg struct {
    path string
    done int
    total int
    err  error
}

type fetchCompleteMsg struct {
    result *engine.ScanResult
    err    error
}

func (m ProgressModel) View() string {
    pct := float64(m.done) / float64(m.total)
    bar := m.bar.ViewAs(pct)
    label := fmt.Sprintf(" %d/%d fetching %s", m.done, m.total, m.current)
    return bar + label
}
```

**Sending progress from engine goroutine:**
```go
// In the cobra RunE, before launching the TUI:
func startScanWithProgress(ctx context.Context, eng *engine.Engine, opts engine.ScanOpts, p *tea.Program) {
    go func() {
        opts.OnProgress = func(evt engine.ProgressEvent) {
            p.Send(fetchProgressMsg{
                path:  evt.Path,
                done:  evt.Done,
                total: evt.Total,
                err:   evt.Error,
            })
        }
        result, err := eng.Scan(ctx, opts)
        p.Send(fetchCompleteMsg{result: result, err: err})
    }()
}
```

### Pattern 6: Non-TTY Fallback

**What:** When stdout is not a TTY (piped, CI), skip bubbletea entirely and print simple log lines.

**When to use:** All commands that support `--auto` or are run in CI.

**Example:**
```go
if !output.IsTTY() || autoFlag {
    // No TUI -- print directly
    result, err := eng.Scan(ctx, scanOpts)
    if err != nil { return err }
    for _, sm := range result.Submodules {
        fmt.Fprintf(cmd.OutOrStdout(), "Fetching %s... %s\n", sm.Path, sm.PrimaryStatus())
    }
    return nil
}
```

### Pattern 7: Ctrl+C with Partial Results

**What:** Wire `context.WithCancel` through `tea.WithContext`. When Ctrl+C arrives, bubbletea sends `tea.InterruptMsg`. The model catches it, marks cancellation, and shows partial results before quitting.

**When to use:** Update and push commands during processing phase.

**Example:**
```go
case tea.InterruptMsg:
    m.cancelled = true
    // Collect partial results from what completed so far
    m.partialResults = m.collectCompletedResults()
    return m, tea.Quit
```

After `p.Run()` returns:
```go
finalModel := result.(UpdateModel)
if finalModel.Cancelled() {
    fmt.Fprintf(os.Stderr, "Cancelled. %d/%d submodules updated before interruption:\n",
        finalModel.CompletedCount(), finalModel.TotalCount())
    finalModel.PrintPartialResults(os.Stderr)
    os.Exit(cli.ExitError)
}
```

### Anti-Patterns to Avoid

- **Launching bubbletea in non-TTY:** Bubbletea will panic or hang if stdin/stdout are not terminals. Always gate on `output.IsTTY()`.
- **Sharing mutable state between goroutine and model:** All communication must flow through `tea.Cmd` / `p.Send()`. The model itself must not be accessed concurrently.
- **Using bubbles/list for multi-select:** The list component is designed for single selection with filtering. It does not support checkboxes, multi-select toggle, or custom key handling for a/A/space. Build custom.
- **Global bubbletea Program variable:** Create the `tea.Program` locally within the command function and let it go out of scope after `Run()`. Do not store it globally.
- **Blocking in Update():** Never do I/O (git calls, file reads) in `Update()`. Use `tea.Cmd` to spawn I/O, and process the result as a `tea.Msg`.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Terminal raw mode + input parsing | Manual termios/stty calls | bubbletea framework | Handles raw mode, escape sequences, mouse, resize, signal restoration |
| Progress bar rendering | Custom `\r` + string manipulation | bubbles/progress | Handles terminal width, animation, color gradients, percentage display |
| Scrollable content area | Custom line offset tracking | bubbles/viewport | Handles scrolling, page up/down, mouse wheel, content wrapping |
| Table rendering with alignment | Manual printf with padding | lipgloss/table | Handles Unicode width, column auto-sizing, borders, per-cell styles |
| Keybinding help display | Manual string formatting | bubbles/help | Handles short/full view, column layout, width truncation, disabled binding hiding |
| Terminal styling | Manual ANSI escape codes | lipgloss.NewStyle() | Handles NO_COLOR, TERM=dumb, non-TTY, true-color vs 256-color detection |
| Side-by-side layout | Manual column calculation | lipgloss.JoinHorizontal() | Handles uneven heights, alignment, multi-line joining |

**Key insight:** The charm ecosystem (bubbletea + bubbles + lipgloss) is designed as a cohesive whole. Using lipgloss styles inside bubbles components inside bubbletea models works seamlessly. Do not mix with other styling approaches (fatih/color is fine for non-TUI output like `--auto` mode, but inside bubbletea views, use lipgloss exclusively).

## Common Pitfalls

### Pitfall 1: Version Incompatibility with Go 1.21
**What goes wrong:** Running `go get github.com/charmbracelet/bubbles@latest` resolves to v0.21.1, which requires Go 1.24.2. `go mod tidy` then auto-upgrades the project's `go` directive to 1.24.2.
**Why it happens:** Go modules enforce the highest `go` directive from any dependency.
**How to avoid:** Pin versions explicitly: `bubbletea@v1.3.0`, `bubbles@v0.20.0`, `lipgloss@v1.0.0`. All three require only Go 1.18.
**Warning signs:** go.mod `go` directive changes from 1.21 to a higher version after running `go mod tidy`.

### Pitfall 2: Launching Bubbletea on Non-TTY
**What goes wrong:** Bubbletea reads from stdin in raw mode. In a pipe or CI environment, this causes hangs, panics, or corrupted output.
**Why it happens:** Bubbletea assumes terminal capabilities. Without a TTY, input reading and rendering fail.
**How to avoid:** Always check `output.IsTTY()` before creating a `tea.NewProgram`. For `--auto` mode and non-TTY, print directly without bubbletea.
**Warning signs:** SSU hangs when called from scripts or CI, or produces garbled output.

### Pitfall 3: Blocking I/O in Update()
**What goes wrong:** Calling engine methods (Scan, Update, Push) directly in the Update function blocks the entire TUI -- no rendering, no input handling.
**Why it happens:** Bubbletea's Update runs synchronously in the event loop. Any blocking call freezes the UI.
**How to avoid:** Spawn engine calls as `tea.Cmd` functions that return results as messages. The engine runs in a separate goroutine managed by bubbletea.
**Warning signs:** TUI freezes during fetch/update operations, progress bar does not animate.

### Pitfall 4: Race Condition with p.Send()
**What goes wrong:** Calling `p.Send()` from the engine's progress callback (running in a goroutine) while the model is being updated causes a race.
**Why it happens:** `p.Send()` is thread-safe in bubbletea v1.x (it sends to a channel), but the message processing in Update() is single-threaded. The risk is minimal, but the pattern must use `p.Send()`, not direct model mutation.
**How to avoid:** Always use `p.Send(msg)` for cross-goroutine communication. Never access the model directly from outside the bubbletea event loop.
**Warning signs:** `-race` flag detects data race in TUI code.

### Pitfall 5: Alt-Screen Cleanup on Panic
**What goes wrong:** If a panic occurs while in alt-screen mode, the terminal is left in a corrupted state (invisible cursor, raw mode).
**Why it happens:** Bubbletea's cleanup runs on normal exit, but panics bypass it.
**How to avoid:** Bubbletea v1.3.0 catches panics by default (unless `WithoutCatchPanics()` is used). Do NOT use `WithoutCatchPanics()`. Also, the `tea.WithContext()` option ensures cleanup on context cancellation.
**Warning signs:** Terminal requires `reset` or `stty sane` after a crash.

### Pitfall 6: lipgloss Styles Inside Bubbletea vs fatih/color Outside
**What goes wrong:** Using fatih/color inside a bubbletea `View()` produces double-escaped or wrong ANSI codes because bubbletea has its own output handling.
**Why it happens:** Bubbletea and lipgloss share the same terminal environment detection (muesli/termenv). fatih/color uses its own detection and may disagree.
**How to avoid:** Inside bubbletea models (`View()`, rendering helpers), use lipgloss exclusively. Outside bubbletea (--auto mode, --json mode, non-TUI paths), fatih/color (existing output package) is fine.
**Warning signs:** Colors look wrong or have escape codes visible in TUI views.

### Pitfall 7: WindowSizeMsg Must Be Handled
**What goes wrong:** TUI renders at wrong width, table columns overflow or are too narrow, split pane is misaligned.
**Why it happens:** Bubbletea sends `WindowSizeMsg` on startup and on resize, but if the model ignores it, it uses default dimensions (0x0).
**How to avoid:** Always handle `tea.WindowSizeMsg` in Update() and store width/height on the model. Propagate dimensions to sub-components (viewport, progress bar).
**Warning signs:** Layout breaks on different terminal sizes, or appears as a single column.

### Pitfall 8: JSON Output Must Be Clean
**What goes wrong:** `ssu status --json` includes ANSI color codes, progress output, or extra text mixed in with JSON.
**Why it happens:** JSON mode and TUI mode share the same command path, and color codes leak into the output.
**How to avoid:** For `--json` mode, never launch bubbletea. Marshal the result directly with `encoding/json` and write to stdout. Disable color output (`color.NoColor = true`) before formatting. Ensure no other writes go to stdout (logging goes to stderr or file).
**Warning signs:** `jq` fails to parse the output, or `--json | jq` shows ANSI codes.

## Code Examples

### Status Table with lipgloss/table

```go
// Source: lipgloss/table official API, adapted for SSU
func renderStatusTable(w io.Writer, result *engine.ScanResult, termWidth int) {
    t := table.New().
        Headers("Path", "Branch", "Target", "Behind", "Feature", "Status").
        Border(lipgloss.NormalBorder()).
        BorderHeader(true).
        BorderColumn(true).
        Width(termWidth)

    headerStyle := lipgloss.NewStyle().Bold(true)
    rootStyle := lipgloss.NewStyle().Bold(true)

    // Root row first
    if result.Root != nil {
        t.Row(
            "(root)",
            result.Root.CurrentBranch,
            result.Root.TargetBranch,
            fmt.Sprintf("%d", result.Root.CommitsBehind),
            "",
            string(result.Root.PrimaryStatus()),
        )
    }

    // Submodule rows
    for _, sm := range result.Submodules {
        feature := "No"
        if sm.IsFeature {
            feature = "Yes"
        }
        t.Row(
            sm.Path,
            sm.CurrentBranch,
            sm.TargetBranch,
            fmt.Sprintf("%d", sm.CommitsBehind),
            feature,
            string(sm.PrimaryStatus()),
        )
    }

    t.StyleFunc(func(row, col int) lipgloss.Style {
        if row == table.HeaderRow {
            return headerStyle
        }
        // Row 0 is root (bold)
        if row == 0 && result.Root != nil {
            return rootStyle
        }
        // Status column coloring
        if col == 5 {
            return statusStyle(row, result)
        }
        return lipgloss.NewStyle()
    })

    fmt.Fprintln(w, t.Render())
}
```

### JSON Output for --json Flag

```go
// Source: encoding/json stdlib
type StatusJSON struct {
    Root       *SubmoduleJSON   `json:"root"`
    Submodules []SubmoduleJSON  `json:"submodules"`
    ScannedAt  string           `json:"scanned_at"`
}

type SubmoduleJSON struct {
    Path          string   `json:"path"`
    CurrentBranch string   `json:"current_branch"`
    TargetBranch  string   `json:"target_branch"`
    CommitsBehind int      `json:"commits_behind"`
    CommitsAhead  int      `json:"commits_ahead"`
    IsFeature     bool     `json:"is_feature"`
    Statuses      []string `json:"statuses"`
}

func printStatusJSON(w io.Writer, result *engine.ScanResult) error {
    out := StatusJSON{
        ScannedAt: time.Now().Format(time.RFC3339),
    }
    if result.Root != nil {
        out.Root = toSubmoduleJSON(result.Root)
    }
    for _, sm := range result.Submodules {
        out.Submodules = append(out.Submodules, *toSubmoduleJSON(sm))
    }
    enc := json.NewEncoder(w)
    enc.SetIndent("", "  ")
    return enc.Encode(out)
}
```

### KeyMap Definitions for Selector

```go
// Source: bubbles/key official API
type SelectorKeyMap struct {
    Up      key.Binding
    Down    key.Binding
    Toggle  key.Binding
    All     key.Binding
    None    key.Binding
    Confirm key.Binding
    Quit    key.Binding
    Filter  key.Binding
    Help    key.Binding
    Detail  key.Binding
    Sort    key.Binding
}

func DefaultSelectorKeyMap() SelectorKeyMap {
    return SelectorKeyMap{
        Up: key.NewBinding(
            key.WithKeys("up", "k"),
            key.WithHelp("\u2191/k", "up"),
        ),
        Down: key.NewBinding(
            key.WithKeys("down", "j"),
            key.WithHelp("\u2193/j", "down"),
        ),
        Toggle: key.NewBinding(
            key.WithKeys(" "),
            key.WithHelp("space", "toggle"),
        ),
        All: key.NewBinding(
            key.WithKeys("a"),
            key.WithHelp("a", "select all"),
        ),
        None: key.NewBinding(
            key.WithKeys("A"),
            key.WithHelp("A", "deselect all"),
        ),
        Confirm: key.NewBinding(
            key.WithKeys("enter"),
            key.WithHelp("enter", "confirm"),
        ),
        Quit: key.NewBinding(
            key.WithKeys("q", "esc"),
            key.WithHelp("q", "quit"),
        ),
        Filter: key.NewBinding(
            key.WithKeys("/"),
            key.WithHelp("/", "filter"),
        ),
        Help: key.NewBinding(
            key.WithKeys("?"),
            key.WithHelp("?", "help"),
        ),
        Detail: key.NewBinding(
            key.WithKeys("tab"),
            key.WithHelp("tab", "toggle detail"),
        ),
        Sort: key.NewBinding(
            key.WithKeys("s"),
            key.WithHelp("s", "change sort"),
        ),
    }
}

// ShortHelp implements help.KeyMap for the help component.
func (k SelectorKeyMap) ShortHelp() []key.Binding {
    return []key.Binding{k.Up, k.Down, k.Toggle, k.Confirm, k.Quit}
}

// FullHelp implements help.KeyMap for the full help display.
func (k SelectorKeyMap) FullHelp() [][]key.Binding {
    return [][]key.Binding{
        {k.Up, k.Down, k.Toggle, k.All, k.None},
        {k.Confirm, k.Quit, k.Filter, k.Help, k.Detail},
    }
}
```

### Progress Bar with Submodule Name

```go
// Source: bubbles/progress ViewAs + custom label
func (m ProgressModel) View() string {
    pct := float64(m.done) / float64(m.total)
    bar := m.bar.ViewAs(pct)

    // Format: [========>    ] 12/25 fetching plugins/auth
    label := fmt.Sprintf(" %d/%d", m.done, m.total)
    if m.current != "" {
        label += fmt.Sprintf(" fetching %s", m.current)
    }

    return bar + lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(label)
}
```

### Exec Command Running Arbitrary Commands

```go
// Source: os/exec stdlib, pattern matching bash SSU's foreach behavior
func execInSubmodules(ctx context.Context, subPaths []string, rootDir string, command string, args []string) error {
    for _, path := range subPaths {
        dir := filepath.Join(rootDir, path)
        cmd := exec.CommandContext(ctx, command, args...)
        cmd.Dir = dir
        cmd.Stdout = os.Stdout
        cmd.Stderr = os.Stderr

        fmt.Printf("==> %s\n", path)
        if err := cmd.Run(); err != nil {
            fmt.Fprintf(os.Stderr, "  error in %s: %v\n", path, err)
            // Continue to next submodule (don't abort)
        }
    }
    return nil
}
```

### Init Wizard Flow

```go
// Source: bubbletea textinput pattern, designed for SSU
func runInitWizard(projectRoot string) error {
    // Check if .ssu.yaml already exists
    configPath := filepath.Join(projectRoot, ".ssu.yaml")
    if _, err := os.Stat(configPath); err == nil {
        return fmt.Errorf("config already exists: %s", configPath)
    }

    // Create default config
    cfg := config.Defaults()

    // Prompt: parallel jobs (default: 8)
    // Prompt: branch priority (default: develop, master, main)
    // Prompt: skip list (default: empty)

    // Write .ssu.yaml
    data, _ := yaml.Marshal(cfg)
    return os.WriteFile(configPath, data, 0644)
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Manual ANSI + stty raw | bubbletea framework | bubbletea v1.0 (2024) | Eliminates all terminal management boilerplate |
| tcell/termbox for TUI | bubbletea + lipgloss | 2022-2023 | Elm Architecture is simpler than imperative widget trees |
| Custom table formatting | lipgloss/table | lipgloss v0.9+ (2024) | Table rendering with auto-column-sizing and per-cell styles |
| fmt.Printf for colors | lipgloss.NewStyle() | lipgloss v0.7+ | Declarative styling with NO_COLOR respect |
| Custom progress strings | bubbles/progress | bubbles v0.9+ | Animated spring-physics progress bar |

**Deprecated/outdated:**
- `bubbletea v2 / charm.land` module path: Not yet stable (RC status). Use v1.x with `github.com/charmbracelet/` path.
- `bubbles v0.21.x`: Requires Go 1.23+. Use v0.20.0 for Go 1.21 compatibility.
- `lipgloss v1.1.0`: Pulls in Go 1.18+ but works; however, v1.0.0 is sufficient and avoids any risk of transitive Go version bumps.
- `termbox-go` / `tcell`: Obsoleted by bubbletea for new projects. bubbletea's Elm Architecture is superior for TUI state management.

## Discretionary Decisions (Recommendations)

### Alt-Screen vs Inline Rendering

**Recommendation: Use inline rendering (no alt-screen) for the TUI selector.**

Rationale:
- Alt-screen clears the terminal and loses scroll-back history. Users may want to reference previous command output while selecting submodules.
- Inline rendering preserves context above the TUI.
- The progress bar and result streaming naturally integrate with inline mode (print above, TUI below).
- The bubbletea `Program.Println()` method allows printing above the inline TUI.

Use `tea.NewProgram(model)` without `tea.WithAltScreen()`. Handle terminal height with `tea.WindowSizeMsg`.

### Detail Pane Toggle Key

**Recommendation: Use `tab` key.**

Rationale:
- Tab is universally associated with "switch pane" or "toggle panel" in terminal tools.
- It does not conflict with any other keybinding in the selector (arrows, j/k, space, enter, q, /, ?, a/A).
- Easy to discover via the ? help overlay.

### Sort Order Toggle Key

**Recommendation: Use `s` key.**

Rationale:
- Mnemonic for "sort".
- Cycles through: path (default) -> status -> behind count -> path (loop).
- Does not conflict with other bindings.

### Progress Bar Visual Style

**Recommendation: Use `progress.WithDefaultGradient()` for a colorful gradient bar, with the `ViewAs` method for static rendering (no animation needed -- progress events come from the engine, not a timer).**

Rationale:
- The gradient stands out visually and matches the charm ecosystem aesthetic.
- `ViewAs` is simpler than animated `SetPercent` for externally-driven progress. No spring animation needed since progress events arrive discretely.
- Width auto-adjusts from `WindowSizeMsg`, capped at 60 characters to leave room for the label.

### Init Wizard Flow

**Recommendation: Simple sequential prompts with defaults, no full TUI needed.**

Flow:
1. Check if `.ssu.yaml` exists -- if so, print message and exit
2. Detect submodules in current project, show count
3. Prompt for parallel jobs (default: 8, or CPU count)
4. Prompt for branch priority (default: develop, master, main)
5. Prompt for skip list (default: none, show detected submodules)
6. Write `.ssu.yaml` with chosen values
7. Print success message with "Run `ssu status` to see your submodules"

Use simple `fmt.Scanln` prompts -- a full bubbletea model is overkill for a one-time wizard with 3-4 questions.

## Open Questions

1. **bubbletea v1.3.0 vs v1.3.10 compatibility**
   - What we know: v1.3.0 requires Go 1.18 and compiles fine. v1.3.10 is on `main` branch which now has Go 1.24 in go.mod but the v1.3.10 TAG itself was released in Sep 2024 and may still be Go 1.18.
   - What's unclear: Whether v1.3.10 introduces any Go 1.24-only features in its transitive deps.
   - Recommendation: Pin to v1.3.0 which is verified Go 1.21 compatible. If specific bug fixes from v1.3.4-v1.3.10 are needed, test each individually.

2. **Non-TTY detection with bubbletea**
   - What we know: The project already has `output.IsTTY()` using `mattn/go-isatty`. Bubbletea also does its own TTY detection.
   - What's unclear: Whether bubbletea gracefully handles being launched on a non-TTY (does it return an error or panic?).
   - Recommendation: Always gate on `output.IsTTY()` before creating a `tea.Program`. Never let bubbletea launch on a non-TTY.

3. **Engine goroutine lifecycle with tea.WithContext**
   - What we know: `tea.WithContext(ctx)` cancels the program when ctx is done. Engine operations use the same context.
   - What's unclear: Whether a long-running engine Scan that's already in flight will block the program from exiting promptly on Ctrl+C.
   - Recommendation: Engine already uses `context.Context` for all git operations. Context cancellation will propagate to `exec.CommandContext` calls, killing hung git processes. This should work correctly.

4. **Confirmation step UX**
   - What we know: CONTEXT.md requires a y/N confirmation after pressing enter.
   - What's unclear: Whether this should be a separate bubbletea model (transition from selector to confirmation view) or a state within the selector model.
   - Recommendation: State within the selector model (add `confirming bool` field). When enter is pressed, switch to a confirmation view showing selected items and a y/N prompt. This avoids the complexity of chaining multiple tea.Programs.

## Sources

### Primary (HIGH confidence)
- [bubbletea v1.3.0 go.mod](https://raw.githubusercontent.com/charmbracelet/bubbletea/v1.3.0/go.mod) - Go 1.18 minimum verified
- [bubbles v0.20.0 go.mod](https://raw.githubusercontent.com/charmbracelet/bubbles/v0.20.0/go.mod) - Go 1.18 minimum, bubbletea v1.1.0 dependency verified
- [lipgloss v1.0.0 go.mod](https://raw.githubusercontent.com/charmbracelet/lipgloss/v1.0.0/go.mod) - Go 1.18 minimum verified
- [bubbletea pkg.go.dev](https://pkg.go.dev/github.com/charmbracelet/bubbletea@v1.3.0) - Model interface, Program, Cmd, Msg types, WithAltScreen, WithContext, WindowSizeMsg, KeyMsg API
- [lipgloss/table pkg.go.dev](https://pkg.go.dev/github.com/charmbracelet/lipgloss@v1.0.0/table) - Table API, Headers, Row, StyleFunc, Border, Width, Render
- [bubbles/progress pkg.go.dev](https://pkg.go.dev/github.com/charmbracelet/bubbles@v0.20.0/progress) - Progress.ViewAs, SetPercent, FrameMsg, WithDefaultGradient
- [bubbles/viewport pkg.go.dev](https://pkg.go.dev/github.com/charmbracelet/bubbles@v0.20.0/viewport) - Viewport.New, SetContent, Update, View, scrolling methods
- [bubbles/help pkg.go.dev](https://pkg.go.dev/github.com/charmbracelet/bubbles@v0.20.0/help) - Help.View, ShortHelp, FullHelp, KeyMap interface
- [bubbles/key pkg.go.dev](https://pkg.go.dev/github.com/charmbracelet/bubbles@v0.20.0/key) - NewBinding, WithKeys, WithHelp, Matches
- Local compilation test: bubbletea@v1.3.0 + bubbles@v0.20.0 + lipgloss@v1.0.0 compile and run with `go 1.21` directive

### Secondary (MEDIUM confidence)
- [bubbletea GitHub releases](https://github.com/charmbracelet/bubbletea/releases) - v1.3.0 release notes and changelog
- [bubbles v0.20.0 release](https://github.com/charmbracelet/bubbles/releases/tag/v0.20.0) - Focus-blur support, component inventory
- [lipgloss JoinHorizontal](https://pkg.go.dev/github.com/charmbracelet/lipgloss) - Layout composition functions
- [Cobra + bubbletea integration discussion](https://github.com/charmbracelet/bubbletea/discussions/940) - Community pattern for cobra RunE integration

### Tertiary (LOW confidence)
- Web search results on bubbletea split-pane layouts and multi-select patterns - Community patterns, not official guidance
- [bubblelayout](https://pkg.go.dev/github.com/winder/bubblelayout) - Third-party layout manager (not recommended -- lipgloss.JoinHorizontal suffices)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Versions verified by local compilation test with Go 1.21 directive; API documented from pkg.go.dev
- Architecture: HIGH - Patterns derived from official bubbletea examples, pkg.go.dev documentation, and Phase 3 engine API
- Pitfalls: HIGH - Version compatibility verified empirically; TTY/non-TTY behavior documented in bubbletea source; alt-screen cleanup documented in issue tracker
- Discretionary decisions: MEDIUM - Alt-screen vs inline is preference-based; recommendations follow community patterns

**Research date:** 2026-02-09
**Valid until:** 2026-03-11 (30 days -- bubbletea v1.x is stable, no major releases expected; version pins protect against breaking changes)
