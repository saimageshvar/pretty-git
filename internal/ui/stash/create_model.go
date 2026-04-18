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
	msg string
}

// paneRows is the fixed visible height for scrollable panes in the create wizard.
const paneRows = 10

// CreateModel is the Bubble Tea model for the stash creation wizard.
type CreateModel struct {
	phase createPhase

	// Type selector (left pane)
	files          []git.FileStatus
	typeCursor     int
	stagedCount    int
	unstagedCount  int
	allCount       int
	rightPaneFocus bool // true when right pane (file list / preview) has focus

	// Right pane — file list (custom) or preview scroll (other types)
	fileCursor   int
	fileOffset   int
	fileSelected []bool

	// Message phase
	stashType  int
	msgInput   textinput.Model
	defaultMsg string

	// Executing phase
	spinner spinner.Model
	result  string
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

	// Custom mode starts with nothing selected — user must explicitly pick files.
	allSelected := make([]bool, len(files))

	return CreateModel{
		phase:         phaseTypeSelect,
		files:         files,
		typeCursor:    startCursor,
		stagedCount:   staged,
		unstagedCount: unstaged,
		allCount:      all,
		fileSelected:  allSelected,
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
		m.result = "✓ stash created: " + msg.msg
		return m, tea.Quit

	case tea.KeyMsg:
		if key.Matches(msg, m.keys.Quit) {
			return m, tea.Quit
		}

		switch m.phase {
		case phaseTypeSelect:
			return m.updateTypeSelect(msg)
		case phaseMessage:
			return m.updateMessage(msg)
		}
	}

	return m, nil
}

func (m CreateModel) updateTypeSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.rightPaneFocus {
		return m.updateRightPane(msg)
	}
	return m.updateLeftPane(msg)
}

func (m CreateModel) updateLeftPane(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

	case key.Matches(msg, m.keys.FocusRight):
		m.rightPaneFocus = true
		m.fileCursor = 0
		m.fileOffset = 0

	case key.Matches(msg, m.keys.Select):
		if m.typeCursor == stashTypeCustom {
			// Force user into the file picker — never skip it for custom.
			m.rightPaneFocus = true
			m.fileCursor = 0
			m.fileOffset = 0
		} else {
			m.stashType = m.typeCursor
			m.phase = phaseMessage
		}
	}

	// Reset scroll when type changes
	if m.typeCursor != prev {
		m.fileCursor = 0
		m.fileOffset = 0
	}
	return m, nil
}

func (m CreateModel) updateRightPane(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	isCustom := m.typeCursor == stashTypeCustom

	switch {
	case key.Matches(msg, m.keys.Back), key.Matches(msg, m.keys.FocusLeft):
		m.rightPaneFocus = false

	case key.Matches(msg, m.keys.Up):
		if m.fileCursor > 0 {
			m.fileCursor--
			m.clampFileScroll()
		}

	case key.Matches(msg, m.keys.Down):
		files := m.filesForType(m.typeCursor)
		if m.fileCursor < len(files)-1 {
			m.fileCursor++
			m.clampFileScroll()
		}

	case key.Matches(msg, m.keys.Toggle):
		if isCustom && m.fileCursor < len(m.fileSelected) {
			m.fileSelected[m.fileCursor] = !m.fileSelected[m.fileCursor]
		}

	case key.Matches(msg, m.keys.All):
		if isCustom {
			for i := range m.fileSelected {
				m.fileSelected[i] = true
			}
		}

	case key.Matches(msg, m.keys.None):
		if isCustom {
			for i := range m.fileSelected {
				m.fileSelected[i] = false
			}
		}

	case key.Matches(msg, m.keys.Select):
		m.stashType = m.typeCursor
		if isCustom {
			count := 0
			for _, sel := range m.fileSelected {
				if sel {
					count++
				}
			}
			if count > 0 {
				m.phase = phaseMessage
				m.rightPaneFocus = false
			}
		} else {
			m.phase = phaseMessage
			m.rightPaneFocus = false
		}
	}

	return m, nil
}

func (m *CreateModel) clampFileScroll() {
	rows := paneRows - 4 // matches fileRowsAvail in viewTypeSelect
	if m.fileCursor < m.fileOffset {
		m.fileOffset = m.fileCursor
	}
	if m.fileCursor >= m.fileOffset+rows {
		m.fileOffset = m.fileCursor - rows + 1
	}
}

