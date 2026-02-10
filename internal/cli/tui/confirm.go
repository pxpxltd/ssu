package tui

import (
	"fmt"
	"strings"
)

// renderConfirmation renders the confirmation view when m.confirming is true.
// This is a View helper, not a separate bubbletea Model.
func renderConfirmation(m SelectorModel) string {
	var b strings.Builder

	// Count selected items.
	count := m.selectedCount()

	b.WriteString(TitleStyle.Render(fmt.Sprintf("Selected %d submodule(s) for %s:", count, m.operation)))
	b.WriteString("\n\n")

	// List each selected item path.
	for idx, sel := range m.allSelected {
		if !sel || idx < 0 || idx >= len(m.items) {
			continue
		}
		b.WriteString("  ")
		b.WriteString(m.items[idx].Path())
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString("Proceed? [y/N] ")

	return b.String()
}
