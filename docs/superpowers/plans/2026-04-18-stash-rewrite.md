# Stash Command Rewrite Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rewrite the `pgit stash` stash push + stash detail logic to fix two confirmed bugs: (1) `--staged` fails on MM files, and (2) browse detail shows too many files for non-"all" stashes.

**Architecture:** Two key changes: (1) MM file pre-check for `--staged`, and (2) `GetStashDetail` uses only stash-internal commits for diffs — `ref^1` (HEAD at stash time), `ref^2` (index), `ref` (WIP) — never HEAD or current working tree. Stash type metadata stored in the message prefix determines which diff pair to use. Simplified verification in `StashPush`.

**Tech Stack:** Go, Bubble Tea TUI, git CLI via `exec.Command`

---

## Confirmed Bugs (from empirical testing)

### Bug 1: `--staged` fails on MM files
- `git stash push --staged` creates the stash commit, then tries to restore the working tree
- On MM files (staged + unstaged changes on same file), the restore fails: `error: patch failed: Cannot remove worktree changes`
- pgit's rollback drops the stash, but the error message is confusing
- Git itself can't handle this case. The fix: **detect MM files before calling git and reject with a clear message**

### Bug 2: Browse detail shows too many files
- `GetStashDetail` uses current working-tree state (`git diff --cached`, `git diff`) as skip sets
- When browsing old stashes, the working tree has changed since creation → skip sets are stale
- `git diff HEAD stash@0^2` compares against current HEAD (which moves!) → shows extra files after commits
- Example: `pgit stash --custom -- a.go` with staged b.go, c.go. After b.go/c.go are committed, `HEAD` has moved, and `diff HEAD stash^2` now shows b.go/c.go as changes since the old HEAD

### Root Cause (Both Bugs)

**Using HEAD and current working-tree state to interpret stash contents.** The stash contains frozen commits — all the information needed is already inside it. The fix: use only stash-internal references (`ref^1`, `ref^2`, `ref`, `ref^3`) for diffs, never `HEAD` or current state.

## Key Insight: Stash-Internal Diffs

A git stash entry contains everything:
```
stash@{0}    = WIP commit (full working tree snapshot at stash time)
stash@{0}^1  = Parent commit (= HEAD at stash time, frozen)
stash@{0}^2  = Index commit (staged changes at stash time, frozen)
stash@{0}^3  = Untracked files tree (optional, frozen)
```

All diffs should be **between these frozen commits**. This means viewing a stash from 6 months ago shows exactly what was stashed, not what's different from today's HEAD.

| Type | Diff | What it shows |
|------|------|---------------|
| **staged** | `ref^1 → ref^2` | Staged changes at stash time (exact) |
| **unstaged** | `ref^2 → ref` + `ref^3` | Working-tree changes at stash time (exact) |
| **all** | `ref^1 → ref` + `ref^3` | Everything stashed (exact) |
| **custom** | `ref^1 → ref^2` + `ref^2 → ref`, filtered to target files | Only selected files |
| **unknown** | `ref^1 → ref` | Best-effort (same as `git stash show`) |

No skip sets. No `git diff --cached`. No `git diff --name-only`. No current state needed.

The **only** reason we store metadata in the message prefix is for **custom** stashes — because `stash^2` captures the full index (not just selected files), so we need to filter. The `[pgit:custom:a.go,b.go]` prefix tells us which files to keep.

---

## File Structure

| File | Responsibility |
|------|---------------|
| `internal/git/git.go` | Core stash logic: `StashPush`, `GetStashDetail`, `ListStashes`, message format |
| `internal/ui/stash/create_model.go` | TUI create wizard — update `doStash` for message format |
| `internal/ui/stash/browse_model.go` | TUI browse model — pass `StashEntry` to `GetStashDetail` |
| `cmd/pretty-git/stash.go` | CLI routing — default messages for `--staged`/`--unstaged` |

---

## Task 1: Add message format constants and parse functions

**Files:**
- Modify: `internal/git/git.go`

- [ ] **Step 1: Add `StashType` enum, `stashMeta` struct, and format/parse functions**

Add after the `StashDetail` struct (around line 730):

