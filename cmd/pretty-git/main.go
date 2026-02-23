package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: pgit <command>")
		fmt.Fprintln(os.Stderr, "  branch                    browse & switch branches")
		fmt.Fprintln(os.Stderr, "  checkout                  browse & switch branches")
		fmt.Fprintln(os.Stderr, "  checkout <name>           switch to branch (create if missing)")
		fmt.Fprintln(os.Stderr, "  checkout -b [name]        create new branch")
		fmt.Fprintln(os.Stderr, "  log                       browse commit log")
		fmt.Fprintln(os.Stderr, "  stash                     interactive stash create wizard")
		fmt.Fprintln(os.Stderr, "  stash apply|pop|drop      browse stashes and apply/pop/drop")
		fmt.Fprintln(os.Stderr, "  stash \"msg\"               quick stash all with message")
		fmt.Fprintln(os.Stderr, "  stash --staged \"msg\"      quick stash staged only")
		fmt.Fprintln(os.Stderr, "  stash --unstaged \"msg\"    quick stash unstaged only")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "branch":
		runBranch()
	case "checkout":
		runCheckout(os.Args[2:])
	case "log":
		runLog()
	case "stash":
		runStash(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}
