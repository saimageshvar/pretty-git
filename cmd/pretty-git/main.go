package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: pgit <command>")
		fmt.Fprintln(os.Stderr, "  branch              browse & switch branches")
		fmt.Fprintln(os.Stderr, "  checkout            browse & switch branches")
		fmt.Fprintln(os.Stderr, "  checkout <name>     switch to branch (create if missing)")
		fmt.Fprintln(os.Stderr, "  checkout -b [name]  create new branch")
		fmt.Fprintln(os.Stderr, "  log                 browse commit log")
		fmt.Fprintln(os.Stderr, "  prompt              current branch & description for shell prompt")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "branch":
		runBranch()
	case "checkout":
		runCheckout(os.Args[2:])
	case "log":
		runLog()
	case "prompt":
		runPrompt(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}
