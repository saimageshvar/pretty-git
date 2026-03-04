package stash

import "github.com/charmbracelet/bubbles/key"

// createPhase identifies which phase of the create wizard is active.
type createPhase int

const (
	phaseTypeSelect createPhase = 0 // Phase 0: choose type & configure custom files
	phaseMessage    createPhase = 1 // Phase 1: enter stash message
	phaseExecuting  createPhase = 2 // Phase 2: running git stash, show spinner
)

type createKeyMap struct {
	Up         key.Binding
	Down       key.Binding
	FocusRight key.Binding
	FocusLeft  key.Binding
	Select     key.Binding // enter — advance to message
	Toggle     key.Binding // space — toggle file checkbox
	All        key.Binding // a — select all
	None       key.Binding // n — deselect all
	Confirm    key.Binding // enter — confirm message
	Back       key.Binding // esc — back / quit
	Quit       key.Binding // ctrl+c — quit
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
		FocusRight: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→", "file pane"),
		),
		FocusLeft: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←", "type pane"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "next"),
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
			key.WithHelp("enter", "create stash"),
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

func (k createKeyMap) leftPaneHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.FocusRight, k.Select, k.Back}
}

func (k createKeyMap) rightPaneCustomHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Toggle, k.All, k.None, k.FocusLeft, k.Select}
}

func (k createKeyMap) rightPanePreviewHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.FocusLeft, k.Select}
}

func (k createKeyMap) messageHelp() []key.Binding {
	return []key.Binding{k.Confirm, k.Back}
}
