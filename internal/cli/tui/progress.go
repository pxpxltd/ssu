package tui

import (
	"context"
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

// ---------------------------------------------------------------------------
// Progress message types (sent from external goroutines via p.Send)
// ---------------------------------------------------------------------------

// FetchProgressMsg reports progress during parallel fetch.
type FetchProgressMsg struct {
	Path  string
	Done  int
	Total int
	Err   error
}

// FetchCompleteMsg signals that the fetch/scan operation finished.
type FetchCompleteMsg struct {
	Result interface{} // *engine.ScanResult (type-assert by caller)
	Err    error
}

// ProcessProgressMsg reports progress during update/push processing.
type ProcessProgressMsg struct {
	Path   string
	Done   int
	Total  int
	Action string // e.g. "updating", "pushing"
	Err    error
}

// ProcessCompleteMsg signals that the update/push operation finished.
type ProcessCompleteMsg struct {
	Result interface{} // *engine.UpdateResult or *engine.PushResult
	Err    error
}

// ---------------------------------------------------------------------------
// ProgressModel
// ---------------------------------------------------------------------------

// ProgressModel implements tea.Model for displaying an animated progress bar
// with a count label and currently-processing submodule name.
type ProgressModel struct {
	bar      progress.Model
	total    int
	done     int
	current  string   // currently-fetching/processing submodule path
	failed   []string // paths that failed
	err      error    // fatal error
	complete bool
	result   interface{} // holds the final result
	width    int
}

// NewProgressModel creates a new progress bar model.
func NewProgressModel(total int) ProgressModel {
	bar := progress.New(progress.WithGradient("#76B947", "#2E7D32"))
	return ProgressModel{
		bar:   bar,
		total: total,
	}
}

// Init implements tea.Model. Progress is driven by external messages.
func (m ProgressModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m ProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		barWidth := m.width - 30
		if barWidth > 60 {
			barWidth = 60
		}
		if barWidth < 10 {
			barWidth = 10
		}
		m.bar.Width = barWidth
		return m, nil

	case FetchProgressMsg:
		m.done = msg.Done
		m.total = msg.Total
		m.current = msg.Path
		if msg.Err != nil {
			m.failed = append(m.failed, msg.Path)
		}
		return m, nil

	case FetchCompleteMsg:
		m.complete = true
		m.result = msg.Result
		m.err = msg.Err
		return m, tea.Quit

	case ProcessProgressMsg:
		m.done = msg.Done
		m.total = msg.Total
		m.current = msg.Path
		if msg.Err != nil {
			m.failed = append(m.failed, msg.Path)
		}
		return m, nil

	case ProcessCompleteMsg:
		m.complete = true
		m.result = msg.Result
		m.err = msg.Err
		return m, tea.Quit

	case progress.FrameMsg:
		// Forward animation frame to the progress bar.
		progressModel, cmd := m.bar.Update(msg)
		m.bar = progressModel.(progress.Model)
		return m, cmd

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.err = context.Canceled
			return m, tea.Quit
		}
	}

	return m, nil
}

// View implements tea.Model.
func (m ProgressModel) View() string {
	if m.total == 0 {
		return "Scanning..."
	}

	pct := float64(m.done) / float64(m.total)
	bar := m.bar.ViewAs(pct)

	label := fmt.Sprintf(" %d/%d", m.done, m.total)
	if m.current != "" {
		label += " fetching " + m.current
	}

	line := bar + MutedStyle.Render(label)

	if len(m.failed) > 0 {
		failedLabel := fmt.Sprintf("\n%s %d failed", StatusErrorStyle.Render("!"), len(m.failed))
		line += failedLabel
	}

	return line
}

// ---------------------------------------------------------------------------
// Public accessors (used by callers after Run() completes)
// ---------------------------------------------------------------------------

// Result returns the stored result for the caller to type-assert.
func (m ProgressModel) Result() interface{} { return m.result }

// Err returns any fatal error that occurred.
func (m ProgressModel) Err() error { return m.err }

// Complete reports whether the operation finished.
func (m ProgressModel) Complete() bool { return m.complete }

// ---------------------------------------------------------------------------
// Non-TTY fallback helper
// ---------------------------------------------------------------------------

// PrintProgressLine writes a simple text progress line for non-TTY/auto mode.
// Format: "Fetching done/total path..."
func PrintProgressLine(w io.Writer, done, total int, path string) {
	fmt.Fprintf(w, "Fetching %d/%d %s...\n", done, total, path)
}
