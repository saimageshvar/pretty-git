package log

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/sai/pretty-git/internal/git"
	ui "github.com/sai/pretty-git/internal/ui"
)

// ── Layout constants ────────────────────────────────────────────────────────

const (
	maxVisible    = 15 // max list rows visible at once
	colHash       = 8  // short hash column
	colAuthor     = 16 // author name column
	colTime       = 13 // "3 days ago   "
	colPad        = 2  // space between columns
	detailPanePct = 45 // right pane as % of terminal width
)

// ── Messages ────────────────────────────────────────────────────────────────

type detailLoadedMsg struct {
	hash   string
	detail git.CommitDetail
	err    error
}

// ── Model ───────────────────────────────────────────────────────────────────

type Model struct {
	commits     []git.Commit
	cursor      int
	offset      int
	visibleRows int
	width       int
	height      int
	repoName    string
	ref         string

	focusedPane pane // paneList or paneDetail

	detail       *git.CommitDetail
	detailHash   string
	detailLines  []string // word-wrapped lines for the detail pane
	detailOffset int

	loading bool

	help    help.Model
	keys    keyMap
	spinner spinner.Model
}

func New(commits []git.Commit, repoName, ref string, termWidth, termHeight int) Model {
	vis := min(len(commits), maxVisible)
	if termHeight > 5 && vis > termHeight-5 {
		vis = termHeight - 5
	}
	if vis < 1 {
		vis = 1
	}

	h := help.New()
	h.Styles.ShortKey = lipgloss.NewStyle().Bold(true).Foreground(ui.ColorKeyHint)
	h.Styles.ShortDesc = lipgloss.NewStyle().Foreground(ui.ColorHeader)
	h.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(ui.ColorTreeConnector)
	h.Width = termWidth

	sp := spinner.New()
	sp.Spinner = spinner.MiniDot
	sp.Style = lipgloss.NewStyle().Bold(true).Foreground(ui.ColorAccent)

	m := Model{
		commits:     commits,
		visibleRows: vis,
		width:       termWidth,
		height:      termHeight,
		repoName:    repoName,
		ref:         ref,
		help:        h,
		keys:        defaultKeyMap(),
		spinner:     sp,
	}
	// Pre-set detailHash so the first load's response is accepted immediately.
	if len(commits) > 0 {
		m.detailHash = commits[0].Hash
		m.loading = true
	}
	return m
}

func (m Model) Init() tea.Cmd {
	if len(m.commits) > 0 {
		return tea.Batch(m.spinner.Tick, doLoadDetail(m.commits[0].Hash))
	}
	return m.spinner.Tick
}

// ── Update ──────────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		vis := min(len(m.commits), maxVisible)
		if msg.Height > 5 && vis > msg.Height-5 {
			vis = msg.Height - 5
		}
		m.visibleRows = max(vis, 1)
		m.clampScroll()
		// Rebuild detail lines at new width.
		if m.detail != nil {
			m.detailLines = buildDetailLines(m.detail, m.detailWidth())
			m.clampDetailScroll()
		}
		return m, nil

	case detailLoadedMsg:
		m.loading = false
		if msg.err == nil && msg.hash == m.detailHash {
			m.detail = &msg.detail
			m.detailLines = buildDetailLines(m.detail, m.detailWidth())
			m.detailOffset = 0
		}
		return m, nil

	case tea.KeyMsg:
		// ── Detail pane has focus ──────────────────────────────────────────
		if m.focusedPane == paneDetail {
			switch {
			case key.Matches(msg, m.keys.Quit):
				return m, tea.Quit
			case key.Matches(msg, m.keys.FocusList):
				m.focusedPane = paneList
			case key.Matches(msg, m.keys.Up):
				if m.detailOffset > 0 {
					m.detailOffset--
				}
			case key.Matches(msg, m.keys.Down):
				m.clampDetailScroll()
				if maxOff := max(len(m.detailLines)-m.visibleRows, 0); m.detailOffset < maxOff {
					m.detailOffset++
				}
			case key.Matches(msg, m.keys.PageUp):
				m.detailOffset = max(m.detailOffset-m.visibleRows, 0)
			case key.Matches(msg, m.keys.PageDown):
				m.detailOffset += m.visibleRows
				m.clampDetailScroll()
			}
			return m, nil
		}

		// ── List pane has focus ───────────────────────────────────────────
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.FocusDetail):
			if m.detail != nil {
				m.focusedPane = paneDetail
			}
		case key.Matches(msg, m.keys.Up):
			if m.cursor > 0 {
				m.cursor--
				m.clampScroll()
				return m, m.triggerLoad()
			}
		case key.Matches(msg, m.keys.Down):
			if m.cursor < len(m.commits)-1 {
				m.cursor++
				m.clampScroll()
				return m, m.triggerLoad()
			}
		case key.Matches(msg, m.keys.PageUp):
			m.cursor = max(m.cursor-m.visibleRows, 0)
			m.clampScroll()
			return m, m.triggerLoad()
		case key.Matches(msg, m.keys.PageDown):
			m.cursor = min(m.cursor+m.visibleRows, len(m.commits)-1)
			m.clampScroll()
			return m, m.triggerLoad()
		}
	}

	return m, nil
}

