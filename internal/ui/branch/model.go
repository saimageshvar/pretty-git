package branch

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sai/pretty-git/internal/git"
	ui "github.com/sai/pretty-git/internal/ui"
)

// ── Column widths ──────────────────────────────────────────────────────────

const (
	colMarker  = 1
	colName    = 32
	colStatus  = 12 // "✓ merged", "↑99 ↓99", etc.
	colDesc    = 25 // branch description (truncated)
	colPad     = 2
	maxVisible = 15 // max rows shown at once (inline / fzf-style)

	// edit form
	editFocusParent   = 0
	editFocusDesc     = 1
	editLabelW        = 15
	editInputW        = 36
	maxEditPickerRows = 9
)

// ── renderItem ─────────────────────────────────────────────────────────────

// renderItem wraps a Branch with pre-computed tree display metadata.
type renderItem struct {
	branch     git.Branch
	treePrefix string // pre-colored ANSI string, e.g. "│  ├─ "
	depth      int    // 0 = root
}

// ── Model ──────────────────────────────────────────────────────────────────

type Model struct {
	branches    []git.Branch // raw, unchanged
	treeItems   []renderItem // full tree, built once at startup
	filtered    []renderItem // active list (tree mode or flat filter mode)
	cursor      int
	offset      int
	err         string
	switching   bool
	done        bool
	quitting    bool
	switchedTo  string
	width       int
	visibleRows int
	repoName    string
	localCount  int
	remoteCount int

	filterInput textinput.Model
	help        help.Model
	keys        keyMap

	// unified edit mode (replaces old settingParent + settingDesc modes)
	editing           bool
	editTargetBranch  string
	editFocused       int // editFocusParent | editFocusDesc
	editParentFilter  textinput.Model
	editAllItems      []renderItem // full picker tree (local only, excl. target)
	editPickerItems   []renderItem // filtered subset
	editParentCursor  int
	editParentOffset  int
	editSelectedParent string
	editDescInput     textinput.Model
	editSaving        bool

	spinner spinner.Model
}

type switchDoneMsg struct {
	err  error
	name string
}

type editSavedMsg struct {
	err    error
	branch string
	parent string // "" = cleared
	desc   string // "" = cleared
	ahead  int
	behind int
}

func New(branches []git.Branch, repoName string, termWidth, termHeight int) Model {
	local, remote := 0, 0
	for _, b := range branches {
		if b.IsRemote {
			remote++
		} else {
			local++
		}
	}

	treeItems := buildRenderItems(branches)

	vis := len(treeItems)
	if vis > maxVisible {
		vis = maxVisible
	}
	// 5 = header(1) + 2 dividers + footer(1) + blank line
	if termHeight > 5 && vis > termHeight-5 {
		vis = termHeight - 5
	}
	if vis < 1 {
		vis = 1
	}

	// ── filter textinput ───────────────────────────────────────────────────
	ti := textinput.New()
	ti.Prompt = ""
	ti.PromptStyle = lipgloss.NewStyle()
	ti.TextStyle = lipgloss.NewStyle().Foreground(ui.ColorHeader)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(ui.ColorDim)
	ti.Placeholder = "type to filter…"
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(ui.ColorAccent)
	_ = ti.Focus()

	// ── edit-form textinputs (initialised empty; populated on open) ────────
	epf := textinput.New()
	epf.Prompt = ""
	epf.PromptStyle = lipgloss.NewStyle()
	epf.TextStyle = lipgloss.NewStyle().Foreground(ui.ColorHeader)
	epf.PlaceholderStyle = lipgloss.NewStyle().Foreground(ui.ColorDim)
	epf.Placeholder = "type to filter…"
	epf.Cursor.Style = lipgloss.NewStyle().Foreground(ui.ColorAccent)

	edi := textinput.New()
	edi.Prompt = ""
	edi.PromptStyle = lipgloss.NewStyle()
	edi.TextStyle = lipgloss.NewStyle().Foreground(ui.ColorHeader)
	edi.PlaceholderStyle = lipgloss.NewStyle().Foreground(ui.ColorDim)
	edi.Placeholder = "short description…"
	edi.Cursor.Style = lipgloss.NewStyle().Foreground(ui.ColorAccent)
	edi.CharLimit = 120

	// ── help ───────────────────────────────────────────────────────────────
	h := help.New()
	h.Styles.ShortKey = lipgloss.NewStyle().Bold(true).Foreground(ui.ColorKeyHint)
	h.Styles.ShortDesc = lipgloss.NewStyle().Foreground(ui.ColorHeader)
	h.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(ui.ColorTreeConnector)
	h.Width = termWidth

	sp := spinner.New()
	sp.Spinner = spinner.MiniDot
	sp.Style = lipgloss.NewStyle().Bold(true).Foreground(ui.ColorAccent)

	// Start with the cursor on the current branch.
	initialCursor := 0
	for i, item := range treeItems {
		if item.branch.IsCurrent {
			initialCursor = i
			break
		}
	}

	return Model{
		branches:         branches,
		treeItems:        treeItems,
		filtered:         treeItems,
		cursor:           initialCursor,
		width:            termWidth,
		visibleRows:      vis,
		repoName:         repoName,
		localCount:       local,
		remoteCount:      remote,
		filterInput:      ti,
		editParentFilter: epf,
		editDescInput:    edi,
		help:             h,
		keys:             defaultKeyMap(),
		spinner:          sp,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.filterInput.Focus())
}

