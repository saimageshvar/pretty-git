package checkout

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sai/pretty-git/internal/git"
	ui "github.com/sai/pretty-git/internal/ui"
)

// ── Constants ──────────────────────────────────────────────────────────────

const (
	focusName   = 0
	focusParent = 1
	focusDesc   = 2

	labelW        = 15 // visual width of label column
	maxPickerRows = 9  // max rows shown in parent picker
	inputMinW     = 36 // minimum textinput visual width
)

// inputWidth returns the available width for text input fields, filling the
// terminal width minus the indent (2), space (1), and label column (labelW).
func (m Model) inputWidth() int {
	w := m.width - 2 - 1 - labelW
	if w < inputMinW {
		w = inputMinW
	}
	return w
}

// parentFilterWidth returns the display width for the parent filter input.
// When a parent is selected, extra prefix text ("  filter: " + name) reduces
// the available space.
func (m Model) parentFilterWidth() int {
	iw := m.inputWidth()
	if m.selectedParent != "" {
		// prefix: selectedParent + "  " + "filter: " = len(selectedParent) + 10
		taken := lipgloss.Width(m.selectedParent) + 10
		iw -= taken
		if iw < 10 {
			iw = 10
		}
	}
	return iw
}

// syncInputWidths keeps the parent-filter textinput sized to the current layout.
// nameInput and descInput are rendered with custom hard-wrapping so their
// textinput Width is set to 0 (disables internal scrolling viewport).
func (m *Model) syncInputWidths() {
	m.nameInput.Width = 0
	m.descInput.Width = 0
	m.parentFilter.Width = m.parentFilterWidth()
}

// renderWrappingInput hard-wraps a textinput value at iw columns.
// The first wrapped line is returned as-is; continuation lines are prefixed
// with indent so they align with the input start column.
// Uses Value() and Position() — does NOT call the textinput's own View().
func renderWrappingInput(ti textinput.Model, focused bool, iw int, indent string) string {
	value := ti.Value()
	runes := []rune(value)

	textSt   := lipgloss.NewStyle().Foreground(ui.ColorHeader)
	cursorSt := lipgloss.NewStyle().Foreground(ui.ColorAccent).Reverse(true)
	dimSt    := lipgloss.NewStyle().Foreground(ui.ColorDim)

	// Empty field — show placeholder (+ cursor when focused)
	if len(runes) == 0 {
		if !focused {
			return dimSt.Render(ti.Placeholder)
		}
		return cursorSt.Render(" ") + dimSt.Render(ti.Placeholder)
	}

	// Hard-wrap into iw-wide chunks
	var chunks [][]rune
	tmp := runes
	for len(tmp) > 0 {
		if len(tmp) <= iw {
			chunks = append(chunks, tmp)
			break
		}
		chunks = append(chunks, tmp[:iw])
		tmp = tmp[iw:]
	}

	cursorPos := ti.Position()
	cursorLine := cursorPos / iw
	cursorCol  := cursorPos % iw

	// Cursor sits exactly at a chunk boundary (e.g. position 102 with iw=102):
	// cursorLine points past the last chunk — add an empty chunk for the cursor.
	if cursorLine >= len(chunks) {
		chunks = append(chunks, []rune{})
	}

	var lines []string
	for i, chunk := range chunks {
		if !focused || i != cursorLine {
			lines = append(lines, textSt.Render(string(chunk)))
			continue
		}
		var sb strings.Builder
		if cursorCol < len(chunk) {
			sb.WriteString(textSt.Render(string(chunk[:cursorCol])))
			sb.WriteString(cursorSt.Render(string(chunk[cursorCol])))
			sb.WriteString(textSt.Render(string(chunk[cursorCol+1:])))
		} else {
			sb.WriteString(textSt.Render(string(chunk)))
			sb.WriteString(cursorSt.Render(" "))
		}
		lines = append(lines, sb.String())
	}

	for i := range lines {
		if i > 0 {
			lines[i] = indent + lines[i]
		}
	}
	return strings.Join(lines, "\n")
}

