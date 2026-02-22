# Bubbles — Complete Reference

**Package:** `github.com/charmbracelet/bubbles`
**Docs:** https://pkg.go.dev/github.com/charmbracelet/bubbles
**Source:** https://github.com/charmbracelet/bubbles

Bubbles is a collection of ready-made TUI components for Bubble Tea applications. Each component follows the same pattern: it has a `Model`, an `Update(tea.Msg) (Model, tea.Cmd)` method, and a `View() string` method.

---

## General Usage Pattern

Every bubble component is integrated the same way:

```go
type appModel struct {
    input textinput.Model
    // ... other components
}

func (m appModel) Init() tea.Cmd {
    return m.input.Focus() // or textinput.Blink, spinner.Tick, etc.
}

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmd tea.Cmd
    m.input, cmd = m.input.Update(msg)
    return m, cmd
}

func (m appModel) View() string {
    return m.input.View()
}
```

---

## Spinner

**Import:** `github.com/charmbracelet/bubbles/spinner`

Indicates ongoing operations with an animated spinner.

### Built-in Spinners

| Constant | Appearance | FPS |
|----------|-----------|-----|
| `spinner.Line` | `\|`, `/`, `-`, `\` | 10 |
| `spinner.Dot` | Braille dots | 10 |
| `spinner.MiniDot` | Braille mini-dots | 12 |
| `spinner.Jump` | Braille jumping | 10 |
| `spinner.Pulse` | `█`, `▓`, `▒`, `░` | 8 |
| `spinner.Points` | `∙∙∙`, `●∙∙`, `∙●∙`, `∙∙●` | 7 |
| `spinner.Globe` | 🌍🌎🌏 | 4 |
| `spinner.Moon` | 🌑..🌘 | 8 |
| `spinner.Monkey` | 🙈🙉🙊 | 3 |
| `spinner.Meter` | `▱▱▱`, `▰▱▱`, … | 7 |
| `spinner.Hamburger` | `☱`, `☲`, `☴` | 3 |
| `spinner.Ellipsis` | `.`, `..`, `...` | 3 |

### Custom Spinner

```go
custom := spinner.Spinner{
    Frames: []string{"◐ ", "◓ ", "◑ ", "◒ "},
    FPS:    time.Second / 10,
}
```

### Model Fields

| Field | Type | Description |
|-------|------|-------------|
| `Spinner` | `spinner.Spinner` | The spinner type to use |
| `Style` | `lipgloss.Style` | Style applied to spinner frame |

### Usage

```go
s := spinner.New()
s.Spinner = spinner.Dot
s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

// Init: start ticking
func (m model) Init() tea.Cmd {
    return m.spinner.Tick
}

// Update: pass messages through
m.spinner, cmd = m.spinner.Update(msg)

// View:
m.spinner.View() + " Loading..."
```

### Constructor Options

```go
s := spinner.New(
    spinner.WithSpinner(spinner.Dot),
    spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("63"))),
)
```

---

## Text Input

**Import:** `github.com/charmbracelet/bubbles/textinput`

Single-line text input with cursor, scrolling, placeholder, validation, and autocomplete.

### Model Fields

| Field | Type | Description |
|-------|------|-------------|
| `Prompt` | `string` | Prefix before the input (default: `"> "`) |
| `Placeholder` | `string` | Shown when empty |
| `EchoMode` | `EchoMode` | Display mode (Normal/Password/None) |
| `EchoCharacter` | `rune` | Character for password masking (default: `*`) |
| `CharLimit` | `int` | Max characters (0 = unlimited) |
| `Width` | `int` | Visible width (0 = unlimited, enables scrolling when set) |
| `Cursor` | `cursor.Model` | Cursor style/behavior |
| `PromptStyle` | `lipgloss.Style` | Style for the prompt |
| `TextStyle` | `lipgloss.Style` | Style for input text |
| `PlaceholderStyle` | `lipgloss.Style` | Style for placeholder text |
| `CompletionStyle` | `lipgloss.Style` | Style for suggestion completion |
| `Validate` | `func(string) error` | Validation function; sets `Err` field |
| `ShowSuggestions` | `bool` | Enable autocomplete suggestions |
| `KeyMap` | `KeyMap` | Configurable keybindings |

### EchoMode Constants

```go
textinput.EchoNormal   // show text as-is (default)
textinput.EchoPassword // show EchoCharacter mask
textinput.EchoNone     // show nothing
```

### Key Methods

```go
ti := textinput.New()

// Focus/Blur
cmd := ti.Focus()   // returns blink command
ti.Blur()