// ── Update ─────────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.help.Width = msg.Width
		vis := len(m.filtered)
		if vis > maxVisible {
			vis = maxVisible
		}
		if msg.Height > 5 && vis > msg.Height-5 {
			vis = msg.Height - 5
		}
		if vis < 1 {
			vis = 1
		}
		m.visibleRows = vis
		m.clampScroll()
		return m, nil

	case switchDoneMsg:
		m.switching = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.switchedTo = msg.name
		m.done = true
		return m, tea.Quit

	case editSavedMsg:
		m.editSaving = false
		if msg.err != nil {
			m.err = msg.err.Error()
			m.closeEditForm()
			return m, m.filterInput.Focus()
		}
		// Update in-memory branch data so the view reflects changes immediately.
		for i := range m.branches {
			if m.branches[i].Name == msg.branch {
				m.branches[i].Parent = msg.parent
				m.branches[i].ParentAhead = msg.ahead
				m.branches[i].ParentBehind = msg.behind
				m.branches[i].Description = msg.desc
				break
			}
		}
		m.treeItems = buildRenderItems(m.branches)
		m.filtered = m.treeItems
		savedName := m.editTargetBranch
		m.applyFilter()
		m.restoreCursor(savedName)
		m.closeEditForm()
		return m, m.filterInput.Focus()

	case tea.KeyMsg:
		if m.editing {
			return m.updateEdit(msg)
		}
		return m.updateKeys(msg)
	}
	return m, nil
}

func (m Model) updateKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
			m.clampScroll()
		}
		return m, nil

	case key.Matches(msg, m.keys.Down):
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
			m.clampScroll()
		}
		return m, nil

	case key.Matches(msg, m.keys.Edit):
		if len(m.filtered) == 0 {
			return m, nil
		}
		b := m.filtered[m.cursor].branch
		if b.IsRemote {
			return m, nil
		}
		return m.openEditForm(b)

	case key.Matches(msg, m.keys.EscBack):
		// esc: clear filter if active, otherwise quit
		if m.filterInput.Value() != "" {
			var savedName string
			if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
				savedName = m.filtered[m.cursor].branch.Name
			}
			m.filterInput.Reset()
			m.applyFilter()
			m.restoreCursor(savedName)
			return m, nil
		}
		m.quitting = true
		return m, tea.Quit

	case key.Matches(msg, m.keys.Quit):
		m.quitting = true
		return m, tea.Quit

	case key.Matches(msg, m.keys.Switch):
		if m.switching || len(m.filtered) == 0 {
			return m, nil
		}
		b := m.filtered[m.cursor].branch
		if b.IsCurrent {
			return m, nil
		}
		m.switching = true
		m.err = ""
		name := b.Name
		if b.IsRemote {
			parts := strings.SplitN(name, "/", 3)
			if len(parts) == 3 {
				name = parts[2]
			}
		}
		return m, doSwitch(name)
	}

	// All other keys (letters, numbers, symbols, backspace, etc.) go to the filter input.
	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	m.applyFilter()
	return m, cmd
}