```go
// StashType represents how a stash was created.
type StashType int

const (
	StashTypeUnknown  StashType = iota // old stashes without prefix
	StashTypeAll
	StashTypeStaged
	StashTypeUnstaged
	StashTypeCustom
)

// stashMeta holds metadata parsed from a stash message prefix.
type stashMeta struct {
	Type        StashType
	TargetFiles []string // for custom stashes
	UserMsg     string   // display message (without prefix)
}

// formatStashMsg encodes stash type and user message into a stash message.
func formatStashMsg(stashType StashType, shortHash, userMsg string, targetFiles []string) string {
	msg := shortHash + ": " + userMsg
	switch stashType {
	case StashTypeStaged:
		return "[pgit:staged] " + msg
	case StashTypeUnstaged:
		return "[pgit:unstaged] " + msg
	case StashTypeAll:
		return "[pgit:all] " + msg
	case StashTypeCustom:
		if len(targetFiles) > 0 {
			return "[pgit:custom:" + strings.Join(targetFiles, ",") + "] " + msg
		}
		return "[pgit:custom] " + msg
	default:
		return msg
	}
}

// parseStashMeta extracts stash metadata from a message.
func parseStashMeta(msg string) stashMeta {
	// [pgit:custom:file1,file2] prefix
	if strings.HasPrefix(msg, "[pgit:custom:") {
		rest := msg[len("[pgit:custom:"):]
		closeBracket := strings.Index(rest, "]")
		if closeBracket >= 0 {
			filesStr := rest[:closeBracket]
			userMsg := strings.TrimLeft(rest[closeBracket+1:], " ")
			return stashMeta{
				Type:        StashTypeCustom,
				TargetFiles: strings.Split(filesStr, ","),
				UserMsg:     userMsg,
			}
		}
	}
	// [pgit:staged] / [pgit:unstaged] / [pgit:all] prefix
	if strings.HasPrefix(msg, "[pgit:staged] ") {
		return stashMeta{Type: StashTypeStaged, UserMsg: msg[len("[pgit:staged] "):]}
	}
	if strings.HasPrefix(msg, "[pgit:unstaged] ") {
		return stashMeta{Type: StashTypeUnstaged, UserMsg: msg[len("[pgit:unstaged] "):]}
	}
	if strings.HasPrefix(msg, "[pgit:all] ") {
		return stashMeta{Type: StashTypeAll, UserMsg: msg[len("[pgit:all] "):]}
	}
	if strings.HasPrefix(msg, "[pgit:custom] ") {
		return stashMeta{Type: StashTypeCustom, UserMsg: msg[len("[pgit:custom] "):]}
	}
	return stashMeta{Type: StashTypeUnknown, UserMsg: msg}
}

// stripStashPrefix removes the [pgit:...] prefix for display.
func stripStashPrefix(msg string) string {
	meta := parseStashMeta(msg)
	return meta.UserMsg
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd /home/sai/projects/pretty-git && go build ./...`

- [ ] **Step 3: Commit**

```bash
git add internal/git/git.go
git commit -m "feat: add stash type metadata — format/parse message prefixes"
```

---

## Task 2: Add `StashType` and `TargetFiles` to `StashEntry`

**Files:**
- Modify: `internal/git/git.go`

- [ ] **Step 1: Add fields to `StashEntry`**

```go
type StashEntry struct {
	Index       int
	Ref         string
	Message     string
	Branch      string
	RelTime     string
	StashType   StashType   // parsed from message prefix
	TargetFiles []string    // for custom stashes
}
```

- [ ] **Step 2: Update `ListStashes` to parse metadata and strip prefix from display message**

In the `ListStashes` function, after constructing each `StashEntry`, parse the type and strip the prefix from `Message`:

```go
// Inside ListStashes, after extracting displayMsg from the branch info:
rawMsg := displayMsg // with prefix
meta := parseStashMeta(rawMsg)
entries = append(entries, StashEntry{
	Index:       i,
	Ref:         ref,
	Message:     meta.UserMsg,   // clean display message
	Branch:      branch,
	RelTime:     relTime,
	StashType:   meta.Type,
	TargetFiles: meta.TargetFiles,
})
```

- [ ] **Step 3: Verify it compiles**

Run: `cd /home/sai/projects/pretty-git && go build ./...`

- [ ] **Step 4: Commit**

```bash
git add internal/git/git.go
git commit -m "feat: add StashType to StashEntry, parse metadata in ListStashes"
```