// ── Types ──────────────────────────────────────────────────────────────────

type renderItem struct {
	branch     git.Branch
	treePrefix string // pre-built box-drawing prefix
}

// Result holds the confirmed form values after submission.
type Result struct {
	Name        string
	Parent      string
	Description string
}

type createDoneMsg struct {
	err    error
	result Result
}

// ── Model ──────────────────────────────────────────────────────────────────

type Model struct {
	// Branch data
	branches    []git.Branch
	allItems    []renderItem // full local-branch tree for picker
	pickerItems []renderItem // filtered subset
	repoName    string

	// Layout
	width       int


	// Form state
	focused        int // focusName | focusParent | focusDesc
	nameInput      textinput.Model
	parentFilter   textinput.Model // filter textinput shown in parent field
	selectedParent string          // confirmed parent branch name (empty = none)
	descInput      textinput.Model

	// Picker navigation
	pickerCursor int
	pickerOffset int

	// Lifecycle
	submitting bool
	err        string
	done       bool
	quitting   bool
	result     Result

	spinner spinner.Model
}

// New creates a checkout form model. Pre-fill any known values; the TUI will
// focus the first unfilled field automatically.
func New(
	branches []git.Branch,
	repoName string,
	width, height int,
	initialName, initialParent, initialDesc string,
) Model {
	allItems := buildPickerItems(branches)

	makeTextInput := func(placeholder string, charLimit int) textinput.Model {
		ti := textinput.New()
		ti.Prompt = ""
		ti.PromptStyle = lipgloss.NewStyle()
		ti.TextStyle = lipgloss.NewStyle().Foreground(ui.ColorHeader)
		ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(ui.ColorDim)
		ti.Placeholder = placeholder
		ti.Cursor.Style = lipgloss.NewStyle().Foreground(ui.ColorAccent)
		ti.CharLimit = charLimit
		return ti
	}

	ni := makeTextInput("feature/my-branch", 250)
	ni.SetValue(initialName)

	pf := makeTextInput("type to filter…", 80)

	di := makeTextInput("short description…", 1000)
	di.SetValue(initialDesc)

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Bold(true).Foreground(ui.ColorAccent)

	// Auto-advance focus past pre-filled fields
	focused := focusName
	if initialName != "" {
		focused = focusParent
	}
	if initialName != "" && initialParent != "" {
		focused = focusDesc
	}

	m := Model{
		branches:    branches,
		allItems:    allItems,
		pickerItems: allItems,
		repoName:    repoName,
		width:       width,
		focused:     focused,
		nameInput:      ni,
		parentFilter:   pf,
		selectedParent: initialParent,
		descInput:      di,
		spinner:        sp,
	}

	m.syncInputWidths()
	m.applyPickerFilter()
	m.preselectParent()

	switch focused {
	case focusName:
		_ = m.nameInput.Focus()
	case focusParent:
		_ = m.parentFilter.Focus()
	case focusDesc:
		_ = m.descInput.Focus()
	}

	return m
}

func (m Model) Init() tea.Cmd { return m.spinner.Tick }

// ── Update ─────────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.syncInputWidths()
		return m, nil

	case createDoneMsg:
		if msg.err != nil {
			m.submitting = false
			m.err = msg.err.Error()
			m.focused = focusName
			m.parentFilter.Blur()
			m.descInput.Blur()
			return m, m.nameInput.Focus()
		}
		m.done = true
		m.result = msg.result
		return m, tea.Quit

	case tea.KeyMsg:
		return m.updateKey(msg)
	}
	return m, nil
}

