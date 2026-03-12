package merge

import (
	"github.com/charmbracelet/bubbles/key"
)

// keyMap defines all keybindings for the merge view.
type keyMap struct {
	Up      key.Binding
	Down    key.Binding
	Select  key.Binding
	EscBack key.Binding
	Quit    key.Binding
	Confirm key.Binding
	Cancel  key.Binding
}

// defaultKeyMap returns the keybindings used by the merge view.
func defaultKeyMap() keyMap {
	return keyMap{
		Up: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("↑", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("↓", "down"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		EscBack: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back/quit"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "confirm merge"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "cancel"),
		),
	}
}

// ShortHelp implements help.KeyMap — shown in normal navigation mode.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Select, k.EscBack}
}

// FullHelp implements help.KeyMap — not used, but required by the interface.
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp()}
}