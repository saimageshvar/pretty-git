# pretty-git

Prettifies git command output with aesthetic, information-dense TUI views.

## Goals
- Ease of access — fewer commands, more insight
- Aesthetic, colourful, vibrant UI
- Clarity over raw git output

## Language & Tooling
- **Go** — all source code
- **Bubble Tea** — TUI framework (see constraint below)
- **Packaging** — `.deb` for Ubuntu

## Critical TUI Constraint
**Inline rendering only — like fzf. Never use `tea.WithAltScreen()`.**
The program renders inside the terminal's scrollback. Users see output after quitting, history is preserved.

## Project Structure
```
cmd/pretty-git-revamp/     # One file per subcommand (main.go, browse.go, log.go, ...)
internal/git/       # Git operations (git.go)
internal/ui/        # Shared TUI logic (tui.go, render.go, style.go)
```

## Test Repository
`~/projects/pg-test` — a real git repo for testing commands.
Agent may freely create branches and commits there.

For TUI features the agent cannot observe directly, ask the human to verify the output and confirm before proceeding.

## Style Rules
- Colourful, vibrant output — use lipgloss or similar for styling
- Box-drawing characters for trees (`├─`, `└─`, `│`)
- Status markers, icons, and colour-coded branch states
- Keep output scannable — visual hierarchy matters