// Value
ti.SetValue("hello")
val := ti.Value()        // returns string
pos := ti.Position()     // cursor position (int)

// Cursor movement
ti.SetCursor(pos)
ti.CursorStart()
ti.CursorEnd()

// State
ti.Focused()             // bool
ti.Reset()               // clear value

// Autocomplete
ti.ShowSuggestions = true
ti.SetSuggestions([]string{"apple", "banana", "cherry"})
ti.AvailableSuggestions()   // []string
ti.MatchedSuggestions()     // []string
ti.CurrentSuggestion()      // string
ti.CurrentSuggestionIndex() // int

// Paste from clipboard command
return m, textinput.Paste
```

### Default Keybindings

| Action | Keys |
|--------|------|
| Move char forward | `→`, `Ctrl+F` |
| Move char backward | `←`, `Ctrl+B` |
| Move word forward | `Alt+→`, `Ctrl+→`, `Alt+F` |
| Move word backward | `Alt+←`, `Ctrl+←`, `Alt+B` |
| Delete word backward | `Alt+Backspace`, `Ctrl+W` |
| Delete word forward | `Alt+Delete`, `Alt+D` |
| Delete after cursor | `Ctrl+K` |
| Delete before cursor | `Ctrl+U` |
| Delete char backward | `Backspace`, `Ctrl+H` |
| Delete char forward | `Delete`, `Ctrl+D` |
| Go to line start | `Home`, `Ctrl+A` |
| Go to line end | `End`, `Ctrl+E` |
| Paste | `Ctrl+V` |
| Accept suggestion | `Tab` |
| Next suggestion | `↓`, `Ctrl+N` |
| Prev suggestion | `↑`, `Ctrl+P` |

### Example

```go
ti := textinput.New()
ti.Placeholder = "Enter name"
ti.CharLimit = 50
ti.Width = 30
ti.EchoMode = textinput.EchoPassword // for password fields
ti.Validate = func(s string) error {
    if len(s) > 20 { return errors.New("too long") }
    return nil
}
cmd := ti.Focus()
```

---

## Text Area

**Import:** `github.com/charmbracelet/bubbles/textarea`

Multi-line text editor with vertical scrolling, configurable size, and line numbers.

### Key Model Fields

| Field | Type | Description |
|-------|------|-------------|
| `Prompt` | `string` | Line prefix (default: `"│ "`) |
| `Placeholder` | `string` | Placeholder text when empty |
| `ShowLineNumbers` | `bool` | Display line numbers |
| `EndOfBufferCharacter` | `rune` | Character shown after last line |
| `KeyMap` | `KeyMap` | Customisable keybindings |
| `FocusedStyle` | `textarea.Style` | Styles when focused |
| `BlurredStyle` | `textarea.Style` | Styles when blurred |
| `CharLimit` | `int` | Max characters (0 = unlimited) |
| `MaxHeight` | `int` | Max height in lines |
| `MaxWidth` | `int` | Max width in characters |

### Key Methods

```go
ta := textarea.New()
ta.SetWidth(60)
ta.SetHeight(10)

cmd := ta.Focus()
ta.Blur()
ta.Focused() // bool

ta.SetValue("initial text")
val := ta.Value()   // full text as string
ta.Reset()

ta.InsertRune('a')
ta.InsertString("hello")

// Cursor position
line, col := ta.Line(), ta.LineOffset() // 0-indexed
ta.CursorDown() / CursorUp() / etc.

// Init
func (m model) Init() tea.Cmd { return textarea.Blink }
```

---

## Table

**Import:** `github.com/charmbracelet/bubbles/table`

Scrollable, navigable table for tabular data.

### Column Definition

```go
columns := []table.Column{
    {Title: "Name",   Width: 20},
    {Title: "Email",  Width: 30},
    {Title: "Status", Width: 10},
}
```

### Row Type

```go
type Row []string

rows := []table.Row{
    {"Alice", "alice@example.com", "active"},
    {"Bob",   "bob@example.com",   "inactive"},
}
```

### Constructor & Options

```go
t := table.New(
    table.WithColumns(columns),
    table.WithRows(rows),
    table.WithFocused(true),
    table.WithHeight(10),
)

// Style
s := table.DefaultStyles()
s.Header = s.Header.
    BorderStyle(lipgloss.NormalBorder()).
    BorderForeground(lipgloss.Color("240")).
    BorderBottom(true).
    Bold(false)
s.Selected = s.Selected.
    Foreground(lipgloss.Color("229")).
    Background(lipgloss.Color("57")).
    Bold(false)
