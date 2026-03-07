package stash

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
	browseMaxVisible = 15
	browseMinVisible = 8  // detail pane is always at least this tall
	browseDetailPct  = 45
	colIdx           = 4
	colMsg           = 28
	colBranch        = 14
	colWhen          = 13
	browsePad        = 2
)

// ── Messages ────────────────────────────────────────────────────────────────

type browseDetailLoadedMsg struct {
	ref    string
	detail git.StashDetail
	err    error
}

type browseActionDoneMsg struct {
	err error
}

// ── Model ───────────────────────────────────────────────────────────────────

// BrowseModel is the Bubble Tea model for the stash list browser.
type BrowseModel struct {
	stashes     []git.StashEntry
	cursor      int
	offset      int
	visibleRows int
	width       int
	height      int
	repoName    string
	mode        BrowseMode

	focusedPane browsePaneMode

	detail       *git.StashDetail
	detailRef    string
	detailLines  []string
	detailOffset int
	loading      bool

	// Confirmation modal (drop=warning, pop=accent)
	confirmingDrop bool
	confirmingPop  bool
	confirmFocus   int // 0=No (default, safe), 1=Yes

	// Result message
	actionErr    error
	actionResult string

	help    help.Model
	keys    browseKeyMap
	spinner spinner.Model
}

// NewBrowse initialises the stash browse model.
func NewBrowse(stashes []git.StashEntry, repoName string, mode BrowseMode, termWidth, termHeight int) BrowseModel {
	vis := min(len(stashes), browseMaxVisible)
	if termHeight > 5 && vis > termHeight-5 {
		vis = termHeight - 5
	}
	if vis < browseMinVisible {
		vis = browseMinVisible
	}

	h := help.New()
	h.Styles.ShortKey = lipgloss.NewStyle().Bold(true).Foreground(ui.ColorKeyHint)
	h.Styles.ShortDesc = lipgloss.NewStyle().Foreground(ui.ColorHeader)
	h.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(ui.ColorTreeConnector)
	h.Width = termWidth

	sp := spinner.New()
	sp.Spinner = spinner.MiniDot
	sp.Style = lipgloss.NewStyle().Bold(true).Foreground(ui.ColorAccent)

	m := BrowseModel{
		stashes:     stashes,
		visibleRows: vis,
		width:       termWidth,
		height:      termHeight,
		repoName:    repoName,
		mode:        mode,
		help:        h,
		keys:        defaultBrowseKeyMap(),
		spinner:     sp,
	}

	if len(stashes) > 0 {
		m.detailRef = stashes[0].Ref
		m.loading = true
	}
	return m
}


func (m BrowseModel) Init() tea.Cmd {
	if len(m.stashes) > 0 {
		return tea.Batch(m.spinner.Tick, doLoadBrowseDetail(m.stashes[0]))
	}
	return m.spinner.Tick
}

// ── Update ──────────────────────────────────────────────────────────────────

