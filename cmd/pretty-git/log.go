package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"pretty-git/internal/git"
	"pretty-git/internal/ui"
)

func NewLogCmd() *cobra.Command {
	var bySection bool
	var multiline bool
	var noColor bool
	var ascii bool
	var maxCommits int
	var usePager bool
	var forcePager bool

	cmd := &cobra.Command{
		Use:   "log",
		Short: "Show commits with parent-awareness and smart scoping",
		Long: `Display commits from the current branch with visual distinction between
commits unique to the branch and commits inherited from the parent.

By default, shows commits in chronological order from the current branch,
including both branch-unique commits and common commits shared with the parent.
Parent-only commits (commits in parent but not in current branch) are excluded.

Formats:
  Default: Chronological order (time-based)
  --by-section: Group commits by type (Current Branch Only, Common Commits)
  --multiline: Detailed format with author, date, and stats`,
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

			// If no parent, inform user
			if !hasParent {
				fmt.Fprintln(os.Stderr, "Note: No parent configured. Showing all commits on this branch.")
				fmt.Fprintln(os.Stderr, "Use 'pretty-git set --parent <branch>' to set a parent.\n")
			}

			// Apply color toggle
			ui.EnableColor = !noColor

			// Render the log (chronological is now default, bySection is opt-in)
			out, err := ui.RenderCommitLog(branch, parent, hasParent, bySection, multiline, ascii, maxCommits)
			if err != nil {
				return err
			}

			// Use pager by default unless --no-pager is set
			if !usePager {
				if err := ui.DisplayWithPager(out, forcePager); err != nil {
					// Fallback to direct output if pager fails
					fmt.Print(out)
				}
			} else {
				fmt.Print(out)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&bySection, "by-section", false, "group commits by section instead of chronological order")
	cmd.Flags().BoolVar(&multiline, "multiline", false, "use detailed multiline format")
	cmd.Flags().BoolVar(&noColor, "no-color", false, "disable colored output")
	cmd.Flags().BoolVar(&ascii, "ascii", false, "use ASCII symbols instead of Unicode")
	cmd.Flags().IntVar(&maxCommits, "max-commits", 300, "maximum commits per section (0 for unlimited)")
	cmd.Flags().BoolVar(&usePager, "no-pager", false, "disable pager (pager is enabled by default)")
	cmd.Flags().BoolVar(&forcePager, "force-pager", false, "force pager to stay open even for small outputs")

	return cmd
}