// ── Edit form ──────────────────────────────────────────────────────────────

func (m Model) openEditForm(b git.Branch) (tea.Model, tea.Cmd) {
	m.editing = true
	m.editTargetBranch = b.Name
	m.editFocused = editFocusParent
	m.editSelectedParent = b.Parent
	m.editSaving = false
	m.err = ""

	m.filterInput.Blur()
	m.editParentFilter.Reset()
	m.editDescInput.SetValue(b.Description)

	// Build picker items (local branches, excluding the target)
	m.editAllItems = buildEditPickerItems(m.branches, b.Name)
	m.editPickerItems = m.editAllItems
	m.editParentCursor = 0
	m.editParentOffset = 0
	m.editPreselectParent()

	return m, m.editParentFilter.Focus()
}

func (m *Model) closeEditForm() {
	m.editing = false
	m.editTargetBranch = ""
	m.editSelectedParent = ""
	m.editParentFilter.Blur()
	m.editParentFilter.Reset()
	m.editDescInput.Blur()
	m.editDescInput.Reset()
	m.editAllItems = nil
	m.editPickerItems = nil
	m.editParentCursor = 0
	m.editParentOffset = 0
}

func (m Model) updateEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.editSaving {
		return m, nil
	}

	k := msg.String()
	switch k {
	case "ctrl+c", "esc":
		savedName := m.editTargetBranch
		m.closeEditForm()
		m.restoreCursor(savedName)
		return m, m.filterInput.Focus()

	case "tab":
		return m.editMoveFocus(1)

	case "shift+tab":
		return m.editMoveFocus(-1)

	case "enter":
		return m.editHandleEnter()

	case "up":
		if m.editFocused == editFocusParent && len(m.editPickerItems) > 0 {
			if m.editParentCursor > 0 {
				m.editParentCursor--
				m.editClampScroll()
			}
			return m, nil
		}

	case "down":
		if m.editFocused == editFocusParent && len(m.editPickerItems) > 0 {
			if m.editParentCursor < len(m.editPickerItems)-1 {
				m.editParentCursor++
				m.editClampScroll()
			}
			return m, nil
		}

	case "ctrl+d":
		if m.editFocused == editFocusParent {
			m.editSelectedParent = ""
			m.editParentFilter.Reset()
			m.editApplyFilter()
			return m, nil
		}
	}

	// Route keypress to the focused textinput
	switch m.editFocused {
	case editFocusParent:
		prev := m.editParentFilter.Value()
		var cmd tea.Cmd
		m.editParentFilter, cmd = m.editParentFilter.Update(msg)
		if m.editParentFilter.Value() != prev {
			m.editApplyFilter()
		}
		return m, cmd
	case editFocusDesc:
		var cmd tea.Cmd
		m.editDescInput, cmd = m.editDescInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) editMoveFocus(delta int) (tea.Model, tea.Cmd) {
	m.editFocused = (m.editFocused + delta + 2) % 2

	m.editParentFilter.Blur()
	m.editDescInput.Blur()

	var cmd tea.Cmd
	switch m.editFocused {
	case editFocusParent:
		m.editPreselectParent()
		cmd = m.editParentFilter.Focus()
	case editFocusDesc:
		cmd = m.editDescInput.Focus()
	}
	return m, cmd
}

func (m Model) editHandleEnter() (tea.Model, tea.Cmd) {
	switch m.editFocused {
	case editFocusParent:
		// Confirm highlighted picker item (if any visible)
		if len(m.editPickerItems) > 0 {
			m.editSelectedParent = m.editPickerItems[m.editParentCursor].branch.Name
			m.editParentFilter.Reset()
			m.editApplyFilter()
		}
		return m.editMoveFocus(1)

	case editFocusDesc:
		return m.editSave()
	}
	return m, nil
}

