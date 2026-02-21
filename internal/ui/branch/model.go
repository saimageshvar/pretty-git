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
	"github.com/sai/pretty-git-revamp/internal/git"
	ui "github.com/sai/pretty-git-revamp/internal/ui"
)

// ── Column widths ──────────────────────────────────────────────────────────

const (
	colMarker           = 1
	colName             = 32
	colStatus           = 12 // "✓ merged", "↑99 ↓99", etc.
	colDesc             = 25 // branch description (truncated)
	colPad              = 2
	maxVisible          = 15 // max rows shown at once (inline / fzf-style)
	maxParentCandidates = 4  // suggestion lines shown in set-parent mode
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
	filtering   bool
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

	// set-parent mode
	settingParent    bool
	parentInput      textinput.Model
	parentCandidates []string // filtered local branch names
	parentCursor     int
	parentOffset     int
	targetBranch     string // branch we're setting parent for

	// set-desc mode
	settingDesc      bool
	descInput        textinput.Model
	targetDescBranch string // branch we're setting description for

	spinner spinner.Model
}

type switchDoneMsg struct {
	err  error
	name string
}

type parentSetMsg struct {
	err    error
	child  string
	parent string // "" means parent was cleared
	ahead  int
	behind int
}

type descSetMsg struct {
	err    error
	branch string
	desc   string // "" means description was cleared
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

	// ── textinput ──────────────────────────────────────────────────────────
	ti := textinput.New()
	ti.Prompt = ""
	ti.PromptStyle = lipgloss.NewStyle()
	ti.TextStyle = lipgloss.NewStyle().Foreground(ui.ColorHeader)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(ui.ColorDim)
	ti.Placeholder = "type to filter…"
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(ui.ColorAccent)

	// ── parentInput ────────────────────────────────────────────────────────
	pi := textinput.New()
	pi.Prompt = ""
	pi.PromptStyle = lipgloss.NewStyle()
	pi.TextStyle = lipgloss.NewStyle().Foreground(ui.ColorHeader)
	pi.PlaceholderStyle = lipgloss.NewStyle().Foreground(ui.ColorDim)
	pi.Placeholder = "type to filter branches…"
	pi.Cursor.Style = lipgloss.NewStyle().Foreground(ui.ColorAccent)

	// ── descInput ──────────────────────────────────────────────────────────
	di := textinput.New()
	di.Prompt = ""
	di.PromptStyle = lipgloss.NewStyle()
	di.TextStyle = lipgloss.NewStyle().Foreground(ui.ColorHeader)
	di.PlaceholderStyle = lipgloss.NewStyle().Foreground(ui.ColorDim)
	di.Placeholder = "short description…"
	di.Cursor.Style = lipgloss.NewStyle().Foreground(ui.ColorAccent)
	di.CharLimit = 120

	// ── help ───────────────────────────────────────────────────────────────
	h := help.New()
	h.Styles.ShortKey = lipgloss.NewStyle().Bold(true).Foreground(ui.ColorKeyHint)
	h.Styles.ShortDesc = lipgloss.NewStyle().Foreground(ui.ColorHeader)
	h.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(ui.ColorTreeConnector)
	h.Width = termWidth

	sp := spinner.New()
	sp.Spinner = spinner.MiniDot
	sp.Style = lipgloss.NewStyle().Bold(true).Foreground(ui.ColorAccent)

	return Model{
		branches:    branches,
		treeItems:   treeItems,
		filtered:    treeItems,
		width:       termWidth,
		visibleRows: vis,
		repoName:    repoName,
		localCount:  local,
		remoteCount: remote,
		filterInput: ti,
		parentInput: pi,
		descInput:   di,
		help:        h,
		keys:        defaultKeyMap(),
		spinner:     sp,
	}
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

	case parentSetMsg:
		m.settingParent = false
		m.parentInput.Blur()
		m.parentInput.Reset()
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		// Update the branch in-memory so the tree reflects the new parent immediately.
		for i := range m.branches {
			if m.branches[i].Name == msg.child {
				m.branches[i].Parent = msg.parent
				m.branches[i].ParentAhead = msg.ahead
				m.branches[i].ParentBehind = msg.behind
				break
			}
		}
		m.treeItems = buildRenderItems(m.branches)
		m.filtered = m.treeItems
		m.applyFilter()
		return m, nil

	case descSetMsg:
		m.settingDesc = false
		m.descInput.Blur()
		m.descInput.Reset()
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		// Update the branch in-memory so the footer reflects the new desc immediately.
		for i := range m.branches {
			if m.branches[i].Name == msg.branch {
				m.branches[i].Description = msg.desc
				break
			}
		}
		m.treeItems = buildRenderItems(m.branches)
		m.filtered = m.treeItems
		m.applyFilter()
		return m, nil

	case tea.KeyMsg:
		if m.settingDesc {
			return m.updateSetDesc(msg)
		}
		if m.settingParent {
			return m.updateSetParent(msg)
		}
		if m.filtering {
			return m.updateFilter(msg)
		}
		return m.updateNormal(msg)
	}
	return m, nil
}