func (m *Model) triggerLoad() tea.Cmd {
	if len(m.commits) == 0 {
		return nil
	}
	hash := m.commits[m.cursor].Hash
	if hash == m.detailHash {
		return nil
	}
	m.detailHash = hash
	m.detail = nil
	m.detailLines = nil
	m.detailOffset = 0
	m.loading = true
	return doLoadDetail(hash)
}

func (m *Model) clampScroll() {
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+m.visibleRows {
		m.offset = m.cursor - m.visibleRows + 1
	}
}

func (m *Model) clampDetailScroll() {
	maxOff := max(len(m.detailLines)-m.visibleRows, 0)
	if m.detailOffset > maxOff {
		m.detailOffset = maxOff
	}
}

// ── View ────────────────────────────────────────────────────────────────────

func (m Model) View() string {
	lw := m.listWidth()
	dw := m.detailWidth()

	var sb strings.Builder

	// ── Header ──────────────────────────────────────────────────────────────
	badge := ui.StyleCountBadge.Render(fmt.Sprintf("%d commits", len(m.commits)))
	refLabel := lipgloss.NewStyle().Foreground(ui.ColorAccent).Bold(true).Italic(true).Render(m.ref)
	sb.WriteString(ui.StyleHeader.Render("  Log") + "  " + refLabel + "  " +
		ui.StyleAccent.Render(m.repoName) + "  " + badge + "\n")
	sb.WriteString(ui.StyleDivider.Render(strings.Repeat("─", m.width)) + "\n")

	// ── Column headers ───────────────────────────────────────────────────────
	sb.WriteString(m.renderColHeaders(lw, dw) + "\n")

	// ── Body rows ────────────────────────────────────────────────────────────
	sb.WriteString(m.renderBody(lw, dw))

	// ── Footer ──────────────────────────────────────────────────────────────
	sb.WriteString(ui.StyleDivider.Render(strings.Repeat("─", m.width)) + "\n")
	if m.focusedPane == paneDetail {
		sb.WriteString("  " + m.help.ShortHelpView(m.keys.detailHelp()))
	} else {
		sb.WriteString("  " + m.help.ShortHelpView(m.keys.listHelp()))
	}

	return sb.String()
}

// listWidth returns the pixel/char width of the left (list) pane.
func (m Model) listWidth() int {
	dw := m.width * detailPanePct / 100
	lw := m.width - dw - 3 // " │ "
	if lw < 30 {
		lw = 30
	}
	return lw
}

// detailWidth returns the char width of the right (detail) pane.
func (m Model) detailWidth() int {
	dw := m.width - m.listWidth() - 3
	if dw < 20 {
		dw = 20
	}
	return dw
}

// renderColHeaders renders the column header row for both panes.
func (m Model) renderColHeaders(lw, dw int) string {
	sep := strings.Repeat(" ", colPad)
	sw := subjectColWidth(lw)

	hashH    := fixedWidth(ui.StyleDim.Render("Hash"), colHash)
	subjectH := fixedWidth(ui.StyleDim.Render("Subject"), sw)
	authorH  := fixedWidth(ui.StyleDim.Render("Author"), colAuthor)
	timeH    := fixedWidth(ui.StyleDim.Render("When"), colTime)

	listPart := fixedWidth("  "+hashH+sep+subjectH+sep+authorH+sep+timeH, lw)

	// Detail pane header — shows focus state and scroll indicator.
	divColor := ui.ColorDivider
	if m.focusedPane == paneDetail {
		divColor = ui.ColorAccent
	}
	divider := lipgloss.NewStyle().Foreground(divColor).Bold(m.focusedPane == paneDetail).Render("│")

	detailTitle := m.detailPaneTitle(dw)

	return listPart + " " + divider + " " + detailTitle
}

