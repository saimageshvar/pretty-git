package stash

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

// stashType constants.
const (
	stashTypeStaged   = 0
	stashTypeUnstaged = 1
	stashTypeAll      = 2
	stashTypeCustom   = 3
)

// stashDoneMsg is returned when the stash operation completes.
type stashDoneMsg struct {
	err error
}

// paneRows is the fixed visible height for scrollable panes in the create wizard.
const paneRows = 10

// CreateModel is the Bubble Tea model for the 3-phase stash creation wizard.
type CreateModel struct {
	phase createPhase

	// Phase 0 — type selector
	files         []git.FileStatus
	typeCursor    int
	stagedCount   int
	unstagedCount int
	allCount      int
	previewOffset int // scroll offset for right-pane file preview

	// Phase 1 — file multi-select
	fileCursor   int
	fileOffset   int // scroll offset for file list
	fileSelected []bool

	// Phase 2 — message input
	stashType  int // resolved stashType constant
	msgInput   textinput.Model
	defaultMsg string

	// Phase 3 — executing
	spinner spinner.Model
	result  string // success message shown after quit
	execErr error

	width    int
	height   int
	repoName string

	help help.Model
	keys createKeyMap
}

// typeOption describes one stash-type option in phase 0.
type typeOption struct {
	label string
	desc  string
	count int
}

func (m *CreateModel) typeOptions() []typeOption {
	return []typeOption{
		{"Staged files", "only indexed changes", m.stagedCount},
		{"Unstaged files", "working-tree changes only", m.unstagedCount},
		{"All modified", "staged + unstaged + untracked", m.allCount},
		{"Custom files…", "pick specific files", m.allCount},
	}
}

// filesForType returns the file list that would be stashed for a given type index.
func (m *CreateModel) filesForType(t int) []git.FileStatus {
	switch t {
	case stashTypeStaged:
		var out []git.FileStatus
		for _, f := range m.files {
			if len(f.Code) > 0 && f.Code[0] != ' ' && f.Code[0] != '?' {
				out = append(out, f)
			}
		}
		return out
	case stashTypeUnstaged:
		var out []git.FileStatus
		for _, f := range m.files {
			if len(f.Code) > 1 && f.Code[1] != ' ' && f.Code != "??" {
				out = append(out, f)
			}
		}
		return out
	case stashTypeAll:
		return m.files
	case stashTypeCustom:
		return m.files // preview all; user picks in next phase
	}
	return nil
}

// NewCreate initialises the create wizard.
func NewCreate(files []git.FileStatus, repoName string, termWidth, termHeight int) CreateModel {
	staged := git.CountStagedFiles(files)
	unstaged := git.CountUnstagedFiles(files)
	all := len(files)

	ti := textinput.New()
	ti.Placeholder = git.LastCommitOneLiner()
	ti.Focus()
	ti.CharLimit = 200
	ti.Width = termWidth - 6

	sp := spinner.New()
	sp.Spinner = spinner.MiniDot
	sp.Style = lipgloss.NewStyle().Bold(true).Foreground(ui.ColorAccent)

	h := help.New()
	h.Styles.ShortKey = lipgloss.NewStyle().Bold(true).Foreground(ui.ColorKeyHint)
	h.Styles.ShortDesc = lipgloss.NewStyle().Foreground(ui.ColorHeader)
	h.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(ui.ColorTreeConnector)
	h.Width = termWidth

	// Start cursor on first option with files
	startCursor := 2 // default to "All modified"
	if staged > 0 {
		startCursor = 0
	}

	return CreateModel{
		phase:         phaseTypeSelect,
		files:         files,
		typeCursor:    startCursor,
		stagedCount:   staged,
		unstagedCount: unstaged,
		allCount:      all,
		fileSelected:  make([]bool, len(files)),
		msgInput:      ti,
		defaultMsg:    git.LastCommitOneLiner(),
		spinner:       sp,
		width:         termWidth,
		height:        termHeight,
		repoName:      repoName,
		help:          h,
		keys:          defaultCreateKeyMap(),
	}
}