func (m BrowseModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		vis := min(len(m.stashes), browseMaxVisible)
		if msg.Height > 5 && vis > msg.Height-5 {
			vis = msg.Height - 5
		}
		if vis < browseMinVisible {
			vis = browseMinVisible
		}
		m.visibleRows = vis
		m.clampScroll()
		if m.detail != nil {
			m.detailLines = buildBrowseDetailLines(m.detail, m.detailWidth())
			m.clampDetailScroll()
		}
		return m, nil

	case browseDetailLoadedMsg:
		m.loading = false
		if msg.err == nil && msg.ref == m.detailRef {
			m.detail = &msg.detail
			m.detailLines = buildBrowseDetailLines(m.detail, m.detailWidth())
			m.detailOffset = 0
		}
		return m, nil

	case browseActionDoneMsg:
		if msg.err != nil {
			m.actionErr = msg.err
			m.loading = false
			return m, nil
		}
		if m.mode == BrowseModeDrop {
			// Remove the dropped entry and re-index
			if m.cursor < len(m.stashes) {
				m.stashes = append(m.stashes[:m.cursor], m.stashes[m.cursor+1:]...)
				// Re-index
				for i := range m.stashes {
					m.stashes[i].Index = i
					m.stashes[i].Ref = fmt.Sprintf("stash@{%d}", i)
				}
			}
			if len(m.stashes) == 0 {
				m.actionResult = "all stashes cleared"
				return m, tea.Quit
			}
			// Clamp cursor
			if m.cursor >= len(m.stashes) {
				m.cursor = len(m.stashes) - 1
			}
			m.clampScroll()
			// Reload detail for new cursor
			m.detail = nil
			m.detailLines = nil
			m.detailRef = m.stashes[m.cursor].Ref
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, doLoadBrowseDetail(m.stashes[m.cursor]))
		}
		// apply / pop: quit with success
		m.actionResult = fmt.Sprintf("✓ stash %s applied", m.mode.String())
		return m, tea.Quit

	case tea.KeyMsg:
		// Confirmation modal (drop / pop) — handled BEFORE general quit so that
		// esc cancels the dialog rather than quitting the whole program.
		if m.confirmingDrop || m.confirmingPop {
			switch {
			case key.Matches(msg, m.keys.CancelDrop): // n/esc → cancel
				m.confirmingDrop, m.confirmingPop = false, false
				m.confirmFocus = 0
			case key.Matches(msg, m.keys.FocusList): // ← → No
				m.confirmFocus = 0
			case key.Matches(msg, m.keys.FocusDetail): // → → Yes
				m.confirmFocus = 1
			case key.Matches(msg, m.keys.Action): // enter → activate focused button
				if m.confirmFocus == 1 {
					m.confirmingDrop, m.confirmingPop = false, false
					m.confirmFocus = 0
					m.actionErr = nil
					m.loading = true
					ref := m.stashes[m.cursor].Ref
					return m, tea.Batch(m.spinner.Tick, doStashAction(ref, m.mode))
				}
				m.confirmingDrop, m.confirmingPop = false, false
				m.confirmFocus = 0
			}
			return m, nil
		}

		if key.Matches(msg, m.keys.Quit) {
			return m, tea.Quit
		}

		if m.focusedPane == browseDetail {
			switch {
			case key.Matches(msg, m.keys.FocusList):
				m.focusedPane = browseList
			case key.Matches(msg, m.keys.Up):
				if m.detailOffset > 0 {
					m.detailOffset--
				}
			case key.Matches(msg, m.keys.Down):
				m.clampDetailScroll()
				if maxOff := max(len(m.detailLines)-m.visibleRows, 0); m.detailOffset < maxOff {
					m.detailOffset++
				}
			}
			return m, nil
		}

		// List pane
		switch {
		case key.Matches(msg, m.keys.Up):
			if m.cursor > 0 {
				m.cursor--
				m.clampScroll()
				return m, m.triggerLoad()
			}
		case key.Matches(msg, m.keys.Down):
			if m.cursor < len(m.stashes)-1 {
				m.cursor++
				m.clampScroll()
				return m, m.triggerLoad()
			}
		case key.Matches(msg, m.keys.FocusDetail):
			if m.detail != nil {
				m.focusedPane = browseDetail
			}
		case key.Matches(msg, m.keys.Action):
			if len(m.stashes) == 0 {
				break
			}
			m.actionErr = nil
			switch m.mode {
			case BrowseModeDrop:
				m.confirmingDrop = true
				m.confirmFocus = 0
			case BrowseModePop:
				m.confirmingPop = true
				m.confirmFocus = 0
			default:
				m.loading = true
				ref := m.stashes[m.cursor].Ref
				return m, tea.Batch(m.spinner.Tick, doStashAction(ref, m.mode))
			}
		}
	}

	return m, nil
}

func (m *BrowseModel) triggerLoad() tea.Cmd {
	if len(m.stashes) == 0 {
		return nil
	}
	entry := m.stashes[m.cursor]
	if entry.Ref == m.detailRef {
		return nil
	}
	m.detailRef = entry.Ref
	m.detail = nil
	m.detailLines = nil
	m.detailOffset = 0
	m.loading = true
	return doLoadBrowseDetail(entry)
}

func (m *BrowseModel) clampScroll() {
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+m.visibleRows {
		m.offset = m.cursor - m.visibleRows + 1
	}
}

func (m *BrowseModel) clampDetailScroll() {
	maxOff := max(len(m.detailLines)-m.visibleRows, 0)
	if m.detailOffset > maxOff {
		m.detailOffset = maxOff
	}
}

// ── View ─────────────────────────────────────────────────────────────────────

