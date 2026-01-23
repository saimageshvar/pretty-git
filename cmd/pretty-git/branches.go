package main

import (
    "fmt"

    "github.com/spf13/cobra"

    "pretty-git/internal/git"
    "pretty-git/internal/ui"
)

func NewBranchesCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "branches",
        Short: "Render branch parent→child tree",
        RunE: func(cmd *cobra.Command, args []string) error {
            parents, err := git.AllParents()
            if err != nil {
                return err
            }

            current, err := git.GetCurrentBranch()
            if err != nil {
                // allow rendering even when detached? propagate for now
                return err
            }

            out, err := ui.RenderBranchesTree(parents, current)
            if err != nil {
                return err
            }

            fmt.Print(out)
            return nil
        },
    }
}
