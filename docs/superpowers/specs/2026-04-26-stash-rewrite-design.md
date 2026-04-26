# Design: Stash Rewrite — Lean Native Wrapper

## Problem

The current stash implementation is buggy despite multiple iterations:
- Sometimes stashes incorrect files
- Shows wrong stash info in list/details
- Over-engineered: encodes custom `[pgit:...]` metadata into stash messages, then parses it back
- ~300 lines of stash-internal commit diffing (`ref^1`, `ref^2`, `ref^3`) that are fragile and git-version-dependent

The fix is not another patch — it's a rewrite that **leverages native git stash** as much as possible, acting as a pure aesthetic wrapper.

## Core Philosophy

- No custom metadata in stash messages. Clean user messages go straight to `git stash push -m`.
- No type badges in the stash list. The user knows what they stashed.
- `git stash show --stat` and `git stash show` for detail view — git owns correctness.
- All ~300 lines of stash-internal diffing code deleted from `internal/git/git.go`.

## Architecture

### What Stays

- 3-phase TUI wizard for creation (type selection → message → execute)
- Dual-pane browse UI (list + detail)
- Bubble Tea model pattern
- `StashApply`, `StashPop`, `StashDrop` wrappers (already clean)
- Keymap files (`create_keymap.go`, `browse_keymap.go`) — mostly unchanged
- CLI router structure in `cmd/pretty-git/stash.go`

### What Changes

| File | Action |
|------|--------|
| `cmd/pretty-git/stash.go` | Rewrite: simpler routing, no metadata concerns |
| `internal/ui/stash/create_model.go` | Rewrite: simpler wizard, MM-aware staging option |
| `internal/ui/stash/browse_model.go` | Rewrite: native `git stash show` for detail |
| `internal/git/git.go` | Delete ~300 lines; add `StashShowFiles()`, `StashShow()`; simplify `StashPush()`, `ListStashes()` |

### Deleted from `internal/git/git.go`

- `StashType` enum, `stashMeta` struct
- `formatStashMsg()`, `parseStashMeta()`, `stripStashPrefix()`
- `GetStashDetailTyped()`, `parseStashDiff()`, `parseStashDiffFiltered()`
- `stashUntrackedFiles()`, `stashUntrackedFilesFiltered()`
- `computeStashStats()`, `statDiff()`, `statDiffFiltered()`, `stashStatusCode()`
- `parseNumStat()` (if no other callers remain)

### Added to `internal/git/git.go`

```go
// StashShowFiles returns files changed in a stash via `git stash show --name-status <ref>`.
func StashShowFiles(ref string) ([]StashDetailFile, error)

// StashShowSummary returns total insertions/deletions count via `git stash show <ref>`.
func StashShowSummary(ref string) (files, insertions, deletions int, error)
```

### Rewritten in `internal/git/git.go`

```go
// StashPush runs the appropriate `git stash push` command. No metadata encoding.
// stashType: "all", "staged", "unstaged", "custom"
func StashPush(msg, stashType string, customFiles []string) (string, error)

// ListStashes returns stash entries. No metadata parsing. No StashType field.
func ListStashes() ([]StashEntry, error)
```

### Simplified Types

```go
type StashEntry struct {
    Index   int
    Ref     string    // "stash@{0}"
    Message string    // clean user message
    Branch  string
    RelTime string
}

type StashDetailFile struct {
    Status string // "M", "A", "D", … from git diff --name-status
    Path   string
}

type StashDetail struct {
    StashEntry
    FilesChanged int
    Insertions   int
    Deletions    int
    Files        []StashDetailFile
}
```

## Stash Creation Flow

### Quick Stash (non-TUI)

Fully testable without TUI — each maps 1:1 to a `git stash push` variant:

```
pgit stash "my message"              → git stash push --include-untracked -m "my message"
pgit stash --staged "my message"     → git stash push --staged -m "my message"
pgit stash --unstaged "my message"   → git stash push --keep-index -m "my message"
pgit stash --custom "msg" -- f1 f2   → git stash push --include-untracked -m "msg" -- f1 f2
```

### TUI Wizard (`pgit stash` with no args)

**Phase 0 — Type selection:**