func (m CreateModel) updateMessage(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back):
		m.execErr = nil
		m.phase = phaseTypeSelect
		m.rightPaneFocus = false

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
		userMsg := strings.TrimSpace(m.msgInput.Value())
		if userMsg == "" {
			userMsg = m.defaultMsg
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

		result, err := git.StashPush(userMsg, stashTypeStr, customFiles)
		return stashDoneMsg{err: err, msg: result}
	}
}

// View renders the current phase.
func (m CreateModel) View() string {
	var sb strings.Builder

	// Header
	sb.WriteString(ui.StyleHeader.Render("  ✦ Stash") + "  " + ui.StyleAccent.Render(m.repoName))
	switch m.phase {
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
	case phaseMessage:
		sb.WriteString(m.viewMessage())
	case phaseExecuting:
		sb.WriteString("\n  " + m.spinner.View() + " Creating stash…\n")
	}

	sb.WriteString(ui.StyleDivider.Render(strings.Repeat("─", m.width)) + "\n")

	// Footer help
	switch m.phase {
	case phaseTypeSelect:
		if m.rightPaneFocus {
			if m.typeCursor == stashTypeCustom {
				sb.WriteString("  " + m.help.ShortHelpView(m.keys.rightPaneCustomHelp()))
			} else {
				sb.WriteString("  " + m.help.ShortHelpView(m.keys.rightPanePreviewHelp()))
			}
		} else {
			sb.WriteString("  " + m.help.ShortHelpView(m.keys.leftPaneHelp()))
		}
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

	// Build right-pane: fixed paneRows height with scroll indicator at bottom
	// 4 header/footer lines (blank + title + rule + scroll), rest is file rows
	fileRowsAvail := paneRows - 4
	if fileRowsAvail < 1 {
		fileRowsAvail = 1
	}

	// Divider color: accent when right pane focused, dim otherwise
	dividerColor := ui.ColorDivider
	if m.rightPaneFocus {
		dividerColor = ui.ColorAccent
	}

	var rightLines []string
	rightLines = append(rightLines, "") // blank row aligns with left pane's first blank

	if m.typeCursor == stashTypeCustom {
		// Custom mode: show interactive checkbox list in right pane
		selectedCount := 0
		for _, sel := range m.fileSelected {
			if sel {
				selectedCount++
			}
		}

		totalFiles := len(m.files)
		titleCount := ui.StyleCountBadge.Render(fmt.Sprintf("%d total  %d selected", totalFiles, selectedCount))
		titleStyle := lipgloss.NewStyle().Foreground(ui.ColorAccent).Bold(true)
		rightLines = append(rightLines, titleStyle.Render("  custom files")+"  "+titleCount)
		rightLines = append(rightLines, ui.StyleDim.Render(strings.Repeat("─", dw-2)))

		above := m.fileOffset
		below := len(m.files) - m.fileOffset - fileRowsAvail
		if below < 0 {
			below = 0
		}

		end := m.fileOffset + fileRowsAvail
		if end > len(m.files) {
			end = len(m.files)
		}
		for i := m.fileOffset; i < end; i++ {
			f := m.files[i]
			isCursor := m.rightPaneFocus && i == m.fileCursor
			isSelected := i < len(m.fileSelected) && m.fileSelected[i]
			checkTxt := "[ ]"
			if isSelected {
				checkTxt = "[✓]"
			}

			var line string
			if isCursor {
				bg := ui.ColorCursorBg
				checkS  := lipgloss.NewStyle().Background(bg).Foreground(ui.ColorAccent).Bold(true).Render(checkTxt)
				spaceS  := lipgloss.NewStyle().Background(bg).Render(" ")
				statusS := lipgloss.NewStyle().Background(bg).Foreground(statusFg(f.StatusDisplay())).Bold(true).Render(f.StatusDisplay())
				pathS   := lipgloss.NewStyle().Background(bg).Foreground(ui.ColorCursorFg).Bold(true).Render(truncateCreateStr(f.Path, dw-9))
				line = "  " + checkS + spaceS + statusS + spaceS + pathS
				used := lipgloss.Width(line)
				if dw > used {
					line += lipgloss.NewStyle().Background(bg).Render(strings.Repeat(" ", dw-used))
				}
			} else {
				checkS := ui.StyleDim.Render(checkTxt)
				if isSelected {
					checkS = lipgloss.NewStyle().Foreground(ui.ColorAccent).Bold(true).Render(checkTxt)
				}
				line = "  " + checkS + " " + statusColor(f.StatusDisplay()) + " " + ui.StyleDim.Render(truncateCreateStr(f.Path, dw-9))
			}
			rightLines = append(rightLines, line)
		}

		// Scroll footer
		hintStyle := lipgloss.NewStyle().Foreground(ui.ColorKeyHint).Bold(true)
		var scrollFooter string
		switch {
		case above > 0 && below > 0:
			scrollFooter = hintStyle.Render(fmt.Sprintf("  ↑%d above · ↓%d more files", above, below))
		case below > 0:
			scrollFooter = hintStyle.Render(fmt.Sprintf("  ↓ %d more files", below))
		case above > 0:
			scrollFooter = hintStyle.Render(fmt.Sprintf("  ↑ %d files above", above))
		}
		rightLines = append(rightLines, scrollFooter)
	} else {
		// Preview mode: file list with cursor highlight for focused right pane
		previewFiles := m.filesForType(m.typeCursor)
		totalFiles := len(previewFiles)
		titleCount := ui.StyleCountBadge.Render(fmt.Sprintf("%d total", totalFiles))
		rightLines = append(rightLines, lipgloss.NewStyle().Foreground(ui.ColorAccent).Bold(true).Render("  files to stash")+"  "+titleCount)
		rightLines = append(rightLines, ui.StyleDim.Render(strings.Repeat("─", dw-2)))

		if len(previewFiles) == 0 {
			rightLines = append(rightLines, ui.StyleDim.Render("  (none)"))
		} else {
			// Clamp fileOffset
			maxOff := len(previewFiles) - fileRowsAvail
			if maxOff < 0 {
				maxOff = 0
			}
			offset := m.fileOffset
			if offset > maxOff {
				offset = maxOff
			}

			end := offset + fileRowsAvail
			if end > len(previewFiles) {
				end = len(previewFiles)
			}
			for i := offset; i < end; i++ {
				f := previewFiles[i]
				isCursor := m.rightPaneFocus && i == m.fileCursor
				if isCursor {
					bg := ui.ColorCursorBg
					statusS := lipgloss.NewStyle().Background(bg).Foreground(statusFg(f.StatusDisplay())).Bold(true).Render(f.StatusDisplay())
					spaceS  := lipgloss.NewStyle().Background(bg).Render("  ")
					pathS   := lipgloss.NewStyle().Background(bg).Foreground(ui.ColorCursorFg).Bold(true).Render(truncateCreateStr(f.Path, dw-6))
					line := "  " + statusS + spaceS + pathS
					used := lipgloss.Width(line)
					if dw > used {
						line += lipgloss.NewStyle().Background(bg).Render(strings.Repeat(" ", dw-used))
					}
					rightLines = append(rightLines, line)
				} else {
					rightLines = append(rightLines, "  "+statusColor(f.StatusDisplay())+"  "+ui.StyleDim.Render(truncateCreateStr(f.Path, dw-6)))
				}
			}

			// Scroll footer
			above := m.fileOffset
			below := len(previewFiles) - m.fileOffset - fileRowsAvail
			if below < 0 {
				below = 0
			}
			hintStyle := lipgloss.NewStyle().Foreground(ui.ColorKeyHint).Bold(true)
			var scrollFooter string
			switch {
			case above > 0 && below > 0:
				scrollFooter = hintStyle.Render(fmt.Sprintf("  ↑%d above · ↓%d more files", above, below))
			case below > 0:
				scrollFooter = hintStyle.Render(fmt.Sprintf("  ↓ %d more files", below))
			case above > 0:
				scrollFooter = hintStyle.Render(fmt.Sprintf("  ↑ %d files above", above))
			}
			rightLines = append(rightLines, scrollFooter)
		}
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
	divider := lipgloss.NewStyle().Foreground(dividerColor).Render("│")
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
