# Lip Gloss — Complete Reference

**Package:** `github.com/charmbracelet/lipgloss`
**Docs:** https://pkg.go.dev/github.com/charmbracelet/lipgloss
**Source:** https://github.com/charmbracelet/lipgloss

Lip Gloss provides declarative, CSS-like styling and layout for terminal output. It handles colors, borders, padding, margins, alignment, and layout composition.

---

## Quick Start

```go
import "github.com/charmbracelet/lipgloss"

style := lipgloss.NewStyle().
    Bold(true).
    Foreground(lipgloss.Color("#FAFAFA")).
    Background(lipgloss.Color("#7D56F4")).
    PaddingTop(2).
    PaddingLeft(4).
    Width(22)

fmt.Println(style.Render("Hello, kitty"))
```

---

## Colors

### ANSI 16 Colors (4-bit)

```go
lipgloss.Color("0")   // black
lipgloss.Color("1")   // red
lipgloss.Color("2")   // green
lipgloss.Color("3")   // yellow
lipgloss.Color("4")   // blue
lipgloss.Color("5")   // magenta
lipgloss.Color("6")   // cyan
lipgloss.Color("7")   // white
// 8–15: bright variants
lipgloss.Color("9")   // bright red
lipgloss.Color("12")  // bright blue
```

### ANSI 256 Colors (8-bit)

```go
lipgloss.Color("86")   // aqua
lipgloss.Color("201")  // hot pink
lipgloss.Color("202")  // orange
lipgloss.Color("63")   // medium purple
lipgloss.Color("99")   // light purple
```

### True Color (24-bit hex)

```go
lipgloss.Color("#FF0000")   // red
lipgloss.Color("#04B575")   // green
lipgloss.Color("#7D56F4")   // purple
lipgloss.Color("#3C3C3C")   // dark gray
```

### Adaptive Color (light/dark background detection)

```go
lipgloss.AdaptiveColor{
    Light: "236",   // used on light backgrounds
    Dark:  "248",   // used on dark backgrounds
}
```

### Complete Color (explicit per-profile)

```go
lipgloss.CompleteColor{
    TrueColor: "#0000FF",
    ANSI256:   "86",
    ANSI:      "5",
}
// Disables automatic color degradation
```

### Complete Adaptive Color

```go
lipgloss.CompleteAdaptiveColor{
    Light: lipgloss.CompleteColor{TrueColor: "#d7ffae", ANSI256: "193", ANSI: "11"},
    Dark:  lipgloss.CompleteColor{TrueColor: "#d75fee", ANSI256: "163", ANSI: "5"},
}
```

### Force Color Profile

```go
import "github.com/muesli/termenv"

lipgloss.SetColorProfile(termenv.TrueColor)
// Profiles: termenv.TrueColor, termenv.ANSI256, termenv.ANSI, termenv.Ascii
```

---

## Style — All Options

`lipgloss.NewStyle()` returns a `Style` — a value type (copy-safe).

### Text Formatting

```go
style.Bold(true)
style.Italic(true)
style.Underline(true)
style.Strikethrough(true)
style.Blink(true)
style.Faint(true)
style.Reverse(true)          // swap foreground/background
style.Overline(true)
```

### Colors

```go
style.Foreground(lipgloss.Color("#FF0000"))
style.Background(lipgloss.Color("#0000FF"))
```

### Width & Height

```go
style.Width(40)       // minimum width
style.Height(10)      // minimum height
style.MaxWidth(80)    // maximum width (truncate/wrap)
style.MaxHeight(20)   // maximum height (truncate)
```

### Padding (inner space)

```go
style.Padding(2)               // all sides
style.Padding(2, 4)            // top/bottom, left/right
style.Padding(1, 4, 2)         // top, left/right, bottom
style.Padding(2, 4, 3, 1)      // top, right, bottom, left (clockwise)

style.PaddingTop(2)
style.PaddingRight(4)
style.PaddingBottom(2)
style.PaddingLeft(4)
```

### Margin (outer space)

```go
style.Margin(2)                // all sides
style.Margin(2, 4)             // top/bottom, left/right
style.Margin(1, 4, 2)          // top, left/right, bottom
style.Margin(2, 4, 3, 1)       // clockwise

style.MarginTop(2)
style.MarginRight(4)
style.MarginBottom(2)
style.MarginLeft(4)
style.MarginBackground(lipgloss.Color("63"))  // fill margin with color
```