func (m Model) updateNormal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
			m.clampScroll()
		}
	case key.Matches(msg, m.keys.Down):
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
			m.clampScroll()
		}
	case key.Matches(msg, m.keys.Filter):
		m.filtering = true
		m.err = ""
		return m, m.filterInput.Focus()
	case key.Matches(msg, m.keys.SetParent):
		if len(m.filtered) == 0 {
			return m, nil
		}
		b := m.filtered[m.cursor].branch
		if b.IsRemote {
			return m, nil
		}
		m.targetBranch = b.Name
		m.settingParent = true
		m.err = ""
		// Always start with an empty filter input; sentinel provides the clear option.
		m.parentInput.Reset()
		m.buildParentCandidates()
		return m, m.parentInput.Focus()

	case key.Matches(msg, m.keys.SetDesc):
		if len(m.filtered) == 0 {
			return m, nil
		}
		b := m.filtered[m.cursor].branch
		if b.IsRemote {
			return m, nil
		}
		m.targetDescBranch = b.Name
		m.settingDesc = true
		m.err = ""
		// Pre-fill with existing description so user can edit rather than retype.
		m.descInput.SetValue(b.Description)
		return m, m.descInput.Focus()

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
	return m, nil
}

func (m Model) updateFilter(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Clear):
		m.filtering = false
		m.filterInput.Blur()
		// Save cursor position by branch name before clearing
		var savedName string
		if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
			savedName = m.filtered[m.cursor].branch.Name
		}
		m.filterInput.Reset()
		m.applyFilter()
		// Restore cursor to same branch in tree view
		if savedName != "" {
			for i, item := range m.filtered {
				if item.branch.Name == savedName {
					m.cursor = i
					m.clampScroll()
					break
				}
			}
		}
		return m, nil

	case key.Matches(msg, m.keys.Quit):
		m.quitting = true
		return m, tea.Quit

	case key.Matches(msg, m.keys.Confirm):
		if len(m.filtered) > 0 {
			b := m.filtered[m.cursor].branch
			if !b.IsCurrent {
				m.filtering = false
				m.filterInput.Blur()
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
		}
		m.filtering = false
		m.filterInput.Blur()
		return m, nil

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
	}

	// All other keys go to the textinput
	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	m.applyFilter()
	return m, cmd
}

func (m Model) updateSetParent(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Clear):
		if m.parentInput.Value() != "" {
			// Clear filter, stay in set-parent mode.
			m.parentInput.Reset()
			m.buildParentCandidates()
			return m, nil
		}
		// Already empty — exit set-parent mode.
		m.settingParent = false
		m.parentInput.Blur()
		m.parentCandidates = nil
		m.parentCursor = 0
		m.parentOffset = 0
		return m, nil

	case key.Matches(msg, m.keys.Quit):
		m.quitting = true
		return m, tea.Quit

	case key.Matches(msg, m.keys.ClearParent):
		child := m.targetBranch
		m.settingParent = false
		m.parentInput.Blur()
		m.parentInput.Reset()
		m.parentCandidates = nil
		m.parentCursor = 0
		m.parentOffset = 0
		return m, doSetParent(child, "")

	case key.Matches(msg, m.keys.Confirm):
		chosen := ""
		if len(m.parentCandidates) > 0 {
			chosen = m.parentCandidates[m.parentCursor]
		}
		child := m.targetBranch
		return m, doSetParent(child, chosen)

	case key.Matches(msg, m.keys.Up):
		if m.parentCursor > 0 {
			m.parentCursor--
			m.clampParentScroll()
		}
		return m, nil

	case key.Matches(msg, m.keys.Down):
		if m.parentCursor < len(m.parentCandidates)-1 {
			m.parentCursor++
			m.clampParentScroll()
		}
		return m, nil
	}

	// All other keys go to the parent textinput; rebuild candidates on change.
	prev := m.parentInput.Value()
	var cmd tea.Cmd
	m.parentInput, cmd = m.parentInput.Update(msg)
	if m.parentInput.Value() != prev {
		m.buildParentCandidates()
	}
	return m, cmd
}

