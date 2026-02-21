package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"

	"github.com/sai/pretty-git/internal/git"
	logui "github.com/sai/pretty-git/internal/ui/log"
)

func runLog() {
	// Optional ref argument: pgit log [<ref>]
	ref := "HEAD"
	if len(os.Args) >= 3 {
		ref = os.Args[2]
	}

	commits, err := git.ListCommits(ref, 200)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pgit: %v\n", err)
		os.Exit(1)
	}
	if len(commits) == 0 {
		fmt.Fprintln(os.Stderr, "pgit: no commits found")
		os.Exit(0)
	}

	repoName := git.RepoName()

	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		width, height = 160, 40
	}

	m := logui.New(commits, repoName, ref, width, height)

	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "pgit: %v\n", err)
		os.Exit(1)
	}
}
