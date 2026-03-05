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
		fmt.Fprintln(os.Stderr, "  branch              browse & switch branches")
		fmt.Fprintln(os.Stderr, "  checkout            browse & switch branches")
		fmt.Fprintln(os.Stderr, "  checkout <name>     switch to branch (create if missing)")
		fmt.Fprintln(os.Stderr, "  checkout -b [name]  create new branch")
		fmt.Fprintln(os.Stderr, "  log                 browse commit log")
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
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}