func (m Model) updateSetDesc(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Clear):
		m.settingDesc = false
		m.descInput.Blur()
		m.descInput.Reset()
		return m, nil

	case key.Matches(msg, m.keys.Quit):
		m.quitting = true
		return m, tea.Quit

	case key.Matches(msg, m.keys.Confirm):
		desc := strings.TrimSpace(m.descInput.Value())
		branch := m.targetDescBranch
		return m, doSetDesc(branch, desc)
	}

	var cmd tea.Cmd
	m.descInput, cmd = m.descInput.Update(msg)
	return m, cmd
}

// buildParentCandidates filters local branch names (excluding targetBranch)
// by the current parentInput value and resets the candidate cursor.
func (m *Model) buildParentCandidates() {
	q := strings.ToLower(m.parentInput.Value())
	var candidates []string
	for _, b := range m.branches {
		if b.IsRemote || b.Name == m.targetBranch {
			continue
		}
		if q == "" || strings.Contains(strings.ToLower(b.Name), q) {
			candidates = append(candidates, b.Name)
		}
	}
	m.parentCandidates = candidates
	m.parentCursor = 0
	m.parentOffset = 0
}

func (m *Model) clampParentScroll() {
	vis := maxParentCandidates
	if m.parentCursor < m.parentOffset {
		m.parentOffset = m.parentCursor
	}
	if m.parentCursor >= m.parentOffset+vis {
		m.parentOffset = m.parentCursor - vis + 1
	}
}