### Alignment

```go
style.Align(lipgloss.Left)     // horizontal alignment
style.Align(lipgloss.Center)
style.Align(lipgloss.Right)

style.AlignVertical(lipgloss.Top)    // vertical alignment (within height)
style.AlignVertical(lipgloss.Center)
style.AlignVertical(lipgloss.Bottom)
```

Position constants:
```go
lipgloss.Left    = 0.0
lipgloss.Center  = 0.5
lipgloss.Right   = 1.0
lipgloss.Top     = 0.0
lipgloss.Bottom  = 1.0
```

### Borders

```go
// Built-in border styles:
lipgloss.NormalBorder()     // ─ │ ┌ ┐ └ ┘
lipgloss.RoundedBorder()    // ─ │ ╭ ╮ ╰ ╯
lipgloss.BlockBorder()      // full-block style
lipgloss.InnerHalfBlockBorder()
lipgloss.OuterHalfBlockBorder()
lipgloss.ThickBorder()      // ━ ┃ ┏ ┓ ┗ ┛
lipgloss.DoubleBorder()     // ═ ║ ╔ ╗ ╚ ╝
lipgloss.HiddenBorder()     // invisible (takes up space)
lipgloss.ASCIIBorder()      // + - |
lipgloss.MarkdownBorder()   // markdown table style

// Apply border:
style.Border(lipgloss.NormalBorder())             // all sides
style.Border(lipgloss.ThickBorder(), true, false) // top and bottom only
style.Border(lipgloss.DoubleBorder(), true, false, false, true) // clockwise

// Individual sides:
style.BorderStyle(lipgloss.RoundedBorder())
style.BorderTop(true)
style.BorderRight(true)
style.BorderBottom(true)
style.BorderLeft(true)

// Border colors:
style.BorderForeground(lipgloss.Color("63"))
style.BorderBackground(lipgloss.Color("228"))

// Per-side colors:
style.BorderTopForeground(lipgloss.Color("63"))
style.BorderRightForeground(...)
style.BorderBottomForeground(...)
style.BorderLeftForeground(...)
// Same for BorderXBackground()

// Custom border:
custom := lipgloss.Border{
    Top:         "._.:*:",
    Bottom:      "._.:*:",
    Left:        "|*",
    Right:       "|*",
    TopLeft:     "*",
    TopRight:    "*",
    BottomLeft:  "*",
    BottomRight: "*",
}
style.Border(custom)
```

### Tab Width

```go
style.TabWidth(4)                    // default: 4 spaces
style.TabWidth(2)                    // render as 2 spaces
style.TabWidth(0)                    // remove tabs
style.TabWidth(lipgloss.NoTabConversion) // leave tabs as-is
```

### Inline / Constraints

```go
style.Inline(true)          // force single line, ignore margin/padding/border
style.MaxWidth(5)           // also limit to 5 cells wide
style.MaxHeight(5)
```

---

## Rendering

```go
// Basic render
result := style.Render("text")

// Render with preset string
style = style.SetString("Hello,")
fmt.Println(style.Render("kitty."))   // Hello, kitty.

// Stringer interface
var style = lipgloss.NewStyle().SetString("hello").Bold(true)
fmt.Println(style)  // prints styled "hello"

// Render multiple strings (joined with no separator):
style.Render("one", "two", "three")
```

---

## Style Copying & Inheritance

```go
// Copy (styles are value types, assignment is a true copy)
original := lipgloss.NewStyle().Foreground(lipgloss.Color("219"))
copied   := original              // true copy
extended := original.Blink(true)  // also a copy, with blink added

// Inheritance (unset rules only)
base  := lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Background(lipgloss.Color("63"))
child := lipgloss.NewStyle().Foreground(lipgloss.Color("201")).Inherit(base)
// child has Foreground=201 (own), Background=63 (inherited)
```

---

## Unsetting Rules

```go
style.UnsetBold()
style.UnsetItalic()
style.UnsetForeground()
style.UnsetBackground()
style.UnsetBorderStyle()
style.UnsetBorderTop() / UnsetBorderRight() / etc.
style.UnsetPaddingTop() / etc.
style.UnsetMarginTop() / etc.
style.UnsetWidth()
style.UnsetHeight()
style.UnsetAlign()
// ... all setters have a corresponding Unset variant
```

---

## Getters

