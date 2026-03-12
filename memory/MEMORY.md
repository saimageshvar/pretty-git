# pretty-git Memory

## Build & Install
After every code change, always build and install the binary as **`pgit_local`**:
```bash
go build -o ~/bin/pgit_local ./cmd/pretty-git
```
`go build ./...` is compile-check only — it does NOT update the invokable binary.
The installed binary lives at `~/bin/pgit_local`. Test in `~/projects/pg-test`. Do **not** build to `pgit`.

## Memory files
Project memory lives in `memories/` (NOT `memory/`) at repo root:
- `memories/pgit-branch.md` — branch UI design decisions
- `memories/pgit-checkout.md` — checkout routing + form
- `memories/pgit-log.md` — log UI
- `memories/mouse-support.md` — future mouse support notes

## Project Structure
- `cmd/pretty-git/` — one file per subcommand: main.go, branch.go, checkout.go, log.go
- `internal/git/git.go` — all git operations
- `internal/ui/branch/` — branch switcher TUI (model.go, keymap.go)
- `internal/ui/checkout/` — new branch form TUI (model.go)
- `internal/ui/log/` — commit log TUI (model.go, keymap.go)
- `internal/ui/style.go` — shared colours and styles

## Key Design Decisions
- Inline rendering only (no `tea.WithAltScreen()`) — fzf-style, scrollback preserved
- `pgit branch` filter is always-on: typing directly filters, no mode switch
- `pgit checkout` routes: no args → branch switcher; `<name>` → switch/create; `-b` → create form

## pgit checkout routing
- `pgit checkout`          → branch switcher UI (same as pgit branch)
- `pgit checkout <name>`   → switch directly if exists; open create form if not
- `pgit checkout -b [name]`→ create-branch form (existing behaviour)
