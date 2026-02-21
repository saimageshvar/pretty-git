package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: pgit <command>")
		fmt.Fprintln(os.Stderr, "commands: branch")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "branch":
		runBranch()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}
