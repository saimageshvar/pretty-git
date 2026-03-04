package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"

	"github.com/sai/pretty-git/internal/git"
	stashui "github.com/sai/pretty-git/internal/ui/stash"
)

// runStash is the entry point for `pgit stash [subcommand] [args…]`.
// Routing:
//
//	pgit stash                         → create wizard (interactive)
//	pgit stash apply                   → browse in apply mode
//	pgit stash pop                     → browse in pop mode
//	pgit stash drop                    → browse in drop mode
//	pgit stash list                    → browse in apply mode (alias)
//	pgit stash "msg"                   → quick: stash all + message (no TUI)
//	pgit stash --staged "msg"          → quick: stash staged only + message
//	pgit stash --unstaged "msg"        → quick: stash unstaged only + message
//	pgit stash --custom "msg" -- f1 f2 → quick: stash specified files + message
func runStash(args []string) {
	repoName := git.RepoName()
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		width, height = 160, 40
	}

	// No args → interactive create wizard
	if len(args) == 0 {
		runStashCreate(repoName, width, height)
		return
	}

	sub := args[0]

	// Browse modes
	switch sub {
	case "apply", "list":
		runStashBrowse(repoName, stashui.BrowseModeApply, width, height)
		return
	case "pop":
		runStashBrowse(repoName, stashui.BrowseModePop, width, height)
		return
	case "drop":
		runStashBrowse(repoName, stashui.BrowseModeDrop, width, height)
		return
	}

	// Quick stash flags
	stashType := "all"
	msgArgs := args
	var customFiles []string

	switch sub {
	case "--staged":
		stashType = "staged"
		msgArgs = args[1:]
	case "--unstaged":
		stashType = "unstaged"
		msgArgs = args[1:]
	case "--custom":
		// pgit stash --custom "msg" -- file1 file2 ...
		stashType = "custom"
		rest := args[1:]
		sep := -1
		for i, a := range rest {
			if a == "--" {
				sep = i
				break
			}
		}
		if sep < 0 {
			fmt.Fprintln(os.Stderr, "pgit: --custom requires -- separator: pgit stash --custom \"msg\" -- file1 file2")
			os.Exit(1)
		}
		msgArgs = rest[:sep]
		customFiles = rest[sep+1:]
		if len(customFiles) == 0 {
			fmt.Fprintln(os.Stderr, "pgit: no files specified after --")
			os.Exit(1)
		}
	}

	// Join remaining args as the message
	msg := strings.Join(msgArgs, " ")
	if msg == "" {
		// No message given — fall back to interactive wizard
		runStashCreate(repoName, width, height)
		return
	}

	// Quick stash: check we have files to stash
	files, err := git.ListModifiedFiles()
	if err != nil {
		fmt.Fprintf(os.Stderr, "pgit: %v\n", err)
		os.Exit(1)
	}
	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "pgit: nothing to stash")
		os.Exit(0)
	}

	shortHash := git.LastCommitShortHash()
	finalMsg := shortHash + ": " + msg

	if err := git.StashPush(finalMsg, stashType, customFiles); err != nil {
		fmt.Fprintf(os.Stderr, "pgit: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "✓ stash created: %s\n", finalMsg)
}

func runStashCreate(repoName string, width, height int) {
	files, err := git.ListModifiedFiles()
	if err != nil {
		fmt.Fprintf(os.Stderr, "pgit: %v\n", err)
		os.Exit(1)
	}
	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "pgit: nothing to stash")
		os.Exit(0)
	}

	m := stashui.NewCreate(files, repoName, width, height)
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	result, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "pgit: %v\n", err)
		os.Exit(1)
	}
	if cm, ok := result.(stashui.CreateModel); ok {
		if msg := cm.Result(); msg != "" {
			fmt.Fprintln(os.Stderr, msg)
		}
	}
}

func runStashBrowse(repoName string, mode stashui.BrowseMode, width, height int) {
	stashes, err := git.ListStashes()
	if err != nil {
		fmt.Fprintf(os.Stderr, "pgit: %v\n", err)
		os.Exit(1)
	}
	if len(stashes) == 0 {
		fmt.Fprintln(os.Stderr, "pgit: no stashes found")
		os.Exit(0)
	}

	m := stashui.NewBrowse(stashes, repoName, mode, width, height)
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	result, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "pgit: %v\n", err)
		os.Exit(1)
	}
	if bm, ok := result.(stashui.BrowseModel); ok {
		if msg := bm.Result(); msg != "" {
			fmt.Fprintln(os.Stderr, msg)
		}
	}
}
