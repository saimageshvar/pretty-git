package ui

import "github.com/charmbracelet/lipgloss"

// ── Adaptive color palette ─────────────────────────────────────────────────
// Each color degrades gracefully on light terminals.

var (
	ColorCurrentBranch = lipgloss.AdaptiveColor{Light: "#007700", Dark: "#00FF87"}
	ColorCursorBg      = lipgloss.AdaptiveColor{Light: "#C8DAFF", Dark: "#2D3561"}
	ColorCursorFg      = lipgloss.AdaptiveColor{Light: "#0A0A0A", Dark: "#F0F0F0"}
	ColorAccent        = lipgloss.AdaptiveColor{Light: "#3355CC", Dark: "#7C9EFF"}
	ColorRemote        = lipgloss.AdaptiveColor{Light: "#888888", Dark: "#666666"}
	ColorHash          = lipgloss.AdaptiveColor{Light: "#996600", Dark: "#FFD700"}
	ColorSubject       = lipgloss.AdaptiveColor{Light: "#444444", Dark: "#999999"}
	ColorRelTime       = lipgloss.AdaptiveColor{Light: "#006688", Dark: "#5FD7D7"}
	ColorError         = lipgloss.AdaptiveColor{Light: "#CC0000", Dark: "#FF5555"}
	ColorHeader        = lipgloss.AdaptiveColor{Light: "#111111", Dark: "#EEEEEE"}
	ColorDim           = lipgloss.AdaptiveColor{Light: "#888888", Dark: "#4A4A4A"}
	ColorKeyHint       = lipgloss.AdaptiveColor{Light: "#3355CC", Dark: "#7C9EFF"}
	ColorCountBadge    = lipgloss.AdaptiveColor{Light: "#666666", Dark: "#888888"}
	ColorBadgeBg       = lipgloss.AdaptiveColor{Light: "#E8E8E8", Dark: "#2A2A2A"}
	ColorDivider       = lipgloss.AdaptiveColor{Light: "#CCCCCC", Dark: "#333333"}
)

// ── Styles ─────────────────────────────────────────────────────────────────

var (
	StyleCurrentBranch = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorCurrentBranch)

	StyleRemoteName = lipgloss.NewStyle().
			Italic(true).
			Foreground(ColorRemote)

	StyleHash = lipgloss.NewStyle().
			Foreground(ColorHash)

	StyleSubject = lipgloss.NewStyle().
			Foreground(ColorSubject)

	StyleRelTime = lipgloss.NewStyle().
			Foreground(ColorRelTime)

	StyleError = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorError)

	StyleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorHeader)

	StyleAccent = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorAccent)

	StyleDim = lipgloss.NewStyle().
			Foreground(ColorDim)

	StyleKeyHint = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorKeyHint)

	StyleCountBadge = lipgloss.NewStyle().
			Foreground(ColorCountBadge).
			Background(ColorBadgeBg).
			PaddingLeft(1).PaddingRight(1)

	StyleDivider = lipgloss.NewStyle().
			Foreground(ColorDivider)
)
