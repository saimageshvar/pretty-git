package stash

import "github.com/charmbracelet/bubbles/key"

// browsePaneMode identifies which pane the browse model is focused on.
type browsePaneMode int

const (
	browseList   browsePaneMode = 0
	browseDetail browsePaneMode = 1
)

// BrowseMode is the action the browser was launched with.
type BrowseMode int

const (
	BrowseModeApply BrowseMode = iota
	BrowseModePop
	BrowseModeDrop
)

func (m BrowseMode) String() string {
	switch m {
	case BrowseModeApply:
		return "apply"
	case BrowseModePop:
		return "pop"
	case BrowseModeDrop:
		return "drop"
	}
	return "apply"
}

type browseKeyMap struct {
	Up           key.Binding
	Down         key.Binding
	FocusDetail  key.Binding
	FocusList    key.Binding
	Action       key.Binding // enter — apply/pop/drop
	ConfirmDrop  key.Binding // y — confirm drop
	CancelDrop   key.Binding // n/esc — cancel drop
	Quit         key.Binding
}

func defaultBrowseKeyMap() browseKeyMap {
	return browseKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		FocusDetail: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "detail"),
		),
		FocusList: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "back"),
		),
		Action: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "action"),
		),
		ConfirmDrop: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "confirm"),
		),
		CancelDrop: key.NewBinding(
			key.WithKeys("n", "esc"),
			key.WithHelp("n/esc", "cancel"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

func (k browseKeyMap) listHelp(mode BrowseMode) []key.Binding {
	action := key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", mode.String()))
	return []key.Binding{k.Up, k.Down, k.FocusDetail, action, k.Quit}
}

func (k browseKeyMap) detailHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.FocusList, k.Quit}
}

func (k browseKeyMap) confirmHelp() []key.Binding {
	return []key.Binding{k.ConfirmDrop, k.CancelDrop}
}

func (k browseKeyMap) ShortHelp() []key.Binding        { return k.listHelp(BrowseModeApply) }
func (k browseKeyMap) FullHelp() [][]key.Binding        { return [][]key.Binding{k.listHelp(BrowseModeApply)} }