func (m BrowseModel) View() string {
	lw := m.listWidth()
	dw := m.detailWidth()

	var sb strings.Builder

	// Header
	badge := ui.StyleCountBadge.Render(fmt.Sprintf("%d stashes", len(m.stashes)))
	modeStyle := lipgloss.NewStyle().Foreground(ui.ColorAccent).Bold(true)
	if m.mode == BrowseModeDrop {
		modeStyle = lipgloss.NewStyle().Foreground(ui.ColorWarning).Bold(true)
	}
	modeLabel := modeStyle.Render("mode: " + m.mode.String())
	sb.WriteString(ui.StyleHeader.Render("  ✦ Stashes") + "  " + ui.StyleAccent.Render(m.repoName) +
		"  " + badge + "  " + modeLabel + "\n")
	sb.WriteString(ui.StyleDivider.Render(strings.Repeat("─", m.width)) + "\n")

	// Column headers
	sb.WriteString(m.renderColHeaders(lw, dw) + "\n")

	// Body
	sb.WriteString(m.renderBody(lw, dw))

	// Footer
	sb.WriteString(ui.StyleDivider.Render(strings.Repeat("─", m.width)) + "\n")

	if m.confirmingDrop || m.confirmingPop {
		sb.WriteString(m.renderConfirmModal())
	} else if m.actionErr != nil {
		sb.WriteString("  " + ui.StyleError.Render("✗ "+m.actionErr.Error()))
	} else if m.focusedPane == browseDetail {
		sb.WriteString("  " + m.help.ShortHelpView(m.keys.detailHelp()))
	} else {
		sb.WriteString("  " + m.help.ShortHelpView(m.keys.listHelp(m.mode)))
	}

	return sb.String()
}

func (m BrowseModel) renderColHeaders(lw, dw int) string {
	sep := strings.Repeat(" ", browsePad)
	msgW := colMsg
	if lw > colIdx+colBranch+colWhen+browsePad*3+10 {
		msgW = lw - colIdx - colBranch - colWhen - browsePad*3 - 2
	}

	idxH := fixedW(ui.StyleDim.Render("Idx"), colIdx)
	msgH := fixedW(ui.StyleDim.Render("Message"), msgW)
	branchH := fixedW(ui.StyleDim.Render("Branch"), colBranch)
	whenH := fixedW(ui.StyleDim.Render("When"), colWhen)

	listPart := fixedW("  "+idxH+sep+msgH+sep+branchH+sep+whenH, lw)

	divColor := ui.ColorDivider
	if m.focusedPane == browseDetail {
		divColor = ui.ColorAccent
	}
	divider := lipgloss.NewStyle().Foreground(divColor).Bold(m.focusedPane == browseDetail).Render("│")

	detailTitle := ui.StyleDim.Render("  Detail")
	if m.focusedPane == browseDetail {
		detailTitle = lipgloss.NewStyle().Foreground(ui.ColorAccent).Bold(true).Render("› Detail")
	}
	if len(m.stashes) > 0 {
		ref := m.stashes[m.cursor].Ref
		detailTitle = ui.StyleDim.Render("  ") + ui.StyleHash.Render(ref)
		if m.focusedPane == browseDetail {
			detailTitle = lipgloss.NewStyle().Foreground(ui.ColorAccent).Bold(true).Render("› ") + ui.StyleHash.Render(ref)
		}
	}

	return listPart + " " + divider + " " + detailTitle
}

func (m BrowseModel) renderBody(lw, dw int) string {
	end := min(m.offset+m.visibleRows, len(m.stashes))

	divColor := ui.ColorDivider
	if m.focusedPane == browseDetail {
		divColor = ui.ColorAccent
	}
	divider := lipgloss.NewStyle().Foreground(divColor).Bold(m.focusedPane == browseDetail).Render("│")

	var rows []string
	for i := m.offset; i < end; i++ {
		listRow := m.renderListRow(i, lw)
		detailRow := m.browseDetailRow(i-m.offset, dw)
		rows = append(rows, listRow+divider+" "+detailRow)
	}

	emptyList := strings.Repeat(" ", lw+1)
	for i := end - m.offset; i < m.visibleRows; i++ {
		detailRow := m.browseDetailRow(i, dw)
		rows = append(rows, emptyList+divider+" "+detailRow)
	}

	return strings.Join(rows, "\n") + "\n"
}

