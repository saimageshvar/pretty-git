package branch

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sai/pretty-git-revamp/internal/git"
	ui "github.com/sai/pretty-git-revamp/internal/ui"
)

// ── Column widths ──────────────────────────────────────────────────────────

const (
	colMarker  = 1
	colName    = 32
	colHash    = 7
	colTime    = 13
	colPad     = 2
	maxVisible = 15 // max rows shown at once (inline / fzf-style)
)

func subjectWidth(termWidth int) int {
	// 2(indent) + marker + pad + name + pad + hash + pad + subject + pad + time
	fixed := 2 + colMarker + colPad + colName + colPad + colHash + colPad + colPad + colTime
	w := termWidth - fixed
	if w < 15 {
		return 15
	}
	if w > 55 {
		return 55
	}
	return w
}

// ── renderItem ─────────────────────────────────────────────────────────────

// renderItem wraps a Branch with pre-computed tree display metadata.
type renderItem struct {
	branch     git.Branch
	treePrefix string // pre-colored ANSI string, e.g. "│  ├─ "
	depth      int    // 0 = root
}

// ── Model ──────────────────────────────────────────────────────────────────

type Model struct {
	branches    []git.Branch   // raw, unchanged
	treeItems   []renderItem   // full tree, built once at startup
	filtered    []renderItem   // active list (tree mode or flat filter mode)
	cursor      int
	offset      int
	filter      string
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
}

type switchDoneMsg struct {
	err  error
	name string
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

	return Model{
		branches:    branches,
		treeItems:   treeItems,
		filtered:    treeItems,
		width:       termWidth,
		visibleRows: vis,
		repoName:    repoName,
		localCount:  local,
		remoteCount: remote,
	}
}

func (m Model) Init() tea.Cmd { return nil }

// ── Update ─────────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
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

	case tea.KeyMsg:
		if m.filtering {
			return m.updateFilter(msg)
		}
		return m.updateNormal(msg)
	}
	return m, nil
}

func (m Model) updateNormal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			m.clampScroll()
		}
	case "down", "j":
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
			m.clampScroll()
		}
	case "/":
		m.filtering = true
		m.err = ""
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "enter":
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
	switch msg.String() {
	case "esc":
		m.filtering = false
		var savedName string
		if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
			savedName = m.filtered[m.cursor].branch.Name
		}
		m.filter = ""
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
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "enter":
		// Confirm filter and switch to navigation mode
		if len(m.filtered) > 0 {
			b := m.filtered[m.cursor].branch
			if !b.IsCurrent {
				m.filtering = false
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
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			m.clampScroll()
		}
	case "down", "j":
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
			m.clampScroll()
		}
	case "backspace", "ctrl+h":
		if len(m.filter) > 0 {
			m.filter = string([]rune(m.filter)[:len([]rune(m.filter))-1])
			m.applyFilter()
		}
	default:
		if len(msg.Runes) == 1 {
			m.filter += string(msg.Runes)
			m.applyFilter()
		}
	}
	return m, nil
}