func (m Model) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.submitting {
		return m, nil
	}

	k := msg.String()
	switch k {
	case "ctrl+c", "esc":
		m.quitting = true
		return m, tea.Quit

	case "tab":
		return m.moveFocus(1)

	case "shift+tab":
		return m.moveFocus(-1)

	case "enter":
		return m.handleEnter()

	case "up":
		if m.focused == focusParent && len(m.pickerItems) > 0 {
			if m.pickerCursor > 0 {
				m.pickerCursor--
				m.clampPickerScroll()
			}
			return m, nil
		}

	case "down":
		if m.focused == focusParent && len(m.pickerItems) > 0 {
			if m.pickerCursor < len(m.pickerItems)-1 {
				m.pickerCursor++
				m.clampPickerScroll()
			}
			return m, nil
		}

	case "ctrl+d":
		if m.focused == focusParent {
			m.selectedParent = ""
			m.parentFilter.Reset()
			m.applyPickerFilter()
			m.syncInputWidths()
			return m, nil
		}
	}

	// Route keypress to the focused input
	switch m.focused {
	case focusName:
		var cmd tea.Cmd
		m.nameInput, cmd = m.nameInput.Update(msg)
		m.err = ""
		return m, cmd
	case focusParent:
		prev := m.parentFilter.Value()
		var cmd tea.Cmd
		m.parentFilter, cmd = m.parentFilter.Update(msg)
		if m.parentFilter.Value() != prev {
			m.applyPickerFilter()
		}
		return m, cmd
	case focusDesc:
		var cmd tea.Cmd
		m.descInput, cmd = m.descInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) moveFocus(delta int) (tea.Model, tea.Cmd) {
	m.focused = (m.focused + delta + 3) % 3

	m.nameInput.Blur()
	m.parentFilter.Blur()
	m.descInput.Blur()

	var cmd tea.Cmd
	switch m.focused {
	case focusName:
		cmd = m.nameInput.Focus()
	case focusParent:
		// Pre-highlight the currently selected parent in the picker
		m.preselectParent()
		cmd = m.parentFilter.Focus()
	case focusDesc:
		cmd = m.descInput.Focus()
	}

	return m, cmd
}

func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	switch m.focused {
	case focusName:
		if strings.TrimSpace(m.nameInput.Value()) == "" {
			m.err = "branch name is required"
			return m, nil
		}
		return m.moveFocus(1)

	case focusParent:
		// Confirm highlighted picker item (if any)
		if len(m.pickerItems) > 0 {
			m.selectedParent = m.pickerItems[m.pickerCursor].branch.Name
			m.parentFilter.Reset()
			m.applyPickerFilter()
			m.syncInputWidths()
		}
		return m.moveFocus(1)

	case focusDesc:
		return m.submit()
	}
	return m, nil
}

func (m Model) submit() (tea.Model, tea.Cmd) {
	name := strings.TrimSpace(m.nameInput.Value())
	if name == "" {
		m.err = "branch name is required"
		m.focused = focusName
		m.parentFilter.Blur()
		m.descInput.Blur()
		return m, m.nameInput.Focus()
	}

	m.submitting = true
	m.err = ""
	result := Result{
		Name:        name,
		Parent:      m.selectedParent,
		Description: strings.TrimSpace(m.descInput.Value()),
	}
	return m, doCreateBranch(result)
}

// Result returns the form result after a successful submission.
func (m Model) Result() Result { return m.result }

// WasQuit returns true if the user cancelled without creating a branch.
func (m Model) WasQuit() bool { return m.quitting }

// ── Picker helpers ─────────────────────────────────────────────────────────

func (m *Model) applyPickerFilter() {
	q := strings.ToLower(m.parentFilter.Value())
	if q == "" {
		m.pickerItems = m.allItems
	} else {
		var out []renderItem
		for _, item := range m.allItems {
			b := item.branch
			if strings.Contains(strings.ToLower(b.Name), q) ||
				strings.Contains(strings.ToLower(b.Description), q) {
				out = append(out, renderItem{branch: b})
			}
		}
		m.pickerItems = out
	}
	m.pickerCursor = 0
	m.pickerOffset = 0
}