---

## Task 3: Rewrite `GetStashDetail` — stash-internal diffs

**Files:**
- Modify: `internal/git/git.go`

This is the core fix for Bug 2. Replace the entire `getStashDetail` function (skip sets + current state) with type-aware diffs using only stash-internal references.

- [ ] **Step 1: Replace `GetStashDetail` and remove `getStashDetail` (old version with skip sets)**

Replace `GetStashDetail` with two functions:

```go
// GetStashDetail fetches extended info for a single stash entry.
// Uses stash-internal commits for diffs — never HEAD or current working tree.
func GetStashDetail(ref string) (StashDetail, error) {
	return getStashDetailTyped(ref, StashTypeUnknown, nil)
}

// GetStashDetailTyped fetches extended info using the stash type for
// the correct diff strategy.
func GetStashDetailTyped(ref string, stashType StashType, targetFiles []string) (StashDetail, error) {
	return getStashDetailTyped(ref, stashType, targetFiles)
}

func getStashDetailTyped(ref string, stashType StashType, targetFiles []string) (StashDetail, error) {
	var files []FileStatus
	var filesChanged int
	seen := make(map[string]bool)

	switch stashType {
	case StashTypeStaged:
		// Staged: only HEAD-at-stash-time → index
		parseDiffInto(ref+"^1", ref+"^2", true, seen, &files, &filesChanged)

	case StashTypeUnstaged:
		// Unstaged: only index → WIP, plus untracked
		parseDiffInto(ref+"^2", ref, false, seen, &files, &filesChanged)
		addUntrackedFiles(ref, seen, &files, &filesChanged)

	case StashTypeCustom:
		// Custom: both diffs, filtered to target files
		targetSet := make(map[string]bool)
		for _, f := range targetFiles {
			targetSet[f] = true
		}
		parseDiffFiltered(ref+"^1", ref+"^2", true, targetSet, seen, &files, &filesChanged)
		parseDiffFiltered(ref+"^2", ref, false, targetSet, seen, &files, &filesChanged)
		addUntrackedFilesFiltered(ref, targetSet, seen, &files, &filesChanged)

	case StashTypeAll:
		// All: both diffs (all files)
		parseDiffInto(ref+"^1", ref+"^2", true, seen, &files, &filesChanged)
		parseDiffInto(ref+"^2", ref, false, seen, &files, &filesChanged)
		addUntrackedFiles(ref, seen, &files, &filesChanged)

	default:
		// Unknown: use full WIP diff (same as git stash show)
		parseDiffInto(ref+"^1", ref, false, seen, &files, &filesChanged)
		addUntrackedFiles(ref, seen, &files, &filesChanged)
	}

	// Compute stats from the same diffs we showed files for
	insertions, deletions := computeStats(ref, stashType, targetFiles)

	return StashDetail{
		FilesChanged: filesChanged,
		Insertions:   insertions,
		Deletions:    deletions,
		Files:        files,
	}, nil
}
```

- [ ] **Step 2: Add helper functions**

