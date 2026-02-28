package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"

	branchui "github.com/sai/pretty-git/internal/ui/branch"
	"github.com/sai/pretty-git/internal/git"
)

// runBranchCmd parses flags for `pgit branch` and launches the TUI.
func runBranchCmd(args []string) {
	fs := flag.NewFlagSet("branch", flag.ExitOnError)
	splitFlag := fs.Bool("split", false, "show vertical detail pane")
	fs.BoolVar(splitFlag, "s", false, "show vertical detail pane (shorthand)")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "pgit: %v\n", err)
		os.Exit(1)
	}
	runBranchImpl(*splitFlag)
}

// runBranch launches the branch switcher without a split pane.
// Used by `pgit checkout` (no args) so it never gets the split flag.
func runBranch() {
	runBranchImpl(false)
}

func runBranchImpl(splitPane bool) {
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

	m := branchui.New(branches, repoName, width, height, splitPane)

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
