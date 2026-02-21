package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"

	branchui "github.com/sai/pretty-git/internal/ui/branch"
	"github.com/sai/pretty-git/internal/git"
)

func runBranch() {
	branches, err := git.ListBranches()
	if err != nil {
		fmt.Fprintf(os.Stderr, "pgit: %v\n", err)
		os.Exit(1)
	}
	if len(branches) == 0 {
		fmt.Fprintln(os.Stderr, "pgit: no branches found")
		os.Exit(0)
	}

	repoName := git.RepoName()

	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		width, height = 120, 40
	}

	m := branchui.New(branches, repoName, width, height)

	// Inline mode — no WithAltScreen
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	result, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "pgit: %v\n", err)
		os.Exit(1)
	}

	final, ok := result.(branchui.Model)
	if ok && final.SwitchedTo() != "" {
		fmt.Printf("✓ Switched to '%s'\n", final.SwitchedTo())
	}
}
