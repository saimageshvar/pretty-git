package ghpr

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sai/pretty-git/internal/gh"
	ui "github.com/sai/pretty-git/internal/ui"
)

// ── Key bindings ───────────────────────────────────────────────────────────

type keyMap struct {
	Up      key.Binding
	Down    key.Binding
	Open    key.Binding
	Refresh key.Binding
	Quit    key.Binding
	Back    key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Open: key.NewBinding(
			key.WithKeys("enter", "o"),
			key.WithHelp("enter/o", "open in browser"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r", "ctrl+r"),
			key.WithHelp("r", "refresh"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back/quit"),
		),
	}
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Open, k.Refresh, k.Quit}
}

// ── Messages ───────────────────────────────────────────────────────────────

type prsLoadedMsg struct {
	prs []gh.PR
	err error
}

type openBrowserMsg struct {
	url string
}

// ── Model ────────────────────────────────────────────────────────────────────

type Model struct {
	prs         []gh.PR
	cursor      int
	offset      int
	loading     bool
	err         string
	width       int
	height      int
	visibleRows int
	repoName    string

	help    help.Model
	keys    keyMap
	spinner spinner.Model
}

const maxVisible = 15

func New(repoName string, termWidth, termHeight int) Model {
	h := help.New()
	h.Styles.ShortKey = lipgloss.NewStyle().Bold(true).Foreground(ui.ColorKeyHint)
	h.Styles.ShortDesc = lipgloss.NewStyle().Foreground(ui.ColorHeader)
	h.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(ui.ColorTreeConnector)
	h.Width = termWidth

	sp := spinner.New()
	sp.Spinner = spinner.MiniDot
	sp.Style = lipgloss.NewStyle().Bold(true).Foreground(ui.ColorAccent)

	vis := maxVisible
	if termHeight > 6 && vis > termHeight-6 {
		vis = termHeight - 6
	}
	if vis < 1 {
		vis = 1
	}

	return Model{
		repoName:    repoName,
		width:       termWidth,
		height:      termHeight,
		visibleRows: vis,
		help:        h,
		keys:        defaultKeyMap(),
		spinner:     sp,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.loadPRs())
}

func (m Model) loadPRs() tea.Cmd {
	return func() tea.Msg {
		prs, err := gh.ListMyPRs(50)
		return prsLoadedMsg{prs: prs, err: err}
	}
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
		m.height = msg.Height
		m.help.Width = msg.Width
		vis := maxVisible
		if msg.Height > 6 && vis > msg.Height-6 {
			vis = msg.Height - 6
		}
		if vis < 1 {
			vis = 1
		}
		m.visibleRows = vis
		m.clampScroll()
		return m, nil

	case prsLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.prs = msg.prs
		m.cursor = 0
		m.offset = 0
		return m, nil

	case openBrowserMsg:
		gh.OpenInBrowser(msg.url)
		return m, nil

	case tea.KeyMsg:
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
		if m.cursor < len(m.prs)-1 {
			m.cursor++
			m.clampScroll()
		}
		return m, nil

	case key.Matches(msg, m.keys.Open):
		if len(m.prs) == 0 || m.cursor >= len(m.prs) {
			return m, nil
		}
		pr := m.prs[m.cursor]
		return m, func() tea.Msg { return openBrowserMsg{url: pr.URL} }

	case key.Matches(msg, m.keys.Refresh):
		m.loading = true
		m.err = ""
		return m, m.loadPRs()

	case key.Matches(msg, m.keys.Quit), key.Matches(msg, m.keys.Back):
		return m, tea.Quit
	}
	return m, nil
}

func (m *Model) clampScroll() {
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+m.visibleRows {
		m.offset = m.cursor - m.visibleRows + 1
	}
}

// ── View ───────────────────────────────────────────────────────────────────

