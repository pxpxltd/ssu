package tui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// SpinnerModel implements tea.Model for displaying a MiniDot spinner with
// a counter and currently-processing path during the scan/fetch phase.
//
// View: "⠹ Fetching 12/24 plugins/blog..."
type SpinnerModel struct {
	spinner  spinner.Model
	label    string // Verb shown before the counter (default "Fetching")
	total    int
	done     int
	current  string
	failed   int
	err      error
	complete bool
	result   interface{}
}

// NewSpinnerModel creates a new spinner model for scan progress.
func NewSpinnerModel() SpinnerModel {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = StatusCurrentStyle // cyan
	return SpinnerModel{
		spinner: s,
		label:   "Fetching",
	}
}

// NewSpinnerModelWithLabel creates a spinner model that shows label instead
// of "Fetching" before the counter (e.g. "Checking PRs 3/12 plugins/auth...").
func NewSpinnerModelWithLabel(label string) SpinnerModel {
	m := NewSpinnerModel()
	m.label = label
	return m
}

// Init implements tea.Model. Starts the spinner tick.
func (m SpinnerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update implements tea.Model.
func (m SpinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case FetchProgressMsg:
		m.done = msg.Done
		m.total = msg.Total
		m.current = msg.Path
		if msg.Err != nil {
			m.failed++
		}
		return m, nil

	case FetchCompleteMsg:
		m.complete = true
		m.result = msg.Result
		m.err = msg.Err
		return m, tea.Quit

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.err = context.Canceled
			return m, tea.Quit
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

// View implements tea.Model.
func (m SpinnerModel) View() string {
	if m.total == 0 {
		return m.spinner.View() + " Scanning..."
	}

	line := fmt.Sprintf("%s %s %d/%d", m.spinner.View(), m.label, m.done, m.total)
	if m.current != "" {
		line += " " + MutedStyle.Render(m.current+"...")
	}

	if m.failed > 0 {
		line += "  " + StatusErrorStyle.Render(fmt.Sprintf("! %d failed", m.failed))
	}

	return line
}

// Result returns the stored result for the caller to type-assert.
func (m SpinnerModel) Result() interface{} { return m.result }

// Err returns any fatal error that occurred.
func (m SpinnerModel) Err() error { return m.err }

// Complete reports whether the operation finished.
func (m SpinnerModel) Complete() bool { return m.complete }
