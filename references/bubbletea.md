# Bubble Tea — Complete Reference

**Package:** `github.com/charmbracelet/bubbletea`
**Docs:** https://pkg.go.dev/github.com/charmbracelet/bubbletea
**Source:** https://github.com/charmbracelet/bubbletea

Bubble Tea is a Go framework for building terminal applications based on **The Elm Architecture (TEA)**. State is managed via a model, and the UI is updated by a pure function. It supports inline and full-screen TUIs, mouse input, focus reporting, and framerate-based rendering.

---

## Core Concepts

### The Elm Architecture

Every Bubble Tea program implements the `tea.Model` interface:

```go
type Model interface {
    Init() Cmd
    Update(Msg) (Model, Cmd)
    View() string
}
```

| Method   | Role |
|----------|------|
| `Init`   | Returns the initial `Cmd` to run on startup (use `nil` for nothing) |
| `Update` | Handles incoming `Msg`s, returns updated model + optional `Cmd` |
| `View`   | Renders the entire UI as a plain string |

---

## Minimal Example

```go
package main

import (
    "fmt"
    "os"
    tea "github.com/charmbracelet/bubbletea"
)

type model struct {
    count int
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "up":   m.count++
        case "down": m.count--
        case "q", "ctrl+c": return m, tea.Quit
        }
    }
    return m, nil
}

func (m model) View() string {
    return fmt.Sprintf("Count: %d\n\nPress up/down to change, q to quit.\n", m.count)
}

func main() {
    p := tea.NewProgram(model{})
    if _, err := p.Run(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

---

## Program Options (`tea.NewProgram(model, ...Option)`)

| Option | Description |
|--------|-------------|
| `tea.WithAltScreen()` | Use the alternate screen buffer (full-screen, restored on exit) |
| `tea.WithMouseAllMotion()` | Enable all mouse motion events |
| `tea.WithMouseCellMotion()` | Enable mouse motion only when a button is held |
| `tea.WithoutCatchPanics()` | Don't recover from panics |
| `tea.WithInput(r io.Reader)` | Use a custom reader for input |
| `tea.WithOutput(w io.Writer)` | Use a custom writer for output |
| `tea.WithContext(ctx)` | Use a context for cancellation |
| `tea.WithFilter(fn)` | Intercept/transform messages before they reach Update |
| `tea.WithReportFocus()` | Enable window focus/blur events |
| `tea.WithFPS(fps int)` | Set max render framerate (default: 60) |

### Running

```go
p := tea.NewProgram(initialModel(), tea.WithAltScreen())
finalModel, err := p.Run()
```

`p.Run()` blocks until quit and returns the final model state and any error.

---

## Messages (`tea.Msg`)

Messages are plain Go values. Built-in messages:

| Type | When it's sent |
|------|----------------|
| `tea.KeyMsg` | A key was pressed |
| `tea.MouseMsg` | Mouse event |
| `tea.WindowSizeMsg` | Terminal was resized |
| `tea.FocusMsg` | Window gained focus |
| `tea.BlurMsg` | Window lost focus |
| `tea.QuitMsg` | Quit requested (via `tea.Quit`) |

### KeyMsg

```go
case tea.KeyMsg:
    switch msg.String() {
    case "ctrl+c", "q":
        return m, tea.Quit
    case "up", "k":
        // move up
    case "down", "j":
        // move down
    case "enter", " ":
        // select
    }

    // Or use msg.Type for special keys
    switch msg.Type {
    case tea.KeyEnter:
    case tea.KeyBackspace:
    case tea.KeyCtrlC:
    case tea.KeyEsc:
    case tea.KeyUp, tea.KeyDown, tea.KeyLeft, tea.KeyRight:
    }
