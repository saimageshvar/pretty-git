# pgit checkout -b — Implementation Summary

## What it does
Inline TUI form for creating a branch with optional parent + description.
If all three are supplied via flags, no TUI opens — branch is created directly.

## Command
```
pgit checkout -b [name] [-p parent] [-d description]
```
- All args optional; TUI opens for any missing field
- All-provided path: creates branch, sets parent/desc, prints summary to stdout

## Key files
- `cmd/pretty-git-revamp/checkout.go` — flag parsing, TUI runner, printCreated
- `internal/ui/checkout/model.go` — Bubble Tea model
- `internal/git/git.go` — `CreateBranch(name)` added

## TUI form — 3 fields, Tab/Shift+Tab to navigate
| Field    | Widget      | Notes                                  |
|----------|-------------|----------------------------------------|
| Branch   | textinput   | required; Enter validates + advances   |
| Parent   | textinput   | type-to-filter; picker opens below     |
| Desc     | textinput   | optional; Enter submits                |

## Parent picker (below layout only)
- Appears below form divider when Parent field is focused
- Branches in DFS tree order (`├─` / `└─`), same as `pgit branch`
- Description uses all remaining line width: `descW = width - indent - prefixLen - nameMax - 2`
- `↑/↓` navigate, `enter` selects & advances to Desc, `ctrl+d` clears, `tab` advances without selecting

## Focus auto-advance
Constructor skips pre-filled fields: name given → start on Parent; name+parent given → start on Desc.
`preselectParent()` scrolls picker to the already-selected branch when re-entering the field.

## Git ops sequence (async tea.Cmd)
`CreateBranch` → `SetParent` (if set) → `SetDescription` (if set) → `createDoneMsg`
Errors at any step surface in the footer; focus returns to Branch field.