func (m CreateModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m CreateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		m.msgInput.Width = msg.Width - 6
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case stashDoneMsg:
		if msg.err != nil {
			m.execErr = msg.err
			m.phase = phaseMessage // go back to message phase to show error
			return m, nil
		}
		m.result = "✓ stash created"
		return m, tea.Quit

	case tea.KeyMsg:
		if key.Matches(msg, m.keys.Quit) {
			return m, tea.Quit
		}

		switch m.phase {
		case phaseTypeSelect:
			return m.updateTypeSelect(msg)
		case phaseFileSelect:
			return m.updateFileSelect(msg)
		case phaseMessage:
			return m.updateMessage(msg)
		}
	}

	return m, nil
}

func (m CreateModel) updateTypeSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	opts := m.typeOptions()
	prev := m.typeCursor
	switch {
	case key.Matches(msg, m.keys.Back):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Up):
		for i := m.typeCursor - 1; i >= 0; i-- {
			if i == stashTypeCustom || opts[i].count > 0 {
				m.typeCursor = i
				break
			}
		}

	case key.Matches(msg, m.keys.Down):
		for i := m.typeCursor + 1; i < len(opts); i++ {
			if i == stashTypeCustom || opts[i].count > 0 {
				m.typeCursor = i
				break
			}
		}

	case key.Matches(msg, m.keys.Select):
		m.stashType = m.typeCursor
		if m.typeCursor == stashTypeCustom {
			m.phase = phaseFileSelect
			for i := range m.fileSelected {
				m.fileSelected[i] = true
			}
		} else {
			m.phase = phaseMessage
		}
	}

	// Reset preview scroll when type changes
	if m.typeCursor != prev {
		m.previewOffset = 0
	}
	return m, nil
}

func (m CreateModel) updateFileSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back):
		m.phase = phaseTypeSelect

	case key.Matches(msg, m.keys.Up):
		if m.fileCursor > 0 {
			m.fileCursor--
			m.clampFileScroll()
		}

	case key.Matches(msg, m.keys.Down):
		if m.fileCursor < len(m.files)-1 {
			m.fileCursor++
			m.clampFileScroll()
		}

	case key.Matches(msg, m.keys.Toggle):
		if m.fileCursor < len(m.fileSelected) {
			m.fileSelected[m.fileCursor] = !m.fileSelected[m.fileCursor]
		}

	case key.Matches(msg, m.keys.All):
		for i := range m.fileSelected {
			m.fileSelected[i] = true
		}

	case key.Matches(msg, m.keys.None):
		for i := range m.fileSelected {
			m.fileSelected[i] = false
		}

	case key.Matches(msg, m.keys.Confirm):
		count := 0
		for _, sel := range m.fileSelected {
			if sel {
				count++
			}
		}
		if count > 0 {
			m.phase = phaseMessage
		}
	}

	return m, nil
}

func (m *CreateModel) clampFileScroll() {
	if m.fileCursor < m.fileOffset {
		m.fileOffset = m.fileCursor
	}
	if m.fileCursor >= m.fileOffset+paneRows {
		m.fileOffset = m.fileCursor - paneRows + 1
	}
}

func (m CreateModel) updateMessage(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back):
		m.execErr = nil
		if m.stashType == stashTypeCustom {
			m.phase = phaseFileSelect
		} else {
			m.phase = phaseTypeSelect
		}

	case key.Matches(msg, m.keys.Confirm):
		m.phase = phaseExecuting
		return m, tea.Batch(m.spinner.Tick, m.doStash())
	}

	// Forward key to text input
	var cmd tea.Cmd
	m.msgInput, cmd = m.msgInput.Update(msg)
	return m, cmd
}

