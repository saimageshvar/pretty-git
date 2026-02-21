# pgit log — Implementation Summary

## What it does
Inline split-pane TUI for browsing git log. Left pane = commit list; right pane = focused commit detail. No alt screen.

## Command
```
pgit log [<ref>]
```
- `ref` defaults to `HEAD`; any branch/tag/hash works
- Fetches up to 200 commits via `git.ListCommits`

## Key files
- `cmd/pretty-git/log.go` — runner (parses ref, gets terminal size, runs TUI)
- `internal/git/git.go` — `Commit`, `CommitDetail`, `ListCommits`, `GetCommitDetail`
- `internal/ui/log/model.go` — Bubble Tea model
- `internal/ui/log/keymap.go` — key bindings

## Layout
```
  Log  <ref>  <repo>  [N commits]
  ─────────────────────────────────────────────────────────────
  Hash      Subject                  Author          When    │  Detail
  <list rows × visibleRows>                                   │  <detail rows × visibleRows>
  ─────────────────────────────────────────────────────────────
  <help hints>
```
- Left pane ≈ 55% of terminal width; right pane ≈ 45%
- Separator `│` turns accent-colored + bold when detail pane is focused
- Max 15 visible rows (`maxVisible`); adjusts to terminal height

## Left pane columns
| Col     | Width | Notes                                      |
|---------|-------|--------------------------------------------|
| Hash    | 8     | short hash, amber                          |
| Subject | dynamic | `lw - 2 - colHash - colAuthor - colTime - colPad*3` |
| Author  | 16    | dim/slate                                  |
| When    | 13    | relative time, teal                        |

## Right pane (detail)
Built by `buildDetailLines(d, dw)` — all content is **word-wrapped** to `dw-4` chars.
Lines stored in `m.detailLines []string`; rebuilt on window resize and on detail load.

Content order:
1. `commit <shortHash>  …<rest of full hash>`
2. Subject (bold, word-wrapped)
3. Body paragraphs (word-wrapped, if present)
4. Author name + email, Date (relative + absolute)
5. `── changes ──` header
6. Summary: `N files changed, +M, -K` (colored)
7. Per-file diff-stat lines with colored `+`/`-` bar

## Scroll indicators (two locations)
- **Column header** (`detailPaneTitle`): `↓ N more lines — press → to scroll` / `↑N ↓N more`
- **Last visible row** of detail pane: bold italic `↓ N more lines` in amber replaces the row when content extends below; disappears when fully scrolled

## Navigation & pane focus
| Key        | Context      | Action                          |
|------------|-------------|----------------------------------|
| `↑/↓` `k/j` | list focused | navigate commit list            |
| `→` / `l`  | list focused | focus detail pane (if loaded)   |
| `↑/↓` `k/j` | detail focused | scroll detail pane            |
| `←` / `h`  | detail focused | return to list pane            |
| `PgUp/PgDn` `ctrl+u/d` | either | page scroll               |
| `q` `esc` `ctrl+c` | either | quit                       |

Footer help hints update to reflect current pane context.

## Git data layer
### `git.ListCommits(ref string, limit int) ([]Commit, error)`
- `git log <ref> --format=<fmt> -n <limit>`
- Format uses `%x1e` (ASCII Record Separator 0x1e) as field delimiter — NUL-safe for exec args
- Fields per commit (6): `%H %x1e %h %x1e %an %x1e %cr %x1e %s %x1e %b %x1e`
- Parsed by splitting on `\x1e`, trimming leading newlines, grouping in 6s

### `git.GetCommitDetail(hash string) (CommitDetail, error)`
- `git show --no-patch --format=<fmt> <hash>` — same `%x1e` delimiter
- Fields: author, email, absolute date, subject, body, full hash, short hash, reltime
- `git diff-tree --stat --no-commit-id -r <hash>` for per-file stats
- Insertions/deletions parsed from summary line by scanning `fields[pi-1]` before "insertion"/"deletion" tokens

## Spinner
`spinner.MiniDot` (braille ⠋⠙⠹…) shown in detail pane row 0 while async load is in flight.
Detail hash is pre-set in `New()` to ensure the first load's response is accepted (`detailHash = commits[0].Hash`, `loading = true`).
