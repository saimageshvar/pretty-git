package main

import (
	"context"
	"fmt"
	"os"

	"github.com/sai/pretty-git/internal/update"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: pgit <command>")
		fmt.Fprintln(os.Stderr, "  branch                    browse & switch branches")
		fmt.Fprintln(os.Stderr, "  checkout                  browse & switch branches")
		fmt.Fprintln(os.Stderr, "  checkout <name>           switch to branch (create if missing)")
		fmt.Fprintln(os.Stderr, "  checkout -b [name]        create new branch")
		fmt.Fprintln(os.Stderr, "  log                       browse commit log")
		fmt.Fprintln(os.Stderr, "  merge                     merge current branch into selected branch")
		fmt.Fprintln(os.Stderr, "  prompt                    current branch & description for shell prompt")
		fmt.Fprintln(os.Stderr, "  stash                     interactive stash create wizard")
		fmt.Fprintln(os.Stderr, "  stash apply|pop|drop      browse stashes and apply/pop/drop")
		fmt.Fprintln(os.Stderr, "  stash \"msg\"               quick stash all with message")
		fmt.Fprintln(os.Stderr, "  stash --staged \"msg\"      quick stash staged only")
		fmt.Fprintln(os.Stderr, "  stash --unstaged \"msg\"    quick stash unstaged only")
		fmt.Fprintln(os.Stderr, "  merge                     pick & merge a branch")
		os.Exit(1)
	}

	runWithUpdate := func(command string, run func()) {
		update.MaybeNotifyAndUpdate(context.Background(), command, version)
		run()
	}

	switch os.Args[1] {
	case "branch":
		runWithUpdate("branch", runBranch)
	case "checkout":
		runWithUpdate("checkout", func() { runCheckout(os.Args[2:]) })
	case "log":
		runWithUpdate("log", runLog)
	case "merge":
		runWithUpdate("merge", runMerge)
	case "prompt":
		runWithUpdate("prompt", func() { runPrompt(os.Args[2:]) })
	case "stash":
		runWithUpdate("stash", func() { runStash(os.Args[2:]) })
	case "merge":
		runWithUpdate("merge", runMerge)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}