// detailPaneTitle builds the header for the detail column including scroll hint.
func (m Model) detailPaneTitle(dw int) string {
	var title string
	focused := m.focusedPane == paneDetail

	if focused {
		title = lipgloss.NewStyle().Foreground(ui.ColorAccent).Bold(true).Render("› Detail")
	} else {
		title = ui.StyleDim.Render("  Detail")
	}

	// Scroll indicator — how many lines are below the visible window.
	below := len(m.detailLines) - m.detailOffset - m.visibleRows
	if below < 0 {
		below = 0
	}
	above := m.detailOffset

	var scrollHint string
	switch {
	case below > 0 && above > 0:
		scrollHint = lipgloss.NewStyle().Foreground(ui.ColorKeyHint).Bold(true).
			Render(fmt.Sprintf("  ↑%d ↓%d more", above, below))
	case below > 0:
		scrollHint = lipgloss.NewStyle().Foreground(ui.ColorKeyHint).Bold(true).
			Render(fmt.Sprintf("  ↓ %d more lines — press → to scroll", below))
	case above > 0:
		scrollHint = lipgloss.NewStyle().Foreground(ui.ColorKeyHint).Bold(true).
			Render(fmt.Sprintf("  ↑ %d lines above", above))
	}

	result := title + scrollHint
	// If focused, also show key hint
	if focused && scrollHint == "" {
		result += ui.StyleDim.Render("  (↑↓ scroll · ← back)")
	}
	return result
}

// renderBody renders all visible rows: list on left, detail on right.
func (m Model) renderBody(lw, dw int) string {
	end := min(m.offset+m.visibleRows, len(m.commits))

	// Pick divider color based on focus.
	divColor := ui.ColorDivider
	if m.focusedPane == paneDetail {
		divColor = ui.ColorAccent
	}
	divider := lipgloss.NewStyle().Foreground(divColor).Bold(m.focusedPane == paneDetail).Render("│")

	var rows []string
	for i := m.offset; i < end; i++ {
		listRow   := m.renderListRow(i, lw)
		detailRow := m.detailRow(i-m.offset, dw)
		rows = append(rows, listRow+divider+" "+detailRow)
	}

	// Fill remaining visible rows when fewer commits than visibleRows.
	emptyList := strings.Repeat(" ", lw+1) // lw + space before divider
	for i := end - m.offset; i < m.visibleRows; i++ {
		detailRow := m.detailRow(i, dw)
		rows = append(rows, emptyList+divider+" "+detailRow)
	}

	return strings.Join(rows, "\n") + "\n"
}

// detailRow returns the i-th visible line of the detail pane, padded to dw chars.
// When i is the last visible row and more content exists below, a scroll hint
// is shown instead of (or over) the actual line.
func (m Model) detailRow(i, dw int) string {
	isLastRow := i == m.visibleRows-1
	below := len(m.detailLines) - m.detailOffset - m.visibleRows

	// Last visible row: show "↓ N more" hint when there's content below.
	if isLastRow && below > 0 {
		hint := fmt.Sprintf("  ↓ %d more lines", below)
		styled := lipgloss.NewStyle().
			Foreground(ui.ColorKeyHint).
			Bold(true).
			Italic(true).
			Render(hint)
		return fitToWidth(styled, dw)
	}

	if m.loading && i == 0 {
		return fitToWidth("  "+m.spinner.View()+" loading…", dw)
	}
	if m.detail == nil && !m.loading && i == 0 {
		return fitToWidth("  "+ui.StyleDim.Render("select a commit"), dw)
	}
	idx := m.detailOffset + i
	if idx < len(m.detailLines) {
		return fitToWidth(m.detailLines[idx], dw)
	}
	return strings.Repeat(" ", dw)
}

