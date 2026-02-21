# pgit branch — Implementation Summary

## What it does
Interactive inline branch switcher (fzf-style, no alt screen).
Arrow navigation, Enter to switch, `/` to filter, `q` to quit.

## Key files
- `cmd/pretty-git-revamp/branch.go` — runner
- `internal/git/git.go` — ListBranches, SwitchBranch, readAllParents, abbreviateRelTime
- `internal/ui/branch/model.go` — Bubble Tea model
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
"  " + marker(1) + sep(2) + prefix + name + sep(2) + hash(7) + sep(2) + subject(40) + sep(2) + time(5)
```
- `nameColWidth(termWidth) = termWidth - 63` — name gets all remaining space
- `nameW = nameColWidth - lipgloss.Width(treePrefix)` — prefix borrows from name budget
- Filter mode: tree stripped, flat matches with no prefix

## Footer
Two lines: key hints + `▶ <full-branch-name>` of focused item (shows full name even when truncated in list).

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