```
  ▼ All changes            3 files  │  files to stash  3 total
  ○ Unstaged changes       2 files  │  ──────────────────────────
  ○ Pick specific files…            │    M  src/a.go
  ○ Staged changes         1 file   │     M  src/b.go
                                    │     ?? README.md
↳ Staged disabled: 2 files have changes in both staged and unstaged areas
  (src/foo.go, src/bar.go). Use All changes or Unstaged changes instead.
```

Options:
- **All changes**: staged + unstaged + untracked
- **Unstaged changes**: working tree only (uses `--keep-index`)
- **Pick specific files…**: checkbox list, all unselected by default
- **Staged changes**: index only (uses `--staged`). **Disabled with warning when MM files exist.** MM files are those where `git status --porcelain` shows different non-`?` characters in both X and Y positions (e.g., `MM`, `AM`, `MD`).

**Phase 1 — Message input:**

Same as current: textinput with rounded border, file summary above, placeholder from last commit message.

**Phase 2 — Execute:**

Spinner + `git stash push` with appropriate flags. Quit on success; back to message phase on error.

### Custom File Picker

- All files unselected by default (counter: `0 of 5 selected`)
- `a` key → select all
- `n` key → deselect all
- `space` → toggle individual file
- Enter with 0 selected → ignored (nothing happens)

## Stash List & Detail (Browse)

### List

`git stash list --format="%gd|%gs|%cr"` — same format string, but parsing no longer extracts stash type metadata. Branch extraction from the `On <branch>: <msg>` prefix works the same.

### Detail Pane

Two native git calls replace all custom diffing:

```bash
git stash show stash@{0}                    # one-line summary line
git stash show --name-status stash@{0}      # file list with status letters (M, A, D)
```

`--name-status` output format (clean, tab-separated, no ambiguity with spaces in paths):

```
M	src/auth/login.go
A	src/auth/test_helper.go
D	src/auth/old_file.go
```

For insertions/deletions count, parse the `git stash show` summary line:

```
 src/auth/login.go | 5 +++--
```
→ Extract `3 files changed, +12, -8` from the trailing summary.

Parsing is straightforward:
- `--name-status` gives status + path on each line (tab-separated, reliable)
- `git stash show` summary gives total insertions/deletions count

### Detail View Rendering

```
› stash@{0}
  fix login bug

  Branch  feature/auth
  Date    2 hours ago

  3 files changed, +12, −8

  ── changes ──────────────────────────
  M  src/auth/login.go
  M  src/auth/session.go
  A  src/auth/test_helper.go
```

No type badge. No `[pgit:...]` prefix. Clean.

### Actions

- Enter → apply/pop (immediate for apply; confirmation modal for pop/drop)
- Stash conflicts: git native behavior. Clean files applied, conflicted files left with `<<<<<<<` markers. Pop does NOT drop on conflict. Error shown in browse footer.
- After drop: remove entry from list, refresh from git

## Edge Cases

| Case | Behavior |
|------|----------|
| Empty working tree | Print "nothing to stash" to stderr, exit 0 |
| No stashes exist | Print "no stashes found" to stderr, exit 0 |
| Stash conflict on apply/pop | Show git's error in browse footer, stash stays |
| Custom with 0 selected | Enter ignored, no stash created |
| Unknown stashes (vanilla git) | Shown normally, `git stash show` works regardless |
| MM files (mixed staged+unstaged) | "Staged changes" option disabled with warning text |
| User picks staged but stash fails | Show error in message phase, let user retry |

## Testing

### Test Repo

`~/projects/pg-test` — a small Go project with known state. Current state: 2 staged files (b.go, c.go), 1 existing stash.

### Test Commands (no TUI needed for quick stash)

```bash
# In ~/projects/pg-test:

# Quick stash all
pgit stash "test all stash"
git stash list          # verify it appears
git stash show stash@{0}  # verify contents

# Quick stash staged
pgit stash --staged "test staged"
git stash show stash@{0}

# Quick stash unstaged
pgit stash --unstaged "test unstaged"

# Quick stash custom
pgit stash --custom "test custom" -- b.go c.go

# Apply/pop (TUI required here)
pgit stash apply        # TUI opens, pick stash
pgit stash pop          # TUI opens, pick stash

# Clean up
git stash drop stash@{0}
```

### What Makes This Testable

- No custom metadata to verify — just check `git stash show` output
- Every operation maps 1:1 to a `git stash` subcommand
- No state to get out of sync between pgit and git
- Quick stash paths work with zero TUI interaction