t.SetStyles(s)
```

### Key Methods

```go
t.SelectedRow()       // table.Row — currently selected row
t.Cursor()            // int — row index of cursor
t.SetRows(rows)       // update rows
t.SetColumns(cols)    // update columns
t.Focus()             // enable keyboard navigation
t.Blur()              // disable keyboard navigation
t.GotoTop()           // jump to first row
t.GotoBottom()        // jump to last row
t.MoveUp(n) / MoveDown(n)  // move cursor
```

### Default Keybindings

| Action | Keys |
|--------|------|
| Move up | `↑`, `k` |
| Move down | `↓`, `j` |
| Go to top | `Home`, `g` |
| Go to bottom | `End`, `G` |
| Page up | `PgUp`, `b` |
| Page down | `PgDn`, `f` |

---

## List

**Import:** `github.com/charmbracelet/bubbles/list`

Fully-featured list with pagination, fuzzy filtering, spinner, help, and status messages.

### Item Interface

```go
type Item interface {
    FilterValue() string  // string used for fuzzy filtering
}

// Implement your own item:
type myItem struct {
    title, desc string
}
func (i myItem) FilterValue() string { return i.title }

// Implement ItemDelegate for custom rendering:
type myDelegate struct{}
func (d myDelegate) Height() int                             { return 1 }
func (d myDelegate) Spacing() int                            { return 0 }
func (d myDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d myDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
    i, _ := listItem.(myItem)
    fmt.Fprintf(w, "%s", i.title)
}
```

### Constructor

```go
items := []list.Item{myItem{"foo", "bar"}, myItem{"baz", "qux"}}
l := list.New(items, list.NewDefaultDelegate(), 80, 20) // width, height

// Or with custom delegate:
l := list.New(items, myDelegate{}, 80, 20)
```

### Key Model Fields / Methods

| Field/Method | Description |
|--------------|-------------|
| `l.SetItems(items)` | Replace all items |
| `l.InsertItem(index, item)` | Insert at position |
| `l.RemoveItem(index)` | Remove item at index |
| `l.SetItem(index, item)` | Replace item at index |
| `l.Items()` | Get all items |
| `l.VisibleItems()` | Filtered visible items |
| `l.SelectedItem()` | Currently selected item |
| `l.Index()` | Index of selected item |
| `l.SetWidth(w) / SetHeight(h)` | Resize |
| `l.SetFilteringEnabled(bool)` | Enable/disable fuzzy filter |
| `l.SetShowFilter(bool)` | Show/hide filter input |
| `l.SetShowStatusBar(bool)` | Show/hide status bar |
| `l.SetShowPagination(bool)` | Show/hide pagination |
| `l.SetShowHelp(bool)` | Show/hide help |
| `l.SetShowTitle(bool)` | Show/hide title |
| `l.Title` | String title |
| `l.Styles` | `list.Styles` — comprehensive style object |
| `l.KeyMap` | `list.KeyMap` — all keybindings |
| `l.SetDelegate(d)` | Change item delegate |
| `l.NewStatusMessage(s)` | Show a status message |
| `l.StartSpinner()` / `StopSpinner()` | Control spinner |
| `l.Paginator` | `paginator.Model` — direct access |
| `l.FilterState()` | `list.FilterState` — Unfiltered/Filtering/FilterApplied |

### Default Delegate Methods (list.DefaultDelegate)

```go
d := list.NewDefaultDelegate()
d.ShowDescription = true
d.Styles.NormalTitle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
d.Styles.NormalDesc  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
d.Styles.SelectedTitle = /* ... */
d.Styles.SelectedDesc  = /* ... */
d.SetHeight(2)     // title + desc = 2 lines
d.SetSpacing(1)    // gap between items
```

`list.DefaultItem` implements both title and description:

```go
type myItem string
func (i myItem) Title() string       { return string(i) }
func (i myItem) Description() string { return "some desc" }
func (i myItem) FilterValue() string { return string(i) }
```

---

## Progress Bar

**Import:** `github.com/charmbracelet/bubbles/progress`

Simple animated or static progress indicator.

### Constructor Options

```go
p := progress.New(
    progress.WithDefaultGradient(),        // green→blue gradient
    progress.WithGradient("#ff7f7f", "#7fff7f"), // custom gradient
    progress.WithSolidFill("#ff0000"),     // solid color
    progress.WithoutPercentage(),          // hide percentage
    progress.WithWidth(40),               // explicit width
    progress.WithScaledGradient(),        // gradient scales with value
    progress.WithColorProfile(termenv.TrueColor),
)
```

### Usage

```go
p := progress.New(progress.WithDefaultGradient())
p.Width = 60  // can also set directly