// preselectParent sets pickerCursor to the position of selectedParent in pickerItems.
func (m *Model) preselectParent() {
	if m.selectedParent == "" {
		return
	}
	for i, item := range m.pickerItems {
		if item.branch.Name == m.selectedParent {
			m.pickerCursor = i
			m.clampPickerScroll()
			return
		}
	}
}

func (m *Model) clampPickerScroll() {
	vis := maxPickerRows
	if m.pickerCursor < m.pickerOffset {
		m.pickerOffset = m.pickerCursor
	}
	if m.pickerCursor >= m.pickerOffset+vis {
		m.pickerOffset = m.pickerCursor - vis + 1
	}
}

// ── Git command ─────────────────────────────────────────────────────────────

func doCreateBranch(result Result) tea.Cmd {
	return func() tea.Msg {
		if err := git.CreateBranch(result.Name); err != nil {
			return createDoneMsg{err: err, result: result}
		}
		if result.Parent != "" {
			if err := git.SetParent(result.Name, result.Parent); err != nil {
				return createDoneMsg{err: err, result: result}
			}
		}
		if result.Description != "" {
			if err := git.SetDescription(result.Name, result.Description); err != nil {
				return createDoneMsg{err: err, result: result}
			}
		}
		return createDoneMsg{err: nil, result: result}
	}
}

// ── View ───────────────────────────────────────────────────────────────────

func (m Model) View() string {
	if m.done || m.quitting {
		return ""
	}
	var sb strings.Builder

	sb.WriteString(m.renderHeader() + "\n")
	sb.WriteString(ui.StyleDivider.Render(strings.Repeat("─", m.width)) + "\n")

	sb.WriteString(m.renderFieldName() + "\n")
	sb.WriteString(m.renderFieldParent() + "\n")
	sb.WriteString(m.renderFieldDesc() + "\n")

	// Show picker panel when parent field is focused
	if m.focused == focusParent {
		sb.WriteString(ui.StyleDivider.Render(strings.Repeat("─", m.width)) + "\n")
		for _, l := range m.pickerLines(m.width) {
			sb.WriteString(l + "\n")
		}
	}

	sb.WriteString(ui.StyleDivider.Render(strings.Repeat("─", m.width)) + "\n")
	sb.WriteString(m.renderFooter())

	return sb.String()
}

// ── Shared field renderers ──────────────────────────────────────────────────

func (m Model) renderHeader() string {
	badge := ui.StyleCountBadge.Render("new branch")
	return ui.StyleHeader.Render("  ✦ New Branch") + "  " +
		ui.StyleAccent.Render(m.repoName) + "  " + badge
}

func (m Model) renderFieldName() string {
	focused := m.focused == focusName
	label := fieldLabel("⎇ Branch:", focused)
	indent := strings.Repeat(" ", 2+labelW+1)
	return "  " + label + " " + renderWrappingInput(m.nameInput, focused, m.inputWidth(), indent)
}

func (m Model) renderFieldParent() string {
	focused := m.focused == focusParent
	label := fieldLabel("⎇ Parent:", focused)
	iw := m.inputWidth()

	if focused {
		// Show filter input; prepend selected branch name if one is confirmed
		if m.selectedParent != "" {
			selS := lipgloss.NewStyle().Foreground(ui.ColorAccent).Bold(true).
				Render(m.selectedParent)
			return "  " + label + " " + selS + "  " + ui.StyleDim.Render("filter: ") + m.parentFilter.View()
		}
		return "  " + label + " " + m.parentFilter.View()
	}

	if m.selectedParent != "" {
		val := lipgloss.NewStyle().Foreground(ui.ColorAccent).Bold(true).
			Width(iw).Render(m.selectedParent)
		return "  " + label + " " + val
	}
	return "  " + label + " " + lipgloss.NewStyle().Foreground(ui.ColorDim).Width(iw).Render("(none)")
}