func (m BrowseModel) renderListRow(idx, lw int) string {
	s := m.stashes[idx]
	isSelected := idx == m.cursor
	sep := strings.Repeat(" ", browsePad)

	msgW := colMsg
	if lw > colIdx+colBranch+colWhen+browsePad*3+10 {
		msgW = lw - colIdx - colBranch - colWhen - browsePad*3 - 2
	}

	idxText := fmt.Sprintf("%d", s.Index)
	msgText := truncateStr(s.Message, msgW)
	branchText := truncateStr(s.Branch, colBranch)
	whenText := truncateStr(s.RelTime, colWhen)

	if isSelected {
		bg := ui.ColorCursorBg
		bgS := func(text string, w int) string {
			return lipgloss.NewStyle().Background(bg).Width(w).Render(text)
		}
		bgSep := lipgloss.NewStyle().Background(bg).Render(sep)
		left := lipgloss.NewStyle().Background(bg).Render("  ")
		idxS := lipgloss.NewStyle().Background(bg).Foreground(ui.ColorHash).Bold(true).Width(colIdx).Render(idxText)
		msgS := lipgloss.NewStyle().Background(bg).Foreground(ui.ColorCursorFg).Bold(true).Width(msgW).Render(msgText)
		branchS := bgS(branchText, colBranch)
		whenS := lipgloss.NewStyle().Background(bg).Foreground(ui.ColorRelTime).Width(colWhen).Render(whenText)

		row := left + idxS + bgSep + msgS + bgSep + branchS + bgSep + whenS
		used := lipgloss.Width(row)
		if lw > used {
			row += lipgloss.NewStyle().Background(bg).Render(strings.Repeat(" ", lw-used))
		}
		return row + " "
	}

	idxS := lipgloss.NewStyle().Foreground(ui.ColorHash).Width(colIdx).Render(idxText)
	msgS := lipgloss.NewStyle().Width(msgW).Render(msgText)
	branchS := lipgloss.NewStyle().Foreground(ui.ColorDesc).Width(colBranch).Render(branchText)
	whenS := ui.StyleRelTime.Width(colWhen).Render(whenText)

	row := "  " + idxS + sep + msgS + sep + branchS + sep + whenS
	used := lipgloss.Width(row)
	if lw > used {
		row += strings.Repeat(" ", lw-used)
	}
	return row + " "
}

func (m BrowseModel) browseDetailRow(i, dw int) string {
	isLastRow := i == m.visibleRows-1
	below := len(m.detailLines) - m.detailOffset - m.visibleRows

	if isLastRow && below > 0 {
		hint := fmt.Sprintf("  ↓ %d more lines", below)
		styled := lipgloss.NewStyle().Foreground(ui.ColorKeyHint).Bold(true).Italic(true).Render(hint)
		return fitW(styled, dw)
	}

	if m.loading && i == 0 {
		return fitW("  "+m.spinner.View()+" loading…", dw)
	}
	if m.detail == nil && !m.loading && i == 0 {
		return fitW("  "+ui.StyleDim.Render("select a stash"), dw)
	}

	idx := m.detailOffset + i
	if idx < len(m.detailLines) {
		return fitW(m.detailLines[idx], dw)
	}
	return strings.Repeat(" ", dw)
}

// buildBrowseDetailLines converts StashDetail into lines for the detail pane.
func buildBrowseDetailLines(d *git.StashDetail, dw int) []string {
	inner := dw - 4
	if inner < 10 {
		inner = 10
	}

	var lines []string
	add := func(s string) { lines = append(lines, s) }

	dimS := func(s string) string { return ui.StyleDim.Render(s) }
	valS := func(s string) string {
		return lipgloss.NewStyle().Foreground(ui.ColorHeader).Render(s)
	}

	// Message (ref is already shown in the column header)
	add("  " + valS(d.Message))
	add("")

	// Metadata
	if d.Branch != "" {
		add("  " + dimS("Branch  ") + valS(d.Branch))
	}
	if d.RelTime != "" {
		add("  " + dimS("Date    ") + ui.StyleRelTime.Render(d.RelTime))
	}

	if len(d.Files) == 0 {
		add("")
		add("  " + dimS("(no file changes)"))
		return lines
	}

	// Summary line
	add("")
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
	add("  " + dimS("── changes "+strings.Repeat("─", max(0, inner-10))))

	// File list
	for _, f := range d.Files {
		statusStr := statusColor(f.StatusDisplay())
		pathStr := dimS(truncateStr(f.Path, inner-6))
		add("  " + statusStr + "  " + pathStr)
	}

	return lines
}

