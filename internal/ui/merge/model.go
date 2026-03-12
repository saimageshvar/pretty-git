package merge

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

// ── Constants ──────────────────────────────────────────────────────────────

const (
	colMarker   = 1
	colNameMin  = 20
	colNameMax  = 60
	colDescMax  = 55
	colStatus   = 12
	colPad      = 2
	maxVisible  = 15
)

// ── renderItem ─────────────────────────────────────────────────────────────

type renderItem struct {
	branch     git.Branch
	treePrefix string
	depth      int
}

// ── Model ──────────────────────────────────────────────────────────────────

type Model struct {
	branches    []git.Branch
	treeItems   []renderItem
	filtered    []renderItem
	cursor      int
	offset      int
	err         string
	merging     bool
	done        bool
	quitting    bool
	mergedTo    string
	mergedFrom  string
	width       int
	visibleRows int
	repoName    string
	localCount  int
	remoteCount int

	filterInput textinput.Model
	help        help.Model
	keys        keyMap
	spinner     spinner.Model

	// Modal state
	showModal       bool
	showConflictModal bool
	selectedBranch  string
	currentBranch   string
	filesChanged    int
	lastCommits     []git.Commit
	loadingPreview  bool
	conflicts       []string
}

type mergeDoneMsg struct {
	err          error
	targetBranch string
	sourceBranch string
	conflicts    []string
}

type previewLoadedMsg struct {
	filesChanged int
	commits      []git.Commit
	err          error
}

// New creates a new merge model.
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
	if termHeight > 5 && vis > termHeight-5 {
		vis = termHeight - 5
	}
	if vis < 1 {
		vis = 1
	}

	// Filter input
	ti := textinput.New()
	ti.Prompt = ""
	ti.PromptStyle = lipgloss.NewStyle()
	ti.TextStyle = lipgloss.NewStyle().Foreground(ui.ColorHeader)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(ui.ColorDim)
	ti.Placeholder = "type to filter…"
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(ui.ColorAccent)
	_ = ti.Focus()

	// Help
	h := help.New()
	h.Styles.ShortKey = lipgloss.NewStyle().Bold(true).Foreground(ui.ColorKeyHint)
	h.Styles.ShortDesc = lipgloss.NewStyle().Foreground(ui.ColorHeader)
	h.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(ui.ColorTreeConnector)
	h.Width = termWidth

	// Spinner
	sp := spinner.New()
	sp.Spinner = spinner.MiniDot
	sp.Style = lipgloss.NewStyle().Bold(true).Foreground(ui.ColorAccent)

	// Start with cursor on current branch
	initialCursor := 0
	for i, item := range treeItems {
		if item.branch.IsCurrent {
			initialCursor = i
			break
		}
	}

	return Model{
		branches:    branches,
		treeItems:   treeItems,
		filtered:    treeItems,
		cursor:      initialCursor,
		width:       termWidth,
		visibleRows: vis,
		repoName:    repoName,
		localCount:  local,
		remoteCount: remote,
		filterInput: ti,
		help:        h,
		keys:        defaultKeyMap(),
		spinner:     sp,
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

	case previewLoadedMsg:
		m.loadingPreview = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.filesChanged = msg.filesChanged
		m.lastCommits = msg.commits
		m.showModal = true
		return m, nil

	case mergeDoneMsg:
		m.merging = false
		if msg.err != nil {
			if len(msg.conflicts) > 0 {
				// Conflict case - show conflict modal
				m.conflicts = msg.conflicts
				m.showModal = false
				m.showConflictModal = true
			} else {
				// Other error
				m.err = msg.err.Error()
				m.showModal = false
			}
			return m, nil
		}
		m.mergedTo = msg.targetBranch
		m.mergedFrom = msg.sourceBranch
		m.done = true
		return m, tea.Quit

	case tea.KeyMsg:
		if m.showConflictModal {
			return m.updateConflictModal(msg)
		}
		if m.showModal {
			return m.updateModal(msg)
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

	case key.Matches(msg, m.keys.Select):
		if len(m.filtered) == 0 || m.loadingPreview {
			return m, nil
		}
		b := m.filtered[m.cursor].branch
		if b.IsCurrent {
			m.err = "cannot merge branch into itself"
			return m, nil
		}
		m.selectedBranch = b.Name
		m.currentBranch = git.CurrentBranch()
		m.loadingPreview = true
		m.err = ""
		return m, loadPreview(m.currentBranch, b.Name)

	case key.Matches(msg, m.keys.EscBack):
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
	}

	// All other keys go to filter input
	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	m.applyFilter()
	return m, cmd
}