func (m Model) editSave() (tea.Model, tea.Cmd) {
	m.editSaving = true
	return m, doSaveEdit(
		m.editTargetBranch,
		m.editSelectedParent,
		strings.TrimSpace(m.editDescInput.Value()),
	)
}

// ── Edit picker helpers ────────────────────────────────────────────────────

func (m *Model) editApplyFilter() {
	q := strings.ToLower(m.editParentFilter.Value())
	if q == "" {
		m.editPickerItems = m.editAllItems
	} else {
		var out []renderItem
		for _, item := range m.editAllItems {
			if strings.Contains(strings.ToLower(item.branch.Name), q) {
				out = append(out, renderItem{branch: item.branch})
			}
		}
		m.editPickerItems = out
	}
	m.editParentCursor = 0
	m.editParentOffset = 0
}

func (m *Model) editPreselectParent() {
	if m.editSelectedParent == "" {
		return
	}
	for i, item := range m.editPickerItems {
		if item.branch.Name == m.editSelectedParent {
			m.editParentCursor = i
			m.editClampScroll()
			return
		}
	}
}

func (m *Model) editClampScroll() {
	vis := maxEditPickerRows
	if m.editParentCursor < m.editParentOffset {
		m.editParentOffset = m.editParentCursor
	}
	if m.editParentCursor >= m.editParentOffset+vis {
		m.editParentOffset = m.editParentCursor - vis + 1
	}
}

// ── View ───────────────────────────────────────────────────────────────────

func (m Model) View() string {
	if m.done || m.quitting {
		return ""
	}

	if m.editing {
		return m.viewEditForm()
	}

	var sb strings.Builder

	// Header
	badge := ui.StyleCountBadge.Render(fmt.Sprintf("%d local · %d remote", m.localCount, m.remoteCount))
	sb.WriteString(ui.StyleHeader.Render("  Branches") + "  " +
		ui.StyleAccent.Render(m.repoName) + "  " + badge + "\n")
	sb.WriteString(ui.StyleDivider.Render(strings.Repeat("─", m.width)) + "\n")

	// Column headers
	sb.WriteString(renderHeaders() + "\n")

	// Branch rows
	end := m.offset + m.visibleRows
	if end > len(m.filtered) {
		end = len(m.filtered)
	}
	if len(m.filtered) == 0 {
		sb.WriteString(ui.StyleDim.Render("  no branches match\n"))
	} else {
		for i := m.offset; i < end; i++ {
			sb.WriteString(renderRow(m.filtered[i], i == m.cursor, m.width) + "\n")
		}
	}

	// Footer
	sb.WriteString(ui.StyleDivider.Render(strings.Repeat("─", m.width)) + "\n")
	sb.WriteString(m.footer())

	return sb.String()
}

// viewEditForm renders the checkout-style edit form as the full view.
func (m Model) viewEditForm() string {
	var sb strings.Builder

	// Header
	badge := ui.StyleCountBadge.Render("edit branch")
	targetS := lipgloss.NewStyle().Foreground(ui.ColorAccent).Bold(true).Render(m.editTargetBranch)
	sb.WriteString(ui.StyleHeader.Render("  ✦ Edit Branch") + "  " + targetS + "  " + badge + "\n")
	sb.WriteString(ui.StyleDivider.Render(strings.Repeat("─", m.width)) + "\n")

	sb.WriteString(m.editRenderParentField() + "\n")
	sb.WriteString(m.editRenderDescField() + "\n")

	// Picker panel — shown when parent field is focused
	if m.editFocused == editFocusParent && !m.editSaving {
		sb.WriteString(ui.StyleDivider.Render(strings.Repeat("─", m.width)) + "\n")
		for _, l := range m.editPickerLines() {
			sb.WriteString(l + "\n")
		}
	}

	sb.WriteString(ui.StyleDivider.Render(strings.Repeat("─", m.width)) + "\n")
	sb.WriteString(m.editFooter())

	return sb.String()
}

