package git

import (
	"fmt"
	"strings"
	"time"

	"pretty-git/internal/cmdutil"
)

// Note: Git config uses quoted section names for [branch "name"] which handles all special characters.
// No encoding/decoding needed - branch names with /, _, -, ~ all work transparently.

// ListBranches returns local branch names
func ListBranches() ([]string, error) {
	out, _, _, err := cmdutil.RunGit("for-each-ref", "refs/heads", "--format=%(refname:short)")
	if err != nil {
		return nil, err
	}

	if out == "" {
		return []string{}, nil
	}

	lines := strings.Split(out, "\n")
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
	}
	return lines, nil
}

// GetCurrentBranch returns the current branch name or error if detached
func GetCurrentBranch() (string, error) {
	out, _, _, err := cmdutil.RunGit("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	out = strings.TrimSpace(out)
	if out == "HEAD" {
		return "", fmt.Errorf("detached HEAD")
	}
	return out, nil
}

// SetParent writes branch.<child>.pretty-git-parent in local git config
func SetParent(child, parent string) error {
	// if a parent already exists for this child, create a simple backup
	if existing, ok, err := GetParent(child); err != nil {
		return err
	} else if ok {
		// store previous value under branch.<child>.pretty-git-parent-backup
		_, _, _, err := cmdutil.RunGit("config", "--local", fmt.Sprintf("branch.%s.pretty-git-parent-backup", child), existing)
		if err != nil {
			return err
		}
	}

	_, _, _, err := cmdutil.RunGit("config", "--local", fmt.Sprintf("branch.%s.pretty-git-parent", child), parent)
	return err
}

// AllParents reads all branch.<branch>.pretty-git-parent entries and returns map[child]=parent
func AllParents() (map[string]string, error) {
	out, stderr, code, err := cmdutil.RunGit("config", "--local", "--get-regexp", "^branch\\..*\\.pretty-git-parent$")
	if err != nil {
		// git returns exit code 1 when no matches are found; treat as empty
		if code == 1 {
			return map[string]string{}, nil
		}
		// if stderr contains known messages also tolerate
		if strings.Contains(stderr, "no matching") || strings.Contains(stderr, "No such file or directory") {
			return map[string]string{}, nil
		}
		return nil, err
	}

	parents := map[string]string{}
	if out == "" {
		return parents, nil
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	for _, l := range lines {
		parts := strings.Fields(l)
		if len(parts) < 2 {
			continue
		}
		key := parts[0]
		val := parts[1]
		// key is branch.<child>.pretty-git-parent
		const prefix = "branch."
		const suffix = ".pretty-git-parent"
		if strings.HasPrefix(key, prefix) && strings.HasSuffix(key, suffix) {
			// Extract child branch name from "branch.CHILD.pretty-git-parent"
			child := strings.TrimPrefix(key, prefix)
			child = strings.TrimSuffix(child, suffix)
			parents[child] = val
		}
	}

	return parents, nil
}

// GetParent returns parent of a child if set
func GetParent(child string) (string, bool, error) {
	out, _, code, err := cmdutil.RunGit("config", "--local", "--get", fmt.Sprintf("branch.%s.pretty-git-parent", child))
	if err != nil {
		if code == 1 {
			return "", false, nil
		}
		return "", false, err
	}
	return strings.TrimSpace(out), true, nil
}

// SetDescription writes branch.<branch>.pretty-git-description in local git config
func SetDescription(branch, description string) error {
	_, _, _, err := cmdutil.RunGit("config", "--local", fmt.Sprintf("branch.%s.pretty-git-description", branch), description)
	return err
}

// GetDescription returns description of a branch if set
func GetDescription(branch string) (string, bool, error) {
	out, _, code, err := cmdutil.RunGit("config", "--local", "--get", fmt.Sprintf("branch.%s.pretty-git-description", branch))
	if err != nil {
		if code == 1 {
			return "", false, nil
		}
		return "", false, err
	}
	return strings.TrimSpace(out), true, nil
}

// AllDescriptions reads all branch.<branch>.pretty-git-description entries and returns map[branch]=description
func AllDescriptions() (map[string]string, error) {
	out, stderr, code, err := cmdutil.RunGit("config", "--local", "--get-regexp", "^branch\\..*\\.pretty-git-description$")
	if err != nil {
		// git returns exit code 1 when no matches are found; treat as empty
		if code == 1 {
			return map[string]string{}, nil
		}
		// if stderr contains known messages also tolerate
		if strings.Contains(stderr, "no matching") || strings.Contains(stderr, "No such file or directory") {
			return map[string]string{}, nil
		}
		return nil, err
	}

	descriptions := map[string]string{}
	if out == "" {
		return descriptions, nil
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	for _, l := range lines {
		parts := strings.SplitN(l, " ", 2)
		if len(parts) < 2 {
			continue
		}
		key := parts[0]
		val := parts[1]
		// key is branch.<branch>.pretty-git-description
		const prefix = "branch."
		const suffix = ".pretty-git-description"
		if strings.HasPrefix(key, prefix) && strings.HasSuffix(key, suffix) {
			// Extract branch name from "branch.BRANCH.pretty-git-description"
			branch := strings.TrimPrefix(key, prefix)
			branch = strings.TrimSuffix(branch, suffix)
			descriptions[branch] = val
		}
	}

	return descriptions, nil
}

// CheckoutBranch performs git checkout. If create is true, behaves like 'git checkout -b <branch> [parent]'
func CheckoutBranch(branch string, create bool, parent string) error {
	if create {
		if parent != "" {
			_, _, _, err := cmdutil.RunGit("checkout", "-b", branch, parent)
			return err
		}
		_, _, _, err := cmdutil.RunGit("checkout", "-b", branch)
		return err
	}
	_, _, _, err := cmdutil.RunGit("checkout", branch)
	return err
}

// BranchStatus contains status information about a branch
type BranchStatus struct {
	Merged   bool
	Tracking bool
	Ahead    int
	Behind   int
	LastDays int  // days since last commit
	IsStale  bool // true if last activity was more than StaleThreshold days ago
}

// StaleThreshold is the number of days a branch must be inactive to be considered stale (default: 30 days)
const StaleThreshold = 30

// GetBranchStatus returns status information for a branch
// parent: the direct parent branch (from pretty-git metadata), if any
func GetBranchStatus(branch string, parent string) *BranchStatus {
	status := &BranchStatus{}

	// Check if merged into its parent (if parent exists)
	// A branch is "merged" when all its commits are already in its parent
	if parent != "" {
		_, _, code, _ := cmdutil.RunGit("merge-base", "--is-ancestor", branch, parent)
		status.Merged = (code == 0)
	}

	// Check if tracking upstream
	upstreamOut, _, upstreamCode, _ := cmdutil.RunGit("rev-parse", "--abbrev-ref", branch+"@{u}")
	if upstreamCode == 0 {
		status.Tracking = true
		upstreamRef := strings.TrimSpace(upstreamOut)

		// Get ahead/behind counts
		countOut, _, _, err := cmdutil.RunGit("rev-list", "--left-right", "--count", branch+"..."+upstreamRef)
		if err == nil {
			parts := strings.Fields(strings.TrimSpace(countOut))
			if len(parts) == 2 {
				fmt.Sscanf(parts[0], "%d", &status.Ahead)
				fmt.Sscanf(parts[1], "%d", &status.Behind)
			}
		}
	}

	// Get days since last commit on this branch
	timeOut, _, timeCode, _ := cmdutil.RunGit("log", "-1", "--format=%at", branch)
	if timeCode == 0 {
		// Parse timestamp
		var timestamp int64
		fmt.Sscanf(strings.TrimSpace(timeOut), "%d", &timestamp)
		if timestamp > 0 {
			lastCommitTime := time.Unix(timestamp, 0)
			daysSinceCommit := int(time.Since(lastCommitTime).Hours() / 24)
			status.LastDays = daysSinceCommit
			status.IsStale = daysSinceCommit > StaleThreshold
		}
	}

	return status
}

// GetMainBranch detects the main branch (master, main, or develop)
func GetMainBranch() (string, error) {
	branches, err := ListBranches()
	if err != nil {
		return "", err
	}

	// Check for main, master, develop in order
	for _, candidate := range []string{"main", "master", "develop"} {
		for _, b := range branches {
			if b == candidate {
				return candidate, nil
			}
		}
	}

	// Fallback to first branch
	if len(branches) > 0 {
		return branches[0], nil
	}

	return "", fmt.Errorf("no branches found")
}
