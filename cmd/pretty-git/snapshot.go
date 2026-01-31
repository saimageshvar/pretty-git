package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"pretty-git/internal/git"
	"pretty-git/internal/ui"

	"github.com/spf13/cobra"
)

// Checkpoint represents a saved snapshot
type Checkpoint struct {
	Name       string    `json:"name"`
	Hash       string    `json:"hash"`
	Timestamp  time.Time `json:"timestamp"`
	Message    string    `json:"message"`
	BranchName string    `json:"branch_name"`
}

// CheckpointStore manages the checkpoint storage
type CheckpointStore struct {
	filePath string
}

// NewCheckpointStore creates a new checkpoint store
func NewCheckpointStore() (*CheckpointStore, error) {
	gitDir, err := git.GetGitDir()
	if err != nil {
		return nil, fmt.Errorf("not in a git repository: %w", err)
	}

	return &CheckpointStore{
		filePath: filepath.Join(gitDir, "pretty-git-snapshots.json"),
	}, nil
}

// Load reads all checkpoints from disk
func (s *CheckpointStore) Load() ([]Checkpoint, error) {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Checkpoint{}, nil
		}
		return nil, err
	}

	var checkpoints []Checkpoint
	if err := json.Unmarshal(data, &checkpoints); err != nil {
		return nil, fmt.Errorf("failed to parse checkpoints: %w", err)
	}

	return checkpoints, nil
}

// Save writes all checkpoints to disk
func (s *CheckpointStore) Save(checkpoints []Checkpoint) error {
	data, err := json.MarshalIndent(checkpoints, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.filePath, data, 0644)
}

// Add adds a new checkpoint
func (s *CheckpointStore) Add(checkpoint Checkpoint) error {
	checkpoints, err := s.Load()
	if err != nil {
		return err
	}

	// Check if name already exists
	for i, cp := range checkpoints {
		if cp.Name == checkpoint.Name {
			// Replace existing checkpoint with same name
			checkpoints[i] = checkpoint
			return s.Save(checkpoints)
		}
	}

	checkpoints = append(checkpoints, checkpoint)
	return s.Save(checkpoints)
}

// Get retrieves a checkpoint by name
func (s *CheckpointStore) Get(name string) (*Checkpoint, error) {
	checkpoints, err := s.Load()
	if err != nil {
		return nil, err
	}

	for _, cp := range checkpoints {
		if cp.Name == name {
			return &cp, nil
		}
	}

	return nil, fmt.Errorf("checkpoint '%s' not found", name)
}

// Delete removes a checkpoint by name
func (s *CheckpointStore) Delete(name string) error {
	checkpoints, err := s.Load()
	if err != nil {
		return err
	}

	newCheckpoints := make([]Checkpoint, 0, len(checkpoints))
	found := false
	for _, cp := range checkpoints {
		if cp.Name != name {
			newCheckpoints = append(newCheckpoints, cp)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("checkpoint '%s' not found", name)
	}

	return s.Save(newCheckpoints)
}

// Clear removes all checkpoints
func (s *CheckpointStore) Clear() error {
	return s.Save([]Checkpoint{})
}

func NewSnapshotCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshot",
		Short: "Save and restore working directory snapshots without committing",
		Long: `Create periodic checkpoints of changes without committing them.
Snapshots are stored using git stash under the hood but don't affect your working directory.

You can create named snapshots, list them, restore to any previous snapshot, or clear them.`,
	}

	cmd.AddCommand(newSnapshotCreateCmd())
	cmd.AddCommand(newSnapshotListCmd())
	cmd.AddCommand(newSnapshotRestoreCmd())
	cmd.AddCommand(newSnapshotClearCmd())

	return cmd
}

// validateSnapshotName checks if a snapshot name is valid
func validateSnapshotName(name string) error {
	if name == "" {
		return fmt.Errorf("snapshot name cannot be empty")
	}
	if len(name) > 100 {
		return fmt.Errorf("snapshot name too long (max 100 characters)")
	}
	// Check for problematic characters that might cause issues
	if strings.ContainsAny(name, "\n\r\t") {
		return fmt.Errorf("snapshot name cannot contain newlines or tabs")
	}
	return nil
}