func (m Model) editRenderParentField() string {
	focused := m.editFocused == editFocusParent && !m.editSaving
	label := editFieldLabel("⎇ Parent:", focused)

	if focused {
		if m.editSelectedParent != "" {
			selS := lipgloss.NewStyle().Foreground(ui.ColorAccent).Bold(true).
				Render(m.editSelectedParent)
			filterS := lipgloss.NewStyle().
				Width(editInputW - lipgloss.Width(m.editSelectedParent) - 2).
				Render(m.editParentFilter.View())
			return "  " + label + " " + selS + "  " + ui.StyleDim.Render("filter: ") + filterS
		}
		return "  " + label + " " + lipgloss.NewStyle().Width(editInputW).Render(m.editParentFilter.View())
	}

	if m.editSelectedParent != "" {
		val := lipgloss.NewStyle().Foreground(ui.ColorAccent).Bold(true).
			Width(editInputW).Render(m.editSelectedParent)
		return "  " + label + " " + val
	}
	return "  " + label + " " + lipgloss.NewStyle().Foreground(ui.ColorDim).Width(editInputW).Render("(none)")
}

func (m Model) editRenderDescField() string {
	focused := m.editFocused == editFocusDesc && !m.editSaving
	label := editFieldLabel("✎ Desc:", focused)
	return "  " + label + " " + lipgloss.NewStyle().Width(editInputW).Render(m.editDescInput.View())
}

func editFieldLabel(text string, focused bool) string {
	s := lipgloss.NewStyle().Width(editLabelW)
	if focused {
		return s.Bold(true).Foreground(ui.ColorKeyHint).Render(text)
	}
	return s.Foreground(ui.ColorDim).Render(text)
}

func (m Model) editPickerLines() []string {
	var lines []string

	if len(m.editPickerItems) == 0 {
		lines = append(lines, ui.StyleDim.Render("  (no branches)"))
		return lines
	}

	end := m.editParentOffset + maxEditPickerRows
	if end > len(m.editPickerItems) {
		end = len(m.editPickerItems)
	}

	const indent = 4
	const nameMax = 28
	const descSep = 2

	for i := m.editParentOffset; i < end; i++ {
		item := m.editPickerItems[i]
		b := item.branch

		prefixLen := len([]rune(item.treePrefix))
		avail := m.width - indent - prefixLen
		nameW := avail
		if nameW > nameMax {
			nameW = nameMax
		}
		if nameW < 4 {
			nameW = 4
		}
		name := truncate(b.Name, nameW)

		descW := m.width - indent - prefixLen - nameW - descSep
		desc := ""
		if b.Description != "" && descW > 4 {
			desc = "  " + ui.StyleDesc.Italic(true).Render(truncate(b.Description, descW))
		}

		if i == m.editParentCursor {
			row := lipgloss.NewStyle().
				Background(ui.ColorCursorBg).
				Foreground(ui.ColorCursorFg).
				Bold(true).
				Width(m.width - 2).
				Render("  » " + item.treePrefix + name + desc)
			lines = append(lines, row)
		} else {
			prefixS := ui.StyleTreeConnector.Render(item.treePrefix)
			nameS := lipgloss.NewStyle().Foreground(ui.ColorHeader).Render(name)
			lines = append(lines, "    "+prefixS+nameS+desc)
		}
	}

	if len(m.editPickerItems) > maxEditPickerRows {
		shown := fmt.Sprintf("  %d–%d of %d", m.editParentOffset+1, end, len(m.editPickerItems))
		lines = append(lines, ui.StyleDim.Render(shown))
	}

	return lines
}

