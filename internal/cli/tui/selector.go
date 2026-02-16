package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pxpxltd/ssu/internal/engine"
	"github.com/pxpxltd/ssu/internal/git"
)

// Sort modes cycle through: path -> status -> behind -> path.
const (
	SortByPath   = 0
	SortByStatus = 1
	SortByBehind = 2
)

// ---------------------------------------------------------------------------
// SelectorModel
// ---------------------------------------------------------------------------

// SelectorModel implements tea.Model for the multi-select submodule picker.
// It supports checkboxes, cursor navigation, filtering, split-pane changelog
// detail, help overlay, sorting, and a confirmation step.
type SelectorModel struct {
	title       string
	subtitle    string
	operation   string // operation name for confirmation (e.g. "update")
	items       []SelectorItem
	cursor      int
	allSelected map[int]bool // selection state by original item index
	filterInput string
	filtering   bool
	showHelp    bool
	showDetail  bool
	sortMode    int
	viewport    viewport.Model
	help        help.Model
	keys        SelectorKeyMap
	width       int
	height      int
	confirming  bool
	confirmed   bool
	cancelled   bool
}

// NewSelectorModel creates a new multi-select selector model.
func NewSelectorModel(items []SelectorItem, opts SelectorOpts) SelectorModel {
	// Apply pre-filter if provided.
	var filtered []SelectorItem
	if opts.FilterFn != nil {
		for _, item := range items {
			if opts.FilterFn(item) {
				filtered = append(filtered, item)
			}
		}
	} else {
		filtered = make([]SelectorItem, len(items))
		copy(filtered, items)
	}

	// Sort by path initially.
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Path() < filtered[j].Path()
	})

	vp := viewport.New(0, 0)

	h := help.New()

	op := opts.Operation
	if op == "" {
		op = "operation"
	}

	m := SelectorModel{
		title:       opts.Title,
		subtitle:    opts.Subtitle,
		operation:   op,
		items:       filtered,
		allSelected: make(map[int]bool),
		showDetail:  opts.ShowDetail,
		viewport:    vp,
		help:        h,
		keys:        DefaultSelectorKeyMap(),
	}

	// Pre-select all items if requested.
	if opts.SelectAll {
		for i := range filtered {
			m.allSelected[i] = true
		}
	}

	// Set initial detail content if items exist.
	if len(m.items) > 0 {
		m.viewport.SetContent(m.items[0].DetailContent())
	}

	return m
}

// SetTitle sets the operation name used in the confirmation prompt.
func (m *SelectorModel) SetTitle(title string) {
	m.title = title
}

// SetOperation sets the operation name used in the confirmation prompt.
func (m *SelectorModel) SetOperation(op string) {
	m.operation = op
}

// Init implements tea.Model. Requests window size.
func (m SelectorModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m SelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeViewport()
		m.help.Width = m.width
		return m, nil

	case tea.KeyMsg:
		// Confirmation mode: only y/n/esc accepted.
		if m.confirming {
			return m.updateConfirm(msg)
		}

		// Filter mode: keys become text input.
		if m.filtering {
			return m.updateFilter(msg)
		}

		// Normal mode.
		return m.updateNormal(msg)
	}

	// Forward viewport messages (for scrolling).
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// updateConfirm handles keys during the confirmation prompt.
func (m SelectorModel) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.confirmed = true
		return m, tea.Quit
	case "n", "N", "esc":
		m.confirming = false
		return m, nil
	}
	return m, nil
}

// updateFilter handles keys during filter input mode.
func (m SelectorModel) updateFilter(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape:
		m.filtering = false
		m.filterInput = ""
		m.cursor = 0
		m.updateDetailPane()
		return m, nil
	case tea.KeyEnter:
		m.filtering = false
		m.cursor = 0
		m.updateDetailPane()
		return m, nil
	case tea.KeyBackspace:
		if len(m.filterInput) > 0 {
			m.filterInput = m.filterInput[:len(m.filterInput)-1]
			m.cursor = 0
			m.updateDetailPane()
		}
		return m, nil
	default:
		// Append printable characters.
		if msg.Type == tea.KeyRunes {
			m.filterInput += msg.String()
			m.cursor = 0
			m.updateDetailPane()
		}
		return m, nil
	}
}

