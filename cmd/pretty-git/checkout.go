package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"

	"github.com/sai/pretty-git/internal/git"
	checkoutui "github.com/sai/pretty-git/internal/ui/checkout"
)

// runCheckout handles `pgit checkout -b [name] [-p parent] [-d desc] [--layout=below|right]`.
func runCheckout(args []string) {
	// Only support `-b` sub-command for now.
	if len(args) == 0 || args[0] != "-b" {
		fmt.Fprintln(os.Stderr, "usage: pgit checkout -b [branch-name] [-p parent] [-d description]")
		os.Exit(1)
	}

	fs := flag.NewFlagSet("checkout -b", flag.ExitOnError)
	parentFlag := fs.String("p", "", "parent branch")
	descFlag := fs.String("d", "", "branch description")

	// The first positional arg after "-b" is the branch name (optional).
	// Parse remaining args after "-b".
	remaining := args[1:]
	// Separate branch name from flags (branch name must come before any -flag).
	var branchName string
	if len(remaining) > 0 && !strings.HasPrefix(remaining[0], "-") {
		branchName = remaining[0]
		remaining = remaining[1:]
	}

	if err := fs.Parse(remaining); err != nil {
		fmt.Fprintf(os.Stderr, "pgit: %v\n", err)
		os.Exit(1)
	}

	parent := *parentFlag
	desc := *descFlag

	// Auto-prefill parent from the current branch when not explicitly provided.
	// initialParent is used for the TUI form; parent (flag value) drives the
	// "skip TUI" shortcut so auto-fill never bypasses the form.
	initialParent := parent
	if initialParent == "" {
		initialParent = git.CurrentBranch()
	}

	// ── All three provided via flags: create directly without TUI ──────────
	if branchName != "" && parent != "" && desc != "" {
		if err := git.CreateBranch(branchName); err != nil {
			fmt.Fprintf(os.Stderr, "pgit: %v\n", err)
			os.Exit(1)
		}
		if err := git.SetParent(branchName, parent); err != nil {
			fmt.Fprintf(os.Stderr, "pgit: warning: could not set parent: %v\n", err)
		}
		if err := git.SetDescription(branchName, desc); err != nil {
			fmt.Fprintf(os.Stderr, "pgit: warning: could not set description: %v\n", err)
		}
		printCreated(branchName, parent, desc)
		return
	}

	// ── At least one field missing: open TUI form ───────────────────────────
	branches, err := git.ListBranches()
	if err != nil {
		fmt.Fprintf(os.Stderr, "pgit: %v\n", err)
		os.Exit(1)
	}

	repoName := git.RepoName()

	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		width, height = 120, 40
	}

	m := checkoutui.New(branches, repoName, width, height,
		branchName, initialParent, desc)

	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	result, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "pgit: %v\n", err)
		os.Exit(1)
	}

	final, ok := result.(checkoutui.Model)
	if !ok || final.WasQuit() {
		return
	}

	r := final.Result()
	printCreated(r.Name, r.Parent, r.Description)
}

func printCreated(name, parent, desc string) {
	fmt.Printf("✓ Created branch '%s'\n", name)
	fmt.Printf("  Branch: %s\n", name)
	if parent != "" {
		fmt.Printf("  Parent: %s\n", parent)
	}
	if desc != "" {
		fmt.Printf("  Desc:   %s\n", desc)
	}
}