// renderListRow renders one commit row exactly lw chars wide (+ 1 space before divider).
func (m Model) renderListRow(idx, lw int) string {
	c          := m.commits[idx]
	isSelected := idx == m.cursor
	sep        := strings.Repeat(" ", colPad)
	sw         := subjectColWidth(lw)

	hashText    := truncate(c.ShortHash, colHash)
	subjectText := truncate(c.Subject, sw)
	authorText  := truncate(c.Author, colAuthor)
	timeText    := truncate(c.RelTime, colTime)

	if isSelected {
		bg   := ui.ColorCursorBg
		bgS  := func(s string, w int) string {
			return lipgloss.NewStyle().Background(bg).Width(w).Render(s)
		}
		bgSep := lipgloss.NewStyle().Background(bg).Render(sep)
		left  := lipgloss.NewStyle().Background(bg).Render("  ")
		hashS    := lipgloss.NewStyle().Background(bg).Foreground(ui.ColorHash).Bold(true).Width(colHash).Render(hashText)
		subjectS := lipgloss.NewStyle().Background(bg).Foreground(ui.ColorCursorFg).Bold(true).Width(sw).Render(subjectText)
		authorS  := bgS(authorText, colAuthor)
		timeS    := lipgloss.NewStyle().Background(bg).Foreground(ui.ColorRelTime).Width(colTime).Render(timeText)

		row := left + hashS + bgSep + subjectS + bgSep + authorS + bgSep + timeS
		used := lipgloss.Width(row)
		if lw > used {
			row += lipgloss.NewStyle().Background(bg).Render(strings.Repeat(" ", lw-used))
		}
		return row + " "
	}

	hashS    := lipgloss.NewStyle().Foreground(ui.ColorHash).Width(colHash).Render(hashText)
	subjectS := lipgloss.NewStyle().Width(sw).Render(subjectText)
	authorS  := lipgloss.NewStyle().Foreground(ui.ColorDesc).Width(colAuthor).Render(authorText)
	timeS    := ui.StyleRelTime.Width(colTime).Render(timeText)

	row := "  " + hashS + sep + subjectS + sep + authorS + sep + timeS
	used := lipgloss.Width(row)
	if lw > used {
		row += strings.Repeat(" ", lw-used)
	}
	return row + " "
}

// ── Detail pane builder ──────────────────────────────────────────────────────

// buildDetailLines converts a CommitDetail into word-wrapped lines for the pane.
// dw is the usable character width of the detail pane.
func buildDetailLines(d *git.CommitDetail, dw int) []string {
	inner := dw - 4 // 4 = "  " left margin + 2 padding
	if inner < 10 {
		inner = 10
	}

	var lines []string
	add := func(s string) { lines = append(lines, s) }

	dimS  := func(s string) string { return ui.StyleDim.Render(s) }
	kwS   := func(s string) string {
		return lipgloss.NewStyle().Foreground(ui.ColorKeyHint).Bold(true).Render(s)
	}
	valS  := func(s string) string {
		return lipgloss.NewStyle().Foreground(ui.ColorHeader).Render(s)
	}
	accS  := func(s string) string {
		return lipgloss.NewStyle().Foreground(ui.ColorAccent).Bold(true).Render(s)
	}
	boldS := func(s string) string {
		return lipgloss.NewStyle().Foreground(ui.ColorCursorFg).Bold(true).Render(s)
	}

	// ── Hash ──────────────────────────────────────────────────────────────
	shortRest := ""
	if len(d.Hash) > 8 {
		shortRest = d.Hash[8:]
	}
	add("  " + kwS("commit ") + accS(d.ShortHash) + dimS("  …"+shortRest))
	add("")

	// ── Subject (bold, word-wrapped) ───────────────────────────────────────
	for _, line := range wordWrap(d.Subject, inner) {
		add("  " + boldS(line))
	}

	// ── Body (word-wrapped) ────────────────────────────────────────────────
	if d.Body != "" {
		add("")
		for _, para := range strings.Split(d.Body, "\n") {
			if para == "" {
				add("")
				continue
			}
			for _, line := range wordWrap(para, inner) {
				add("  " + valS(line))
			}
		}
	}

	add("")

	// ── Metadata ────────────────────────────────────────────────────────────
	for _, line := range wordWrap(d.Author+" <"+d.AuthorEmail+">", inner) {
		add("  " + dimS("Author  ") + valS(line))
	}
	add("  " + dimS("Date    ") +
		lipgloss.NewStyle().Foreground(ui.ColorRelTime).Render(d.RelTime) +
		dimS("  ("+shortDate(d.AuthorDate)+")"))

	// ── Diff stats ──────────────────────────────────────────────────────────
	if d.DiffStat != "" {
		add("")
		add("  " + dimS("── changes "+strings.Repeat("─", max(0, inner-10))))

		// Summary line with colored counts
		statLine := fmt.Sprintf("%d file", d.FilesChanged)
		if d.FilesChanged != 1 {
			statLine += "s"
		}
		statLine += " changed"
		if d.Insertions > 0 {
			statLine += ", " + lipgloss.NewStyle().Foreground(ui.ColorParentMerged).Bold(true).
				Render(fmt.Sprintf("+%d", d.Insertions))
		}
		if d.Deletions > 0 {
			statLine += ", " + lipgloss.NewStyle().Foreground(ui.ColorError).Bold(true).
				Render(fmt.Sprintf("-%d", d.Deletions))
		}
		add("  " + statLine)
		add("")

		// Per-file lines from git diff-tree --stat
		statLines := strings.Split(d.DiffStat, "\n")
		fileLines := statLines
		if len(statLines) > 1 {
			fileLines = statLines[:len(statLines)-1] // drop git's own summary line
		}
		for _, sl := range fileLines {
			sl = strings.TrimLeft(sl, " ")
			if sl == "" {
				continue
			}
			add("  " + colorizeStatBar(sl, inner))
		}
	} else {
		add("")
		add("  " + dimS("(no file changes)"))
	}

	return lines
}

