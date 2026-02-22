package branch

import (
	"github.com/charmbracelet/bubbles/key"
)

// keyMap defines all keybindings for the branch view.
type keyMap struct {
	Up      key.Binding
	Down    key.Binding
	Switch  key.Binding
	Edit    key.Binding
	EscBack key.Binding // esc: clear filter if active, else quit
	Quit    key.Binding // ctrl+c: force quit
}

// defaultKeyMap returns the keybindings used by the branch view.
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
		Switch: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "switch"),
		),
		Edit: key.NewBinding(
			key.WithKeys("ctrl+e"),
			key.WithHelp("ctrl+e", "edit"),
		),
		EscBack: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "clear/quit"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
		),
	}
}

// ShortHelp implements help.KeyMap — shown in normal navigation mode.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Switch, k.Edit, k.EscBack}
}

// FullHelp implements help.KeyMap — not used, but required by the interface.
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp()}
}
