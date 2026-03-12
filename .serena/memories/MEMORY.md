# pretty-git Memory

## Project
Go inline TUI (fzf-style, no alt-screen). Prettifies `git branch`, `git checkout -b`, `git log`.

## Key Files
- `internal/ui/style.go` — lipgloss color palette + styles
- `internal/ui/branch/model.go` — branch view (1171 lines)
- `internal/ui/branch/keymap.go` — branch keybindings
- `internal/ui/checkout/model.go` — new-branch checkout form
- `internal/ui/log/model.go` — log view with detail pane
- `internal/ui/log/keymap.go` — log keybindings

## Build & Test
- `go build ./...` — build
- `go run ./cmd/pretty-git branch` — run branch view
- Test repo: `~/projects/pg-test`

## Style Conventions
- lipgloss for all styling; AdaptiveColor for light/dark
- Box-drawing chars for trees (├─ └─ │)
- Inline rendering only (no alt-screen)
- Colourful, vibrant, synthwave neon palette
