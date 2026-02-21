# pgit branch — Implementation Summary

## What it does
Interactive inline branch switcher (fzf-style, no alt screen).
Arrow navigation, Enter to switch, `/` to filter, `q` to quit.

## Key files
- `cmd/pretty-git-revamp/branch.go` — runner
- `internal/git/git.go` — ListBranches, SwitchBranch, readAllParents, abbreviateRelTime
- `internal/ui/branch/model.go` — Bubble Tea model
- `internal/ui/branch/keymap.go` — key.Binding definitions + help.KeyMap impl
- `internal/ui/style.go` — shared lipgloss styles

## Tree view
Branches with `branch.<name>.pgit-parent` git config are rendered as a hierarchy
using box-drawing characters (`├─`, `└─`, `│`).

- `Branch.Parent` populated via single `git config --local --list` call (O(1))
- `renderItem{branch, treePrefix, depth}` wraps each row
- `buildRenderItems()` — DFS from virtual root, children[""] = roots
- Prefix per depth level = 3 chars (`│  ` or `   ` for trunk, `├─ ` / `└─ ` for connector)
- **Width measurement**: use `lipgloss.Width(treePrefix)` — NOT `len([]rune())`.
  Box-drawing characters are Unicode-ambiguous and may render as 2 columns;
  rune count underestimates, causing columns to drift right per depth level.

## Column layout
```
"  " + marker(1) + sep(2) + prefix + name(32) + sep(2) + status(12) + sep(2) + time
```
- Hash and subject columns removed; replaced by `vs parent` status column
- Dim column header row rendered above branch rows: Branch · vs parent · Last commit
- Filter mode: tree stripped, flat matches with no prefix

## Parent status column
`Branch.ParentAhead/ParentBehind` computed concurrently (one goroutine per branch) via
`git rev-list --left-right --count branch...parent`. Same mechanism as git's own ahead/behind.
- `ParentAhead==0` → `✓ merged` (green); `↑N` (yellow); `↑N ↓M` (yellow+red)
- Empty for branches with no parent or remote branches

## Footer
`footerInfoLines []footerInfoLine` — ordered slice of info line functions, easy to reorder/extend.
Current items: `footerNamePin` (full branch name), `footerParentStatusDesc` (plain-English parent status).

## Timestamps
`abbreviateRelTime()` in git.go: "30 minutes ago" → "30m", "2 hours ago" → "2h", "11 months ago" → "11mo", etc.

## Cursor highlight
Selected row: background fills to terminal edge via:
```go
if w := lipgloss.Width(content); termWidth > w {
    trail = lipgloss.NewStyle().Background(bg).Render(strings.Repeat(" ", termWidth-w))
}
```

## Test repo
`~/projects/pg-test` — 50 branches, 4 levels deep. Root names use dashes (`feat-auth`)
because git cannot have both `feat/auth` and `feat/auth/login` as refs simultaneously.
