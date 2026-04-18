package main

import (
	"fmt"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"

	"github.com/sai/pretty-git/internal/git"
	branchui "github.com/sai/pretty-git/internal/ui/branch"
)

func runMerge() {
	// ── Pre-flight checks ──────────────────────────────────────────────
	currentBranch := git.CurrentBranch()
	if currentBranch == "" {
		fmt.Fprintln(os.Stderr, "pgit: not a git repository or detached HEAD")
		os.Exit(1)
	}

	branches, err := git.ListLocalBranches()
	if err != nil {
		fmt.Fprintf(os.Stderr, "pgit: %v\n", err)
		os.Exit(1)
	}

	// Count branches excluding the current one
	mergeable := 0
	for _, b := range branches {
		if !b.IsCurrent {
			mergeable++
		}
	}
	if mergeable == 0 {
		fmt.Fprintln(os.Stderr, "pgit: no other local branches to merge")
		os.Exit(0)
	}

	repoName := git.RepoName()

	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		width, height = 120, 40
	}

	// ── Open branch picker in select mode ──────────────────────────────
	m := branchui.New(branches, repoName, width, height, branchui.ModeSelect)

	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	result, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "pgit: %v\n", err)
		os.Exit(1)
	}

	final, ok := result.(branchui.Model)
	if !ok || final.SwitchedTo() == "" {
		return // user quit without selecting
	}

	selectedBranch := final.SwitchedTo()

	// ── Run git merge with passthrough output ──────────────────────────
	cmd := exec.Command("git", "merge", selectedBranch)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			// Non-exit error (e.g. command not found)
			fmt.Fprintf(os.Stderr, "pgit: %v\n", err)
			os.Exit(1)
		}
	}

	switch exitCode {
	case 0:
		fmt.Printf("✓ Merged '%s' into '%s'\n", selectedBranch, currentBranch)
	case 1:
		// Conflicts — git already printed instructions (git merge --continue, --abort)
		// Stay silent and let the user follow git's own guidance.
	default:
		os.Exit(exitCode)
	}
}