func (m Model) editFooter() string {
	dc := lipgloss.NewStyle().Foreground(ui.ColorHeader)

	if m.editSaving {
		return "  " + m.spinner.View() + " " +
			lipgloss.NewStyle().Foreground(ui.ColorParentAhead).Bold(true).Render("saving…")
	}

	if m.editFocused == editFocusParent {
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
		ui.StyleKeyHint.Render("enter") + dc.Render(" save  ") +
		ui.StyleKeyHint.Render("esc") + dc.Render(" cancel")
}

// focusedName returns the full branch name under the cursor, or "" if none.
func (m Model) focusedName() string {
	if len(m.filtered) == 0 || m.cursor >= len(m.filtered) {
		return ""
	}
	return m.filtered[m.cursor].branch.Name
}

// ── Footer info lines ──────────────────────────────────────────────────────

type footerInfoLine func(m Model) string

var footerInfoLines = []footerInfoLine{
	footerNamePin,
	footerBranchDesc,
	footerParentStatusDesc,
}

func footerNamePin(m Model) string {
	name := m.focusedName()
	if name == "" {
		return ""
	}
	label := ui.StyleDim.Render("  branch  ")
	value := lipgloss.NewStyle().Foreground(ui.ColorAccent).Bold(true).Italic(true).Render(name)
	return label + value
}

func footerBranchDesc(m Model) string {
	if len(m.filtered) == 0 || m.cursor >= len(m.filtered) {
		return ""
	}
	b := m.filtered[m.cursor].branch
	if b.Description == "" || b.IsRemote {
		return ""
	}
	label := ui.StyleDim.Render("  desc    ")
	value := lipgloss.NewStyle().Foreground(ui.ColorDesc).Italic(true).Render(b.Description)
	return label + value
}

func footerParentStatusDesc(m Model) string {
	if len(m.filtered) == 0 || m.cursor >= len(m.filtered) {
		return ""
	}
	b := m.filtered[m.cursor].branch
	if b.Parent == "" || b.IsRemote {
		return ""
	}

	parent := lipgloss.NewStyle().Foreground(ui.ColorAccent).Bold(true).Render(b.Parent)
	switch {
	case b.ParentAhead == 0:
		return "  " + lipgloss.NewStyle().Foreground(ui.ColorParentMerged).Bold(true).Render("✓") +
			ui.StyleDim.Render(" all commits merged into ") + parent
	case b.ParentBehind == 0:
		ahead := lipgloss.NewStyle().Foreground(ui.ColorParentAhead).Bold(true).Render(
			fmt.Sprintf("↑%d", b.ParentAhead),
		)
		return "  " + ahead + ui.StyleDim.Render(" ahead of ") + parent + ui.StyleDim.Render(" · ready to merge")
	default:
		ahead := lipgloss.NewStyle().Foreground(ui.ColorParentAhead).Bold(true).Render(
			fmt.Sprintf("↑%d", b.ParentAhead),
		)
		behind := lipgloss.NewStyle().Foreground(ui.ColorParentDiverged).Bold(true).Render(
			fmt.Sprintf("↓%d", b.ParentBehind),
		)
		return "  " + ahead + ui.StyleDim.Render(" ahead · ") + behind + ui.StyleDim.Render(" behind ") +
			parent + ui.StyleDim.Render(" — rebase needed")
	}
}

func (m Model) renderInfoLines() string {
	var lines []string
	for _, fn := range footerInfoLines {
		if l := fn(m); l != "" {
			lines = append(lines, l)
		}
	}
	if len(lines) == 0 {
		return ""
	}
	divider := "\n" + ui.StyleDivider.Render(strings.Repeat("─", m.width))
	return divider + "\n" + strings.Join(lines, "\n")
}

func (m Model) footer() string {
	info := m.renderInfoLines()
	filterPrompt := "  " + ui.StyleDim.Render("filter: ") + m.filterInput.View()
	hints := "  " + m.help.ShortHelpView(m.keys.ShortHelp())

	switch {
	case m.switching:
		return lipgloss.NewStyle().Foreground(ui.ColorParentAhead).Bold(true).Render("  " + m.spinner.View() + " switching branch…")

	case m.err != "":
		return "  " + ui.StyleError.Render("✗ "+m.err) + "\n" + filterPrompt + "\n" + hints + info

	default:
		return filterPrompt + "\n" + hints + info
	}
}

// SwitchedTo returns the branch name after a successful switch.
func (m Model) SwitchedTo() string { return m.switchedTo }

// ── Row rendering ──────────────────────────────────────────────────────────

func renderHeaders() string {
	sep := strings.Repeat(" ", colPad)
	nameH   := lipgloss.NewStyle().Width(colName).Render(ui.StyleDim.Render("Branch"))
	statusH := lipgloss.NewStyle().Width(colStatus).Render(ui.StyleDim.Render("vs parent"))
	descH   := lipgloss.NewStyle().Width(colDesc).Render(ui.StyleDim.Render("Description"))
	timeH   := ui.StyleDim.Render("Last commit")
	return "   " + sep + nameH + sep + statusH + sep + descH + sep + timeH
}

func parentStatusText(b git.Branch) (text string, style lipgloss.Style) {
	if b.Parent == "" || b.IsRemote {
		return padRight("", colStatus), lipgloss.NewStyle()
	}
	switch {
	case b.ParentAhead == 0:
		return padRight("✓ merged", colStatus),
			lipgloss.NewStyle().Foreground(ui.ColorParentMerged)
	case b.ParentBehind == 0:
		return padRight(fmt.Sprintf("↑%d", b.ParentAhead), colStatus),
			lipgloss.NewStyle().Foreground(ui.ColorParentAhead)
	default:
		text = fmt.Sprintf("↑%d ↓%d", b.ParentAhead, b.ParentBehind)
		return padRight(text, colStatus),
			lipgloss.NewStyle().Foreground(ui.ColorParentDiverged)
	}
}

func renderRow(item renderItem, isSelected bool, termWidth int) string {
	b := item.branch

	prefixW := len([]rune(item.treePrefix))
	nameW := colName - prefixW
	if nameW < 8 {
		nameW = 8
	}

	markerChar := " "
	switch {
	case b.IsCurrent:
		markerChar = "★"
	case isSelected:
		markerChar = "»"
	}

	nameText := padRight(truncate(b.Name, nameW), nameW)
	statusText, statusStyle := parentStatusText(b)
	descText := padRight(truncate(b.Description, colDesc), colDesc)
	timeText := b.RelTime

	sep := strings.Repeat(" ", colPad)

	if isSelected {
		bg := ui.ColorCursorBg
		bgSep   := lipgloss.NewStyle().Background(bg).Render(sep)
		markerS := lipgloss.NewStyle().Background(bg).Bold(true).Foreground(ui.ColorAccent).Render(markerChar)
		prefixS := lipgloss.NewStyle().Background(bg).Foreground(ui.ColorDim).Render(item.treePrefix)
		nameS   := lipgloss.NewStyle().Background(bg).Bold(true).Foreground(ui.ColorCursorFg).Width(nameW).Render(nameText)
		statusS := statusStyle.Background(bg).Width(colStatus).Render(statusText)
		descS   := lipgloss.NewStyle().Background(bg).Foreground(ui.ColorDesc).Italic(true).Width(colDesc).Render(descText)
		timeS   := lipgloss.NewStyle().Background(bg).Foreground(ui.ColorRelTime).Render(timeText)
		leftPad := lipgloss.NewStyle().Background(bg).Render("  ")
		used := 2 + colMarker + colPad + colName + colPad + colStatus + colPad + colDesc + colPad + len([]rune(timeText))
		trail := ""
		if termWidth > used {
			trail = lipgloss.NewStyle().Background(bg).Render(strings.Repeat(" ", termWidth-used))
		}
		return leftPad + markerS + bgSep + prefixS + nameS + bgSep + statusS + bgSep + descS + bgSep + timeS + trail
	}

	prefixS := ui.StyleTreeConnector.Render(item.treePrefix)
	var markerS, nameS string
	if b.IsCurrent {
		markerS = ui.StyleCurrentBranch.Render(markerChar)
		nameS   = ui.StyleCurrentBranch.Width(nameW).Render(nameText)
	} else if b.IsRemote {
		markerS = " "
		nameS   = ui.StyleRemoteName.Width(nameW).Render(nameText)
	} else {
		markerS = " "
		nameS   = lipgloss.NewStyle().Width(nameW).Render(nameText)
	}

	statusS := statusStyle.Width(colStatus).Render(statusText)
	descS   := ui.StyleDesc.Italic(true).Width(colDesc).Render(descText)
	timeS   := ui.StyleRelTime.Render(timeText)

	return "  " + markerS + sep + prefixS + nameS + sep + statusS + sep + descS + sep + timeS
}

// ── Git commands ───────────────────────────────────────────────────────────

func doSwitch(name string) tea.Cmd {
	return func() tea.Msg {
		return switchDoneMsg{err: git.SwitchBranch(name), name: name}
	}
}

func doSaveEdit(branch, parent, desc string) tea.Cmd {
	return func() tea.Msg {
		// Set or clear parent
		if parent == "" {
			if err := git.UnsetParent(branch); err != nil {
				return editSavedMsg{err: err, branch: branch}
			}
		} else {
			if err := git.SetParent(branch, parent); err != nil {
				return editSavedMsg{err: err, branch: branch}
			}
		}

		// Set or clear description
		if desc == "" {
			if err := git.UnsetDescription(branch); err != nil {
				return editSavedMsg{err: err, branch: branch}
			}
		} else {
			if err := git.SetDescription(branch, desc); err != nil {
				return editSavedMsg{err: err, branch: branch}
			}
		}

		var ahead, behind int
		if parent != "" {
			ahead, behind = git.ParentAheadBehind(branch, parent)
		}
		return editSavedMsg{branch: branch, parent: parent, desc: desc, ahead: ahead, behind: behind}
	}
}

// ── Helpers ────────────────────────────────────────────────────────────────

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

func padRight(s string, w int) string {
	r := []rune(s)
	if len(r) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-len(r))
}

