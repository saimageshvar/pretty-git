package log

import "github.com/charmbracelet/bubbles/key"

// pane identifies which pane currently has focus.
type pane int

const (
	paneList    pane = 0
	paneDetail  pane = 1
	paneFilters pane = 2
)

type keyMap struct {
	// List pane navigation
	Up       key.Binding
	Down     key.Binding
	PageUp   key.Binding
	PageDown key.Binding

	// Pane switching
	FocusDetail  key.Binding // → enter detail pane
	FocusList    key.Binding // ← return to list pane
	FocusFilters key.Binding // f  enter filter bar

	// Filter bar navigation (active when paneFilters focused)
	FilterLeft   key.Binding // ← move cursor left
	FilterRight  key.Binding // → move cursor right
	FilterToggle key.Binding // space toggle focused checkbox
	FilterExit   key.Binding // esc/f exit filter bar

	Quit key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "ctrl+u"),
			key.WithHelp("PgUp", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "ctrl+d"),
			key.WithHelp("PgDn", "page dn"),
		),
		FocusDetail: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→", "expand"),
		),
		FocusList: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←", "back"),
		),
		FocusFilters: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "filters"),
		),
		FilterLeft: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/→", "move"),
		),
		FilterRight: key.NewBinding(
			key.WithKeys("right", "l"),
		),
		FilterToggle: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "toggle"),
		),
		FilterExit: key.NewBinding(
			key.WithKeys("esc", "f", "enter"),
			key.WithHelp("esc", "done"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

// listHelp is shown when the list pane is focused.
func (k keyMap) listHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.FocusDetail, k.FocusFilters, k.Quit}
}

// detailHelp is shown when the detail pane is focused.
func (k keyMap) detailHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.FocusList, k.Quit}
}

// filterHelp is shown when the filter bar is focused.
func (k keyMap) filterHelp() []key.Binding {
	return []key.Binding{k.FilterLeft, k.FilterToggle, k.FilterExit, k.Quit}
}

func (k keyMap) ShortHelp() []key.Binding        { return k.listHelp() }
func (k keyMap) FullHelp() [][]key.Binding        { return [][]key.Binding{k.listHelp()} }
