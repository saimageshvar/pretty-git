package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"pretty-git/internal/ui"
)

func NewBrowseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "browse",
		Short: "Interactive TUI for browsing and managing branches",
		Long: `Launch an interactive terminal UI for navigating branch hierarchy,
expanding/collapsing parent-child trees, and performing quick actions
like checkout, set parent, and inspect metadata.

Controls:
  ↑/k, ↓/j   - Navigate branches
  Space      - Toggle expand/collapse
  Enter      - Checkout selected branch
  p          - Set parent for selected branch
  i          - Inspect branch metadata
  q, Ctrl+C  - Quit`,
		RunE: func(cmd *cobra.Command, args []string) error {
			model, err := ui.NewTUIModel()
			if err != nil {
				return err
			}

			p := tea.NewProgram(model)
			if _, err := p.Run(); err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}