func (m Model) updateModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Confirm), key.Matches(msg, m.keys.Select):
		m.showModal = false
		m.merging = true
		m.err = ""
		return m, doMerge(m.selectedBranch, m.currentBranch)

	case key.Matches(msg, m.keys.Cancel), key.Matches(msg, m.keys.EscBack):
		m.showModal = false
		m.selectedBranch = ""
		return m, nil
	}
	return m, nil
}

func (m Model) updateConflictModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Select), key.Matches(msg, m.keys.EscBack):
		m.showConflictModal = false
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
}

// ── View ───────────────────────────────────────────────────────────────────

func (m Model) View() string {
	if m.done || m.quitting {
		return ""
	}

	if m.showConflictModal {
		return m.viewConflictModal()
	}

	if m.showModal {
		return m.viewModal()
	}

	var sb strings.Builder

	// Header
	badge := ui.StyleCountBadge.Render(fmt.Sprintf("%d local · %d remote", m.localCount, m.remoteCount))
	sb.WriteString(ui.StyleHeader.Render("  Merge into") + "  " +
		ui.StyleAccent.Render(m.repoName) + "  " + badge + "\n")
	sb.WriteString(ui.StyleDivider.Render(strings.Repeat("─", m.width)) + "\n")

	lw := m.listWidth()
	sb.WriteString(m.renderColHeaders(lw) + "\n")
	if len(m.filtered) == 0 {
		sb.WriteString(ui.StyleDim.Render("  no branches match\n"))
	} else {
		sb.WriteString(m.renderBody(lw))
	}

	// Footer
	sb.WriteString(ui.StyleDivider.Render(strings.Repeat("─", m.width)) + "\n")
	sb.WriteString(m.footer())

	return sb.String()
}

func (m Model) viewModal() string {
	var sb strings.Builder

	// Modal dimensions
	modalW := 60
	if m.width < modalW+4 {
		modalW = m.width - 4
	}
	if modalW < 30 {
		modalW = 30
	}

	// Header
	title := fmt.Sprintf("Merge %s into %s?", m.currentBranch, m.selectedBranch)
	sb.WriteString(ui.StyleHeader.Render("  "+title) + "\n")
	sb.WriteString(ui.StyleDivider.Render(strings.Repeat("─", modalW)) + "\n\n")

	// Files changed
	sb.WriteString(ui.StyleDim.Render("  Files changed: ") +
		lipgloss.NewStyle().Foreground(ui.ColorAccent).Bold(true).Render(fmt.Sprintf("%d", m.filesChanged)) + "\n\n")

	// Last commits
	sb.WriteString(ui.StyleDim.Render("  Last commits from " + m.currentBranch + ":") + "\n")
	for _, c := range m.lastCommits {
		line := fmt.Sprintf("    %s %s", c.ShortHash, c.Subject)
		if len(line) > modalW-2 {
			line = line[:modalW-5] + "…"
		}
		sb.WriteString(lipgloss.NewStyle().Foreground(ui.ColorHeader).Render(line) + "\n")
	}

	sb.WriteString("\n")

	// Buttons
	sb.WriteString(ui.StyleDivider.Render(strings.Repeat("─", modalW)) + "\n")
	sb.WriteString("  " + ui.StyleKeyHint.Render("enter/y") +
		ui.StyleDim.Render(" confirm    ") +
		ui.StyleKeyHint.Render("esc/n") +
		ui.StyleDim.Render(" cancel"))

	return sb.String()
}

