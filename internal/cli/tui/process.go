package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pxpxltd/ssu/internal/engine"
)

// ProcessItemMsg reports per-item state changes during push/update processing.
type ProcessItemMsg struct {
	Type   engine.ProgressEventType // Started, Completed, Failed
	Path   string
	Action string // result text for completed items
	Err    error
	Done   int
	Total  int
}

// itemState tracks the display state of a single processing item.
type itemState int

const (
	itemWaiting    itemState = iota
	itemProcessing
	itemDone
	itemFailed
)

type processItem struct {
	path   string
	state  itemState
	action string // result text (set on done/failed)
	err    error
}

// ProcessModel implements tea.Model for multi-line live status during
// push/update processing.
type ProcessModel struct {
	items     []processItem
	spinner   spinner.Model
	bar       progress.Model
	operation string // "push" or "update"
	done      int
	total     int
	err       error
	complete  bool
	result    interface{}
	height    int // terminal height for scrolling
}

// NewProcessModel creates a new multi-line process status model.
// paths is the list of submodule paths to process, operation is "push" or "update".
func NewProcessModel(paths []string, operation string) ProcessModel {
	items := make([]processItem, len(paths))
	for i, p := range paths {
		items[i] = processItem{path: p, state: itemWaiting}
	}

	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = StatusCurrentStyle

	bar := progress.New(
		progress.WithDefaultGradient(),
		progress.WithoutPercentage(),
	)
	bar.Width = 40

	return ProcessModel{
		items:     items,
		spinner:   s,
		bar:       bar,
		operation: operation,
		total:     len(paths),
	}
}

// Init implements tea.Model.
func (m ProcessModel) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update implements tea.Model.
func (m ProcessModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		return m, nil

	case ProcessItemMsg:
		m.done = msg.Done
		m.total = msg.Total
		for i := range m.items {
			if m.items[i].path == msg.Path {
				switch msg.Type {
				case engine.EventStarted:
					m.items[i].state = itemProcessing
				case engine.EventCompleted:
					m.items[i].state = itemDone
					m.items[i].action = msg.Action
				case engine.EventFailed:
					m.items[i].state = itemFailed
					m.items[i].action = msg.Action
					m.items[i].err = msg.Err
				}
				break
			}
		}
		return m, nil

	case ProcessCompleteMsg:
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

	case progress.FrameMsg:
		barModel, cmd := m.bar.Update(msg)
		m.bar = barModel.(progress.Model)
		return m, cmd
	}

	return m, nil
}

// View implements tea.Model.
func (m ProcessModel) View() string {
	var b strings.Builder

	// Determine visible window for scrolling.
	maxVisible := len(m.items)
	if m.height > 0 && maxVisible > m.height-3 { // leave room for progress bar + label
		maxVisible = m.height - 3
		if maxVisible < 3 {
			maxVisible = 3
		}
	}

	// Find window: center around the first active item, or show from top.
	start := 0
	if maxVisible < len(m.items) {
		// Find first processing item.
		activeIdx := -1
		for i, item := range m.items {
			if item.state == itemProcessing {
				activeIdx = i
				break
			}
		}
		if activeIdx >= 0 {
			start = activeIdx - maxVisible/2
			if start < 0 {
				start = 0
			}
			if start+maxVisible > len(m.items) {
				start = len(m.items) - maxVisible
			}
		} else {
			// No active items: show last completed items.
			start = m.done - maxVisible
			if start < 0 {
				start = 0
			}
		}
	}

	end := start + maxVisible
	if end > len(m.items) {
		end = len(m.items)
	}

	// Render visible items.
	for i := start; i < end; i++ {
		item := m.items[i]
		var line string

		switch item.state {
		case itemWaiting:
			sym := MutedStyle.Render("  " + symbolWaiting)
			line = sym + "  " + MutedStyle.Render(item.path)

		case itemProcessing:
			sym := "  " + m.spinner.View()
			line = sym + "  " + item.path
			if item.action != "" {
				line += "  " + MutedStyle.Render(item.action)
			} else {
				line += "  " + MutedStyle.Render(m.operation + "ing...")
			}

		case itemDone:
			sym := StatusPendingStyle.Render("  " + symbolDone)
			line = sym + "  " + item.path
			if item.action != "" {
				line += "  " + MutedStyle.Render(item.action)
			}

		case itemFailed:
			sym := StatusConflictStyle.Render("  " + symbolFailed)
			errText := item.action
			if errText == "" && item.err != nil {
				errText = item.err.Error()
			}
			line = sym + "  " + item.path
			if errText != "" {
				line += "  " + StatusConflictStyle.Render(errText)
			}
		}

		b.WriteString(line + "\n")
	}

	// Progress bar + label at bottom.
	opLabel := strings.ToUpper(m.operation[:1]) + m.operation[1:]
	label := fmt.Sprintf("%s %d/%d", opLabel, m.done, m.total)

	pct := float64(0)
	if m.total > 0 {
		pct = float64(m.done) / float64(m.total)
	}
	barView := m.bar.ViewAs(pct)

	b.WriteString(fmt.Sprintf("\n%s %s %s",
		lipgloss.NewStyle().PaddingLeft(18).Render(label),
		barView,
		MutedStyle.Render(fmt.Sprintf("%d%%", int(pct*100))),
	))

	return b.String()
}

// Symbols for item states.
const (
	symbolWaiting = "\u25CB" // ○
	symbolDone    = "\u2713" // ✓
	symbolFailed  = "\u2717" // ✗
)

// Result returns the stored result for the caller to type-assert.
func (m ProcessModel) Result() interface{} { return m.result }

// Err returns any fatal error that occurred.
func (m ProcessModel) Err() error { return m.err }

// Complete reports whether the operation finished.
func (m ProcessModel) Complete() bool { return m.complete }
