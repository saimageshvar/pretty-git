package log

import "github.com/charmbracelet/bubbles/key"

// pane identifies which pane currently has focus.
type pane int

const (
	paneList   pane = 0
	paneDetail pane = 1
)

type keyMap struct {
	// List pane navigation
	Up       key.Binding
	Down     key.Binding
	PageUp   key.Binding
	PageDown key.Binding

	// Pane switching
	FocusDetail key.Binding // → enter detail pane
	FocusList   key.Binding // ← return to list pane

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
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c", "esc"),
			key.WithHelp("q", "quit"),
		),
	}
}

// listHelp is shown when the list pane is focused.
func (k keyMap) listHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.FocusDetail, k.Quit}
}

// detailHelp is shown when the detail pane is focused.
func (k keyMap) detailHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.FocusList, k.Quit}
}

func (k keyMap) ShortHelp() []key.Binding { return k.listHelp() }
func (k keyMap) FullHelp() [][]key.Binding { return [][]key.Binding{k.listHelp()} }