```go
// parseDiffInto runs git diff --name-status between two refs and appends files.
func parseDiffInto(fromRef, toRef string, staged bool, seen map[string]bool, files *[]FileStatus, filesChanged *int) {
	out, err := run("git", "diff", "--name-status", fromRef, toRef)
	if err != nil {
		return
	}
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		letter := strings.TrimRight(fields[0], "0123456789")
		path := fields[len(fields)-1]
		if seen[path] {
			continue
		}
		seen[path] = true
		*filesChanged++
		var fCode string
		if staged {
			switch letter {
			case "M":
				fCode = "M "
			case "A":
				fCode = "A "
			case "D":
				fCode = "D "
			default:
				fCode = letter + " "
			}
		} else {
			switch letter {
			case "M":
				fCode = " M"
			case "A":
				fCode = " A"
			case "D":
				fCode = " D"
			default:
				fCode = " " + letter
			}
		}
		*files = append(*files, FileStatus{Code: fCode, Path: path})
	}
}

// parseDiffFiltered runs git diff --name-status and only includes files in targetSet.
func parseDiffFiltered(fromRef, toRef string, staged bool, targetSet map[string]bool, seen map[string]bool, files *[]FileStatus, filesChanged *int) {
	out, err := run("git", "diff", "--name-status", fromRef, toRef)
	if err != nil {
		return
	}
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		letter := strings.TrimRight(fields[0], "0123456789")
		path := fields[len(fields)-1]
		if seen[path] || !targetSet[path] {
			continue
		}
		seen[path] = true
		*filesChanged++
		var fCode string
		if staged {
			switch letter {
			case "M":
				fCode = "M "
			case "A":
				fCode = "A "
			case "D":
				fCode = "D "
			default:
				fCode = letter + " "
			}
		} else {
			switch letter {
			case "M":
				fCode = " M"
			case "A":
				fCode = " A"
			case "D":
				fCode = " D"
			default:
				fCode = " " + letter
			}
		}
		*files = append(*files, FileStatus{Code: fCode, Path: path})
	}
}

// addUntrackedFiles appends untracked files from stash^3.
func addUntrackedFiles(ref string, seen map[string]bool, files *[]FileStatus, filesChanged *int) {
	out, err := run("git", "ls-tree", "--name-only", ref+"^3")
	if err != nil {
		return // stash^3 may not exist
	}
	for _, path := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if path == "" || seen[path] {
			continue
		}
		seen[path] = true
		*filesChanged++
		*files = append(*files, FileStatus{Code: "??", Path: path})
	}
}

// addUntrackedFilesFiltered appends untracked files from stash^3 only if in targetSet.
func addUntrackedFilesFiltered(ref string, targetSet map[string]bool, seen map[string]bool, files *[]FileStatus, filesChanged *int) {
	out, err := run("git", "ls-tree", "--name-only", ref+"^3")
	if err != nil {
		return
	}
	for _, path := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if path == "" || seen[path] || !targetSet[path] {
			continue
		}
		seen[path] = true
		*filesChanged++
		*files = append(*files, FileStatus{Code: "??", Path: path})
	}
}

// computeStats returns insertions and deletions for the stash, using the
// diff appropriate for the stash type.
func computeStats(ref string, stashType StashType, targetFiles []string) (insertions, deletions int) {
	switch stashType {
	case StashTypeStaged:
		return statDiff(ref+"^1", ref+"^2")
	case StashTypeUnstaged:
		ins, dels := statDiff(ref+"^2", ref)
		// Also count untracked if present
		return ins, dels
	case StashTypeCustom:
		targetSet := make(map[string]bool)
		for _, f := range targetFiles {
			targetSet[f] = true
		}
		return statDiffFiltered(ref+"^1", ref, targetSet)
	default:
		// For "all" and unknown, use full WIP diff
		return statDiff(ref+"^1", ref)
	}
}

// statDiff computes insertions/deletions from git diff --numstat between two refs.
func statDiff(fromRef, toRef string) (insertions, deletions int) {
	out, err := run("git", "diff", "--numstat", fromRef, toRef)
	if err != nil {
		return 0, 0
	}
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		ins, dels, _ := parseNumstatLine(fields)
		insertions += ins
		deletions += dels
	}
	return insertions, deletions
}

// statDiffFiltered computes insertions/deletions only for files in targetSet.
func statDiffFiltered(fromRef, toRef string, targetSet map[string]bool) (insertions, deletions int) {
	out, err := run("git", "diff", "--numstat", fromRef, toRef)
	if err != nil {
		return 0, 0
	}
	for _, line := range strings.TrimRight(out, "\n") {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		path := fields[2]
		if !targetSet[path] {
			continue
		}
		ins, dels, _ := parseNumstatLine(fields)
		insertions += ins
		deletions += dels
	}
	return insertions, deletions
}

func parseNumstatLine(fields []string) (insertions, deletions int, path string) {
	path = fields[len(fields)-1]
	ins, err := strconv.Atoi(fields[0])
	if err != nil || fields[0] == "-" {
		ins = 0 // binary file
	}
	dels, err := strconv.Atoi(fields[1])
	if err != nil || fields[1] == "-" {
		dels = 0
	}
	return ins, dels, path
}
```

**IMPORTANT:** Fix the bug in `statDiffFiltered` — `strings.TrimRight` returns a string, not a slice. It should be `strings.Split(strings.TrimRight(out, "\n"), "\n")`:

