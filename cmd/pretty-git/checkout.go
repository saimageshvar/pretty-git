package main

import (
    "github.com/spf13/cobra"

    "pretty-git/internal/git"
)

func NewCheckoutCmd() *cobra.Command {
    var create bool
    var parent string

    cmd := &cobra.Command{
        Use:   "checkout [branch]",
        Short: "Wrapper around git checkout that records parent metadata when creating branches",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            branch := args[0]

            if create {
                // determine previous current branch
                prev, err := git.GetCurrentBranch()
                if err != nil {
                    return err
                }

                // if --parent provided, use it
                from := parent
                if from == "" {
                    from = prev
                }

                if err := git.CheckoutBranch(branch, true, from); err != nil {
                    return err
                }

                // record parent metadata
                if err := git.SetParent(branch, from); err != nil {
                    return err
                }

                return nil
            }

            // simple checkout
            if err := git.CheckoutBranch(branch, false, ""); err != nil {
                return err
            }

            return nil
        },
    }

    cmd.Flags().BoolVarP(&create, "create", "b", false, "create a new branch (like -b)")
    cmd.Flags().StringVar(&parent, "parent", "", "explicit parent branch to base the new branch on")

    return cmd
}