```go
style.GetBold()          // bool
style.GetForeground()    // lipgloss.TerminalColor
style.GetBackground()    // lipgloss.TerminalColor
style.GetPaddingTop()    // int
style.GetPaddingRight()  // int
style.GetPaddingBottom() // int
style.GetPaddingLeft()   // int
style.GetHorizontalPadding() // left + right
style.GetVerticalPadding()   // top + bottom
style.GetMarginTop()     // etc.
style.GetBorderStyle()   // lipgloss.Border
style.GetBorderTopSize() // int
style.GetWidth()         // int
style.GetHeight()        // int
style.GetAlign()         // lipgloss.Position
// all style properties have getters
```

---

## Utility Functions

### Measuring

```go
width  := lipgloss.Width(renderedBlock)   // int — ANSI-aware width
height := lipgloss.Height(renderedBlock)  // int — line count
w, h   := lipgloss.Size(renderedBlock)    // both at once
```

### Joining

```go
// Horizontal join — align along vertical axis
lipgloss.JoinHorizontal(lipgloss.Top,    a, b, c)
lipgloss.JoinHorizontal(lipgloss.Center, a, b, c)
lipgloss.JoinHorizontal(lipgloss.Bottom, a, b, c)
lipgloss.JoinHorizontal(0.2, a, b, c)  // custom position 0.0–1.0

// Vertical join — align along horizontal axis
lipgloss.JoinVertical(lipgloss.Left,   a, b)
lipgloss.JoinVertical(lipgloss.Center, a, b)
lipgloss.JoinVertical(lipgloss.Right,  a, b)
```

### Placing Text in Whitespace

```go
// Center in 80-wide space
block := lipgloss.PlaceHorizontal(80, lipgloss.Center, content)

// Bottom of 30-tall space
block := lipgloss.PlaceVertical(30, lipgloss.Bottom, content)

// Bottom-right of 30×80 space
block := lipgloss.Place(30, 80, lipgloss.Right, lipgloss.Bottom, content)

// Style the whitespace
ws := lipgloss.NewStyle().Background(lipgloss.Color("240"))
block := lipgloss.PlaceHorizontal(80, lipgloss.Center, content,
    lipgloss.WithWhitespaceChars("░"),
    lipgloss.WithWhitespaceForeground(lipgloss.Color("240")),
)
```

---

## Custom Renderers (multi-output / SSH)

```go
// Create renderer for a specific output (e.g., SSH session)
renderer := lipgloss.NewRenderer(sess) // sess is an io.Writer
style := renderer.NewStyle().Background(lipgloss.AdaptiveColor{Light: "63", Dark: "228"})
io.WriteString(sess, style.Render("Hello"))

// The renderer detects color profile and dark background for the given output
renderer.SetColorProfile(termenv.TrueColor)
renderer.SetHasDarkBackground(true)
```

---

## Tables (`lipgloss/table`)

**Import:** `github.com/charmbracelet/lipgloss/table`

```go
import "github.com/charmbracelet/lipgloss/table"

t := table.New().
    Border(lipgloss.NormalBorder()).
    BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("99"))).
    StyleFunc(func(row, col int) lipgloss.Style {
        if row == table.HeaderRow {
            return headerStyle
        }
        if row%2 == 0 {
            return evenStyle
        }
        return oddStyle
    }).
    Headers("NAME", "AGE", "EMAIL").
    Rows([][]string{
        {"Alice", "30", "alice@example.com"},
        {"Bob",   "25", "bob@example.com"},
    }...)
    // or add row-by-row:
    // .Row("Alice", "30", "alice@example.com")

fmt.Println(t)
```

### Table Methods

| Method | Description |
|--------|-------------|
| `.Border(border)` | Set border style |
| `.BorderTop/Right/Bottom/Left(bool)` | Show/hide individual sides |
| `.BorderHeader(bool)` | Show/hide header separator |
| `.BorderColumn(bool)` | Show/hide column separators |
| `.BorderRow(bool)` | Show/hide row separators |
| `.BorderStyle(style)` | Style for borders |
| `.Headers(cols...)` | Set header columns |
| `.Rows(rows...)` | Set all rows |
| `.Row(vals...)` | Add a single row |
| `.StyleFunc(fn)` | Per-cell style function |
| `.Width(w)` | Total table width |
| `.Height(h)` | Total table height |
| `.Offset(n)` | Row offset (must set after Rows) |
| `.String()` | Render to string |

### Predefined Border Styles for Tables