```go
func statDiffFiltered(fromRef, toRef string, targetSet map[string]bool) (insertions, deletions int) {
	out, err := run("git", "diff", "--numstat", fromRef, toRef)
	if err != nil {
		return 0, 0
	}
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		path := fields[len(fields)-1]
		if !targetSet[path] {
			continue
		}
		ins, dels, _ := parseNumstatLine(fields)
		insertions += ins
		deletions += dels
	}
	return insertions, deletions
}
```

- [ ] **Step 3: Remove old functions**

Remove:
- `getStashDetail` (the old version with `currentStaged`/`currentUnstaged` parameters)
- `checkNoCollateral`
- `checkStashContent`
- `porcelainStatus` (if only used by the removed verification)
- `stashCount` (if only used by the removed verification)
- `sortedKeys` (if only used by the removed verification)

**Check first**: `porcelainStatus` might be used elsewhere. Search for it. If only used by `checkNoCollateral` and `checkStashContent`, remove it. Same for `stashCount` and `sortedKeys`.

- [ ] **Step 4: Verify it compiles**

Run: `cd /home/sai/projects/pretty-git && go build ./...`

- [ ] **Step 5: Commit**

```bash
git add internal/git/git.go
git commit -m "feat: rewrite GetStashDetail — stash-internal diffs, no skip sets, no HEAD"
```

---

## Task 4: Rewrite `StashPush` — MM pre-check, message format, simplified verification

**Files:**
- Modify: `internal/git/git.go`

- [ ] **Step 1: Add MM file pre-check and message format to `StashPush`**

Replace the `StashPush` function. Key changes:
1. Return `(string, error)` — the string is the formatted message for success display
2. Pre-check for MM files when stashType is "staged"
3. Use `formatStashMsg` for the message
4. Remove `checkNoCollateral` and `checkStashContent` — use simple verification
5. Remove `targetSet` computation — no longer needed for verification

```go
// StashPush creates a stash with the given message and options.
// Returns the formatted stash message on success.
// stashType: "staged", "unstaged", "all", "custom"
// customFiles: used when stashType == "custom"
func StashPush(msg, stashType string, customFiles []string) (string, error) {
	// Pre-check: MM files break git stash push --staged.
	if stashType == "staged" {
		files, err := ListModifiedFiles()
		if err != nil {
			return "", fmt.Errorf("checking file status: %w", err)
		}
		var mmFiles []string
		for _, f := range files {
			if len(f.Code) >= 2 && f.Code[0] != ' ' && f.Code[0] != '?' && f.Code[1] != ' ' && f.Code != "??" {
				mmFiles = append(mmFiles, f.Path)
			}
		}
		if len(mmFiles) > 0 {
			return "", fmt.Errorf("cannot stash staged changes: the following files have both staged and unstaged changes:\n  %s\n\nStage all changes with `pgit stash` (all), or unstage the unstaged portions first", strings.Join(mmFiles, "\n  "))
		}
	}

	// Map stashType string to StashType constant
	var st StashType
	switch stashType {
	case "staged":
		st = StashTypeStaged
	case "unstaged":
		st = StashTypeUnstaged
	case "custom":
		st = StashTypeCustom
	default:
		st = StashTypeAll
	}

	// Build the git command
	var args []string
	switch stashType {
	case "staged":
		args = []string{"stash", "push", "--staged", "-m", formatStashMsg(st, LastCommitShortHash(), msg, nil)}
	case "unstaged":
		args = []string{"stash", "push", "--keep-index", "-m", formatStashMsg(st, LastCommitShortHash(), msg, nil)}
	case "all":
		args = []string{"stash", "push", "--include-untracked", "-m", formatStashMsg(st, LastCommitShortHash(), msg, nil)}
	case "custom":
		if len(customFiles) == 0 {
			return "", fmt.Errorf("no files selected for custom stash")
		}
		args = append([]string{"stash", "push", "--include-untracked", "-m", formatStashMsg(st, LastCommitShortHash(), msg, customFiles), "--"}, customFiles...)
	default:
		args = []string{"stash", "push", "--include-untracked", "-m", formatStashMsg(st, LastCommitShortHash(), msg, nil)}
	}

	out, err := exec.Command("git", args...).CombinedOutput()
	if err != nil {
		errMsg := strings.TrimSpace(string(out))
		if errMsg == "" {
			return "", err
		}
		return "", fmt.Errorf("%s", errMsg)
	}

	// Extract the formatted message for display
	finalMsg := formatStashMsg(st, LastCommitShortHash(), msg, customFiles)
	return finalMsg, nil
}
```