func (m *CreateModel) doStash() tea.Cmd {
	return func() tea.Msg {
		// Build the message: "hash: user message" or just hash if empty
		shortHash := git.LastCommitShortHash()
		userMsg := strings.TrimSpace(m.msgInput.Value())
		var finalMsg string
		if userMsg == "" {
			finalMsg = m.defaultMsg
		} else {
			finalMsg = shortHash + ": " + userMsg
		}

		// Determine stash type string
		var stashTypeStr string
		switch m.stashType {
		case stashTypeStaged:
			stashTypeStr = "staged"
		case stashTypeUnstaged:
			stashTypeStr = "unstaged"
		case stashTypeAll:
			stashTypeStr = "all"
		case stashTypeCustom:
			stashTypeStr = "custom"
		}

		// Collect custom file paths
		var customFiles []string
		if m.stashType == stashTypeCustom {
			for i, f := range m.files {
				if i < len(m.fileSelected) && m.fileSelected[i] {
					customFiles = append(customFiles, f.Path)
				}
			}
		}

		err := git.StashPush(finalMsg, stashTypeStr, customFiles)
		return stashDoneMsg{err: err}
	}
}

// View renders the current phase.
func (m CreateModel) View() string {
	var sb strings.Builder

	// Header
	sb.WriteString(ui.StyleHeader.Render("  ✦ Stash") + "  " + ui.StyleAccent.Render(m.repoName))
	switch m.phase {
	case phaseFileSelect:
		sb.WriteString("  " + ui.StyleDim.Render("select files"))
	case phaseMessage:
		sb.WriteString("  " + ui.StyleDim.Render("stash message"))
	case phaseExecuting:
		sb.WriteString("  " + ui.StyleDim.Render("creating…"))
	}
	sb.WriteString("\n")
	sb.WriteString(ui.StyleDivider.Render(strings.Repeat("─", m.width)) + "\n")

	switch m.phase {
	case phaseTypeSelect:
		sb.WriteString(m.viewTypeSelect())
	case phaseFileSelect:
		sb.WriteString(m.viewFileSelect())
	case phaseMessage:
		sb.WriteString(m.viewMessage())
	case phaseExecuting:
		sb.WriteString("\n  " + m.spinner.View() + " Creating stash…\n")
	}

	sb.WriteString(ui.StyleDivider.Render(strings.Repeat("─", m.width)) + "\n")

	// Footer help
	switch m.phase {
	case phaseTypeSelect:
		sb.WriteString("  " + m.help.ShortHelpView(m.keys.typeSelectHelp()))
	case phaseFileSelect:
		sb.WriteString("  " + m.help.ShortHelpView(m.keys.fileSelectHelp()))
	case phaseMessage:
		sb.WriteString("  " + m.help.ShortHelpView(m.keys.messageHelp()))
	}

	return sb.String()
}