```go
table.New().Border(lipgloss.NormalBorder())
table.New().Border(lipgloss.RoundedBorder())
table.New().Border(lipgloss.ThickBorder())
table.New().Border(lipgloss.DoubleBorder())
table.New().Border(lipgloss.ASCIIBorder())
// Markdown table (disable top/bottom for pure markdown):
table.New().Border(lipgloss.MarkdownBorder()).BorderTop(false).BorderBottom(false)
```

---

## Lists (`lipgloss/list`)

**Import:** `github.com/charmbracelet/lipgloss/list`

```go
import "github.com/charmbracelet/lipgloss/list"

l := list.New("A", "B", "C")
fmt.Println(l)
// • A
// • B
// • C
```

### Nested Lists

```go
l := list.New(
    "Fruits",
    list.New("Apple", "Banana", "Cherry"),
    "Veggies",
    list.New("Carrot", "Dill"),
)
```

### Built-in Enumerators

| Enumerator | Style |
|------------|-------|
| `list.Bullet` | `• item` (default) |
| `list.Alphabet` | `a. item` |
| `list.Arabic` | `1. item` |
| `list.Roman` | `i. item` |
| `list.Tree` | tree structure |

### Custom Enumerator

```go
func MyEnumerator(items list.Items, i int) string {
    if i == 0 {
        return "→"
    }
    return " "
}

l := list.New("First", "Second", "Third").
    Enumerator(MyEnumerator)
```

### List Styling

```go
l := list.New("A", "B", "C").
    Enumerator(list.Roman).
    EnumeratorStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("99")).MarginRight(1)).
    ItemStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("212")))
```

### Building Incrementally

```go
l := list.New()
for _, item := range items {
    l.Item(item)
}
```

---

## Trees (`lipgloss/tree`)

**Import:** `github.com/charmbracelet/lipgloss/tree`

```go
import "github.com/charmbracelet/lipgloss/tree"

t := tree.Root(".").
    Child("macOS").
    Child(
        tree.New().
            Root("Linux").
            Child("NixOS").
            Child("Arch Linux"),
    ).
    Child(
        tree.New().
            Root("BSD").
            Child("FreeBSD"),
    )

fmt.Println(t)
// .
// ├── macOS
// ├── Linux
// │   ├── NixOS
// │   └── Arch Linux
// └── BSD
//     └── FreeBSD
```

### Built-in Tree Enumerators

| Constant | Style |
|----------|-------|
| `tree.DefaultEnumerator` | `├──` / `└──` |
| `tree.RoundedEnumerator` | `├──` / `╰──` (rounded last item) |

### Tree Styling

```go
t := tree.Root("Root").
    Child("Item 1", "Item 2").
    Enumerator(tree.RoundedEnumerator).
    RootStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("35"))).
    ItemStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("212"))).
    EnumeratorStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("63")).MarginRight(1))
```

### Building Incrementally

```go
t := tree.New()
for _, item := range items {
    t.Child(item)
}
```

---

## FAQ

### Colors not showing?

Lip Gloss auto-degrades colors. In tests/CI/piped output, colors are stripped. Force a profile:

```go
lipgloss.SetColorProfile(termenv.TrueColor)
```

### Misalignment with CJK characters?

Set `RUNEWIDTH_EASTASIAN=0` in your environment (affects east-Asian wide character width calculation).

### Lip Gloss vs Bubble Tea?

Lip Gloss is for **styling and layout** (what your UI looks like). Bubble Tea is the **application framework** (event loop, state management). Use both together.

---

## Complete Style Reference Summary

| Category | Methods |
|----------|---------|
| Text | `Bold`, `Italic`, `Underline`, `Strikethrough`, `Blink`, `Faint`, `Reverse`, `Overline` |
| Color | `Foreground`, `Background` |
| Spacing | `Padding*`, `Margin*`, `MarginBackground` |
| Size | `Width`, `Height`, `MaxWidth`, `MaxHeight` |
| Alignment | `Align`, `AlignVertical` |
| Border | `Border`, `BorderStyle`, `BorderTop/Right/Bottom/Left`, `BorderForeground`, `BorderBackground` |
| Text wrap | `TabWidth`, `Inline` |
| Setup | `SetString`, `Inherit`, `Copy` (via assignment) |
| Unset | `Unset*` for every setter |
| Get | `Get*` for every setter |
| Render | `Render(strings...)`, `String()` |

---

**License:** MIT