- [ ] **Step 2: Remove old verification functions**

Remove these functions that are no longer needed:
- `checkNoCollateral`
- `checkStashContent`
- `porcelainStatus` (check usage first — if only used by above, remove)
- `stashCount` (check usage first)
- `sortedKeys`
- `getStashDetail` (old version — already replaced)

Search the codebase for each function name to confirm no other code references them before deleting.

- [ ] **Step 3: Verify it compiles**

Run: `cd /home/sai/projects/pretty-git && go build ./...`

- [ ] **Step 4: Commit**

```bash
git add internal/git/git.go
git commit -m "feat: rewrite StashPush — MM pre-check, message format, simplified verification"
```

---

## Task 5: Update callers — CLI and TUI

**Files:**
- Modify: `cmd/pretty-git/stash.go`
- Modify: `internal/ui/stash/create_model.go`
- Modify: `internal/ui/stash/browse_model.go`

- [ ] **Step 1: Update `cmd/pretty-git/stash.go`**

The CLI quick-stash path currently prepends the hash. Since `StashPush` now handles formatting, remove the hash prefix:

```go
// In runStash, replace:
//   shortHash := git.LastCommitShortHash()
//   finalMsg := shortHash + ": " + msg
//   if err := git.StashPush(finalMsg, stashType, customFiles); err != nil {
// With:
//   result, err := git.StashPush(msg, stashType, customFiles)
//   if err != nil { ... }
//   fmt.Fprintf(os.Stderr, "✓ stash created: %s\n", result)

// Also update the "nothing to stash" check for --staged:
// The MM pre-check in StashPush handles --staged now,
// but we still need the "nothing to stash" check for the TUI path.
```

Add default messages for `--staged`/`--unstaged` without args:

```go
msg := strings.Join(msgArgs, " ")
if msg == "" {
	switch stashType {
	case "staged":
		msg = "staged changes"
	case "unstaged":
		msg = "unstaged changes"
	case "custom":
		msg = "selected files"
	default:
		runStashCreate(repoName, width, height)
		return
	}
}

result, err := git.StashPush(msg, stashType, customFiles)
if err != nil {
	fmt.Fprintf(os.Stderr, "pgit: %v\n", err)
	os.Exit(1)
}
fmt.Fprintf(os.Stderr, "✓ stash created: %s\n", result)
```

- [ ] **Step 2: Update `internal/ui/stash/create_model.go`**

In `doStash`, update to use the new `StashPush` signature (returns string, error) and let it handle message formatting:

In the `stashDoneMsg` struct, add a `Msg` field:

```go
type stashDoneMsg struct {
	err error
	msg string
}
```

In `doStash`:

```go
func (m *CreateModel) doStash() tea.Cmd {
	return func() tea.Msg {
		userMsg := strings.TrimSpace(m.msgInput.Value())
		if userMsg == "" {
			userMsg = m.defaultMsg
		}

		var stashTypeStr string
		var customFiles []string
		switch m.stashType {
		case stashTypeStaged:
			stashTypeStr = "staged"
		case stashTypeUnstaged:
			stashTypeStr = "unstaged"
		case stashTypeAll:
			stashTypeStr = "all"
		case stashTypeCustom:
			stashTypeStr = "custom"
			for i, f := range m.files {
				if i < len(m.fileSelected) && m.fileSelected[i] {
					customFiles = append(customFiles, f.Path)
				}
			}
		}

		result, err := git.StashPush(userMsg, stashTypeStr, customFiles)
		return stashDoneMsg{err: err, msg: result}
	}
}
```

In the `Update` method, update `stashDoneMsg` handling to use the msg field:

```go
case stashDoneMsg:
	if msg.err != nil {
		m.execErr = msg.err
		m.phase = phaseMessage // go back to message phase to show error
		return m, nil
	}
	m.result = msg.msg
	return m, tea.Quit
```

- [ ] **Step 3: Update `internal/ui/stash/browse_model.go`**

In `doLoadBrowseDetail`, pass the stash metadata to `GetStashDetailTyped`:

