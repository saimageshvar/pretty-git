package git

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// Branch holds metadata about a git branch.
type Branch struct {
	Name      string
	IsCurrent bool
	IsRemote  bool
	ShortHash string
	Subject   string // first line of commit message
	Upstream  string // e.g. origin/main (empty if none)
	Ahead     int
	Behind    int
	RelTime   string // "2 hours ago"
	Parent    string // value of branch.<name>.pgit-parent config; empty = root
}

// ListBranches returns all local and remote branches, sorted by most recent commit.
// The current branch appears first.
func ListBranches() ([]Branch, error) {
	locals, err := listLocal()
	if err != nil {
		return nil, err
	}
	remotes, err := listRemote()
	if err != nil {
		return nil, err
	}

	// current branch first, then rest of locals, then remotes
	var current []Branch
	var rest []Branch
	for _, b := range locals {
		if b.IsCurrent {
			current = append(current, b)
		} else {
			rest = append(rest, b)
		}
	}
	result := append(current, rest...)
	result = append(result, remotes...)
	return result, nil
}

// listLocal parses `git branch -vv --sort=-committerdate`.
func listLocal() ([]Branch, error) {
	// format: <current> <name> <hash> [<upstream>] <subject>
	out, err := run("git", "branch", "-vv", "--sort=-committerdate")
	if err != nil {
		return nil, err
	}

	// Read all pgit-parent values in one git config call instead of one per branch.
	parents := readAllParents()

	var branches []Branch
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if line == "" {
			continue
		}
		b := parseLocalLine(line)
		if b.Name == "" {
			continue
		}
		t, _ := relTime(b.Name, false)
		b.RelTime = t
		b.Parent = parents[b.Name]
		branches = append(branches, b)
	}
	return branches, nil
}

// parseLocalLine parses one line of `git branch -vv` output.
// Format: "* main           a1b2c3d [origin/main: ahead 1] some message"
//
//	"  feature/login  e5f6a78 some message"
func parseLocalLine(line string) Branch {
	var b Branch
	if len(line) < 2 {
		return b
	}
	marker := line[0]
	b.IsCurrent = marker == '*'
	rest := strings.TrimSpace(line[2:])

	fields := strings.Fields(rest)
	if len(fields) < 2 {
		return b
	}
	b.Name = fields[0]
	b.ShortHash = fields[1]

	// parse optional upstream tracking info [origin/main: ahead N, behind M]
	msgStart := 2
	if len(fields) > 2 && strings.HasPrefix(fields[2], "[") {
		end := -1
		for i := 2; i < len(fields); i++ {
			if strings.HasSuffix(fields[i], "]") {
				end = i
				break
			}
		}
		if end >= 0 {
			tracking := strings.Join(fields[2:end+1], " ")
			tracking = strings.Trim(tracking, "[]")
			parts := strings.SplitN(tracking, ":", 2)
			b.Upstream = strings.TrimSpace(parts[0])
			if len(parts) > 1 {
				info := parts[1]
				fmt.Sscanf(info, " ahead %d", &b.Ahead)
				fmt.Sscanf(info, " behind %d", &b.Behind)
				if strings.Contains(info, "ahead") && strings.Contains(info, "behind") {
					fmt.Sscanf(info, " ahead %d, behind %d", &b.Ahead, &b.Behind)
				}
			}
			msgStart = end + 1
		}
	}
	b.Subject = strings.Join(fields[msgStart:], " ")
	return b
}

// listRemote parses `git branch -r --sort=-committerdate`.
func listRemote() ([]Branch, error) {
	out, err := run("git", "branch", "-r", "--sort=-committerdate")
	if err != nil {
		// remote failures are non-fatal
		return nil, nil
	}
	var branches []Branch
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "->") {
			continue
		}
		// get hash + subject for each remote branch
		info, err := run("git", "log", "-1", "--format=%h %s", line)
		if err != nil {
			continue
		}
		info = strings.TrimSpace(info)
		parts := strings.SplitN(info, " ", 2)
		b := Branch{
			Name:     line,
			IsRemote: true,
		}
		if len(parts) >= 1 {
			b.ShortHash = parts[0]
		}
		if len(parts) >= 2 {
			b.Subject = parts[1]
		}
		t, _ := relTime(line, true)
		b.RelTime = t
		branches = append(branches, b)
	}
	return branches, nil
}

// readAllParents reads all pgit-parent config values in a single git config
// call and returns a map of branch name → parent branch name.
func readAllParents() map[string]string {
	out, err := run("git", "config", "--local", "--list")
	if err != nil {
		return nil
	}
	parents := make(map[string]string)
	for _, line := range strings.Split(out, "\n") {
		// Format: branch.<name>.pgit-parent=<value>
		if !strings.Contains(line, ".pgit-parent=") {
			continue
		}
		kv := strings.SplitN(line, "=", 2)
		if len(kv) != 2 {
			continue
		}
		// key: branch.<name>.pgit-parent  (name may contain dots, e.g. "feat/x.y")
		key := kv[0]
		val := kv[1]
		// Strip "branch." prefix and ".pgit-parent" suffix
		if !strings.HasPrefix(key, "branch.") || !strings.HasSuffix(key, ".pgit-parent") {
			continue
		}
		name := key[len("branch.") : len(key)-len(".pgit-parent")]
		parents[name] = val
	}
	return parents
}

// relTime returns the relative time of the last commit on a branch.
func relTime(ref string, _ bool) (string, error) {
	out, err := run("git", "log", "-1", "--format=%cr", ref)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// SwitchBranch runs `git checkout <name>` and returns any error.
func SwitchBranch(name string) error {
	cmd := exec.Command("git", "checkout", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return err
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}

// RepoName returns the basename of the current git repository root.
func RepoName() string {
	out, err := run("git", "rev-parse", "--show-toplevel")
	if err != nil {
		return ""
	}
	return filepath.Base(strings.TrimSpace(out))
}

// run executes a command and returns stdout, or an error if it fails.
func run(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("%s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", err
	}
	return string(out), nil
}