// updateNormal handles keys in normal (non-filter, non-confirm) mode.
func (m SelectorModel) updateNormal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	filtered := m.filteredItems()

	switch {
	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}
		m.updateDetailPane()

	case key.Matches(msg, m.keys.Down):
		if m.cursor < len(filtered)-1 {
			m.cursor++
		}
		m.updateDetailPane()

	case key.Matches(msg, m.keys.Toggle):
		if len(filtered) > 0 {
			origIdx := m.originalIndex(m.cursor)
			if origIdx >= 0 {
				m.allSelected[origIdx] = !m.allSelected[origIdx]
				if !m.allSelected[origIdx] {
					delete(m.allSelected, origIdx)
				}
			}
		}

	case key.Matches(msg, m.keys.All):
		for _, fi := range m.filteredIndices() {
			m.allSelected[fi] = true
		}

	case key.Matches(msg, m.keys.None):
		for _, fi := range m.filteredIndices() {
			delete(m.allSelected, fi)
		}

	case key.Matches(msg, m.keys.Confirm):
		if m.selectedCount() > 0 {
			m.confirming = true
		}

	case key.Matches(msg, m.keys.Quit):
		m.cancelled = true
		return m, tea.Quit

	case key.Matches(msg, m.keys.Filter):
		m.filtering = true

	case key.Matches(msg, m.keys.Help):
		m.showHelp = !m.showHelp

	case key.Matches(msg, m.keys.Detail):
		m.showDetail = !m.showDetail
		m.resizeViewport()

	case key.Matches(msg, m.keys.Sort):
		m.sortMode = (m.sortMode + 1) % 3
		m.sortItems()
		m.cursor = 0
		m.updateDetailPane()
	}

	return m, nil
}

// View implements tea.Model.
func (m SelectorModel) View() string {
	if m.confirming {
		return renderConfirmation(m)
	}

	var b strings.Builder

	// Title and subtitle header.
	if m.title != "" {
		b.WriteString(TitleStyle.Render(m.title))
		b.WriteString("\n")
		if m.subtitle != "" {
			b.WriteString(MutedStyle.Render(m.subtitle))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Filter line.
	if m.filtering || m.filterInput != "" {
		prefix := MutedStyle.Render("/")
		b.WriteString(prefix)
		b.WriteString(m.filterInput)
		if m.filtering {
			b.WriteString("\u2588") // block cursor
		}
		b.WriteString("\n")
	}

	// Calculate available height for the item list.
	headerLines := 0
	if m.title != "" {
		headerLines += 2
		if m.subtitle != "" {
			headerLines++
		}
	}
	if m.filtering || m.filterInput != "" {
		headerLines++
	}
	footerLines := 2 // selection count + help line
	if m.showHelp {
		footerLines += 3
	}
	listHeight := m.height - headerLines - footerLines
	if listHeight < 1 {
		listHeight = 10 // fallback
	}

	// Render main content.
	filtered := m.filteredItems()
	listContent := m.renderList(filtered, listHeight)

	if m.showDetail && m.width > 40 {
		listWidth := m.width / 2
		detailWidth := m.width - listWidth - 1

		leftStyle := lipgloss.NewStyle().Width(listWidth).MaxHeight(listHeight)
		rightStyle := lipgloss.NewStyle().Width(detailWidth).MaxHeight(listHeight)

		// Build separator.
		sep := MutedStyle.Render(strings.Repeat("\u2502\n", listHeight))

		left := leftStyle.Render(listContent)
		right := rightStyle.Render(m.viewport.View())

		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, left, sep, right))
	} else {
		b.WriteString(listContent)
	}

	b.WriteString("\n")

	// Footer: selection count.
	count := m.selectedCount()
	sortLabel := [3]string{"path", "status", "behind"}[m.sortMode]
	footer := MutedStyle.Render(fmt.Sprintf("%d selected | sort: %s", count, sortLabel))
	b.WriteString(footer)
	b.WriteString("\n")

	// Help line.
	if m.showHelp {
		b.WriteString(m.help.View(m.keys))
	} else {
		b.WriteString(m.help.ShortHelpView(m.keys.ShortHelp()))
	}

	return b.String()
}