func (m Model) renderFieldDesc() string {
	focused := m.focused == focusDesc
	label := fieldLabel("✎ Desc:", focused)
	indent := strings.Repeat(" ", 2+labelW+1)
	return "  " + label + " " + renderWrappingInput(m.descInput, focused, m.inputWidth(), indent)
}

func fieldLabel(text string, focused bool) string {
	s := lipgloss.NewStyle().Width(labelW)
	if focused {
		return s.Bold(true).Foreground(ui.ColorKeyHint).Render(text)
	}
	return s.Foreground(ui.ColorDim).Render(text)
}


// ── Picker column layout ───────────────────────────────────────────────────

// pickerColumnWidths computes fixed name and description column widths for the
// parent picker, mirroring the approach used by the branch list's columnWidths.
//
// Fixed overhead per row: 2 (left pad) + 1 (marker) + 2 (sep) + 2 (sep) = 7
// treePrefix length is variable and subtracted per-row from nameW.
// Description gets ~35% of available space (bounded to [14, 48]).
// Name gets the remainder.
func pickerColumnWidths(termWidth int) (totalNameW, descW int) {
	const fixed = 2 + 1 + 2 + 2 // leftpad + marker + sep×2
	avail := termWidth - fixed
	if avail < 1 {
		return 20, 0
	}
	descW = avail * 35 / 100
	if descW < 14 {
		descW = 14
	}
	if descW > 48 {
		descW = 48
	}
	totalNameW = avail - descW
	if totalNameW < 12 {
		totalNameW = 12
	}
	return
}

// ── Picker panel ───────────────────────────────────────────────────────────

func (m Model) pickerLines(width int) []string {
	var lines []string

	if len(m.pickerItems) == 0 {
		lines = append(lines, ui.StyleDim.Render("  (no branches)"))
		return lines
	}

	end := m.pickerOffset + maxPickerRows
	if end > len(m.pickerItems) {
		end = len(m.pickerItems)
	}

	const colPad = 2
	sep := strings.Repeat(" ", colPad)

	totalNameW, descW := pickerColumnWidths(width)

	for i := m.pickerOffset; i < end; i++ {
		item := m.pickerItems[i]
		b := item.branch

		prefixLen := len([]rune(item.treePrefix))
		nameW := totalNameW - prefixLen
		if nameW < 8 {
			nameW = 8
		}

		nameText := truncate(b.Name, nameW)
		descText := truncate(b.Description, descW)

		if i == m.pickerCursor {
			bg := ui.ColorCursorBg
			bgSep    := lipgloss.NewStyle().Background(bg).Render(sep)
			leftPad  := lipgloss.NewStyle().Background(bg).Render("  ")
			markerS  := lipgloss.NewStyle().Background(bg).Bold(true).Foreground(ui.ColorAccent).Render("»")
			prefixS  := lipgloss.NewStyle().Background(bg).Foreground(ui.ColorDim).Render(item.treePrefix)
			nameS    := lipgloss.NewStyle().Background(bg).Bold(true).Foreground(ui.ColorCursorFg).Width(nameW).Render(nameText)
			descS    := lipgloss.NewStyle().Background(bg).Foreground(ui.ColorDesc).Italic(true).Width(descW).Render(descText)
			used     := 2 + 1 + colPad + totalNameW + colPad + descW
			trail    := ""
			if width > used {
				trail = lipgloss.NewStyle().Background(bg).Render(strings.Repeat(" ", width-used))
			}
			lines = append(lines, leftPad+markerS+bgSep+prefixS+nameS+bgSep+descS+trail)
		} else {
			prefixS := ui.StyleTreeConnector.Render(item.treePrefix)
			nameS   := lipgloss.NewStyle().Foreground(ui.ColorHeader).Width(nameW).Render(nameText)
			descS   := ui.StyleDesc.Italic(true).Render(descText)
			// "  " (leftpad=2) + " " (blank marker=1) + sep(colPad) = same indent as cursor row
			lines = append(lines, "  "+" "+sep+prefixS+nameS+sep+descS)
		}
	}

	// Scroll indicator
	if len(m.pickerItems) > maxPickerRows {
		shown := fmt.Sprintf("  %d–%d of %d", m.pickerOffset+1, end, len(m.pickerItems))
		lines = append(lines, ui.StyleDim.Render(shown))
	}

	return lines
}