func (m CreateModel) viewTypeSelect() string {
	// Split into left (option list) ~45% and right (file preview) ~55%
	lw := m.width * 45 / 100
	if lw < 32 {
		lw = 32
	}
	dw := m.width - lw - 3 // 3 = " │ "
	if dw < 18 {
		dw = 18
	}

	opts := m.typeOptions()
	// desc column width: lw - cursor(2) - radio(1) - space(1) - label(18) - space(1)
	descWidth := lw - 23
	if descWidth < 8 {
		descWidth = 8
	}

	// Build left-pane rows.
	// Rule: never pass pre-styled strings into another Render() — style each
	// element independently, then measure + fill remaining space manually.
	padToLW := func(row string) string {
		used := lipgloss.Width(row)
		if lw > used {
			return row + strings.Repeat(" ", lw-used)
		}
		return row
	}

	var leftRows []string
	leftRows = append(leftRows, strings.Repeat(" ", lw)) // blank line
	leftRows = append(leftRows, padToLW(lipgloss.NewStyle().Bold(true).Foreground(ui.ColorHeader).Render("  Select what to stash:")))
	leftRows = append(leftRows, strings.Repeat(" ", lw)) // blank line

	for i, opt := range opts {
		isCursor := i == m.typeCursor
		isEmpty := (i != stashTypeCustom) && opt.count == 0
		bg := ui.ColorCursorBg

		var row string
		if isCursor && !isEmpty {
			// Each element gets background applied individually (avoids ANSI width miscounts)
			cursorS := lipgloss.NewStyle().Background(bg).Foreground(ui.ColorAccent).Bold(true).Render("» ")
			radioS  := lipgloss.NewStyle().Background(bg).Foreground(ui.ColorAccent).Bold(true).Render("● ")
			labelS  := lipgloss.NewStyle().Background(bg).Foreground(ui.ColorCursorFg).Bold(true).Width(18).Render(opt.label)
			spaceS  := lipgloss.NewStyle().Background(bg).Render(" ")
			descS   := lipgloss.NewStyle().Background(bg).Foreground(ui.ColorDesc).Render(truncateCreateStr(opt.desc, descWidth))
			row = cursorS + radioS + labelS + spaceS + descS
			// Fill remainder with background
			used := lipgloss.Width(row)
			if lw > used {
				row += lipgloss.NewStyle().Background(bg).Render(strings.Repeat(" ", lw-used))
			}
		} else if isCursor {
			// Cursor on an empty option — dim everything, no background
			cursorS := lipgloss.NewStyle().Foreground(ui.ColorDim).Render("» ")
			radioS  := ui.StyleDim.Render("○ ")
			labelS  := lipgloss.NewStyle().Foreground(ui.ColorDim).Width(18).Render(opt.label)
			descS   := ui.StyleDim.Render(truncateCreateStr(opt.desc, descWidth))
			row = padToLW(cursorS + radioS + labelS + " " + descS)
		} else if isEmpty {
			radioS := ui.StyleDim.Render("○ ")
			labelS := lipgloss.NewStyle().Foreground(ui.ColorDim).Width(18).Render(opt.label)
			descS  := ui.StyleDim.Render(truncateCreateStr(opt.desc, descWidth))
			row = padToLW("  " + radioS + labelS + " " + descS)
		} else {
			radioS := lipgloss.NewStyle().Foreground(ui.ColorAccent).Render("○ ")
			labelS := lipgloss.NewStyle().Width(18).Render(opt.label)
			descS  := ui.StyleDim.Render(truncateCreateStr(opt.desc, descWidth))
			row = padToLW("  " + radioS + labelS + " " + descS)
		}
		leftRows = append(leftRows, row)
	}
	leftRows = append(leftRows, strings.Repeat(" ", lw)) // trailing blank

	// Build right-pane: fixed paneRows height with scroll
	previewFiles := m.filesForType(m.typeCursor)
	// 3 header lines (blank + title + rule), rest is file rows
	fileRowsAvail := paneRows - 3
	if fileRowsAvail < 1 {
		fileRowsAvail = 1
	}

	// Build full list of file lines first, then window
	var allFileLines []string
	if len(previewFiles) == 0 {
		allFileLines = append(allFileLines, ui.StyleDim.Render("  (none)"))
	} else {
		for _, f := range previewFiles {
			status := statusColor(f.StatusDisplay())
			path := ui.StyleDim.Render(truncateCreateStr(f.Path, dw-6))
			allFileLines = append(allFileLines, "  "+status+"  "+path)
		}
	}

	// Clamp previewOffset
	maxOff := len(allFileLines) - fileRowsAvail
	if maxOff < 0 {
		maxOff = 0
	}
	offset := m.previewOffset
	if offset > maxOff {
		offset = maxOff
	}

	previewTitle := "  files to stash"
	if m.typeCursor == stashTypeCustom {
		previewTitle = "  all changed files"
	}
	scrollHint := ""
	below := len(allFileLines) - offset - fileRowsAvail
	if below > 0 {
		scrollHint = ui.StyleDim.Render(fmt.Sprintf("  ↓%d", below))
	}
	above := offset
	if above > 0 {
		scrollHint = ui.StyleDim.Render(fmt.Sprintf("  ↑%d", above)) + scrollHint
	}

	var rightLines []string
	rightLines = append(rightLines, strings.Repeat(" ", lw)) // blank row aligns with left pane's first blank
	rightLines = append(rightLines, lipgloss.NewStyle().Foreground(ui.ColorAccent).Bold(true).Render(previewTitle)+scrollHint)
	rightLines = append(rightLines, ui.StyleDim.Render(strings.Repeat("─", dw-2)))

	end := offset + fileRowsAvail
	if end > len(allFileLines) {
		end = len(allFileLines)
	}
	for _, line := range allFileLines[offset:end] {
		rightLines = append(rightLines, line)
	}
	// Pad right pane to exactly paneRows
	for len(rightLines) < paneRows {
		rightLines = append(rightLines, "")
	}

	// Left pane: pad to paneRows too
	for len(leftRows) < paneRows {
		leftRows = append(leftRows, strings.Repeat(" ", lw))
	}

	// Merge left + divider + right (exactly paneRows rows)
	divider := lipgloss.NewStyle().Foreground(ui.ColorDivider).Render("│")
	var sb strings.Builder
	for i := 0; i < paneRows; i++ {
		left := leftRows[i]
		used := lipgloss.Width(left)
		if lw > used {
			left += strings.Repeat(" ", lw-used)
		}
		right := ""
		if i < len(rightLines) {
			right = rightLines[i]
		}
		sb.WriteString(left + " " + divider + " " + right + "\n")
	}
	return sb.String()
}