// Static update (no animation):
view := p.ViewAs(0.75)  // 0.0 to 1.0

// Animated update — send a FrameMsg:
cmd := p.SetPercent(0.75)   // returns tea.Cmd for animation
// Handle in Update:
case progress.FrameMsg:
    pm, cmd := m.progress.Update(msg)
    m.progress = pm.(progress.Model)
    return m, cmd
```

### Model Fields

| Field | Type | Description |
|-------|------|-------------|
| `Width` | `int` | Width in terminal columns |
| `Full` | `rune` | Filled rune (default: `█`) |
| `Empty` | `rune` | Empty rune (default: `░`) |
| `FullColor` | `string` | Color for filled portion |
| `EmptyColor` | `string` | Color for empty portion |
| `ShowPercentage` | `bool` | Show percentage readout |
| `PercentFormat` | `string` | `fmt` format string for percentage |
| `PercentageStyle` | `lipgloss.Style` | Style for percentage text |
| `UseRounding` | `bool` | Round percentage |
| `Spring` | (harmonica) | Animation spring settings |

---

## Viewport

**Import:** `github.com/charmbracelet/bubbles/viewport`

Vertically scrollable content viewport. Ideal for pagers and log viewers.

### Constructor

```go
vp := viewport.New(width, height)
vp.SetContent(longString)

// High performance mode (for alternate screen):
vp := viewport.New(width, height)
vp.HighPerformanceRendering = true
```

### Key Methods

```go
vp.SetContent(s string)       // set new content
vp.Width, vp.Height           // dimensions

// Scrolling
vp.ScrollPercent()            // float64: 0.0–1.0
vp.SetYOffset(offset int)     // scroll to offset
vp.GotoTop()
vp.GotoBottom()
vp.HalfViewUp()
vp.HalfViewDown()
vp.LineUp(n) / LineDown(n)

// In Update: pass messages
vp, cmd = vp.Update(msg)

// Scroll position info
vp.AtTop()        // bool
vp.AtBottom()     // bool
vp.PastBottom()   // bool
vp.ScrollPercent() // float64
```

### Default Keybindings

| Action | Keys |
|--------|------|
| Up 1 line | `↑`, `k` |
| Down 1 line | `↓`, `j` |
| Up half page | `Ctrl+U`, `u` |
| Down half page | `Ctrl+D`, `d` |
| Page up | `PgUp`, `b` |
| Page down | `PgDn`, `f`, `Space` |
| Go to top | `Home`, `g` |
| Go to bottom | `End`, `G` |

---

## Paginator

**Import:** `github.com/charmbracelet/bubbles/paginator`

Handles pagination state and optional rendering.

### Types

```go
paginator.Dots    // ● ● ○ ○ dot-style (iOS-like)
paginator.Arabic  // "1/5" numeric
```

### Usage

```go
p := paginator.New()
p.Type = paginator.Dots
p.PerPage = 10
p.SetTotalPages(len(items))

// Navigation
p.NextPage()
p.PrevPage()
p.Page          // current page (0-indexed)
p.TotalPages()
p.ItemsOnPage(total int) // items on current page
p.GetSliceBounds(total int) // (start, end int) for slicing items
p.OnLastPage() // bool
p.View()       // render the paginator
```

---

## Timer

**Import:** `github.com/charmbracelet/bubbles/timer`

Countdown timer.

```go
t := timer.NewWithInterval(5*time.Minute, time.Second)
// or: timer.New(5 * time.Minute) // default 1s interval

// Init
func (m model) Init() tea.Cmd { return m.timer.Init() }

// Update
case timer.TickMsg:
    var cmd tea.Cmd
    m.timer, cmd = m.timer.Update(msg)
    if m.timer.Timedout() {
        // handle timeout
    }
    return m, cmd
case timer.StartStopMsg:
    var cmd tea.Cmd
    m.timer, cmd = m.timer.Update(msg)
    return m, cmd

// Control
cmd := m.timer.Toggle()  // start/stop
cmd  = m.timer.Start()
cmd  = m.timer.Stop()
m.timer.Timedout()       // bool
m.timer.Running()        // bool
m.timer.Timeout         // time.Duration — time remaining
m.timer.View()          // render
```

---

## Stopwatch

**Import:** `github.com/charmbracelet/bubbles/stopwatch`

Count-up timer.

```go
sw := stopwatch.New()
// or: stopwatch.NewWithInterval(time.Millisecond * 100)

