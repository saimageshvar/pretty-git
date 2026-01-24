package main

import (
    "fmt"
    "os"

    "github.com/spf13/cobra"
)

func main() {
    if err := Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func Execute() error {
    root := &cobra.Command{
        Use:   "pretty-git",
        Short: "Pretty visualization and metadata helpers for git branches",
    }

    root.AddCommand(NewCheckoutCmd())
    root.AddCommand(NewBranchesCmd())
    root.AddCommand(NewSetParentCmd())

    return root.Execute()
}