func (m CreateModel) viewFileSelect() string {
	var sb strings.Builder

	selectedCount := 0
	for _, sel := range m.fileSelected {
		if sel {
			selectedCount++
		}
	}

	// Scroll indicators
	above := m.fileOffset
	below := len(m.files) - m.fileOffset - paneRows
	if below < 0 {
		below = 0
	}

	// Header row with scroll hint
	hint := ""
	if above > 0 {
		hint += ui.StyleDim.Render(fmt.Sprintf(" ↑%d", above))
	}
	if below > 0 {
		hint += ui.StyleDim.Render(fmt.Sprintf(" ↓%d", below))
	}
	sb.WriteString("\n  " + ui.StyleDim.Render(fmt.Sprintf("%d files", len(m.files))) + hint + "\n\n")

	// Windowed file list
	end := m.fileOffset + paneRows
	if end > len(m.files) {
		end = len(m.files)
	}
	for i := m.fileOffset; i < end; i++ {
		f := m.files[i]
		isCursor := i == m.fileCursor
		isSelected := i < len(m.fileSelected) && m.fileSelected[i]

		var line string
		if isCursor {
			bg := ui.ColorCursorBg
			cursorS  := lipgloss.NewStyle().Background(bg).Foreground(ui.ColorAccent).Bold(true).Render("» ")
			checkTxt := "[ ]"
			if isSelected {
				checkTxt = "[✓]"
			}
			checkS  := lipgloss.NewStyle().Background(bg).Foreground(ui.ColorAccent).Bold(true).Render(checkTxt)
			spaceS  := lipgloss.NewStyle().Background(bg).Render("  ")
			statusS := lipgloss.NewStyle().Background(bg).Foreground(statusFg(f.StatusDisplay())).Bold(true).Render(f.StatusDisplay())
			pathS   := lipgloss.NewStyle().Background(bg).Foreground(ui.ColorCursorFg).Bold(true).Render(f.Path)
			line = cursorS + checkS + spaceS + statusS + spaceS + pathS
			used := lipgloss.Width(line)
			if m.width > used {
				line += lipgloss.NewStyle().Background(bg).Render(strings.Repeat(" ", m.width-used))
			}
		} else {
			checkTxt := "[ ]"
			if isSelected {
				checkTxt = "[✓]"
			}
			checkS := ui.StyleDim.Render(checkTxt)
			if isSelected {
				checkS = lipgloss.NewStyle().Foreground(ui.ColorAccent).Bold(true).Render(checkTxt)
			}
			line = "  " + checkS + "  " + statusColor(f.StatusDisplay()) + "  " + f.Path
		}
		sb.WriteString(line + "\n")
	}

	// Pad to fixed height
	rendered := end - m.fileOffset
	for rendered < paneRows {
		sb.WriteString("\n")
		rendered++
	}

	sb.WriteString("\n  ")
	sb.WriteString(ui.StyleCountBadge.Render(fmt.Sprintf("%d selected", selectedCount)))
	sb.WriteString("\n")
	return sb.String()
}