func (m Model) viewConflictModal() string {
	var sb strings.Builder

	// Modal dimensions
	modalW := 60
	if m.width < modalW+4 {
		modalW = m.width - 4
	}
	if modalW < 30 {
		modalW = 30
	}

	// Header
	sb.WriteString(ui.StyleError.Render("  ✗ Merge Conflict") + "\n")
	sb.WriteString(ui.StyleDivider.Render(strings.Repeat("─", modalW)) + "\n\n")

	sb.WriteString(ui.StyleDim.Render("  Conflicts in "+fmt.Sprintf("%d", len(m.conflicts))+" file(s):") + "\n\n")
	for _, f := range m.conflicts {
		line := "    " + f
		if len(line) > modalW-2 {
			line = line[:modalW-5] + "…"
		}
		sb.WriteString(lipgloss.NewStyle().Foreground(ui.ColorHeader).Render(line) + "\n")
	}

	sb.WriteString("\n")
	sb.WriteString(ui.StyleDim.Render("  Resolve conflicts manually, then:") + "\n")
	sb.WriteString(ui.StyleDim.Render("    git add <files>") + "\n")
	sb.WriteString(ui.StyleDim.Render("    git commit") + "\n")
	sb.WriteString("\n")
	sb.WriteString(ui.StyleDim.Render("  Or abort the merge:") + "\n")
	sb.WriteString(ui.StyleDim.Render("    git merge --abort") + "\n")

	sb.WriteString("\n")
	sb.WriteString(ui.StyleDivider.Render(strings.Repeat("─", modalW)) + "\n")
	sb.WriteString("  " + ui.StyleKeyHint.Render("enter") +
		ui.StyleDim.Render(" ok"))

	return sb.String()
}

func (m Model) footer() string {
	filterPrompt := "  " + ui.StyleDim.Render("filter: ") + m.filterInput.View()
	hints := "  " + m.help.ShortHelpView(m.keys.ShortHelp())

	switch {
	case m.merging:
		return lipgloss.NewStyle().Foreground(ui.ColorParentAhead).Bold(true).Render("  " + m.spinner.View() + " merging…")

	case m.err != "":
		return "  " + ui.StyleError.Render("✗ "+m.err) + "\n" + filterPrompt + "\n" + hints

	default:
		return filterPrompt + "\n" + hints
	}
}

// ── Rendering helpers ──────────────────────────────────────────────────────

func (m Model) listWidth() int {
	const detailPanePct = 40
	dw := m.width * detailPanePct / 100
	lw := m.width - dw - 3
	const minLW = 19 + colNameMin + 15
	if lw < minLW {
		lw = minLW
	}
	return lw
}

func (m Model) renderColHeaders(lw int) string {
	sep := strings.Repeat(" ", colPad)
	nw, dw := columnWidths(lw)
	nameH := lipgloss.NewStyle().Width(nw).Render(ui.StyleDim.Render("Branch"))
	statusH := lipgloss.NewStyle().Width(colStatus).Render(ui.StyleDim.Render("vs parent"))
	descH := ui.StyleDim.Render("Description")
	_ = dw
	return "   " + sep + nameH + sep + statusH + sep + descH
}

func (m Model) renderBody(lw int) string {
	end := m.offset + m.visibleRows
	if end > len(m.filtered) {
		end = len(m.filtered)
	}

	var rows []string
	for i := m.offset; i < end; i++ {
		rows = append(rows, renderRow(m.filtered[i], i == m.cursor, lw))
	}
	return strings.Join(rows, "\n") + "\n"
}

func columnWidths(termWidth int) (nameW, descW int) {
	const fixed = 2 + colMarker + 3*colPad + colStatus
	avail := termWidth - fixed
	if avail < 1 {
		return colNameMin, 0
	}
	descW = avail * 40 / 100
	if descW < 15 {
		descW = 15
	}
	if descW > colDescMax {
		descW = colDescMax
	}
	nameW = avail - descW
	if nameW < colNameMin {
		nameW = colNameMin
	}
	if nameW > colNameMax {
		nameW = colNameMax
	}
	return
}

