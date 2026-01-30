package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"pretty-git/internal/git"
)

func NewCheckoutCmd() *cobra.Command {
	var create bool
	var parent string
	var description string
	var updateParent bool
	var yes bool

	cmd := &cobra.Command{
		Use:   "checkout [branch]",
		Short: "Wrapper around git checkout that records parent metadata when creating branches",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			branch := args[0]

			// capture current branch before switching (used as implicit parent)
			prev, err := git.GetCurrentBranch()
			if err != nil {
				return err
			}

			if create {
				// if --parent provided, use it
				from := parent
				if from == "" {
					from = prev
				}

				if err := git.CheckoutBranch(branch, true, from); err != nil {
					return err
				}

				// check existing parent metadata
				if existing, ok, err := git.GetParent(branch); err != nil {
					return err
				} else if ok && !updateParent {
					return fmt.Errorf("parent metadata for branch '%s' already exists (parent=%s); use --update-parent to overwrite", branch, existing)
				}

				// record parent metadata (SetParent will create a backup if overwriting)
				if err := git.SetParent(branch, from); err != nil {
					return err
				}

				// record description if provided
				if description != "" {
					if err := git.SetDescription(branch, description); err != nil {
						return fmt.Errorf("failed to set description: %w", err)
					}
				}

				return nil
			}

			// non-create checkout
			if err := git.CheckoutBranch(branch, false, ""); err != nil {
				return err
			}

			if updateParent {
				// determine which parent to set (explicit flag or previous branch)
				from := parent
				if from == "" {
					from = prev
				}

				// if parent already exists, ask for confirmation unless --yes
				if existing, ok, err := git.GetParent(branch); err != nil {
					return err
				} else if ok {
					if !yes {
						reader := bufio.NewReader(os.Stdin)
						fmt.Printf("branch '%s' already has parent '%s'. Overwrite and create backup? (y/N): ", branch, existing)
						resp, _ := reader.ReadString('\n')
						resp = strings.TrimSpace(resp)
						if strings.ToLower(resp) != "y" && strings.ToLower(resp) != "yes" {
							return fmt.Errorf("aborted by user")
						}
					}
				}

				if err := git.SetParent(branch, from); err != nil {
					return err
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&create, "create", "b", false, "create a new branch (like -b)")
	cmd.Flags().StringVar(&parent, "parent", "", "explicit parent branch to base the new branch on")
	cmd.Flags().StringVar(&description, "desc", "", "description for the new branch")
	cmd.Flags().BoolVar(&updateParent, "update-parent", false, "when switching to an existing branch, update its parent metadata (requires explicit opt-in)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "assume yes for confirmations when updating parent metadata")

	return cmd
}
