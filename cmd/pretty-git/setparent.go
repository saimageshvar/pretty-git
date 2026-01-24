package main

import (
    "bufio"
    "fmt"
    "os"
    "strings"

    "github.com/spf13/cobra"

    "pretty-git/internal/git"
)

func NewSetParentCmd() *cobra.Command {
    var yes bool

    cmd := &cobra.Command{
        Use:   "set-parent [branch] <parent>",
        Short: "Set or update the recorded parent for a branch (defaults to current branch)",
        Args:  cobra.RangeArgs(1, 2),
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