// ── Tree builders ──────────────────────────────────────────────────────────

// buildRenderItems converts a flat branch list into a DFS-ordered []renderItem.
func buildRenderItems(branches []git.Branch) []renderItem {
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

			result = append(result, renderItem{
				branch:     b,
				treePrefix: prefix,
				depth:      depth,
			})

			next := make([]bool, depth+1)
			copy(next, isLastAtDepth)
			next[depth] = isLast
			dfs(b.Name, next)
		}
	}

	dfs("", []bool{})

	// Append remote branches flat
	for _, b := range branches {
		if b.IsRemote {
			result = append(result, renderItem{branch: b})
		}
	}

	return result
}

// buildEditPickerItems builds picker items for the edit form's parent picker.
// Only local branches are included; the branch being edited is excluded.
func buildEditPickerItems(branches []git.Branch, exclude string) []renderItem {
	nameToIdx := make(map[string]int)
	for i, b := range branches {
		if !b.IsRemote && b.Name != exclude {
			nameToIdx[b.Name] = i
		}
	}

	children := make(map[string][]git.Branch)
	for _, b := range branches {
		if b.IsRemote || b.Name == exclude {
			continue
		}
		if b.Parent != "" && b.Parent != exclude {
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

			result = append(result, renderItem{branch: b, treePrefix: prefix, depth: depth})

			next := make([]bool, depth+1)
			copy(next, isLastAtDepth)
			next[depth] = isLast
			dfs(b.Name, next)
		}
	}

	dfs("", []bool{})
	return result
}

func (m *Model) applyFilter() {
	q := strings.ToLower(m.filterInput.Value())
	if q == "" {
		m.filtered = m.treeItems
	} else {
		var out []renderItem
		for _, b := range m.branches {
			if strings.Contains(strings.ToLower(b.Name), q) {
				out = append(out, renderItem{branch: b})
			}
		}
		m.filtered = out
	}
	m.cursor = 0
	m.offset = 0

	vis := len(m.filtered)
	if vis > maxVisible {
		vis = maxVisible
	}
	if vis < 1 {
		vis = 1
	}
	m.visibleRows = vis
}

func (m *Model) clampScroll() {
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+m.visibleRows {
		m.offset = m.cursor - m.visibleRows + 1
	}
}

// restoreCursor moves the cursor to the branch with the given name in filtered,
// falling back to 0 if not found.
func (m *Model) restoreCursor(name string) {
	if name == "" {
		return
	}
	for i, item := range m.filtered {
		if item.branch.Name == name {
			m.cursor = i
			m.clampScroll()
			return
		}
	}
}
