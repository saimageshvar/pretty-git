package main

import (
    "fmt"

    "github.com/spf13/cobra"

    "pretty-git/internal/git"
    "pretty-git/internal/ui"
)

func NewBranchesCmd() *cobra.Command {
    var compact bool
    var verbose bool
        var noColor bool
        var noMarker bool

    cmd := &cobra.Command{
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

                // apply style toggles
                ui.EnableColor = !noColor
                ui.ShowCurrentMarker = !noMarker

            out, err := ui.RenderBranchesTree(parents, current, compact, verbose)
            if err != nil {
                return err
            }

            fmt.Print(out)
            return nil
        },
    }

    cmd.Flags().BoolVar(&compact, "compact", false, "use compact layout with narrower indents")
    cmd.Flags().BoolVar(&verbose, "verbose", false, "show parent metadata inline for each branch")
        cmd.Flags().BoolVar(&noColor, "no-color", false, "disable colored output")
        cmd.Flags().BoolVar(&noMarker, "no-marker", false, "hide current-branch marker")

    return cmd
}
