package git

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// Branch holds metadata about a git branch.
type Branch struct {
	Name         string
	IsCurrent    bool
	IsRemote     bool
	ShortHash    string
	Subject      string // first line of commit message
	Upstream     string // e.g. origin/main (empty if none)
	Ahead        int
	Behind       int
	RelTime      string // "2 hours ago"
	Parent       string // value of branch.<name>.pgit-parent config; empty = root
	ParentAhead  int    // commits in this branch not yet in parent
	ParentBehind int    // commits in parent not yet in this branch
	Description  string // value of branch.<name>.pgit-desc config; empty = none
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

	// Read all pgit-parent and pgit-desc values in one git config call each.
	parents := readAllParents()
	descs := readAllDescriptions()

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
		b.Description = descs[b.Name]
		branches = append(branches, b)
	}

	// Compute parent ahead/behind concurrently for branches that have a parent.
	type parentResult struct {
		name   string
		ahead  int
		behind int
	}
	ch := make(chan parentResult, len(branches))
	var wg sync.WaitGroup
	for _, b := range branches {
		if b.Parent == "" {
			continue
		}
		wg.Add(1)
		go func(name, parent string) {
			defer wg.Done()
			a, beh := ParentAheadBehind(name, parent)
			ch <- parentResult{name, a, beh}
		}(b.Name, b.Parent)
	}
	go func() {
		wg.Wait()
		close(ch)
	}()
	statusMap := make(map[string]parentResult)
	for r := range ch {
		statusMap[r.name] = r
	}
	for i := range branches {
		if r, ok := statusMap[branches[i].Name]; ok {
			branches[i].ParentAhead = r.ahead
			branches[i].ParentBehind = r.behind
		}
	}

	return branches, nil
}