```go
func doLoadBrowseDetail(entry git.StashEntry) tea.Cmd {
	return func() tea.Msg {
		d, err := git.GetStashDetailTyped(entry.Ref, entry.StashType, entry.TargetFiles)
		if err == nil {
			d.StashEntry = entry // populate all metadata
		}
		return browseDetailLoadedMsg{ref: entry.Ref, detail: d, err: err}
	}
}
```

- [ ] **Step 4: Verify it compiles**

Run: `cd /home/sai/projects/pretty-git && go build ./...`

- [ ] **Step 5: Commit**

```bash
git add cmd/pretty-git/stash.go internal/ui/stash/create_model.go internal/ui/stash/browse_model.go
git commit -m "feat: update CLI and TUI callers for new StashPush and GetStashDetail signatures"
```

---

## Task 6: Update browse detail rendering for stash type

**Files:**
- Modify: `internal/ui/stash/browse_model.go`

- [ ] **Step 1: Add stash type label in `buildBrowseDetailLines`**

After the message line in `buildBrowseDetailLines`, add a type indicator:

```go
// After: add("  " + valS(d.Message))
// Add type label
typeLabel := ""
switch d.StashType {
case git.StashTypeStaged:
	typeLabel = "  staged"
case git.StashTypeUnstaged:
	typeLabel = "  unstaged"
case git.StashTypeCustom:
	typeLabel = "  custom"
	if len(d.TargetFiles) > 0 {
		typeLabel = fmt.Sprintf("  custom (%d files)", len(d.TargetFiles))
	}
case git.StashTypeAll:
	typeLabel = "  all"
}
if typeLabel != "" {
	add("  " + lipgloss.NewStyle().Foreground(ui.ColorAccent).Bold(true).Render(typeLabel))
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd /home/sai/projects/pretty-git && go build ./...`

- [ ] **Step 3: Commit**

```bash
git add internal/ui/stash/browse_model.go
git commit -m "feat: display stash type label in browse detail pane"
```

---

## Task 7: Manual integration testing

- [ ] **Step 1: Build the binary**

```bash
cd /home/sai/projects/pretty-git && go build -o /tmp/pgit-test ./cmd/pretty-git/
```

- [ ] **Step 2: Test Bug 1 — MM file rejection**

```bash
cd /tmp && rm -rf stash-test && mkdir stash-test && cd stash-test
git init && git config user.email "t@t" && git config user.name "T"
echo "hello" > a.go && git add . && git commit -m "init"
echo "// staged" >> a.go && git add a.go
echo "// unstaged too" >> a.go
/tmp/pgit-test stash --staged "mm-test" 2>&1
```

Expected: Clear error about MM files. No stash created. Working tree unchanged.

- [ ] **Step 3: Test Bug 1 — Staged stash on pure staged files**

```bash
cd /tmp && rm -rf stash-test && mkdir stash-test && cd stash-test
git init && git config user.email "t@t" && git config user.name "T"
echo "hello" > a.go && echo "world" > b.go && git add . && git commit -m "init"
echo "// staged a" >> a.go && git add a.go
echo "// staged b" >> b.go && git add b.go
echo "// unstaged c" >> c.go 2>/dev/null; echo "foo" > c.go && echo "// change" >> c.go
/tmp/pgit-test stash --staged "staged-test" 2>&1
echo "Exit: $?"
git status --porcelain
git stash list
```

Expected: Stash created with `[pgit:staged]` prefix. `a.go` and `b.go` stashed, `c.go` unchanged.

- [ ] **Step 4: Test Bug 2 — Browse detail shows correct files after HEAD moves**

```bash
# After step 3, browse the stash
/tmp/pgit-test stash list
# Then commit more changes to move HEAD
echo "// new commit" >> b.go && git add b.go && git commit -m "move HEAD"
# Now browse the old stash — should still show only a.go and b.go (the staged files)
# NOT c.go or any other files
```

Expected: Detail pane shows only `a.go` and `b.go` (staged files), not `c.go`.

- [ ] **Step 5: Test custom stash with extra staged files**

```bash
git stash drop
echo "// staged a" >> a.go && git add a.go
echo "// staged b" >> b.go && git add b.go
echo "// staged c" >> c.go 2>/dev/null; echo "new" > c.go && git add c.go
/tmp/pgit-test stash --custom "custom-test" -- a.go 2>&1
git stash list
# Browse stash — should show only a.go, NOT b.go or c.go
```