func renderRow(item renderItem, isSelected bool, termWidth int) string {
	b := item.branch

	_, dw := columnWidths(termWidth)
	prefixW := len([]rune(item.treePrefix))
	nameW := nameColWidth(termWidth) - prefixW
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
	descText := truncate(b.Description, dw)

	sep := strings.Repeat(" ", colPad)

	if isSelected {
		bg := ui.ColorCursorBg
		bgSep := lipgloss.NewStyle().Background(bg).Render(sep)
		markerS := lipgloss.NewStyle().Background(bg).Bold(true).Foreground(ui.ColorAccent).Render(markerChar)
		prefixS := lipgloss.NewStyle().Background(bg).Foreground(ui.ColorDim).Render(item.treePrefix)
		nameS := lipgloss.NewStyle().Background(bg).Bold(true).Foreground(ui.ColorCursorFg).Width(nameW).Render(nameText)
		statusS := statusStyle.Background(bg).Width(colStatus).Render(statusText)
		descS := lipgloss.NewStyle().Background(bg).Foreground(ui.ColorDesc).Italic(true).Width(dw).Render(descText)
		leftPad := lipgloss.NewStyle().Background(bg).Render("  ")
		used := 2 + colMarker + colPad + nameColWidth(termWidth) + colPad + colStatus + colPad + dw
		trail := ""
		if termWidth > used {
			trail = lipgloss.NewStyle().Background(bg).Render(strings.Repeat(" ", termWidth-used))
		}
		return leftPad + markerS + bgSep + prefixS + nameS + bgSep + statusS + bgSep + descS + trail
	}

	prefixS := ui.StyleTreeConnector.Render(item.treePrefix)
	var markerS, nameS string
	if b.IsCurrent {
		markerS = ui.StyleCurrentBranch.Render(markerChar)
		nameS = ui.StyleCurrentBranch.Width(nameW).Render(nameText)
	} else if b.IsRemote {
		markerS = " "
		nameS = ui.StyleRemoteName.Width(nameW).Render(nameText)
	} else {
		markerS = " "
		nameS = lipgloss.NewStyle().Width(nameW).Render(nameText)
	}

	statusS := statusStyle.Width(colStatus).Render(statusText)
	descS := ui.StyleDesc.Italic(true).Render(descText)

	return "  " + markerS + sep + prefixS + nameS + sep + statusS + sep + descS
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

func nameColWidth(termWidth int) int {
	nw, _ := columnWidths(termWidth)
	return nw
}

// ── Git commands ───────────────────────────────────────────────────────────

func loadPreview(currentBranch, targetBranch string) tea.Cmd {
	return func() tea.Msg {
		// Get files changed
		files, err := git.MergePreview(currentBranch, targetBranch)
		if err != nil {
			return previewLoadedMsg{err: err}
		}

		// Get last 3 commits from current branch
		commits, err := git.ListCommits(currentBranch, 3, git.CommitFilters{})
		if err != nil {
			return previewLoadedMsg{err: err}
		}

		return previewLoadedMsg{filesChanged: files, commits: commits}
	}
}

func doMerge(targetBranch, sourceBranch string) tea.Cmd {
	return func() tea.Msg {
		result := git.SwitchAndMergeWithResult(targetBranch, sourceBranch)
		var err error
		if !result.Success {
			err = fmt.Errorf("%s", result.ErrorMessage)
		}
		return mergeDoneMsg{
			err:          err,
			targetBranch: targetBranch,
			sourceBranch: sourceBranch,
			conflicts:    result.Conflicts,
		}
	}
}

// ── Tree builder ───────────────────────────────────────────────────────────

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

func (m *Model) applyFilter() {
	q := strings.ToLower(m.filterInput.Value())
	if q == "" {
		m.filtered = m.treeItems
	} else {
		var out []renderItem
		for _, b := range m.branches {
			if strings.Contains(strings.ToLower(b.Name), q) ||
				strings.Contains(strings.ToLower(b.Description), q) {
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

// MergedTo returns the target branch name after a successful merge.
func (m Model) MergedTo() string { return m.mergedTo }

// MergedFrom returns the source branch name after a successful merge.
func (m Model) MergedFrom() string { return m.mergedFrom }