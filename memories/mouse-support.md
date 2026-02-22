# Mouse Support — Inline Mode Without Bubblezone

## Why Bubblezone Fails Here

Bubblezone assumes content starts at screen coordinate (0,0) — i.e. altscreen. In pgit's
inline mode the TUI renders at whatever Y the terminal cursor is at when the program starts
(e.g. line 35 of a 40-line terminal). Mouse events report absolute screen coordinates, so
every click is offset by the TUI's start row. The offset is unknown to the program.

WIP attempt is on branch `wip/mouse-bubblezone` — do not merge.

---

## Approach: Manual Coordinate Math

### Core Idea

Since pgit's layouts are fully deterministic (fixed row assignments), we can map any click
to an element purely from:

```
relY = msg.Y - m.startRow
relX = msg.X
```

Where `m.startRow` is captured once at program start.

### Getting startRow

Query the terminal cursor row **before** calling `tea.NewProgram`:

```go
// ANSI DSR (Device Status Report) — ask terminal for cursor position
fmt.Fprint(os.Stderr, "\033[6n")
// Terminal responds with \033[<row>;<col>R — parse it
startRow, _ = readCursorPos(os.Stderr)
```

Pass `startRow` into each model's `New()`. Store as `m.startRow int`.

Alternatively, use `golang.org/x/term` raw mode temporarily to read the response.

### Row Maps (per command)

Build a fixed row→element map from the model's known layout. Example for **pgit log**:

```
relY 0 → header
relY 1 → divider
relY 2 → filter bar  (check relX for checkbox 0 vs 1)
relY 3 → column headers
relY 4..4+visibleRows-1 → commit row (relY - 4 + offset = absolute commit index)
relY > 4+visibleRows → detail/footer area
```

For **relX** in filter bar: checkbox 0 starts at col ~2, checkbox 1 at ~2+len("My commits")+4.
Measure widths with `lipgloss.Width()` at render time and store the breakpoints.

### Storing Column Breakpoints

Add to Model:

```go
// Set during View(), read during mouse handling
filterCheckbox0X int  // start col of "My commits" checkbox
filterCheckbox1X int  // start col of "Skip merges" checkbox
listDividerX     int  // X of the │ divider (= listWidth() + 1)
```

Populate these in `View()` (or a shared `layout()` method) so the mouse handler has
exact column positions without re-computing.

### Handling the Mouse Msg

```go
case tea.MouseMsg:
    if msg.Action != tea.MouseActionRelease && msg.Button != tea.MouseButtonWheelUp ... {
        return m, nil
    }
    relY := msg.Y - m.startRow
    relX := msg.X
    return m.handleMouse(relY, relX, msg)
```

### Wheel Scroll

Wheel events don't need startRow precision — just use `msg.X` vs `m.listDividerX`:

```go
case tea.MouseButtonWheelUp, tea.MouseButtonWheelDown:
    if msg.X < m.listDividerX {
        // scroll commit list
    } else {
        // scroll detail pane
    }
```

This already works without any zone tracking. The `wip/` branch has this part right.

### Double-Click Detection

Keep the existing approach — it doesn't depend on bubblezone:

```go
lastClickTime time.Time
lastClickZone string  // e.g. "row:7", "checkbox:0"
```

---

## Per-Command Layout Constants

### pgit log

| relY | Element |
|---|---|
| 0 | header |
| 1 | divider |
| 2 | filter bar |
| 3 | column headers |
| 4 … 4+vis-1 | commit rows |
| rest | below list (footer area) |

Filter bar X breakpoints: measure `"  " + checkbox0_rendered` width after rendering.

### pgit branch

| relY | Element |
|---|---|
| 0 | header |
| 1 | divider |
| 2 | column headers |
| 3 … 3+vis-1 | branch rows |
| 3+vis | divider |
| 3+vis+1 | filter prompt (`filter: [input]`) |
| 3+vis+2 | key hints |
| 3+vis+3 | divider (only if info lines exist) |
| 3+vis+4 … | info lines: branch name, desc, parent status |

No filter mode — filter is always-on. Footer layout is always the same structure.

### pgit checkout

| relY | Element |
|---|---|
| 0 | header |
| 1 | divider |
| 2 | name field |
| 3 | parent field |
| 4 | desc field |
| 5 | picker divider (if parent focused) |
| 6 … 6+pickerRows-1 | picker rows |
| next | footer divider |
| last | footer |

---

## Implementation Plan (when revisiting)

1. Write a `readCursorPos()` helper using ANSI DSR (`\033[6n`) — reads from stderr in raw
   mode, parses `\033[row;colR`. ~30 lines of Go.
2. Add `startRow int` to all three models, thread it through `New()`.
3. Add layout breakpoint fields (`filterCheckbox0X`, `listDividerX`, etc.) populated in
   `View()`.
4. Replace the bubblezone `case tea.MouseMsg:` handlers with the relY/relX approach above.
5. Remove the `zone *zone.Manager` field and all `zone.Mark()` / `zone.Scan()` calls.
6. Remove the `github.com/lrstanley/bubblezone` dependency entirely.

Wheel scrolling is the easiest win and needs no startRow — do that first as a standalone
PR before tackling click support.

---

## Risks / Unknowns

- **startRow accuracy**: if the terminal scrolls between program start and a click, relY
  shifts. Mitigate by capping relY to [0, totalRenderedLines) and ignoring out-of-range.
- **Terminal support**: DSR is widely supported (xterm, kitty, alacritty, wezterm, tmux).
  Fallback: if DSR times out, disable click support but keep wheel scroll.
- **Resize**: on `tea.WindowSizeMsg`, layout row counts may change. Since we recompute
  layout constants in `View()` this is handled automatically.
