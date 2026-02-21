package branch

import (
	"github.com/charmbracelet/bubbles/key"
)

// keyMap defines all keybindings for the branch view.
type keyMap struct {
	Up     key.Binding
	Down   key.Binding
	Filter key.Binding
	Switch key.Binding
	Quit   key.Binding

	// Filter-mode bindings
	Confirm key.Binding
	Clear   key.Binding
}

// defaultKeyMap returns the keybindings used by the branch view.
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
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		Switch: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "switch"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "confirm"),
		),
		Clear: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "clear"),
		),
	}
}

// ShortHelp implements help.KeyMap — shown in normal navigation mode.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Switch, k.Filter, k.Quit}
}

// FullHelp implements help.KeyMap — not used, but required by the interface.
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp()}
}

// filterShortHelp returns the bindings shown while in filter mode.
func (k keyMap) filterShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Confirm, k.Clear}
}
