# Charmbracelet Go Packages — Reference Index

This folder contains comprehensive offline reference docs for the Charmbracelet Go TUI packages.

| File | Package | Import Path |
|------|---------|-------------|
| [bubbletea.md](bubbletea.md) | Bubble Tea | `github.com/charmbracelet/bubbletea` |
| [bubbles.md](bubbles.md) | Bubbles | `github.com/charmbracelet/bubbles` |
| [lipgloss.md](lipgloss.md) | Lip Gloss | `github.com/charmbracelet/lipgloss` |
| [glow.md](glow.md) | Glow | `github.com/charmbracelet/glow` |

---

## Quick Summary

### Bubble Tea (`bubbletea.md`)
The TUI framework. Implements **The Elm Architecture** (Model / Update / View). Covers:
- The `tea.Model` interface and all 3 methods
- `tea.NewProgram` options (alt screen, mouse, FPS, context, filter…)
- All built-in message types (`KeyMsg`, `MouseMsg`, `WindowSizeMsg`, `FocusMsg`…)
- All built-in commands (`Quit`, `Batch`, `Sequence`, `Tick`, `ExecProcess`…)
- Writing custom async commands
- Programmatic `p.Send(msg)` from goroutines
- Composing sub-models
- Logging and debugging (Delve, LogToFile)
- Full examples index (40+ examples)

### Bubbles (`bubbles.md`)
Ready-made components. Covers all packages:
- **spinner** — all 12 built-in spinners + custom, options
- **textinput** — all fields, keybindings, autocomplete, echo modes
- **textarea** — multi-line editor, fields, methods
- **table** — column definitions, row navigation, styles
- **list** — Item interface, delegate, filtering, status messages
- **progress** — animated/static, gradient/solid, custom fill chars
- **viewport** — scrolling, high-performance mode, all methods
- **paginator** — Dots/Arabic style, `GetSliceBounds`
- **timer** and **stopwatch** — full API
- **help** — `ShortHelp` / `FullHelp` interface
- **key** — `NewBinding`, `Matches`, `WithHelp`, `WithKeys`
- **filepicker** — extension filter, directory/file selection
- **cursor** — blink modes, styles

### Lip Gloss (`lipgloss.md`)
Styling and layout. Covers:
- All 4 color types (ANSI, ANSI256, TrueColor, Adaptive, Complete, CompleteAdaptive)
- All style properties: text formatting, colors, padding, margin, width/height, alignment, borders, tab width
- All border styles (Normal, Rounded, Thick, Double, Block, ASCII, Markdown, custom)
- `Render`, `Inherit`, `Unset*`, `Get*`
- Layout utilities: `JoinHorizontal`, `JoinVertical`, `Place*`, `Width`, `Height`
- Custom renderers for multi-output (SSH etc.)
- `lipgloss/table` — full table API with StyleFunc
- `lipgloss/list` — enumerators, nesting, custom enumerators
- `lipgloss/tree` — tree rendering, enumerators, styling

### Glow (`glow.md`)
Markdown reader CLI/TUI. Covers:
- Installation on all platforms
- CLI flags (`--style`, `--width`, `--pager`, `--all`)
- TUI and pager keybindings
- Full `glow.yml` config reference
- Custom JSON stylesheets
- Using Glamour for programmatic rendering in Go
