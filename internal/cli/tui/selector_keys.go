package tui

import "github.com/charmbracelet/bubbles/key"

// SelectorKeyMap defines all keybindings for the multi-select selector.
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

// DefaultSelectorKeyMap returns the default keybindings matching bash-era
// conventions plus new features (filter, help, detail, sort).
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
			key.WithHelp("q/esc", "quit"),
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
			key.WithHelp("tab", "detail pane"),
		),
		Sort: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "sort"),
		),
	}
}

// ShortHelp implements help.KeyMap -- returns bindings for the compact help line.
func (k SelectorKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Toggle, k.Confirm, k.Quit}
}

// FullHelp implements help.KeyMap -- returns bindings for the full help display (two columns).
func (k SelectorKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Toggle, k.All, k.None},
		{k.Confirm, k.Quit, k.Filter, k.Help, k.Detail, k.Sort},
	}
}
