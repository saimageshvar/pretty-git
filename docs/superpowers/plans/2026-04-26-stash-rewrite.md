# Stash Rewrite — Lean Native Wrapper

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rewrite the stash feature to leverage native `git stash` commands exclusively — no custom metadata encoding, no stash-internal commit diffing. A pure aesthetic wrapper.

**Architecture:** Delete ~300 lines of custom stash diffing/metadata from `internal/git/git.go`. Add 2 new native-git functions (`StashShowFiles`, `StashShowSummary`). Simplify `StashPush` and `ListStashes`. Rewrite the browse model to use `git stash show` for detail. Rewrite the create model to disable staged-stash when MM files exist. Rewrite `cmd/pretty-git/stash.go` router.

**Tech Stack:** Go, Bubble Tea (charmbracelet), lipgloss, native `git` CLI

---

### Task 1: Add `StashShowFiles` and `StashShowSummary` to `internal/git/git.go`

**Files:**
- Modify: `internal/git/git.go`

These are new functions that don't break existing code. They'll be used by the new browse model.

- [ ] **Step 1: Add `StashShowFiles` function**

Insert after line 873 (after the `GetStashDetail` function block ends, but before the `parseStashDiff` block — since we'll delete all of that later). Place it right after the closing `}` of `GetStashDetailTyped` (line 873).

```go
// StashShowFiles returns files changed in a stash via `git stash show --name-status <ref>`.
func StashShowFiles(ref string) ([]StashDetailFile, error) {
	out, err := run("git", "stash", "show", "--name-status", ref)
	if err != nil {
		return nil, err
	}
	var files []StashDetailFile
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		// Strip similarity score from renames (e.g. "R100" → "R")
		status := strings.TrimRight(fields[0], "0123456789")
		path := fields[len(fields)-1]
		files = append(files, StashDetailFile{Status: status, Path: path})
	}
	return files, nil
}

// StashDetailFile represents a single file in a stash detail.
type StashDetailFile struct {
	Status string // "M", "A", "D", ...
	Path   string
}
```

- [ ] **Step 2: Add `StashShowSummary` function**

Insert after the new `StashShowFiles` function:

```go
// StashShowSummary extracts files-changed, insertions, deletions from `git stash show <ref>`.
func StashShowSummary(ref string) (files, insertions, deletions int, err error) {
	out, err := run("git", "stash", "show", ref)
	if err != nil {
		return 0, 0, 0, err
	}
	out = strings.TrimSpace(out)
	// `git stash show` output is a one-line diffstat summary, e.g.:
	//  src/a.go | 5 +++--
	//  src/b.go | 3 ++-
	//  2 files changed, 5 insertions(+), 3 deletions(-)
	// We only need the last line (the summary).
	lines := strings.Split(out, "\n")
	if len(lines) == 0 {
		return 0, 0, 0, nil
	}
	last := strings.TrimSpace(lines[len(lines)-1])
	// Parse: "N file(s) changed, X insertion(s)(+), Y deletion(s)(-)"
	return parseDiffStatSummary(last)
}
```

- [ ] **Step 3: Add `parseDiffStatSummary` helper**

```go
// parseDiffStatSummary parses the summary line from git diff --stat / git stash show.
// Format: "N file(s) changed, X insertion(s)(+), Y deletion(s)(-)"
// Some parts may be omitted if zero.
func parseDiffStatSummary(line string) (files, insertions, deletions int, err error) {
	parts := strings.Split(line, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if strings.HasPrefix(p, "file") || strings.HasPrefix(p, "1 file") {
			if n, readErr := fmt.Sscanf(p, "%d file", &files); n == 1 && readErr == nil {
				continue
			}
		}
		if idx := strings.Index(p, " insertion"); idx >= 0 {
			if n, readErr := fmt.Sscanf(p[:idx], "%d", &insertions); n == 1 && readErr == nil {
				continue
			}
		}
		if idx := strings.Index(p, " deletion"); idx >= 0 {
			if n, readErr := fmt.Sscanf(p[:idx], "%d", &deletions); n == 1 && readErr == nil {
				continue
			}
		}
	}
	return files, insertions, deletions, nil
}
```

- [ ] **Step 4: Verify it compiles**

Run: `go build ./...`
Expected: Should compile (new functions don't break existing code).

- [ ] **Step 5: Commit**

```bash
git add internal/git/git.go
git commit -m "feat(stash): add StashShowFiles and StashShowSummary using native git stash show"
```

---

### Task 2: Simplify `StashPush` — remove metadata encoding

**Files:**
- Modify: `internal/git/git.go`

Replace the current `StashPush` (lines 1062–1142) with a version that passes the user message directly to git without any `[pgit:...]` prefix.

- [ ] **Step 1: Replace `StashPush` function**

The new version (replace lines 1062–1142):

```go
// StashPush creates a stash with the given message and options.
// stashType: "all", "staged", "unstaged", "custom"
// customFiles: used when stashType == "custom"
func StashPush(msg, stashType string, customFiles []string) (string, error) {
	var args []string
	switch stashType {
	case "staged":
		args = []string{"stash", "push", "--staged", "-m", msg}
	case "unstaged":
		args = []string{"stash", "push", "--keep-index", "-m", msg}
	case "custom":
		if len(customFiles) == 0 {
			return "", fmt.Errorf("no files selected for custom stash")
		}
		args = append([]string{"stash", "push", "--include-untracked", "-m", msg, "--"}, customFiles...)
	default: // "all"
		args = []string{"stash", "push", "--include-untracked", "-m", msg}
	}

	out, err := exec.Command("git", args...).CombinedOutput()
	if err != nil {
		errMsg := strings.TrimSpace(string(out))
		if errMsg == "" {
			return "", err
		}
		return "", fmt.Errorf("%s", errMsg)
	}

	return strings.TrimSpace(string(out)), nil
}
```

- [ ] **Step 2: Verify it compiles**

The `formatStashMsg` call is removed but `formatStashMsg` still exists in the file (not yet deleted). Should compile.

Run: `go build ./...`

- [ ] **Step 3: Test quick stash with the test repo**

```bash
cd ~/projects/pg-test
# First clean up existing stash
git stash drop stash@{0} 2>/dev/null || true
# Verify current state
git status --porcelain
# Should show staged b.go and c.go
```

- [ ] **Step 4: Commit**

```bash
git add internal/git/git.go
git commit -m "refactor(stash): simplify StashPush — no metadata encoding, clean messages to git"
```

---

### Task 3: Simplify `ListStashes` — remove metadata parsing

**Files:**
- Modify: `internal/git/git.go`
- Modify: `internal/ui/stash/browse_model.go` (uses `StashType` and `TargetFiles` fields)

The current `ListStashes` at lines 769–817 parses `[pgit:...]` prefixes and populates `StashType`/`TargetFiles` on each entry. Replace it with a version that skips all of that.

- [ ] **Step 1: Update `StashEntry` type**

Replace the current `StashEntry` struct (lines 627–635) — remove `StashType` and `TargetFiles`:

```go
// StashEntry holds summary metadata for one stash ref.
type StashEntry struct {
	Index   int
	Ref     string // "stash@{N}"
	Message string
	Branch  string
	RelTime string
}
```

- [ ] **Step 2: Rewrite `ListStashes` function**

Replace lines 769–817:

```go
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

		// Extract branch and clean message from "On <branch>: <msg>" or "WIP on <branch>: <msg>"
		branch := ""
		userMsg := msg
		if after, ok := strings.CutPrefix(msg, "WIP on "); ok {
			if colonIdx := strings.Index(after, ": "); colonIdx >= 0 {
				branch = after[:colonIdx]
				userMsg = after[colonIdx+2:]
			}
		} else if after, ok := strings.CutPrefix(msg, "On "); ok {
			if colonIdx := strings.Index(after, ": "); colonIdx >= 0 {
				branch = after[:colonIdx]
				userMsg = after[colonIdx+2:]
			}
		}

		entries = append(entries, StashEntry{
			Index:   i,
			Ref:     ref,
			Message: userMsg,
			Branch:  branch,
			RelTime: relTime,
		})
	}
	return entries, nil
}
```

- [ ] **Step 3: Temporarily fix `browse_model.go` compile errors**

The browse model currently references `entry.StashType` and `entry.TargetFiles` and calls `git.GetStashDetailTyped`. We're deleting those, but we'll rewrite the browse model in Task 5. For now, to keep the build passing, remove the StashType-dependent code in the browse model's `buildBrowseDetailLines` function (lines 510–524) and in `doLoadBrowseDetail` (line 629).

In `internal/ui/stash/browse_model.go`, replace `doLoadBrowseDetail` (lines 627–635) with a stub that uses the new simple `GetStashDetail`:

```go
func doLoadBrowseDetail(entry git.StashEntry) tea.Cmd {
	return func() tea.Msg {
		d := git.StashDetail{StashEntry: entry}
		files, err := git.StashShowFiles(entry.Ref)
		if err == nil {
			d.Files = make([]git.FileStatus, len(files))
			for i, f := range files {
				d.Files[i] = git.FileStatus{
					Code: f.Status,
					Path: f.Path,
				}
			}
			d.FilesChanged = len(files)
		}
		fc, ins, dels, sumErr := git.StashShowSummary(entry.Ref)
		if sumErr == nil {
			d.FilesChanged = fc
			d.Insertions = ins
			d.Deletions = dels
		}
		return browseDetailLoadedMsg{ref: entry.Ref, detail: d, err: err}
	}
}
```

In `buildBrowseDetailLines` (lines 510–524), remove the `switch d.StashType` block:

```go
	// Message (ref is already shown in the column header)
	add("  " + valS(d.Message))
	add("")  // no type badge
```

Also update the `StashDetail` type's `Files` field — currently `Files []FileStatus` but `StashShowFiles` returns `[]StashDetailFile`. We need to handle this. For now in the temporary fix, map `StashDetailFile` to `FileStatus` as shown above. The proper rewrite in Task 5 will fix this cleanly.

- [ ] **Step 4: Verify it compiles**

Run: `go build ./...`

- [ ] **Step 5: Commit**

```bash
git add internal/git/git.go internal/ui/stash/browse_model.go
git commit -m "refactor(stash): simplify ListStashes — no metadata parsing; temp browse fixes"
```

---

### Task 4: Rewrite `cmd/pretty-git/stash.go` router

**Files:**
- Modify: `cmd/pretty-git/stash.go`

The router stays structurally the same but remove the unused `git.LastCommitOneLiner()` import (no longer needed). Actually the import is `git` — it has LastCommitOneLiner. Wait, `LastCommitOneLiner` is used by `create_model.go` for the placeholder. But `stash.go` doesn't call it directly. Let me check current imports...

Actually the router file doesn't import `strings` if we remove the `--custom` flag parsing (which we're keeping). Let me re-check. The current router (lines 1–175) handles:
- Subcommand routing (apply, pop, drop, list)
- Quick stash routing (--staged, --unstaged, --custom, plain msg)

We're keeping all of that. The router itself doesn't change much — it just calls `git.StashPush()` which is now simpler.

- [ ] **Step 1: Replace `LastCommitOneLiner` with a simpler default message**

In the router's quick stash path (lines 91–106), the default messages for --staged/--unstaged/--custom are simple strings. These are fine. But the "all" case falls back to the interactive wizard if no message is given. This is unchanged.

For the `runStashCreate` function (line 136), the `git.LastCommitOneLiner()` call is now only used in `create_model.go` — no router changes needed.

Actually, looking more carefully at `stash.go`, it does NOT use `LastCommitOneLiner`. The router only calls:
- `git.RepoName()`
- `git.ListModifiedFiles()`
- `git.ListStashes()`
- `git.StashPush()`
- `stashui.NewCreate()`, `stashui.NewBrowse()`
- `stashui.BrowseModeApply`, etc.

The router file doesn't need changes! The functions it calls are simpler internally but the signatures are the same.

- [ ] **Step 1: No changes needed. Skip to commit.**

Wait, let me verify the signatures:
- `StashPush(msg, stashType string, customFiles []string) (string, error)` — same as before
- `ListStashes() ([]StashEntry, error)` — same as before
- `ListModifiedFiles() ([]FileStatus, error)` — unchanged

The router compiles as-is. Skip this task.

---

### Task 5: Rewrite `internal/ui/stash/create_model.go` — MM handling + clean messages

**Files:**
- Modify: `internal/ui/stash/create_model.go`

The create wizard needs these changes:
1. "Staged changes" option is disabled when MM files exist
2. A warning text appears below the options
3. `doStash()` calls simplified `StashPush` (no `shortHash`, no `formatStashMsg`)

- [ ] **Step 1: Add MM detection and disabled state**

In `typeOption`, add a `disabled` field. In `typeOptions()`, check for MM files and mark the staged option disabled. Add a new field `mmFiles []string` to `CreateModel` and a `mmWarning string` field for the warning text.

In `NewCreate` (around line 114), after calculating staged/unstaged counts, detect MM files:

```go
// Detect MM files for staged-stash warning
var mmFiles []string
for _, f := range files {
    if len(f.Code) >= 2 &&
        f.Code[0] != ' ' && f.Code[0] != '?' &&
        f.Code[1] != ' ' && f.Code != "??" {
        mmFiles = append(mmFiles, f.Path)
    }
}
```

Store `mmFiles` in the model.

Update `typeOption`:
```go
type typeOption struct {
    label    string
    desc     string
    count    int
    disabled bool
    disabledReason string
}
```

Update `typeOptions()` to populate `disabled` and `disabledReason` for the staged option when MM files exist.

- [ ] **Step 2: Update left pane navigation to skip disabled options**

In `updateLeftPane` (lines 214–260), when navigating up/down, skip items where `opts[i].disabled` is true (unless it's the custom option, which is always navigable even with 0 files). Also, when pressing Select on a disabled option, do nothing (or show a brief message).

- [ ] **Step 3: Update left pane rendering for disabled state**

In `viewTypeSelect` (lines 431–680), render the disabled staged option differently:
- Use dim colors for the label, radio, and count
- Show the `disabledReason` text below all options
- No cursor background when cursor is on disabled item

Append the MM warning after the last option row in the left pane:
```go
if m.mmWarning != "" {
    leftRows = append(leftRows, padToLW(""))
    warningStyle := lipgloss.NewStyle().Foreground(ui.ColorWarning).Italic(true)
    leftRows = append(leftRows, padToLW("  "+warningStyle.Render("↳ "+m.mmWarning)))
}
```

- [ ] **Step 4: Update `doStash()` to pass clean messages**

In `doStash()` (lines 351–384), remove the `stashTypeStr` switch and the `formatStashMsg` call. Pass the user message directly:

```go
func (m *CreateModel) doStash() tea.Cmd {
    return func() tea.Msg {
        userMsg := strings.TrimSpace(m.msgInput.Value())
        if userMsg == "" {
            userMsg = m.defaultMsg
        }

        var stashTypeStr string
        switch m.stashType {
        case stashTypeStaged:
            stashTypeStr = "staged"
        case stashTypeUnstaged:
            stashTypeStr = "unstaged"
        case stashTypeAll:
            stashTypeStr = "all"
        case stashTypeCustom:
            stashTypeStr = "custom"
        }

        var customFiles []string
        if m.stashType == stashTypeCustom {
            for i, f := range m.files {
                if i < len(m.fileSelected) && m.fileSelected[i] {
                    customFiles = append(customFiles, f.Path)
                }
            }
            if len(customFiles) == 0 {
                return stashDoneMsg{err: fmt.Errorf("no files selected")}
            }
        }

        result, err := git.StashPush(userMsg, stashTypeStr, customFiles)
        return stashDoneMsg{err: err, msg: result}
    }
}
```

Note: remove the `m.defaultMsg` initialization from `LastCommitOneLiner` — use a simple string instead, or keep `LastCommitOneLiner` for the placeholder only. Actually, `LastCommitOneLiner` is still useful for the placeholder text in the message input. Keep it for the placeholder.

Wait, the `NewCreate` sets `defaultMsg = git.LastCommitOneLiner()` on line 153. This is used as the placeholder/fallback. `LastCommitOneLiner` returns `<hash> <subject>` which is a Git-level value, not metadata. Keep it — it's a good default.

- [ ] **Step 5: Remove unused imports**

After the rewrite, check if `LastCommitOneLiner` is still imported from `git` package (it is — used for placeholder). The `stashDoneMsg` type, `paneRows` const, `statusFg`, `statusColor`, `truncateCreateStr` all stay. The `phaseTypeSelect`, `phaseMessage`, `phaseExecuting` constants stay.

- [ ] **Step 6: Verify it compiles**

Run: `go build ./...`

- [ ] **Step 7: Commit**

```bash
git add internal/ui/stash/create_model.go
git commit -m "feat(stash): disable staged option for MM files; clean StashPush calls"
```

---

### Task 6: Rewrite `internal/ui/stash/browse_model.go` — native stash show

**Files:**
- Modify: `internal/ui/stash/browse_model.go`

Full rewrite of the browse model to use `StashShowFiles` and `StashShowSummary` instead of `GetStashDetailTyped`. Remove all stash-type-dependent rendering.

- [ ] **Step 1: Update `StashDetail` type in `git.go` to use `StashDetailFile`**

Before we rewrite browse_model.go, we need the `StashDetail.Files` field to be `[]StashDetailFile` instead of `[]FileStatus`. Update the `StashDetail` struct in `git.go` (lines 638–644):

```go
// StashDetail holds extended info for one stash.
type StashDetail struct {
	StashEntry
	FilesChanged int
	Insertions   int
	Deletions    int
	Files        []StashDetailFile
}
```

- [ ] **Step 2: Remove `GetStashDetail` and `GetStashDetailTyped`**

Delete lines 819–874 (the `GetStashDetail` and `GetStashDetailTyped` functions and all their helper comments). We'll replace them with a single simple function.

- [ ] **Step 3: Add a new simple `GetStashDetail`**

```go
// GetStashDetail fetches stash detail using native git stash show.
func GetStashDetail(ref string) (StashDetail, error) {
	files, err := StashShowFiles(ref)
	if err != nil {
		return StashDetail{}, err
	}
	filesChanged, insertions, deletions, err := StashShowSummary(ref)
	if err != nil {
		// Summary parsing may fail on edge cases; fall back to file count
		filesChanged = len(files)
		insertions, deletions = 0, 0
	}
	return StashDetail{
		FilesChanged: filesChanged,
		Insertions:   insertions,
		Deletions:    deletions,
		Files:        files,
	}, nil
}
```

- [ ] **Step 4: Rewrite `doLoadBrowseDetail` in `browse_model.go`**

Replace lines 627–635:

```go
func doLoadBrowseDetail(entry git.StashEntry) tea.Cmd {
	return func() tea.Msg {
		d, err := git.GetStashDetail(entry.Ref)
		if err == nil {
			d.StashEntry = entry
		}
		return browseDetailLoadedMsg{ref: entry.Ref, detail: d, err: err}
	}
}
```

- [ ] **Step 5: Rewrite `buildBrowseDetailLines`**

Replace lines 493–571. The new version:
- No type badge rendering
- Uses `StashDetailFile.Status` directly for status display
- Same structure otherwise

```go
func buildBrowseDetailLines(d *git.StashDetail, dw int) []string {
	inner := dw - 4
	if inner < 10 {
		inner = 10
	}

	var lines []string
	add := func(s string) { lines = append(lines, s) }

	dimS := func(s string) string { return ui.StyleDim.Render(s) }
	valS := func(s string) string {
		return lipgloss.NewStyle().Foreground(ui.ColorHeader).Render(s)
	}

	// Message (ref is already shown in the column header)
	add("  " + valS(d.Message))
	add("")

	// Metadata
	if d.Branch != "" {
		add("  " + dimS("Branch  ") + valS(d.Branch))
	}
	if d.RelTime != "" {
		add("  " + dimS("Date    ") + ui.StyleRelTime.Render(d.RelTime))
	}

	if len(d.Files) == 0 {
		add("")
		add("  " + dimS("(no file changes)"))
		return lines
	}

	// Summary line
	add("")
	statLine := fmt.Sprintf("%d file", d.FilesChanged)
	if d.FilesChanged != 1 {
		statLine += "s"
	}
	statLine += " changed"
	if d.Insertions > 0 {
		statLine += ", " + lipgloss.NewStyle().Foreground(ui.ColorParentMerged).Bold(true).
			Render(fmt.Sprintf("+%d", d.Insertions))
	}
	if d.Deletions > 0 {
		statLine += ", " + lipgloss.NewStyle().Foreground(ui.ColorError).Bold(true).
			Render(fmt.Sprintf("-%d", d.Deletions))
	}
	add("  " + statLine)
	add("")
	add("  " + dimS("── changes "+strings.Repeat("─", max(0, inner-10))))

	// File list — use StashDetailFile.Status directly
	for _, f := range d.Files {
		statusStr := statusColor(f.Status)
		pathStr := dimS(truncateStr(f.Path, inner-6))
		add("  " + statusStr + "  " + pathStr)
	}

	return lines
}
```

Note: `statusColor` is in `create_model.go` (same package). It takes a single status letter string. `StashDetailFile.Status` from `git stash show --name-status` returns something like "M", "A", "D", etc. — single letters. This works.

Wait, `stash show --name-status` can output status codes like "M\tpath" where status is "M". Actually it outputs `<status>\t<path>`. The `Fields` call already handles this. First field = status letter.

- [ ] **Step 6: Remove drop-then-reindex logic in `BrowseModel.Update`**

In the browse action handling for drop (lines 168–193), remove the manual re-indexing and `stashes[i].Index` reassignment (lines 173–176). Just remove the entry from the slice:

```go
if m.mode == BrowseModeDrop {
    if m.cursor < len(m.stashes) {
        m.stashes = append(m.stashes[:m.cursor], m.stashes[m.cursor+1:]...)
    }
    if len(m.stashes) == 0 {
        m.actionResult = "all stashes cleared"
        return m, tea.Quit
    }
    // Clamp cursor
    if m.cursor >= len(m.stashes) {
        m.cursor = len(m.stashes) - 1
    }
    m.clampScroll()
    // Reload detail for new cursor
    m.detail = nil
    m.detailLines = nil
    m.detailRef = m.stashes[m.cursor].Ref
    m.loading = true
    return m, tea.Batch(m.spinner.Tick, doLoadBrowseDetail(m.stashes[m.cursor]))
}
```

- [ ] **Step 7: Verify it compiles**

Run: `go build ./...`

- [ ] **Step 8: Commit**

```bash
git add internal/git/git.go internal/ui/stash/browse_model.go
git commit -m "feat(stash): rewrite browse model to use native git stash show; remove stash-type logic"
```

---

### Task 7: Delete unused functions and types from `internal/git/git.go`

**Files:**
- Modify: `internal/git/git.go`

Now that nothing references the old stash functions, clean them up.

- [ ] **Step 1: Delete all unused code**

Delete the following block (lines 646–1059):
- `StashType` type and constants (lines 646–655)
- `stashMeta` struct (lines 657–662)
- `formatStashMsg` function (lines 664–682)
- `parseStashMeta` function (lines 684–715)
- `stripStashPrefix` function (lines 717–721)
- `parseStashDiff` function (lines 876–899)
- `parseStashDiffFiltered` function (lines 901–924)
- `stashStatusCode` function (lines 926–950)
- `stashUntrackedFiles` function (lines 952–966)
- `stashUntrackedFilesFiltered` function (lines 968–982)
- `computeStashStats` function (lines 984–1002)
- `statDiff` function (lines 1004–1023)
- `statDiffFiltered` function (lines 1025–1048)
- `parseNumStat` function (lines 1050–1060)

- [ ] **Step 2: Verify it compiles**

Run: `go build ./...`

- [ ] **Step 3: Check for unused imports**

The `strconv` import was used by `parseNumStat`. Check if `strconv` is used elsewhere in the file:

Run: `rg "strconv" internal/git/git.go`

If not, remove the import. Also check `fmt` import — `fmt.Errorf` and `fmt.Sprintf` are used in `StashPush` and the new functions, so it stays. The `strings` package is used throughout. `strconv` — check.

Actually, looking at the file's imports at the top, let me check what's used.

- [ ] **Step 4: Commit**

```bash
git add internal/git/git.go
git commit -m "refactor(stash): delete ~300 lines of unused stash diffing/metadata code"
```

---

### Task 8: End-to-end testing with `~/projects/pg-test`

**Files:**
- Test directory: `~/projects/pg-test`

- [ ] **Step 1: Build the project**

```bash
cd /home/sai/projects/pretty-git
go build -o /tmp/pgit ./cmd/pretty-git/
```

- [ ] **Step 2: Reset test repo to a clean state**

```bash
cd ~/projects/pg-test
git stash drop stash@{0} 2>/dev/null || true
git reset --hard HEAD
git clean -fd
```

- [ ] **Step 3: Create test changes**

```bash
cd ~/projects/pg-test
# Create a staged change
echo "// staged change" >> a.go
git add a.go

# Create an unstaged change
echo "// unstaged change" >> a.go

# Create an untracked file
echo "untracked" > newfile.txt
```

- [ ] **Step 4: Test quick stash all**

```bash
/tmp/pgit stash "test all stash"
# Verify
git stash list
# Should show: stash@{0}: On master: test all stash
git stash show --stat stash@{0}
# Should show a.go and newfile.txt
```

- [ ] **Step 5: Pop and test staged stash**

```bash
git stash pop stash@{0}
# Now a.go has staged+unstaged changes, newfile.txt is back
# Test staged stash
git add a.go
/tmp/pgit stash --staged "test staged"
# Only staged changes should be stashed, unstaged "// unstaged change" should still be in working tree
git diff a.go  # should show the unstaged line
git stash show --stat stash@{0}  # should show a.go only
```

- [ ] **Step 6: Pop and test unstaged stash**

```bash
git stash pop stash@{0}
# Test unstaged stash
echo "// another unstaged" >> b.go
/tmp/pgit stash --unstaged "test unstaged"
git stash show --stat stash@{0}
```

- [ ] **Step 7: Test custom stash**

```bash
git stash pop stash@{0}
/tmp/pgit stash --custom "test custom" -- newfile.txt
git stash show --stat stash@{0}
# Should show only newfile.txt
```

- [ ] **Step 8: Clean up test stashes**

```bash
git stash drop stash@{0}  # drop the custom stash
git stash drop stash@{0}  # drop the unstaged stash
git reset --hard HEAD
git clean -fd
```

- [ ] **Step 9: Commit any test fixes**

If any issues were found and fixed during testing:

```bash
git add -A
git commit -m "fix(stash): fixes from e2e testing"
```

Otherwise, skip.

---

### Task 9: Verify the full build and check for remaining references

**Files:**
- All

- [ ] **Step 1: Full build check**

```bash
cd /home/sai/projects/pretty-git
go build ./...
```

Expected: Clean build, zero errors.

- [ ] **Step 2: Check no remaining references to deleted types**

```bash
rg "StashType|stashMeta|formatStashMsg|parseStashMeta|stripStashPrefix|GetStashDetailTyped|TargetFiles|parseNumStat" internal/ cmd/
```

Expected: No matches (or only in comments/docs).

- [ ] **Step 3: Run any existing lints**

```bash
rg "lint" Makefile README.md RELEASING.md 2>/dev/null
```
If no lint config, run `go vet ./...`.

- [ ] **Step 4: Commit if any cleanup needed**

```bash
git add -A
git commit -m "chore: final cleanup after stash rewrite"
```