func (m Model) View() string {
	var sb strings.Builder

	// Header
	sb.WriteString(ui.StyleHeader.Render("  Pull Requests") + "  " +
		ui.StyleAccent.Render(m.repoName) + "  " +
		ui.StyleCountBadge.Render(fmt.Sprintf("%d PRs", len(m.prs))) + "\n")
	sb.WriteString(ui.StyleDivider.Render(strings.Repeat("─", m.width)) + "\n")

	// Content
	if m.loading {
		sb.WriteString("\n  " + m.spinner.View() + " " +
			lipgloss.NewStyle().Foreground(ui.ColorParentAhead).Render("loading…") + "\n")
	} else if m.err != "" {
		sb.WriteString("\n  " + ui.StyleError.Render("✗ "+m.err) + "\n")
	} else if len(m.prs) == 0 {
		sb.WriteString("\n  " + ui.StyleDim.Render("no pull requests found") + "\n")
	} else {
		sb.WriteString(m.renderList())
	}

	// Footer
	sb.WriteString(ui.StyleDivider.Render(strings.Repeat("─", m.width)) + "\n")
	sb.WriteString(m.footer())

	return sb.String()
}

func (m Model) renderList() string {
	var sb strings.Builder

	// Column headers
	sb.WriteString(ui.StyleDim.Render("   #   State   Review          Files   Age      Title") + "\n")

	end := m.offset + m.visibleRows
	if end > len(m.prs) {
		end = len(m.prs)
	}

	for i := m.offset; i < end; i++ {
		pr := m.prs[i]
		isSelected := i == m.cursor
		sb.WriteString(m.renderRow(pr, isSelected))
	}

	return sb.String()
}

func (m Model) renderRow(pr gh.PR, isSelected bool) string {
	// Number
	num := fmt.Sprintf("#%-5d", pr.Number)

	// State with icon
	stateIcon := gh.StateIcon(pr.State)
	var stateStyle lipgloss.Style
	switch strings.ToUpper(pr.State) {
	case "OPEN":
		stateStyle = lipgloss.NewStyle().Foreground(ui.ColorParentAhead)
	case "MERGED":
		stateStyle = lipgloss.NewStyle().Foreground(ui.ColorParentMerged)
	case "CLOSED":
		stateStyle = lipgloss.NewStyle().Foreground(ui.ColorError)
	}
	state := stateStyle.Render(fmt.Sprintf("%-7s", stateIcon+" "+strings.ToLower(pr.State)))

	// Review status
	review := gh.ReviewStatus(pr.ReviewDecision)
	var reviewStyle lipgloss.Style
	switch strings.ToUpper(pr.ReviewDecision) {
	case "APPROVED":
		reviewStyle = lipgloss.NewStyle().Foreground(ui.ColorParentMerged)
	case "CHANGES_REQUESTED":
		reviewStyle = lipgloss.NewStyle().Foreground(ui.ColorError)
	default:
		reviewStyle = lipgloss.NewStyle().Foreground(ui.ColorDim)
	}
	if review == "" {
		review = "—"
	}
	reviewStr := reviewStyle.Render(fmt.Sprintf("%-15s", review))

	// Files changed
	files := fmt.Sprintf("%d file", pr.ChangedFiles)
	if pr.ChangedFiles != 1 {
		files = fmt.Sprintf("%d files", pr.ChangedFiles)
	}
	filesStr := lipgloss.NewStyle().Foreground(ui.ColorHeader).Render(fmt.Sprintf("%-7s", files))

	// Age
	days := gh.DaysOpen(pr.CreatedAt)
	var age string
	if days == 0 {
		age = "today"
	} else if days == 1 {
		age = "1 day"
	} else {
		age = fmt.Sprintf("%d days", days)
	}
	ageStr := lipgloss.NewStyle().Foreground(ui.ColorRelTime).Render(fmt.Sprintf("%-8s", age))

	// Title (truncate to fit)
	availWidth := m.width - 45 // approximate space used by other columns
	if availWidth < 10 {
		availWidth = 10
	}
	title := pr.Title
	if len(title) > availWidth {
		title = title[:availWidth-1] + "…"
	}
	titleStr := lipgloss.NewStyle().Foreground(ui.ColorHeader).Render(title)

	row := fmt.Sprintf("  %s %s %s %s %s %s", num, state, reviewStr, filesStr, ageStr, titleStr)

	if isSelected {
		return lipgloss.NewStyle().
			Background(ui.ColorCursorBg).
			Foreground(ui.ColorCursorFg).
			Width(m.width).
			Render(row) + "\n"
	}
	return row + "\n"
}

func (m Model) footer() string {
	if m.loading {
		return "  " + m.spinner.View() + " " +
			lipgloss.NewStyle().Foreground(ui.ColorParentAhead).Render("loading…")
	}
	return "  " + m.help.ShortHelpView(m.keys.ShortHelp())
}