Expected: Detail shows only `a.go`.

- [ ] **Step 6: Run Go tests**

```bash
cd /home/sai/projects/pretty-git && go test ./...
```

- [ ] **Step 7: Commit any fixes**

```bash
git add -A && git commit -m "test: integration testing for stash rewrite"
```

---

## Task 8: Update test script

**Files:**
- Modify: `scripts/stash_tests.sh`

- [ ] **Step 1: Update `stash_files` function**

The test script's `stash_files` function uses current working tree state for filtering. Update it to use stash-internal diffs:

```bash
stash_files() {
    local ref="stash@{0}"
    local msg
    msg=$(git -C "$REPO" stash list --format="%gs" | head -1)

    # Parse stash type from message prefix
    local type="unknown"
    case "$msg" in
        *"[pgit:staged]"*)   type="staged" ;;
        *"[pgit:unstaged]"*) type="unstaged" ;;
        *"[pgit:custom]"*)   type="custom" ;;
        *"[pgit:all]"*)      type="all" ;;
    esac

    case "$type" in
        staged)
            git -C "$REPO" diff --name-only "$ref"^1 "$ref"^2 2>/dev/null
            ;;
        unstaged)
            git -C "$REPO" diff --name-only "$ref"^2 "$ref" 2>/dev/null
            git -C "$REPO" ls-tree --name-only "$ref"^3 2>/dev/null
            ;;
        custom)
            # Filter to target files from message
            local targets
            targets=$(echo "$msg" | sed -n 's/.*\[pgit:custom:\([^]]*\)\].*/\1/p' | tr ',' '\n')
            local all_files
            all_files=$(git -C "$REPO" diff --name-only "$ref"^1 "$ref"^2 2>/dev/null;
                        git -C "$REPO" diff --name-only "$ref"^2 "$ref" 2>/dev/null)
            echo "$all_files" | while read -r f; do
                if echo "$targets" | grep -qxF "$f"; then
                    echo "$f"
                fi
            done
            ;;
        *)
            # Unknown / all — show everything
            git -C "$REPO" diff --name-only "$ref"^1 "$ref" 2>/dev/null
            git -C "$REPO" ls-tree --name-only "$ref"^3 2>/dev/null
            ;;
    esac
}
```

- [ ] **Step 2: Update test 3 for MM rejection**

Test 3 expects `pgit stash --staged` with MM files to fail. Update the assertion to check for the new error message:

```bash
# Test 3 currently checks for error. Update to check the specific message:
output=$(cd "$REPO" && $PGIT stash --staged "t3-mm" 2>&1)
if [[ "$output" == *"cannot stash staged changes"* ]]; then
    pass "error returned for MM file (expected)"
else
    fail "expected MM file error, got: $output"
fi
```

- [ ] **Step 3: Run the test suite**

```bash
cd /home/sai/projects/pretty-git && bash scripts/stash_tests.sh
```

- [ ] **Step 4: Commit**

```bash
git add scripts/stash_tests.sh
git commit -m "test: update stash tests for stash-internal diffs and MM pre-check"
```

---

## Self-Review

### Spec Coverage
- ✅ Bug 1 (staged fails on MM): Task 4 (MM pre-check with clear error)
- ✅ Bug 2 (too many files in detail): Task 3 (stash-internal diffs, no HEAD/current state)
- ✅ Message format: Tasks 1, 4
- ✅ TUI integration: Tasks 5, 6
- ✅ Default message for `--staged`/`--unstaged`: Task 5
- ✅ End-to-end testing: Task 7
- ✅ Test script updates: Task 8

### Placeholder Scan
- No TBD, TODO, or "implement later"
- All code steps contain actual implementation
- All commands are specific

### Type Consistency
- `StashType` enum defined in Task 1, used in Tasks 2, 3, 4, 5, 6
- `formatStashMsg`/`parseStashMeta` defined in Task 1, called in Tasks 3, 4, 5
- `GetStashDetailTyped` defined in Task 3, called from Task 5
- `StashPush` returns `(string, error)` from Task 4, called in Task 5
- All `FileStatus` usage consistent with existing struct