// renderConfirmModal renders a centered lipgloss modal for drop/pop confirmation.
func (m BrowseModel) renderConfirmModal() string {
	if len(m.stashes) == 0 {
		return ""
	}
	entry := m.stashes[m.cursor]
	isDrop := m.confirmingDrop

	var accent lipgloss.TerminalColor = ui.ColorAccent
	titleText := "  Pop Stash"
	if isDrop {
		accent = ui.ColorWarning
		titleText = "⚠  Drop Stash"
	}

	titleS := lipgloss.NewStyle().Bold(true).Foreground(accent)
	refS := lipgloss.NewStyle().Foreground(ui.ColorHash).Bold(true)
	msgS := lipgloss.NewStyle().Foreground(ui.ColorHeader)

	// Buttons — No is on the left (safe default), Yes on the right
	noStyle := lipgloss.NewStyle().Bold(true).Padding(0, 3)
	yesStyle := lipgloss.NewStyle().Bold(true).Padding(0, 3)
	if m.confirmFocus == 0 {
		noStyle = noStyle.Background(ui.ColorCursorBg).Foreground(accent)
	} else {
		yesStyle = yesStyle.Background(accent).Foreground(ui.ColorCursorFg)
	}

	buttonRow := noStyle.Render("✕  No") + "      " + yesStyle.Render("✓  Yes")
	hint := ui.StyleDim.Render("← →  navigate   enter  confirm")

	content := lipgloss.JoinVertical(lipgloss.Left,
		titleS.Render(titleText),
		"",
		refS.Render(entry.Ref)+"  "+msgS.Render(truncateStr(entry.Message, 40)),
		ui.StyleDim.Render("branch: "+truncateStr(entry.Branch, 35)),
		"",
		"  "+buttonRow,
		"",
		hint,
	)

	box := lipgloss.NewStyle().
		BorderStyle(lipgloss.DoubleBorder()).
		BorderForeground(accent).
		Padding(1, 3).
		Width(54).
		Render(content)

	return "\n" + lipgloss.PlaceHorizontal(m.width, lipgloss.Center, box) + "\n"
}

// ── Async commands ────────────────────────────────────────────────────────────

func doLoadBrowseDetail(entry git.StashEntry) tea.Cmd {
	return func() tea.Msg {
		d, err := git.GetStashDetail(entry.Ref)
		if err == nil {
			d.StashEntry = entry // populate all metadata fields
		}
		return browseDetailLoadedMsg{ref: entry.Ref, detail: d, err: err}
	}
}

func doStashAction(ref string, mode BrowseMode) tea.Cmd {
	return func() tea.Msg {
		var err error
		switch mode {
		case BrowseModeApply:
			err = git.StashApply(ref)
		case BrowseModePop:
			err = git.StashPop(ref)
		case BrowseModeDrop:
			err = git.StashDrop(ref)
		}
		return browseActionDoneMsg{err: err}
	}
}

// ── Width helpers ─────────────────────────────────────────────────────────────

func (m BrowseModel) listWidth() int {
	dw := m.width * browseDetailPct / 100
	lw := m.width - dw - 3
	if lw < 30 {
		lw = 30
	}
	return lw
}

func (m BrowseModel) detailWidth() int {
	dw := m.width - m.listWidth() - 3
	if dw < 20 {
		dw = 20
	}
	return dw
}

// Result returns the post-quit message (empty if quit without action).
func (m BrowseModel) Result() string { return m.actionResult }

// ── Helpers ───────────────────────────────────────────────────────────────────

func fixedW(s string, w int) string {
	return lipgloss.NewStyle().Width(w).MaxWidth(w).Render(s)
}

func fitW(s string, w int) string {
	vis := lipgloss.Width(s)
	switch {
	case vis == w:
		return s
	case vis < w:
		return s + strings.Repeat(" ", w-vis)
	default:
		plain := stripANSIStr(s)
		r := []rune(plain)
		if len(r) > w {
			r = r[:w]
		}
		return string(r)
	}
}

func stripANSIStr(s string) string {
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

func truncateStr(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
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