func (m *Model) applyFilter() {
	if m.filter == "" {
		// Restore full tree view
		m.filtered = m.treeItems
	} else {
		// Flat filter mode: strip tree connectors, show plain sorted matches
		q := strings.ToLower(m.filter)
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

func (m Model) footer() string {
	hints := "  " +
		ui.StyleKeyHint.Render("↑↓") + ui.StyleDim.Render(" navigate") +
		ui.StyleDim.Render("   ") +
		ui.StyleKeyHint.Render("enter") + ui.StyleDim.Render(" switch") +
		ui.StyleDim.Render("   ") +
		ui.StyleKeyHint.Render("/") + ui.StyleDim.Render(" filter") +
		ui.StyleDim.Render("   ") +
		ui.StyleKeyHint.Render("q") + ui.StyleDim.Render(" quit")

	switch {
	case m.filtering:
		prompt := ui.StyleKeyHint.Render("/") + ui.StyleDim.Render(" filter: ") +
			lipgloss.NewStyle().Foreground(ui.ColorHeader).Render(m.filter) +
			lipgloss.NewStyle().Foreground(ui.ColorAccent).Render("█") // cursor
		hint := ui.StyleDim.Render("  esc clear · enter confirm")
		return "  " + prompt + hint
	case m.switching:
		return ui.StyleDim.Render("  switching branch…")
	case m.err != "":
		return "  " + ui.StyleError.Render("✗ "+m.err) + "\n" + hints
	default:
		return hints
	}
}

// SwitchedTo returns the branch name after a successful switch.
func (m Model) SwitchedTo() string { return m.switchedTo }

// ── Row rendering ──────────────────────────────────────────────────────────

func renderRow(item renderItem, isSelected bool, termWidth int) string {
	b := item.branch
	subjectW := subjectWidth(termWidth)

	// Name column width shrinks by the visual width of the tree prefix (raw rune count).
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
		markerChar = "›"
	}

	nameText := truncate(b.Name, nameW)
	if b.Ahead > 0 || b.Behind > 0 {
		ann := ""
		if b.Ahead > 0 {
			ann += fmt.Sprintf("↑%d", b.Ahead)
		}
		if b.Behind > 0 {
			ann += fmt.Sprintf("↓%d", b.Behind)
		}
		max := nameW - len(ann) - 1
		if max < 8 {
			max = 8
		}
		nameText = truncate(b.Name, max) + " " + ann
	}
	nameText    = padRight(nameText, nameW)
	hashText    := padRight(b.ShortHash, colHash)
	subjectText := truncate(b.Subject, subjectW)
	timeText    := b.RelTime

	sep := strings.Repeat(" ", colPad)

	if isSelected {
		bg := ui.ColorCursorBg
		bgSep   := lipgloss.NewStyle().Background(bg).Render(sep)
		markerS := lipgloss.NewStyle().Background(bg).Bold(true).Foreground(ui.ColorAccent).Render(markerChar)
		prefixS := lipgloss.NewStyle().Background(bg).Foreground(ui.ColorDim).Render(item.treePrefix)
		nameS   := lipgloss.NewStyle().Background(bg).Bold(true).Foreground(ui.ColorCursorFg).Width(nameW).Render(nameText)
		hashS   := lipgloss.NewStyle().Background(bg).Foreground(ui.ColorHash).Width(colHash).Render(hashText)
		subjS   := lipgloss.NewStyle().Background(bg).Foreground(ui.ColorCursorFg).Width(subjectW).Render(subjectText)
		timeS   := lipgloss.NewStyle().Background(bg).Foreground(ui.ColorRelTime).Render(timeText)
		leftPad := lipgloss.NewStyle().Background(bg).Render("  ")
		// prefixW + nameW == colName, so total used width is unchanged
		used := 2 + colMarker + colPad + colName + colPad + colHash + colPad + subjectW + colPad + len([]rune(timeText))
		trail := ""
		if termWidth > used {
			trail = lipgloss.NewStyle().Background(bg).Render(strings.Repeat(" ", termWidth-used))
		}
		return leftPad + markerS + bgSep + prefixS + nameS + bgSep + hashS + bgSep + subjS + bgSep + timeS + trail
	}

	// Normal row
	prefixS := ui.StyleTreeConnector.Render(item.treePrefix)
	var markerS, nameS string
	if b.IsCurrent {
		markerS = ui.StyleCurrentBranch.Render(markerChar)
		nameS   = ui.StyleCurrentBranch.Copy().Width(nameW).Render(nameText)
	} else if b.IsRemote {
		markerS = " "
		nameS   = ui.StyleRemoteName.Copy().Width(nameW).Render(nameText)
	} else {
		markerS = " "
		nameS   = lipgloss.NewStyle().Width(nameW).Render(nameText)
	}

	hashS    := ui.StyleHash.Render(hashText)
	subjectS := ui.StyleSubject.Copy().Width(subjectW).Render(subjectText)
	timeS    := ui.StyleRelTime.Render(timeText)

	return "  " + markerS + sep + prefixS + nameS + sep + hashS + sep + subjectS + sep + timeS
}

// ── Git command ────────────────────────────────────────────────────────────

func doSwitch(name string) tea.Cmd {
	return func() tea.Msg {
		return switchDoneMsg{err: git.SwitchBranch(name), name: name}
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