func (m CreateModel) viewMessage() string {
	var sb strings.Builder
	sb.WriteString("\n")

	// File summary (left aligned, compact)
	typeNames := []string{"staged files", "unstaged files", "modified files", "selected files"}
	var filesToShow []git.FileStatus
	switch m.stashType {
	case stashTypeStaged:
		for _, f := range m.files {
			if len(f.Code) > 0 && f.Code[0] != ' ' && f.Code[0] != '?' {
				filesToShow = append(filesToShow, f)
			}
		}
	case stashTypeUnstaged:
		for _, f := range m.files {
			if len(f.Code) > 1 && f.Code[1] != ' ' && f.Code != "??" {
				filesToShow = append(filesToShow, f)
			}
		}
	case stashTypeAll:
		filesToShow = m.files
	case stashTypeCustom:
		for i, f := range m.files {
			if i < len(m.fileSelected) && m.fileSelected[i] {
				filesToShow = append(filesToShow, f)
			}
		}
	}

	typeLabel := typeNames[m.stashType]
	sb.WriteString("  " + ui.StyleHeader.Render(fmt.Sprintf("Stashing %d %s:", len(filesToShow), typeLabel)) + "\n")
	maxShow := 6
	for i, f := range filesToShow {
		if i >= maxShow {
			sb.WriteString("    " + ui.StyleDim.Render(fmt.Sprintf("… and %d more", len(filesToShow)-maxShow)) + "\n")
			break
		}
		sb.WriteString("    " + statusColor(f.StatusDisplay()) + "  " + ui.StyleDim.Render(f.Path) + "\n")
	}

	sb.WriteString("\n")

	// Prominent message input box
	boxWidth := m.width - 6
	if boxWidth < 20 {
		boxWidth = 20
	}
	m.msgInput.Width = boxWidth - 4 // account for border + padding

	label := lipgloss.NewStyle().
		Foreground(ui.ColorAccent).
		Bold(true).
		Render(" stash message ")

	inputBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorAccent).
		Padding(0, 1).
		Width(boxWidth).
		Render(m.msgInput.View())

	sb.WriteString("  " + label + "\n")
	sb.WriteString("  " + inputBox + "\n")

	// Error from previous attempt
	if m.execErr != nil {
		sb.WriteString("\n  " + ui.StyleError.Render("✗ "+m.execErr.Error()) + "\n")
	}

	sb.WriteString("\n")
	return sb.String()
}

// statusFg returns the foreground color for a status letter.
func statusFg(s string) lipgloss.TerminalColor {
	switch s {
	case "M":
		return lipgloss.AdaptiveColor{Light: "#B45309", Dark: "#FCD34D"}
	case "A":
		return ui.ColorParentMerged
	case "D":
		return ui.ColorError
	case "R":
		return ui.ColorRemote
	default:
		return ui.ColorDim
	}
}

// statusColor returns a bold colored status letter.
func statusColor(s string) string {
	return lipgloss.NewStyle().Foreground(statusFg(s)).Bold(true).Render(s)
}

// Result returns the success message (empty if not done or error).
func (m CreateModel) Result() string { return m.result }

func truncateCreateStr(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