// ── Git async command ────────────────────────────────────────────────────────

func doLoadDetail(hash string) tea.Cmd {
	return func() tea.Msg {
		d, err := git.GetCommitDetail(hash)
		return detailLoadedMsg{hash: hash, detail: d, err: err}
	}
}

// ── Helpers ──────────────────────────────────────────────────────────────────

// subjectColWidth returns how many chars are available for the subject column.
func subjectColWidth(lw int) int {
	w := lw - 2 - colHash - colAuthor - colTime - colPad*3
	if w < 10 {
		w = 10
	}
	return w
}

// fixedWidth renders s into exactly w visible chars (lipgloss handles padding/truncation).
func fixedWidth(s string, w int) string {
	return lipgloss.NewStyle().Width(w).MaxWidth(w).Render(s)
}

// fitToWidth ensures a (possibly ANSI-styled) string occupies exactly w visible chars.
func fitToWidth(s string, w int) string {
	vis := lipgloss.Width(s)
	switch {
	case vis == w:
		return s
	case vis < w:
		return s + strings.Repeat(" ", w-vis)
	default:
		// Visible content exceeds pane width: truncate plain text.
		plain := lipgloss.NewStyle().Render(s) // strip styles via re-render trick won't work; strip manually
		plain = stripANSI(s)
		r := []rune(plain)
		if len(r) > w {
			r = r[:w]
		}
		return string(r)
	}
}

// stripANSI removes ANSI escape sequences for safe rune-level truncation.
func stripANSI(s string) string {
	var b strings.Builder
	inEsc := false
	for _, ch := range s {
		if inEsc {
			if ch == 'm' {
				inEsc = false
			}
			continue
		}
		if ch == '\x1b' {
			inEsc = true
			continue
		}
		b.WriteRune(ch)
	}
	return b.String()
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

// wordWrap breaks s into lines of at most maxW visible chars, on word boundaries.
func wordWrap(s string, maxW int) []string {
	if maxW <= 0 {
		return []string{s}
	}
	words := strings.Fields(s)
	if len(words) == 0 {
		return []string{""}
	}
	var lines []string
	line := ""
	for _, w := range words {
		if line == "" {
			line = w
		} else if len([]rune(line))+1+len([]rune(w)) <= maxW {
			line += " " + w
		} else {
			lines = append(lines, line)
			line = w
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

// colorizeStatBar colors the +/- histogram in a `git diff-tree --stat` line.
func colorizeStatBar(line string, maxW int) string {
	idx := strings.LastIndex(line, "|")
	if idx < 0 {
		return truncate(ui.StyleDim.Render(line), maxW)
	}
	file := strings.TrimRight(line[:idx], " ")
	rest := line[idx+1:]

	var barB strings.Builder
	for _, ch := range rest {
		switch ch {
		case '+':
			barB.WriteString(lipgloss.NewStyle().Foreground(ui.ColorParentMerged).Render("+"))
		case '-':
			barB.WriteString(lipgloss.NewStyle().Foreground(ui.ColorError).Render("-"))
		default:
			barB.WriteRune(ch)
		}
	}

	// Truncate the filename portion if it's too long.
	maxFile := maxW/2 - 3
	if maxFile < 5 {
		maxFile = 5
	}
	return ui.StyleDim.Render(truncate(file, maxFile)) + ui.StyleDim.Render(" │") + barB.String()
}

// shortDate strips seconds and timezone from an absolute date string.
func shortDate(s string) string {
	parts := strings.Fields(s)
	if len(parts) >= 5 {
		return strings.Join(parts[:5], " ")
	}
	return s
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