// Init
func (m model) Init() tea.Cmd { return m.sw.Init() }

// Update
case stopwatch.TickMsg:
    m.sw, cmd = m.sw.Update(msg)
case stopwatch.StartStopMsg:
    m.sw, cmd = m.sw.Update(msg)

// Control
cmd := m.sw.Toggle()
cmd  = m.sw.Start()
cmd  = m.sw.Stop()
cmd  = m.sw.Reset()

m.sw.Running()   // bool
m.sw.Elapsed()   // time.Duration
m.sw.View()
```

---

## Help

**Import:** `github.com/charmbracelet/bubbles/help`

Auto-generated help view from key bindings. Shows compact (single-line) or full (multi-line) mode.

```go
h := help.New()
h.ShowAll = false  // compact mode; user can toggle with '?'
h.Width = 80

// Implement help.KeyMap interface:
type keyMap struct {
    Up    key.Binding
    Down  key.Binding
    Help  key.Binding
    Quit  key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
    return []key.Binding{k.Help, k.Quit}  // shown in compact mode
}

func (k keyMap) FullHelp() [][]key.Binding {
    return [][]key.Binding{
        {k.Up, k.Down},       // first column
        {k.Help, k.Quit},     // second column
    }
}

// Render:
m.help.View(m.keys)  // m.keys implements help.KeyMap
```

---

## Key

**Import:** `github.com/charmbracelet/bubbles/key`

Non-visual component for defining and matching keybindings.

```go
// Define
up := key.NewBinding(
    key.WithKeys("k", "up"),
    key.WithHelp("↑/k", "move up"),
)

// Match in Update
case tea.KeyMsg:
    if key.Matches(msg, up) {
        // handle up
    }

// Disable/enable
up.SetEnabled(false)
up.Enabled()  // bool

// Inspect
up.Keys()    // []string — actual key strings
up.Help()    // key.HelpData{Key, Desc string}
```

---

## File Picker

**Import:** `github.com/charmbracelet/bubbles/filepicker`

Navigate directories and select files with optional extension filtering.

```go
fp := filepicker.New()
fp.CurrentDirectory, _ = os.Getwd()
fp.AllowedTypes = []string{".go", ".md"}  // filter by extension
fp.DirAllowed = false                      // directories selectable?
fp.FileAllowed = true                      // files selectable?
fp.ShowHidden = false
fp.ShowSize = true
fp.ShowPermissions = true
fp.Height = 20
fp.AutoHeight = false

// Init
func (m model) Init() tea.Cmd { return m.fp.Init() }

// Update — check for selection
m.fp, cmd = m.fp.Update(msg)
if didSelect, path := m.fp.DidSelectFile(msg); didSelect {
    m.selectedFile = path
}
if didSelect, path := m.fp.DidSelectDisabledFile(msg); didSelect {
    // user picked a file that wasn't allowed (e.g., wrong extension)
}
```

---

## Cursor

**Import:** `github.com/charmbracelet/bubbles/cursor`

Reusable blinking cursor used internally by textinput and textarea.

```go
c := cursor.New()

// Modes
cursor.CursorBlink   // blinking cursor
cursor.CursorStatic  // static cursor
cursor.CursorHide    // hidden cursor

c.SetMode(cursor.CursorBlink)
cmd := c.Focus()
c.Blur()
cmd = c.BlinkCmd()

// Style
c.Style = lipgloss.NewStyle().Background(lipgloss.Color("63"))
c.TextStyle = lipgloss.NewStyle()  // style of char under cursor
c.SetChar("█")
c.View()
```

---

## Package Summary

| Package | Import path |
|---------|-------------|
| spinner | `github.com/charmbracelet/bubbles/spinner` |
| textinput | `github.com/charmbracelet/bubbles/textinput` |
| textarea | `github.com/charmbracelet/bubbles/textarea` |
| table | `github.com/charmbracelet/bubbles/table` |
| list | `github.com/charmbracelet/bubbles/list` |
| progress | `github.com/charmbracelet/bubbles/progress` |
| viewport | `github.com/charmbracelet/bubbles/viewport` |
| paginator | `github.com/charmbracelet/bubbles/paginator` |
| timer | `github.com/charmbracelet/bubbles/timer` |
| stopwatch | `github.com/charmbracelet/bubbles/stopwatch` |
| help | `github.com/charmbracelet/bubbles/help` |
| key | `github.com/charmbracelet/bubbles/key` |
| filepicker | `github.com/charmbracelet/bubbles/filepicker` |
| cursor | `github.com/charmbracelet/bubbles/cursor` |

---

**License:** MIT