// ParentAheadBehind returns how many commits branch is ahead of and behind parent.
// Uses `git rev-list --left-right --count branch...parent`:
//   - left  (ahead)  = commits in branch not in parent
//   - right (behind) = commits in parent not in branch
func ParentAheadBehind(branch, parent string) (ahead, behind int) {
	out, err := run("git", "rev-list", "--left-right", "--count", branch+"..."+parent)
	if err != nil {
		return 0, 0
	}
	parts := strings.Fields(strings.TrimSpace(out))
	if len(parts) == 2 {
		ahead, _ = strconv.Atoi(parts[0])
		behind, _ = strconv.Atoi(parts[1])
	}
	return
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

// readAllDescriptions reads all pgit-desc config values and returns a map of
// branch name → description.
func readAllDescriptions() map[string]string {
	out, err := run("git", "config", "--local", "--list")
	if err != nil {
		return nil
	}
	descs := make(map[string]string)
	for _, line := range strings.Split(out, "\n") {
		if !strings.Contains(line, ".pgit-desc=") {
			continue
		}
		kv := strings.SplitN(line, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := kv[0]
		val := kv[1]
		if !strings.HasPrefix(key, "branch.") || !strings.HasSuffix(key, ".pgit-desc") {
			continue
		}
		name := key[len("branch.") : len(key)-len(".pgit-desc")]
		descs[name] = val
	}
	return descs
}

// SetDescription writes `branch.<name>.pgit-desc = <desc>` into local git config.
func SetDescription(branch, desc string) error {
	key := fmt.Sprintf("branch.%s.pgit-desc", branch)
	cmd := exec.Command("git", "config", "--local", key, desc)
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

// UnsetDescription removes the `branch.<name>.pgit-desc` key from local git config.
func UnsetDescription(branch string) error {
	key := fmt.Sprintf("branch.%s.pgit-desc", branch)
	cmd := exec.Command("git", "config", "--local", "--unset", key)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 5 {
			return nil
		}
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return err
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}

// relTime returns the relative time of the last commit on a branch.
func relTime(ref string, _ bool) (string, error) {
	out, err := run("git", "log", "-1", "--format=%cr", ref)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// CreateBranch creates a new local branch from HEAD using `git checkout -b <name>`.
func CreateBranch(name string) error {
	cmd := exec.Command("git", "checkout", "-b", name)
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

// SetParent writes `branch.<child>.pgit-parent = <parent>` into local git config.
func SetParent(child, parent string) error {
	key := fmt.Sprintf("branch.%s.pgit-parent", child)
	cmd := exec.Command("git", "config", "--local", key, parent)
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

// UnsetParent removes the `branch.<child>.pgit-parent` key from local git config.
func UnsetParent(child string) error {
	key := fmt.Sprintf("branch.%s.pgit-parent", child)
	cmd := exec.Command("git", "config", "--local", "--unset", key)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// exit code 5 means key didn't exist — not an error for us
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 5 {
			return nil
		}
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return err
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}

// BranchExists returns true if a local branch with the given name exists.
func BranchExists(name string) bool {
	out, err := run("git", "branch", "--list", name)
	if err != nil {
		return false
	}
	return strings.TrimSpace(out) != ""
}

// CurrentBranch returns the name of the currently checked-out branch,
// or "" if HEAD is detached or the command fails.
func CurrentBranch() string {
	out, err := run("git", "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return ""
	}
	name := strings.TrimSpace(out)
	if name == "HEAD" {
		return "" // detached HEAD
	}
	return name
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

// ── Log ────────────────────────────────────────────────────────────────────

// Commit holds metadata for one git commit.
type Commit struct {
Hash      string // full 40-char hash
ShortHash string // 7-char abbreviated hash
Subject   string // first line of commit message
Author    string // author name
RelTime   string // "2 hours ago"
Body      string // full commit message (subject + body, newline-joined)
}

// CommitDetail holds the extended info loaded on demand for the detail pane.
type CommitDetail struct {
Commit
AuthorEmail string
AuthorDate  string // absolute date, e.g. "Mon Jan 2 15:04:05 2006 -0700"
FilesChanged int
Insertions   int
Deletions    int
DiffStat     string // raw `git diff-tree --stat` output
}

// CommitFilters controls optional filtering for ListCommits.
type CommitFilters struct {
	OnlyAuthorEmail string // empty = all authors
	SkipMerges      bool
}

// CurrentUserEmail returns the value of `git config user.email`, or "".
func CurrentUserEmail() string {
	out, err := run("git", "config", "user.email")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// ListCommits returns up to limit commits reachable from ref (default HEAD).
func ListCommits(ref string, limit int, filters CommitFilters) ([]Commit, error) {
	if ref == "" {
		ref = "HEAD"
	}
	if limit <= 0 {
		limit = 100
	}
	// Use ASCII Record Separator (0x1e) as field delimiter via git's %x1e escape.
	// This avoids NUL bytes (which exec rejects) and is safe across all field values.
	// Format per commit: hash RS short RS author RS reltime RS subject RS body RS
	const rs = "\x1e"
	format := "%H%x1e%h%x1e%an%x1e%cr%x1e%s%x1e%b%x1e"
	args := []string{"log", ref, "--format=" + format, "-n", strconv.Itoa(limit)}
	if filters.OnlyAuthorEmail != "" {
		args = append(args, "--author="+filters.OnlyAuthorEmail)
	}
	if filters.SkipMerges {
		args = append(args, "--no-merges")
	}
	out, err := run("git", args...)
	if err != nil {
		return nil, err
	}

	var commits []Commit
	fields := strings.Split(out, rs)
	// Strip leading newlines git inserts between records.
	var clean []string
	for _, f := range fields {
		clean = append(clean, strings.TrimLeft(f, "\n"))
	}
	for i := 0; i+5 < len(clean); i += 6 {
		hash := strings.TrimSpace(clean[i])
		if hash == "" {
			continue
		}
		commits = append(commits, Commit{
			Hash:      hash,
			ShortHash: strings.TrimSpace(clean[i+1]),
			Author:    strings.TrimSpace(clean[i+2]),
			RelTime:   strings.TrimSpace(clean[i+3]),
			Subject:   strings.TrimSpace(clean[i+4]),
			Body:      strings.TrimSpace(clean[i+5]),
		})
	}
	return commits, nil
}

// GetCommitDetail fetches extended info (diff stats) for a single commit.
func GetCommitDetail(hash string) (CommitDetail, error) {
	// Use ASCII Record Separator (0x1e) as field delimiter.
	const rs = "\x1e"
	format := "%an%x1e%ae%x1e%ad%x1e%s%x1e%b%x1e%H%x1e%h%x1e%cr"
	out, err := run("git", "show", "--no-patch", "--format="+format, hash)
	if err != nil {
		return CommitDetail{}, err
	}
	fields := strings.Split(strings.TrimLeft(out, "\n"), rs)
	var d CommitDetail
	if len(fields) >= 8 {
		d.Author      = strings.TrimSpace(fields[0])
		d.AuthorEmail = strings.TrimSpace(fields[1])
		d.AuthorDate  = strings.TrimSpace(fields[2])
		d.Subject     = strings.TrimSpace(fields[3])
		d.Body        = strings.TrimSpace(fields[4])
		d.Hash        = strings.TrimSpace(fields[5])
		d.ShortHash   = strings.TrimSpace(fields[6])
		d.RelTime     = strings.TrimSpace(strings.TrimLeft(fields[7], "\n"))
	}

	// Diff stat
	stat, err := run("git", "diff-tree", "--stat", "--no-commit-id", "-r", hash)
	if err == nil {
		d.DiffStat = strings.TrimRight(stat, "\n")
		lines := strings.Split(d.DiffStat, "\n")
		if len(lines) > 0 {
			summary := lines[len(lines)-1]
			// Parse "N file(s) changed, M insertion(s)(+), K deletion(s)(-)"
			parts := strings.Fields(summary)
			for pi, p := range parts {
				switch {
				case strings.HasPrefix(p, "file") && pi > 0:
					fmt.Sscanf(parts[pi-1], "%d", &d.FilesChanged)
				case strings.HasPrefix(p, "insertion") && pi > 0:
					fmt.Sscanf(parts[pi-1], "%d", &d.Insertions)
				case strings.HasPrefix(p, "deletion") && pi > 0:
					fmt.Sscanf(parts[pi-1], "%d", &d.Deletions)
				}
			}
		}
	}
	return d, nil
}


// ── Stash ──────────────────────────────────────────────────────────────────

// FileStatus represents a file's git status and path.
type FileStatus struct {
	Code string // X or XY code from `git status --porcelain`
	Path string
}

// StatusDisplay returns a single letter for display (M, A, D, ?, R, …).
func (f FileStatus) StatusDisplay() string {
	if len(f.Code) == 0 {
		return "?"
	}
	// Prioritise the index status (first char), fall back to worktree (second char).
	idx := string(f.Code[0])
	if idx != " " && idx != "?" {
		return idx
	}
	if len(f.Code) > 1 {
		wt := string(f.Code[1])
		if wt != " " {
			return wt
		}
	}
	return "?"
}

// IsUntracked returns true if the file is untracked (?? in git status).
func (f FileStatus) IsUntracked() bool {
	return f.Code == "??"
}

// StashEntry holds summary metadata for one stash ref.
type StashEntry struct {
	Index   int
	Ref     string // "stash@{N}"
	Message string
	Branch  string
	RelTime string
}

// StashDetail holds extended info for one stash.
type StashDetail struct {
	StashEntry
	FilesChanged int
	Insertions   int
	Deletions    int
	Files        []FileStatus // files changed in the stash
}

// ListModifiedFiles returns the current working-tree status (staged + unstaged + untracked).
func ListModifiedFiles() ([]FileStatus, error) {
	out, err := run("git", "status", "--porcelain")
	if err != nil {
		return nil, err
	}
	var files []FileStatus
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if len(line) < 4 {
			continue
		}
		code := line[:2]
		path := strings.TrimSpace(line[3:])
		// Handle renames: "R old -> new" — use the new path
		if strings.Contains(path, " -> ") {
			parts := strings.SplitN(path, " -> ", 2)
			path = parts[1]
		}
		files = append(files, FileStatus{Code: code, Path: path})
	}
	return files, nil
}

// CountStagedFiles returns the number of staged (index) changes.
func CountStagedFiles(files []FileStatus) int {
	n := 0
	for _, f := range files {
		if len(f.Code) > 0 && f.Code[0] != ' ' && f.Code[0] != '?' {
			n++
		}
	}
	return n
}

// CountUnstagedFiles returns the number of unstaged (worktree) changes (excludes untracked).
func CountUnstagedFiles(files []FileStatus) int {
	n := 0
	for _, f := range files {
		if len(f.Code) > 1 && f.Code[1] != ' ' && f.Code != "??" {
			n++
		}
	}
	return n
}

// ListStashes returns all stash entries, most recent first.
func ListStashes() ([]StashEntry, error) {
	out, err := run("git", "stash", "list", "--format=%gd|%gs|%cr")
	if err != nil {
		return nil, err
	}
	out = strings.TrimRight(out, "\n")
	if out == "" {
		return nil, nil
	}

	var entries []StashEntry
	for i, line := range strings.Split(out, "\n") {
		parts := strings.SplitN(line, "|", 3)
		if len(parts) < 3 {
			continue
		}
		ref := strings.TrimSpace(parts[0])
		msg := strings.TrimSpace(parts[1])
		relTime := strings.TrimSpace(parts[2])

		// Extract branch from message: "On <branch>: <msg>" or "WIP on <branch>: <msg>"
		branch := ""
		displayMsg := msg
		if after, ok := strings.CutPrefix(msg, "WIP on "); ok {
			if colonIdx := strings.Index(after, ": "); colonIdx >= 0 {
				branch = after[:colonIdx]
				displayMsg = after[colonIdx+2:]
			}
		} else if after, ok := strings.CutPrefix(msg, "On "); ok {
			if colonIdx := strings.Index(after, ": "); colonIdx >= 0 {
				branch = after[:colonIdx]
				displayMsg = after[colonIdx+2:]
			}
		}

		entries = append(entries, StashEntry{
			Index:   i,
			Ref:     ref,
			Message: displayMsg,
			Branch:  branch,
			RelTime: relTime,
		})
	}
	return entries, nil
}

// GetStashDetail fetches extended info for a single stash entry.
func GetStashDetail(ref string) (StashDetail, error) {
	// Get list of changed files with status
	out, err := run("git", "stash", "show", "--name-status", ref)
	if err != nil {
		return StashDetail{}, err
	}

	var files []FileStatus
	var filesChanged, insertions, deletions int
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		code := fields[0]
		path := fields[1]
		// Map single-letter status to our 2-char code (staged perspective)
		var fCode string
		switch code {
		case "M":
			fCode = "M "
		case "A":
			fCode = "A "
		case "D":
			fCode = "D "
		default:
			fCode = code + " "
		}
		files = append(files, FileStatus{Code: fCode, Path: path})
		filesChanged++
	}

	// Get diff stats
	stat, err := run("git", "stash", "show", "--stat", ref)
	if err == nil {
		lines := strings.Split(strings.TrimRight(stat, "\n"), "\n")
		if len(lines) > 0 {
			summary := lines[len(lines)-1]
			parts := strings.Fields(summary)
			for pi, p := range parts {
				switch {
				case strings.HasPrefix(p, "insertion") && pi > 0:
					fmt.Sscanf(parts[pi-1], "%d", &insertions)
				case strings.HasPrefix(p, "deletion") && pi > 0:
					fmt.Sscanf(parts[pi-1], "%d", &deletions)
				}
			}
		}
	}

	return StashDetail{
		FilesChanged: filesChanged,
		Insertions:   insertions,
		Deletions:    deletions,
		Files:        files,
	}, nil
}

// StashPush creates a stash with the given message and options.
// stashType: "staged", "unstaged", "all", "custom"
// customFiles: used when stashType == "custom"
func StashPush(msg, stashType string, customFiles []string) error {
	var args []string
	switch stashType {
	case "staged":
		args = []string{"stash", "push", "--staged", "-m", msg}
	case "unstaged":
		// Stash only unstaged changes by listing their files explicitly
		files, err := ListModifiedFiles()
		if err != nil {
			return err
		}
		var unstaged []string
		for _, f := range files {
			if len(f.Code) > 1 && f.Code[1] != ' ' && f.Code != "??" {
				unstaged = append(unstaged, f.Path)
			}
		}
		if len(unstaged) == 0 {
			return fmt.Errorf("no unstaged files to stash")
		}
		args = append([]string{"stash", "push", "-m", msg, "--"}, unstaged...)
	case "all":
		args = []string{"stash", "push", "--include-untracked", "-m", msg}
	case "custom":
		// Check if any selected file is untracked
		status, _ := ListModifiedFiles()
		untrackedSet := make(map[string]bool)
		for _, f := range status {
			if f.IsUntracked() {
				untrackedSet[f.Path] = true
			}
		}
		hasUntracked := false
		for _, path := range customFiles {
			if untrackedSet[path] {
				hasUntracked = true
				break
			}
		}
		if hasUntracked {
			args = append([]string{"stash", "push", "--include-untracked", "-m", msg, "--"}, customFiles...)
		} else {
			args = append([]string{"stash", "push", "-m", msg, "--"}, customFiles...)
		}
	default:
		args = []string{"stash", "push", "--include-untracked", "-m", msg}
	}
	cmd := exec.Command("git", args...)
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

// StashApply applies a stash without removing it.
func StashApply(ref string) error {
	cmd := exec.Command("git", "stash", "apply", ref)
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

// StashPop applies a stash and removes it if successful.
func StashPop(ref string) error {
	cmd := exec.Command("git", "stash", "pop", ref)
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

// StashDrop removes a stash entry.
func StashDrop(ref string) error {
	cmd := exec.Command("git", "stash", "drop", ref)
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

// LastCommitShortHash returns the 7-char abbreviated hash of HEAD.
func LastCommitShortHash() string {
	out, err := run("git", "log", "-1", "--format=%h")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// LastCommitOneLiner returns "<hash> <subject>" for HEAD.
func LastCommitOneLiner() string {
	out, err := run("git", "log", "-1", "--format=%h %s")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}