// ── Footer ─────────────────────────────────────────────────────────────────

func (m Model) renderFooter() string {
	if m.submitting {
		return "  " + m.spinner.View() + " " +
			lipgloss.NewStyle().Foreground(ui.ColorParentAhead).Bold(true).Render("creating branch…")
	}

	dc := lipgloss.NewStyle().Foreground(ui.ColorHeader)

	if m.err != "" {
		hint := "  " +
			ui.StyleKeyHint.Render("tab") + dc.Render("/") +
			ui.StyleKeyHint.Render("shift+tab") + dc.Render(" navigate  ") +
			ui.StyleKeyHint.Render("enter") + dc.Render(" confirm  ") +
			ui.StyleKeyHint.Render("esc") + dc.Render(" cancel")
		return "  " + ui.StyleError.Render("✗ "+m.err) + "\n" + hint
	}

	if m.focused == focusParent {
		return "  " +
			ui.StyleKeyHint.Render("↑/↓") + dc.Render(" navigate  ") +
			ui.StyleKeyHint.Render("enter") + dc.Render(" select  ") +
			ui.StyleKeyHint.Render("ctrl+d") + dc.Render(" clear  ") +
			ui.StyleKeyHint.Render("tab") + dc.Render(" next field  ") +
			ui.StyleKeyHint.Render("esc") + dc.Render(" cancel")
	}

	return "  " +
		ui.StyleKeyHint.Render("tab") + dc.Render("/") +
		ui.StyleKeyHint.Render("shift+tab") + dc.Render(" navigate  ") +
		ui.StyleKeyHint.Render("enter") + dc.Render(" next/confirm  ") +
		ui.StyleKeyHint.Render("esc") + dc.Render(" cancel")
}

// ── Tree builder ───────────────────────────────────────────────────────────

// buildPickerItems converts local branches to a DFS-ordered tree list.
func buildPickerItems(branches []git.Branch) []renderItem {
	nameToIdx := make(map[string]int)
	for i, b := range branches {
		if !b.IsRemote {
			nameToIdx[b.Name] = i
		}
	}

	children := make(map[string][]git.Branch)
	for _, b := range branches {
		if b.IsRemote {
			continue
		}
		if b.Parent != "" {
			if _, ok := nameToIdx[b.Parent]; ok {
				children[b.Parent] = append(children[b.Parent], b)
				continue
			}
		}
		children[""] = append(children[""], b)
	}

	var result []renderItem

	var dfs func(parentName string, isLastAtDepth []bool)
	dfs = func(parentName string, isLastAtDepth []bool) {
		kids := children[parentName]
		for i, b := range kids {
			isLast := i == len(kids)-1
			depth := len(isLastAtDepth)

			prefix := ""
			for d := 0; d < depth; d++ {
				if isLastAtDepth[d] {
					prefix += "   "
				} else {
					prefix += "│  "
				}
			}
			if isLast {
				prefix += "└─ "
			} else {
				prefix += "├─ "
			}

			result = append(result, renderItem{branch: b, treePrefix: prefix})

			next := make([]bool, depth+1)
			copy(next, isLastAtDepth)
			next[depth] = isLast
			dfs(b.Name, next)
		}
	}

	dfs("", []bool{})
	return result
}

// ── Helpers ────────────────────────────────────────────────────────────────

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return "…"
	}
	return string(r[:n-1]) + "…"
}