```

### MouseMsg

```go
case tea.MouseMsg:
    switch msg.Button {
    case tea.MouseButtonLeft:   // left click
    case tea.MouseButtonRight:  // right click
    case tea.MouseButtonMiddle:
    case tea.MouseButtonWheelUp:
    case tea.MouseButtonWheelDown:
    }
    // msg.X, msg.Y — coordinates
    // msg.Action — tea.MouseActionPress / Release / Motion
```

### WindowSizeMsg

```go
case tea.WindowSizeMsg:
    m.width  = msg.Width
    m.height = msg.Height
```

---

## Commands (`tea.Cmd`)

A `Cmd` is `func() Msg` — a function that runs asynchronously and returns a message.

### Built-in Commands

| Command | Description |
|---------|-------------|
| `tea.Quit` | Signal the program to exit |
| `tea.Batch(cmds...)` | Run multiple commands concurrently |
| `tea.Sequence(cmds...)` | Run commands one after another |
| `tea.Tick(d, fn)` | Fire a message after duration `d` |
| `tea.Every(d, fn)` | Fire messages every duration `d` |
| `tea.Cmd(nil)` | No-op |
| `tea.SetWindowTitle(s)` | Set the terminal window title |
| `tea.EnterAltScreen()` | Switch to alternate screen at runtime |
| `tea.ExitAltScreen()` | Return from alternate screen |
| `tea.EnableMouseAllMotion()` | Enable mouse all-motion at runtime |
| `tea.EnableMouseCellMotion()` | Enable mouse cell-motion at runtime |
| `tea.DisableMouse()` | Disable mouse at runtime |
| `tea.HideCursor()` | Hide the terminal cursor |
| `tea.ShowCursor()` | Show the terminal cursor |
| `tea.Printf(format, args...)` | Print above the TUI (for logging/notifications) |
| `tea.Println(args...)` | Print a line above the TUI |
| `tea.ClearScreen()` | Clear the screen |
| `tea.ClearScrollArea()` | Clear the scroll area |
| `tea.SuspendProcess()` | Suspend (Ctrl+Z) the process |
| `tea.ExecProcess(cmd, fn)` | Run an external process (e.g., editor), restore TUI after |

### Writing Custom Commands

```go
// Simple command
func fetchData() tea.Msg {
    resp, err := http.Get("https://example.com")
    if err != nil {
        return errMsg{err}
    }
    return dataMsg{resp.StatusCode}
}

// Command with arguments (closure pattern)
func fetchURL(url string) tea.Cmd {
    return func() tea.Msg {
        resp, err := http.Get(url)
        if err != nil {
            return errMsg{err}
        }
        return dataMsg{resp.StatusCode}
    }
}

// In Init or Update:
return m, fetchURL("https://example.com")
```

### Batching & Sequencing

```go
return m, tea.Batch(
    fetchData,
    spinner.Tick,
    someOtherCmd,
)

return m, tea.Sequence(
    doFirst,
    doSecond,  // runs after doFirst's message is processed
)
```

---

## Sending Messages Programmatically

Send messages from outside the Bubble Tea event loop:

```go
p := tea.NewProgram(model{})
go func() {
    time.Sleep(2 * time.Second)
    p.Send(myCustomMsg{})
}()
p.Run()
```

---

## Composing Sub-models

A recommended pattern for composable views:

```go
type mainModel struct {
    spinner  spinner.Model
    textarea textarea.Model
    focused  string
}

func (m mainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmds []tea.Cmd
    var cmd tea.Cmd

    m.spinner, cmd = m.spinner.Update(msg)
    cmds = append(cmds, cmd)

    m.textarea, cmd = m.textarea.Update(msg)
    cmds = append(cmds, cmd)

    return m, tea.Batch(cmds...)
}
```

---

## Logging & Debugging

```go
// Log to file (stdout is taken by TUI)
f, err := tea.LogToFile("debug.log", "debug")
if err != nil { log.Fatal(err) }
defer f.Close()
```

```bash
# Watch log in real time
tail -f debug.log
```

### Debugging with Delve

```bash
# Terminal 1
dlv debug --headless --api-version=2 --listen=127.0.0.1:43000 .

