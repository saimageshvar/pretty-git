package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"

	"github.com/sai/pretty-git/internal/git"
	"github.com/sai/pretty-git/internal/gh"
	ghprui "github.com/sai/pretty-git/internal/ui/ghpr"
)

func runGH(args []string) {
	// Check if gh CLI is available
	if !gh.IsAvailable() {
		fmt.Fprintln(os.Stderr, "pgit: GitHub CLI (gh) is not installed or not authenticated")
		fmt.Fprintln(os.Stderr, "  Install: https://cli.github.com/")
		fmt.Fprintln(os.Stderr, "  Authenticate: gh auth login")
		os.Exit(1)
	}

	// Parse subcommand
	if len(args) == 0 {
		runGHPR()
		return
	}

	switch args[0] {
	case "pr":
		runGHPR()
	default:
		fmt.Fprintf(os.Stderr, "pgit gh: unknown subcommand: %s\n", args[0])
		fmt.Fprintln(os.Stderr, "  pgit gh        list your pull requests")
		fmt.Fprintln(os.Stderr, "  pgit gh pr     list your pull requests")
		os.Exit(1)
	}
}

func runGHPR() {
	repoName := git.RepoName()

	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		width, height = 160, 40
	}

	m := ghprui.New(repoName, width, height)

	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "pgit: %v\n", err)
		os.Exit(1)
	}
}