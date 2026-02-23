package stash

import "github.com/charmbracelet/bubbles/key"

// createPhase identifies which phase of the create wizard is active.
type createPhase int

const (
	phaseTypeSelect  createPhase = 0 // Phase 0: choose stash type
	phaseFileSelect  createPhase = 1 // Phase 1: multi-select files (custom only)
	phaseMessage     createPhase = 2 // Phase 2: enter message
	phaseExecuting   createPhase = 3 // Phase 3: running git, show spinner
)

type createKeyMap struct {
	Up      key.Binding
	Down    key.Binding
	Select  key.Binding
	Toggle  key.Binding
	All     key.Binding
	None    key.Binding
	Confirm key.Binding
	Back    key.Binding
	Quit    key.Binding
}

func defaultCreateKeyMap() createKeyMap {
	return createKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Toggle: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "toggle"),
		),
		All: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "all"),
		),
		None: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "none"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "confirm"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "quit"),
		),
	}
}

func (k createKeyMap) typeSelectHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Select, k.Back}
}

func (k createKeyMap) fileSelectHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Toggle, k.All, k.None, k.Confirm, k.Back}
}

func (k createKeyMap) messageHelp() []key.Binding {
	return []key.Binding{k.Confirm, k.Back}
}
