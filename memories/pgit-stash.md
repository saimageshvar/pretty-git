# pgit stash — WIP notes

Branch: `worktree-stash-feature`

## What's implemented

### Files added
- `cmd/pretty-git/stash.go` — CLI router
- `internal/ui/stash/create_keymap.go`
- `internal/ui/stash/create_model.go` — 3-phase create wizard (type → file picker → message)
- `internal/ui/stash/browse_keymap.go`
- `internal/ui/stash/browse_model.go` — dual-pane stash browser (apply / pop / drop)

### Files modified
- `internal/git/git.go` — FileStatus, StashEntry, StashDetail types + all stash git ops
- `internal/ui/style.go` — ColorWarning, StyleWarning added
- `cmd/pretty-git/main.go` — "stash" case + updated usage

### Routing
```
pgit stash                    → create wizard
pgit stash apply / list       → browse in apply mode
pgit stash pop                → browse in pop mode
pgit stash drop               → browse in drop mode (y/n confirmation)
pgit stash "msg"              → quick stash all
pgit stash --staged "msg"     → quick stash staged only
pgit stash --unstaged "msg"   → quick stash unstaged only
```

## Bugs/tweaks fixed during review

| Issue | Fix |
|-------|-----|
| Each type option had a different bullet icon (●○◉☰) — confusing | Unified to ○ (inactive) / ● (focused) radio style |
| Highlight background bled into empty space past the text | Per-element background application (same pattern as log model); never pass pre-styled ANSI into another `Width().Render()` |
| Message phase had a redundant "default: …" line below input | Removed — placeholder text inside the input already shows the default |
| Message input was dull and hard to see | Wrapped in `lipgloss.RoundedBorder()` box with accent-colored border and label |
| Right pane height in create view grew with number of files | Fixed to `paneRows = 10` with scroll offsets (`previewOffset`, `fileOffset`) |
| Browse view detail pane too small when only 1–2 stashes | `browseMinVisible = 8` floor on `visibleRows` |

## Still needs tweaking (user deferred)

- General visual polish pass — spacing, alignment may need tuning after real use
- The file-select phase (phase 1) could benefit from a right-side diff preview of the focused file
- Drop mode confirmation could be more prominent (currently inline footer `⚠ Drop stash@{N}? y/n`)
- `pgit stash --unstaged "msg"` quick path is untested end-to-end
- Consider adding `pgit stash show <n>` non-TUI shortcut

## Key implementation notes

- `git stash push -m "msg" -- file1 file2` (custom type) does NOT require pre-staging; works on both staged + unstaged portions of those paths
- Messages passed via `exec.Command` (not shell) — spaces, quotes, emojis, special chars all safe
- `StashPush` shadows `msg` param in error block (intentional, not a bug)
- Right-pane detail lines built by `buildBrowseDetailLines` — needs `StashEntry` fields populated from the caller (done via `doLoadBrowseDetail(entry git.StashEntry)`)