// ---------------------------------------------------------------------------
// Render helpers
// ---------------------------------------------------------------------------

// renderList renders the item list with cursor and checkboxes.
func (m SelectorModel) renderList(filtered []filteredEntry, maxHeight int) string {
	if len(filtered) == 0 {
		if m.filterInput != "" {
			return MutedStyle.Render("  No matches for \"" + m.filterInput + "\"")
		}
		return MutedStyle.Render("  No items")
	}

	// Viewport scrolling: determine which items to show.
	start := 0
	if m.cursor >= maxHeight {
		start = m.cursor - maxHeight + 1
	}
	end := start + maxHeight
	if end > len(filtered) {
		end = len(filtered)
		start = end - maxHeight
		if start < 0 {
			start = 0
		}
	}

	var lines []string
	for i := start; i < end; i++ {
		entry := filtered[i]
		item := entry.item
		origIdx := entry.originalIndex

		// Cursor indicator.
		var cursorStr string
		if i == m.cursor {
			cursorStr = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Render("> ")
		} else {
			cursorStr = "  "
		}

		// Checkbox.
		var checkbox string
		if m.allSelected[origIdx] {
			checkbox = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render("[\u2713]")
		} else {
			checkbox = "[ ]"
		}

		// Path and metadata.
		path := item.Label()
		meta := MutedStyle.Render(item.Metadata())

		line := fmt.Sprintf("%s%s %s %s", cursorStr, checkbox, path, meta)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// ---------------------------------------------------------------------------
// Filtering and indexing
// ---------------------------------------------------------------------------

// filteredEntry pairs a SelectorItem with its original index in m.items.
type filteredEntry struct {
	item          SelectorItem
	originalIndex int
}

// filteredItems returns items matching the current filter.
func (m SelectorModel) filteredItems() []filteredEntry {
	if m.filterInput == "" {
		entries := make([]filteredEntry, len(m.items))
		for i, item := range m.items {
			entries[i] = filteredEntry{item: item, originalIndex: i}
		}
		return entries
	}

	needle := strings.ToLower(m.filterInput)
	var entries []filteredEntry
	for i, item := range m.items {
		if strings.Contains(strings.ToLower(item.Path()), needle) {
			entries = append(entries, filteredEntry{item: item, originalIndex: i})
		}
	}
	return entries
}

// filteredIndices returns the original indices of all currently visible items.
func (m SelectorModel) filteredIndices() []int {
	entries := m.filteredItems()
	indices := make([]int, len(entries))
	for i, e := range entries {
		indices[i] = e.originalIndex
	}
	return indices
}

// originalIndex returns the original item index for the given filtered cursor position.
// Returns -1 if the cursor is out of range.
func (m SelectorModel) originalIndex(cursor int) int {
	entries := m.filteredItems()
	if cursor < 0 || cursor >= len(entries) {
		return -1
	}
	return entries[cursor].originalIndex
}

// selectedCount returns the number of currently selected items.
func (m SelectorModel) selectedCount() int {
	count := 0
	for _, v := range m.allSelected {
		if v {
			count++
		}
	}
	return count
}

// ---------------------------------------------------------------------------
// Sorting
// ---------------------------------------------------------------------------

// sortItems sorts m.items by the current sort mode and clears allSelected
// mapping since indices change. To preserve selections, we rebuild the map.
func (m *SelectorModel) sortItems() {
	// Build reverse mapping: old index -> item.
	type indexedItem struct {
		item     SelectorItem
		selected bool
	}
	old := make([]indexedItem, len(m.items))
	for i, item := range m.items {
		old[i] = indexedItem{item: item, selected: m.allSelected[i]}
	}

	switch m.sortMode {
	case SortByPath:
		sort.Slice(old, func(i, j int) bool {
			return old[i].item.Path() < old[j].item.Path()
		})
	case SortByStatus:
		sort.Slice(old, func(i, j int) bool {
			si, _ := old[i].item.(SubmoduleItem)
			sj, _ := old[j].item.(SubmoduleItem)
			if si.Info == nil || sj.Info == nil {
				return old[i].item.Path() < old[j].item.Path()
			}
			return statusSortPriority(si.Info.PrimaryStatus()) < statusSortPriority(sj.Info.PrimaryStatus())
		})
	case SortByBehind:
		sort.Slice(old, func(i, j int) bool {
			si, _ := old[i].item.(SubmoduleItem)
			sj, _ := old[j].item.(SubmoduleItem)
			if si.Info == nil || sj.Info == nil {
				return false
			}
			return si.Info.CommitsBehind > sj.Info.CommitsBehind
		})
	}

	// Rebuild items and selection map.
	newSelected := make(map[int]bool)
	for i, ii := range old {
		m.items[i] = ii.item
		if ii.selected {
			newSelected[i] = true
		}
	}
	m.allSelected = newSelected
}

// statusSortPriority returns a sort order for statuses (lower = first).
func statusSortPriority(s git.SubmoduleStatus) int {
	priorities := map[git.SubmoduleStatus]int{
		git.StatusError:    0,
		git.StatusConflict: 1,
		git.StatusModified: 2,
		git.StatusAhead:    3,
		git.StatusPending:  4,
		git.StatusMissing:  5,
		git.StatusSkipped:  6,
		git.StatusCurrent:  7,
	}
	p, ok := priorities[s]
	if !ok {
		return 99
	}
	return p
}

// ---------------------------------------------------------------------------
// Viewport / detail pane helpers
// ---------------------------------------------------------------------------

// resizeViewport adjusts the viewport dimensions based on current layout.
func (m *SelectorModel) resizeViewport() {
	headerLines := 2
	if m.subtitle != "" {
		headerLines++
	}
	if m.filtering || m.filterInput != "" {
		headerLines++
	}
	footerLines := 2
	if m.showHelp {
		footerLines += 3
	}

	vpHeight := m.height - headerLines - footerLines
	if vpHeight < 1 {
		vpHeight = 5
	}

	if m.showDetail && m.width > 40 {
		m.viewport.Width = m.width/2 - 1
	} else {
		m.viewport.Width = m.width
	}
	m.viewport.Height = vpHeight
}

// updateDetailPane sets viewport content to the currently highlighted item's detail.
func (m *SelectorModel) updateDetailPane() {
	filtered := m.filteredItems()
	if m.cursor >= 0 && m.cursor < len(filtered) {
		m.viewport.SetContent(filtered[m.cursor].item.DetailContent())
		m.viewport.GotoTop()
	}
}

// ---------------------------------------------------------------------------
// Public accessors (used by callers after Run() completes)
// ---------------------------------------------------------------------------

// Selected returns the SubmoduleInfo pointers for all selected items.
func (m SelectorModel) Selected() []*engine.SubmoduleInfo {
	var result []*engine.SubmoduleInfo
	for idx, sel := range m.allSelected {
		if !sel || idx < 0 || idx >= len(m.items) {
			continue
		}
		si, ok := m.items[idx].(SubmoduleItem)
		if ok {
			result = append(result, si.Info)
		}
	}
	return result
}

// SelectedPaths returns the Path() of all selected items regardless of concrete type.
func (m SelectorModel) SelectedPaths() []string {
	var paths []string
	for idx, sel := range m.allSelected {
		if !sel || idx < 0 || idx >= len(m.items) {
			continue
		}
		paths = append(paths, m.items[idx].Path())
	}
	return paths
}

// Cancelled reports whether the user cancelled selection.
func (m SelectorModel) Cancelled() bool { return m.cancelled }

// Confirmed reports whether the user confirmed selection.
func (m SelectorModel) Confirmed() bool { return m.confirmed }