func (m *Model) applyFilter() {
	q := strings.ToLower(m.filterInput.Value())
	if q == "" {
		// Restore full tree view
		m.filtered = m.treeItems
	} else {
		// Flat filter mode: strip tree connectors, show plain sorted matches
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

	// Recompute visible rows for new list length
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
	// Keep cursor visible within [offset, offset+visibleRows)
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+m.visibleRows {
		m.offset = m.cursor - m.visibleRows + 1
	}
}

// ── View ───────────────────────────────────────────────────────────────────

func (m Model) View() string {
	if m.done || m.quitting {
		return ""
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

// focusedName returns the full branch name under the cursor, or "" if none.
func (m Model) focusedName() string {
	if len(m.filtered) == 0 || m.cursor >= len(m.filtered) {
		return ""
	}
	return m.filtered[m.cursor].branch.Name
}

// ── Footer info lines ──────────────────────────────────────────────────────
//
// footerInfoLine is a function that produces one contextual footer line.
// Return "" to omit the line entirely. Reorder or remove entries in
// footerInfoLines to change what appears below the primary hint/state row.

type footerInfoLine func(m Model) string

// footerInfoLines is the ordered list of contextual lines rendered below the
// primary state row (key hints / filter prompt / error). Edit this slice to
// add, remove, or reorder footer info items.
var footerInfoLines = []footerInfoLine{
	footerNamePin,
	footerBranchDesc,
	footerParentStatusDesc,
}

// footerNamePin shows the full branch name of the focused row, solving the
// truncation problem in deep tree paths.
func footerNamePin(m Model) string {
	name := m.focusedName()
	if name == "" {
		return ""
	}
	label := ui.StyleDim.Render("  branch  ")
	value := lipgloss.NewStyle().Foreground(ui.ColorAccent).Bold(true).Italic(true).Render(name)
	return label + value
}

// footerBranchDesc shows the pgit-desc of the focused branch, if set.
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

// footerParentStatusDesc shows a human-readable description of the focused
// branch's relationship to its parent branch.
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

// renderInfoLines collects all non-empty footer info lines into one string.
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
	return "\n" + strings.Join(lines, "\n")
}

// renderParentSuggestions renders the candidate list for set-parent mode.
// Up to maxParentCandidates lines are shown; the selected row is highlighted.
func (m Model) renderParentSuggestions() string {
	var sb strings.Builder
	header := lipgloss.NewStyle().Foreground(ui.ColorKeyHint).Bold(true).Render("  ⎇ set parent of ") +
		lipgloss.NewStyle().Foreground(ui.ColorAccent).Bold(true).Italic(true).Render(m.targetBranch)
	sb.WriteString(header + "\n")

	if len(m.parentCandidates) == 0 {
		sb.WriteString(ui.StyleDim.Render("    (no matches)"))
		return sb.String()
	}

	end := m.parentOffset + maxParentCandidates
	if end > len(m.parentCandidates) {
		end = len(m.parentCandidates)
	}
	for i := m.parentOffset; i < end; i++ {
		c := m.parentCandidates[i]
		if i == m.parentCursor {
			row := lipgloss.NewStyle().
				Background(ui.ColorCursorBg).
				Foreground(ui.ColorCursorFg).
				Bold(true).
				Width(m.width - 4).
				Render("  › " + c)
			sb.WriteString("  " + row + "\n")
		} else {
			sb.WriteString("    " + lipgloss.NewStyle().Foreground(ui.ColorHeader).Render(c) + "\n")
		}
	}
	return strings.TrimRight(sb.String(), "\n")
}

func (m Model) footer() string {
	info := m.renderInfoLines()

	switch {
	case m.settingDesc:
		header := lipgloss.NewStyle().Foreground(ui.ColorKeyHint).Bold(true).Render("  "+m.spinner.View()+" set desc of ") +
			lipgloss.NewStyle().Foreground(ui.ColorAccent).Bold(true).Italic(true).Render(m.targetDescBranch)
		prompt := ui.StyleKeyHint.Render("d") +
			ui.StyleDim.Render(" desc: ") +
			m.descInput.View()
		dc := lipgloss.NewStyle().Foreground(ui.ColorHeader)
		hint := "  " +
			ui.StyleKeyHint.Render("enter") + dc.Render(" save  ") +
			ui.StyleKeyHint.Render("esc") + dc.Render(" cancel")
		return header + "\n  " + prompt + hint + info

	case m.settingParent:
		suggestions := m.renderParentSuggestions()
		prompt := lipgloss.NewStyle().Foreground(ui.ColorKeyHint).Bold(true).Render(m.spinner.View()) +
			ui.StyleKeyHint.Render(" p") +
			ui.StyleDim.Render(" parent: ") +
			m.parentInput.View()
		desc := lipgloss.NewStyle().Foreground(ui.ColorHeader)
		hint := "  " +
			ui.StyleKeyHint.Render("↑/↓") + desc.Render(" navigate  ") +
			ui.StyleKeyHint.Render("enter") + desc.Render(" confirm  ") +
			ui.StyleKeyHint.Render("ctrl+d") + desc.Render(" unset parent  ") +
			ui.StyleKeyHint.Render("esc") + desc.Render(" cancel")
		return suggestions + "\n  " + prompt + hint + info

	case m.filtering:
		prompt := ui.StyleKeyHint.Render("/") +
			ui.StyleDim.Render(" filter: ") +
			m.filterInput.View()
		hint := ui.StyleDim.Render("  " + m.help.ShortHelpView(m.keys.filterShortHelp()))
		return "  " + prompt + hint + info

	case m.switching:
		return lipgloss.NewStyle().Foreground(ui.ColorParentAhead).Bold(true).Render("  " + m.spinner.View() + " switching branch…")

	case m.err != "":
		hints := "  " + m.help.ShortHelpView(m.keys.ShortHelp())
		return "  " + ui.StyleError.Render("✗ "+m.err) + "\n" + hints + info

	default:
		return "  " + m.help.ShortHelpView(m.keys.ShortHelp()) + info
	}
}

// SwitchedTo returns the branch name after a successful switch.
func (m Model) SwitchedTo() string { return m.switchedTo }

// ── Row rendering ──────────────────────────────────────────────────────────

// renderHeaders returns a dim column header row aligned to the data columns.
func renderHeaders() string {
	sep := strings.Repeat(" ", colPad)
	nameH   := lipgloss.NewStyle().Width(colName).Render(ui.StyleDim.Render("Branch"))
	statusH := lipgloss.NewStyle().Width(colStatus).Render(ui.StyleDim.Render("vs parent"))
	descH   := lipgloss.NewStyle().Width(colDesc).Render(ui.StyleDim.Render("Description"))
	timeH   := ui.StyleDim.Render("Last commit")
	return "   " + sep + nameH + sep + statusH + sep + descH + sep + timeH
}

// parentStatusText returns the display text and its lipgloss style for the
// parent-status column. Raw text is returned separately so callers can apply
// a background style on the cursor row.
func parentStatusText(b git.Branch) (text string, style lipgloss.Style) {
	if b.Parent == "" || b.IsRemote {
		return padRight("", colStatus), lipgloss.NewStyle()
	}
	switch {
	case b.ParentAhead == 0:
		// All commits merged into parent (or nothing committed yet).
		return padRight("✓ merged", colStatus),
			lipgloss.NewStyle().Foreground(ui.ColorParentMerged)
	case b.ParentBehind == 0:
		// Ahead of parent, parent hasn't moved — clean, just unmerged work.
		return padRight(fmt.Sprintf("↑%d", b.ParentAhead), colStatus),
			lipgloss.NewStyle().Foreground(ui.ColorParentAhead)
	default:
		// Diverged: ahead AND parent has new commits — warning orange.
		text = fmt.Sprintf("↑%d ↓%d", b.ParentAhead, b.ParentBehind)
		return padRight(text, colStatus),
			lipgloss.NewStyle().Foreground(ui.ColorParentDiverged)
	}
}

func renderRow(item renderItem, isSelected bool, termWidth int) string {
	b := item.branch

	// Name column width shrinks by the visual width of the tree prefix (rune count).
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

	nameText   := padRight(truncate(b.Name, nameW), nameW)
	statusText, statusStyle := parentStatusText(b)
	descText   := padRight(truncate(b.Description, colDesc), colDesc)
	timeText   := b.RelTime

	sep := strings.Repeat(" ", colPad)

	if isSelected {
		bg := ui.ColorCursorBg
		bgSep    := lipgloss.NewStyle().Background(bg).Render(sep)
		markerS  := lipgloss.NewStyle().Background(bg).Bold(true).Foreground(ui.ColorAccent).Render(markerChar)
		prefixS  := lipgloss.NewStyle().Background(bg).Foreground(ui.ColorDim).Render(item.treePrefix)
		nameS    := lipgloss.NewStyle().Background(bg).Bold(true).Foreground(ui.ColorCursorFg).Width(nameW).Render(nameText)
		statusS  := statusStyle.Background(bg).Width(colStatus).Render(statusText)
		descS    := lipgloss.NewStyle().Background(bg).Foreground(ui.ColorDesc).Italic(true).Width(colDesc).Render(descText)
		timeS    := lipgloss.NewStyle().Background(bg).Foreground(ui.ColorRelTime).Render(timeText)
		leftPad  := lipgloss.NewStyle().Background(bg).Render("  ")
		used := 2 + colMarker + colPad + colName + colPad + colStatus + colPad + colDesc + colPad + len([]rune(timeText))
		trail := ""
		if termWidth > used {
			trail = lipgloss.NewStyle().Background(bg).Render(strings.Repeat(" ", termWidth-used))
		}
		return leftPad + markerS + bgSep + prefixS + nameS + bgSep + statusS + bgSep + descS + bgSep + timeS + trail
	}

	// Normal row
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

func doSetParent(child, parent string) tea.Cmd {
	return func() tea.Msg {
		var err error
		if parent == "" || parent == "(none — clear parent)" {
			err = git.UnsetParent(child)
			return parentSetMsg{err: err, child: child, parent: ""}
		}
		err = git.SetParent(child, parent)
		if err != nil {
			return parentSetMsg{err: err, child: child, parent: parent}
		}
		ahead, behind := git.ParentAheadBehind(child, parent)
		return parentSetMsg{child: child, parent: parent, ahead: ahead, behind: behind}
	}
}

func doSetDesc(branch, desc string) tea.Cmd {
	return func() tea.Msg {
		var err error
		if desc == "" {
			err = git.UnsetDescription(branch)
		} else {
			err = git.SetDescription(branch, desc)
		}
		return descSetMsg{err: err, branch: branch, desc: desc}
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

// ── Tree builder ───────────────────────────────────────────────────────────

// buildRenderItems converts a flat branch list into a DFS-ordered []renderItem.
// Each local branch's Parent field is used to build the hierarchy; branches
// with no parent (or an unknown parent) become roots. Remote branches are
// appended flat after the local tree with no connectors.
func buildRenderItems(branches []git.Branch) []renderItem {
	// Index local branch names for parent look-up
	nameToIdx := make(map[string]int)
	for i, b := range branches {
		if !b.IsRemote {
			nameToIdx[b.Name] = i
		}
	}

	// Build children map: "" = roots, branchName = children of that branch
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

	// DFS from virtual root, building box-drawing connector strings
	var dfs func(parentName string, isLastAtDepth []bool)
	dfs = func(parentName string, isLastAtDepth []bool) {
		kids := children[parentName]
		for i, b := range kids {
			isLast := i == len(kids)-1
			depth := len(isLastAtDepth)

			// Build raw (no ANSI) connector string
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

	// Append remote branches flat (no tree connectors)
	for _, b := range branches {
		if b.IsRemote {
			result = append(result, renderItem{branch: b})
		}
	}

	return result
}
