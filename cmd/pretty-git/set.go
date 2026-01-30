package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"pretty-git/internal/git"
)

func NewSetCmd() *cobra.Command {
	var yes bool
	var parent string
	var description string

	cmd := &cobra.Command{
		Use:   "set [branch]",
		Short: "Set or update parent and/or description for a branch (defaults to current branch)",
		Long: `Set or update the recorded parent and/or description for a branch.
If no branch is specified, updates the current branch.
At least one of --parent or --desc must be provided.

Examples:
  # Set parent for current branch
  pretty-git set --parent main

  # Set description for current branch
  pretty-git set --desc "Feature implementation"

  # Set both parent and description for current branch
  pretty-git set --parent main --desc "Feature implementation"

  # Set parent and description for a named branch
  pretty-git set feature-x --parent develop --desc "New feature X"

  # Skip confirmation when overwriting existing values
  pretty-git set --parent main --yes`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate that at least one flag is provided
			if parent == "" && description == "" {
				return fmt.Errorf("at least one of --parent or --desc must be provided")
			}

			// Determine target branch
			var branch string
			if len(args) == 1 {
				branch = args[0]
			} else {
				cur, err := git.GetCurrentBranch()
				if err != nil {
					return err
				}
				branch = cur
			}

			// Handle parent update
			if parent != "" {
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

				if err := git.SetParent(branch, parent); err != nil {
					return fmt.Errorf("failed to set parent: %w", err)
				}
				fmt.Printf("Set parent for '%s' to '%s'\n", branch, parent)
			}

			// Handle description update
			if description != "" {
				if existing, ok, err := git.GetDescription(branch); err != nil {
					return err
				} else if ok && existing != "" {
					if !yes {
						reader := bufio.NewReader(os.Stdin)
						fmt.Printf("branch '%s' already has description '%s'. Overwrite? (y/N): ", branch, existing)
						resp, _ := reader.ReadString('\n')
						resp = strings.TrimSpace(resp)
						if strings.ToLower(resp) != "y" && strings.ToLower(resp) != "yes" {
							return fmt.Errorf("aborted by user")
						}
					}
				}

				if err := git.SetDescription(branch, description); err != nil {
					return fmt.Errorf("failed to set description: %w", err)
				}
				fmt.Printf("Set description for '%s'\n", branch)
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "assume yes for confirmations")
	cmd.Flags().StringVar(&parent, "parent", "", "parent branch to set")
	cmd.Flags().StringVar(&description, "desc", "", "description to set for the branch")

	return cmd
}

// NewSetParentCmd maintains backward compatibility with the old set-parent command
func NewSetParentCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:    "set-parent [branch] <parent>",
		Short:  "Set or update the recorded parent for a branch (defaults to current branch)",
		Hidden: true, // Hide from help but keep for backward compatibility
		Args:   cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var child, parent string
			if len(args) == 1 {
				// parent provided, child is current branch
				p := args[0]
				cur, err := git.GetCurrentBranch()
				if err != nil {
					return err
				}
				child = cur
				parent = p
			} else {
				child = args[0]
				parent = args[1]
			}

			if existing, ok, err := git.GetParent(child); err != nil {
				return err
			} else if ok {
				if !yes {
					reader := bufio.NewReader(os.Stdin)
					fmt.Printf("branch '%s' already has parent '%s'. Overwrite and create backup? (y/N): ", child, existing)
					resp, _ := reader.ReadString('\n')
					resp = strings.TrimSpace(resp)
					if strings.ToLower(resp) != "y" && strings.ToLower(resp) != "yes" {
						return fmt.Errorf("aborted by user")
					}
				}
			}

			if err := git.SetParent(child, parent); err != nil {
				return err
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "assume yes for confirmations")

	return cmd
}
