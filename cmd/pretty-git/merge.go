package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"

	mergeui "github.com/sai/pretty-git/internal/ui/merge"
	"github.com/sai/pretty-git/internal/git"
)

func runMerge() {
	branches, err := git.ListLocalBranches()
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

	m := mergeui.New(branches, repoName, width, height)

	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	result, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "pgit: %v\n", err)
		os.Exit(1)
	}

	final, ok := result.(mergeui.Model)
	if ok && final.MergedTo() != "" {
		fmt.Printf("✓ Merged '%s' into '%s'\n", final.MergedFrom(), final.MergedTo())
	}
}