# Terminal 2
dlv connect 127.0.0.1:43000
```

---

## Program.Kill() vs Quit

- `p.Quit()` — sends a `QuitMsg`, triggers graceful shutdown
- `p.Kill()` — forcibly stops the program without cleanup

---

## Full-screen vs Inline

```go
// Full-screen (replaces terminal content, restores on exit)
tea.NewProgram(m, tea.WithAltScreen())

// Inline (renders below the cursor, scrolls with terminal)
tea.NewProgram(m)
```

---

## Examples Index

All examples live at `github.com/charmbracelet/bubbletea/tree/main/examples`:

| Example | What it demonstrates |
|---------|---------------------|
| `simple` | Minimal model/update/view |
| `spinner` | Spinner component with tick |
| `textinput` | Single text input |
| `textinputs` | Multiple text inputs with tab focus |
| `textarea` | Multi-line text area |
| `list-default` | Default list component |
| `list-simple` | Minimal list |
| `list-fancy` | Full-featured list with fuzzy filter |
| `table` | Table component |
| `paginator` | Pagination |
| `pager` | Viewport (scrollable pager) |
| `progress-animated` | Animated progress bar |
| `progress-static` | Static progress bar |
| `progress-download` | Download progress bar |
| `timer` | Timer countdown |
| `stopwatch` | Stopwatch |
| `help` | Help view with keybindings |
| `mouse` | Mouse event handling |
| `http` | HTTP request as a command |
| `realtime` | Real-time data from goroutine |
| `composable-views` | Multiple composable views |
| `tabs` | Tab navigation |
| `fullscreen` | Alternate screen toggle |
| `altscreen-toggle` | Toggle between inline and full-screen |
| `suspend` | Process suspend (Ctrl+Z) |
| `exec` | Running external programs |
| `autocomplete` | Text input with autocomplete |
| `file-picker` | File picker component |
| `credit-card-form` | Form with validation |
| `window-size` | Responding to terminal resize |
| `views` | Multiple independent views |
| `set-window-title` | Setting terminal title |
| `send-msg` | Programmatic message sending |
| `pipe` | Reading from stdin pipe |
| `prevent-quit` | Confirmation before quit |
| `debounce` | Debouncing key events |
| `sequence` | Sequential commands |
| `focus-blur` | Focus/blur events |
| `split-editors` | Side-by-side editors |
| `glamour` | Markdown rendering with Glamour |
| `tui-daemon-combo` | Background daemon + TUI |
| `result` | Returning a value from the program |

---

## Key Types Quick Reference

```go
// Key type constants (tea.KeyType)
tea.KeySpace
tea.KeyEnter
tea.KeyBackspace
tea.KeyDelete
tea.KeyEsc
tea.KeyTab
tea.KeyShiftTab
tea.KeyUp / Down / Left / Right
tea.KeyHome / End
tea.KeyPgUp / PgDown
tea.KeyF1 .. tea.KeyF20
tea.KeyCtrlA .. tea.KeyCtrlZ
tea.KeyCtrlSpace
tea.KeyCtrlBackslash
tea.KeyRunes  // printable characters
```

---

## Related Libraries

| Library | Purpose |
|---------|---------|
| [Bubbles](https://github.com/charmbracelet/bubbles) | Ready-made components (spinner, textinput, list, …) |
| [Lip Gloss](https://github.com/charmbracelet/lipgloss) | Styling and layout |
| [Harmonica](https://github.com/charmbracelet/harmonica) | Spring animations |
| [BubbleZone](https://github.com/lrstanley/bubblezone) | Mouse hit zones for components |
| [Glamour](https://github.com/charmbracelet/glamour) | Markdown rendering |
| [Huh](https://github.com/charmbracelet/huh) | Forms and prompts |

---

**License:** MIT