func newSnapshotCreateCmd() *cobra.Command {
	var message string

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new snapshot with the given name",
		Long: `Create a snapshot of staged changes.
The snapshot is stored with the given name and can be restored later.

IMPORTANT: You must stage the files you want to snapshot using 'git add' before creating a snapshot.

Example:
  git add file1.txt file2.txt
  pretty-git snapshot create checkpoint-1
  
  git add .
  pretty-git snapshot create my-changes -m "Before refactoring"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			// Validate snapshot name
			if err := validateSnapshotName(name); err != nil {
				return err
			}

			store, err := NewCheckpointStore()
			if err != nil {
				return err
			}

			// Get current branch name
			currentBranch, err := git.GetCurrentBranch()
			if err != nil {
				return fmt.Errorf("failed to get current branch: %w", err)
			}

			// Create the snapshot
			if message == "" {
				message = fmt.Sprintf("Snapshot: %s", name)
			}

			hash, err := git.CreateSnapshot(message)
			if err != nil {
				return err
			}

			checkpoint := Checkpoint{
				Name:       name,
				Hash:       hash,
				Timestamp:  time.Now(),
				Message:    message,
				BranchName: currentBranch,
			}

			if err := store.Add(checkpoint); err != nil {
				return err
			}

			fmt.Printf("%s Snapshot '%s' created successfully\n", ui.ColorSuccess("✓"), ui.ColorHighlight(name))
			fmt.Printf("  Hash: %s\n", ui.ColorDim(hash[:12]))
			fmt.Printf("  Time: %s\n", ui.ColorDim(checkpoint.Timestamp.Format("2006-01-02 15:04:05")))

			return nil
		},
	}

	cmd.Flags().StringVarP(&message, "message", "m", "", "Optional message for the snapshot")

	return cmd
}

func newSnapshotListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all saved snapshots",
		Long:  `Display all saved snapshots with their names, timestamps, and hashes.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := NewCheckpointStore()
			if err != nil {
				return err
			}

			checkpoints, err := store.Load()
			if err != nil {
				return err
			}

			if len(checkpoints) == 0 {
				fmt.Println("No snapshots found.")
				fmt.Println("Create one with: pretty-git snapshot create <name>")
				return nil
			}

			// Get current branch for highlighting
			currentBranch, _ := git.GetCurrentBranch()

			// Group snapshots by branch
			branchGroups := make(map[string][]Checkpoint)
			for _, cp := range checkpoints {
				branch := cp.BranchName
				if branch == "" {
					branch = "(unknown branch)"
				}
				branchGroups[branch] = append(branchGroups[branch], cp)
			}

			// Sort snapshots within each branch by timestamp (newest first)
			for branch := range branchGroups {
				snapshots := branchGroups[branch]
				sort.Slice(snapshots, func(i, j int) bool {
					return snapshots[i].Timestamp.After(snapshots[j].Timestamp)
				})
				branchGroups[branch] = snapshots
			}

			fmt.Printf("Saved snapshots (%d):\n\n", len(checkpoints))

			// Sort branch names for consistent output
			var branches []string
			for branch := range branchGroups {
				branches = append(branches, branch)
			}
			// Put current branch first
			if currentBranch != "" {
				for i, b := range branches {
					if b == currentBranch {
						branches[0], branches[i] = branches[i], branches[0]
						break
					}
				}
			}

			// Display snapshots grouped by branch
			for _, branch := range branches {
				branchLabel := branch
				if branch == currentBranch {
					branchLabel = ui.ColorCurrent(branch + " (current)")
				} else {
					branchLabel = ui.ColorDim(branch)
				}
				fmt.Printf("%s %s\n", ui.ColorHighlight("■"), branchLabel)

				for _, cp := range branchGroups[branch] {
					fmt.Printf("  %s %s\n", ui.ColorHighlight("●"), ui.ColorBold(cp.Name))
					fmt.Printf("    Time: %s\n", cp.Timestamp.Format("2006-01-02 15:04:05"))
					fmt.Printf("    Hash: %s\n", ui.ColorDim(cp.Hash[:12]))
					if cp.Message != "" && cp.Message != fmt.Sprintf("Snapshot: %s", cp.Name) {
						fmt.Printf("    Message: %s\n", ui.ColorDim(cp.Message))
					}
					fmt.Println()
				}
			}

			return nil
		},
	}

	return cmd
}

func newSnapshotRestoreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restore <name>",
		Short: "Restore working directory to a saved snapshot",
		Long: `Restore the working directory and staged changes to match a saved snapshot.
This will overwrite your current changes, so use with caution.

Note: If there are conflicts during restore, git will leave conflict markers for you to resolve manually.

Example:
  pretty-git snapshot restore checkpoint-1`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			store, err := NewCheckpointStore()
			if err != nil {
				return err
			}

			// Validate snapshot name
			if err := validateSnapshotName(name); err != nil {
				return err
			}

			checkpoint, err := store.Get(name)
			if err != nil {
				return err
			}

			// Get current branch
			currentBranch, err := git.GetCurrentBranch()
			if err != nil {
				return fmt.Errorf("failed to get current branch: %w", err)
			}

			// Warn if restoring snapshot from different branch
			if checkpoint.BranchName != "" && checkpoint.BranchName != currentBranch {
				fmt.Printf("%s This snapshot is from branch '%s', but you're on '%s'.\n",
					ui.ColorStaleBranch("⚠"),
					ui.ColorHighlight(checkpoint.BranchName),
					ui.ColorHighlight(currentBranch))
				fmt.Printf("  Restoring may cause unexpected results.\n")
				fmt.Print("Continue? (y/N): ")
				var response string
				fmt.Scanln(&response)
				if response != "y" && response != "Y" {
					fmt.Println("Restore cancelled.")
					return nil
				}
			}

			// Warn if there are uncommitted changes
			hasChanges, err := git.HasUncommittedChanges()
			if err != nil {
				return fmt.Errorf("failed to check working directory status: %w", err)
			}
			if hasChanges {
				fmt.Printf("%s Working directory has uncommitted changes. These will be overwritten.\n",
					ui.ColorStaleBranch("⚠"))
				fmt.Print("Continue? (y/N): ")
				var response string
				fmt.Scanln(&response)
				if response != "y" && response != "Y" {
					fmt.Println("Restore cancelled.")
					return nil
				}
			}

			fmt.Printf("Restoring snapshot '%s' from %s...\n",
				ui.ColorHighlight(name),
				ui.ColorDim(checkpoint.Timestamp.Format("2006-01-02 15:04:05")))

			if err := git.RestoreSnapshot(checkpoint.Hash); err != nil {
				return err
			}

			fmt.Printf("%s Successfully restored to snapshot '%s'\n", ui.ColorSuccess("✓"), ui.ColorHighlight(name))
			return nil
		},
	}

	return cmd
}

func newSnapshotClearCmd() *cobra.Command {
	var all bool
	var yes bool

	cmd := &cobra.Command{
		Use:   "clear [name]",
		Short: "Clear one or all snapshots",
		Long: `Remove saved snapshots. Specify a name to remove a single snapshot,
or use --all to remove all snapshots.

Examples:
  pretty-git snapshot clear checkpoint-1       # Remove one snapshot
  pretty-git snapshot clear --all              # Remove all snapshots
  pretty-git snapshot clear --all --yes        # Remove all without confirmation`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := NewCheckpointStore()
			if err != nil {
				return err
			}

			if all {
				// Clear all snapshots
				if !yes {
					fmt.Print("Are you sure you want to clear all snapshots? (y/N): ")
					var response string
					fmt.Scanln(&response)
					if response != "y" && response != "Y" {
						fmt.Println("Cancelled.")
						return nil
					}
				}

				if err := store.Clear(); err != nil {
					return err
				}

				fmt.Printf("%s All snapshots cleared\n", ui.ColorSuccess("✓"))
				return nil
			}

			// Clear specific snapshot
			if len(args) == 0 {
				return fmt.Errorf("specify a snapshot name or use --all to clear all snapshots")
			}

			name := args[0]
			// Validate snapshot name
			if err := validateSnapshotName(name); err != nil {
				return err
			}

			if err := store.Delete(name); err != nil {
				return err
			}

			fmt.Printf("%s Snapshot '%s' removed\n", ui.ColorSuccess("✓"), ui.ColorHighlight(name))
			return nil
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "Clear all snapshots")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation when clearing all")

	return cmd
}
