package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"pretty-git/internal/git"
	"pretty-git/internal/ui"
)

func NewLogCmd() *cobra.Command {
	var showAll bool
	var multiline bool
	var noColor bool
	var chronological bool
	var ascii bool

	cmd := &cobra.Command{
		Use:   "log",
		Short: "Show commits with parent-awareness and smart scoping",
		Long: `Display commits with visual distinction between commits unique to the current branch
and commits inherited from the parent branch.

By default, shows only commits unique to the current branch (branch-only commits).
Use --all to show full history including inherited commits from the parent.

Formats:
  Default: Oneline format (hash message [markers])
  --multiline: Detailed format with author, date, and stats
  --chronological: Show commits in time order (only with --all)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get current branch
			branch, err := git.GetCurrentBranch()
			if err != nil {
				// Handle detached HEAD - just show message and continue with HEAD
				if branch == "" {
					fmt.Fprintln(os.Stderr, "Warning: In detached HEAD state. Showing commits from HEAD.")
					branch = "HEAD"
				} else {
					return fmt.Errorf("failed to get current branch: %w", err)
				}
			}

			// Get parent branch
			parent, hasParent, err := git.GetParent(branch)
			if err != nil {
				return err
			}

			// If no parent and --all is specified, inform user
			if showAll && !hasParent {
				fmt.Fprintln(os.Stderr, "Note: No parent configured. Use 'pretty-git set --parent <branch>' to set one.")
				fmt.Fprintln(os.Stderr, "Showing all commits on this branch.\n")
			}

			// Apply color toggle
			ui.EnableColor = !noColor

			// Render the log
			out, err := ui.RenderCommitLog(branch, parent, hasParent, showAll, multiline, chronological, ascii)
			if err != nil {
				return err
			}

			fmt.Print(out)
			return nil
		},
	}

	cmd.Flags().BoolVar(&showAll, "all", false, "show full history including parent commits")
	cmd.Flags().BoolVar(&multiline, "multiline", false, "use detailed multiline format")
	cmd.Flags().BoolVar(&noColor, "no-color", false, "disable colored output")
	cmd.Flags().BoolVar(&chronological, "chronological", false, "show commits in chronological order (only with --all)")
	cmd.Flags().BoolVar(&ascii, "ascii", false, "use ASCII symbols instead of Unicode")

	return cmd
}
