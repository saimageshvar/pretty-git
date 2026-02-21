package ui

import "github.com/charmbracelet/lipgloss"

// ── Color palette ──────────────────────────────────────────────────────────
//
// Dark:  synthwave / neon-dusk — vivid neons on deep backgrounds.
// Light: jewel-toned on white — rich saturated hues, never grey.

var (
	// Branch name / marker for current branch — neon mint / deep emerald
	ColorCurrentBranch = lipgloss.AdaptiveColor{Light: "#047857", Dark: "#00FF87"}

	// Cursor / selection row
	ColorCursorBg = lipgloss.AdaptiveColor{Light: "#EDE9FE", Dark: "#2D1B69"}
	ColorCursorFg = lipgloss.AdaptiveColor{Light: "#0F0A1E", Dark: "#FAFAFA"}

	// Accent — repo name, focused labels, active input text
	ColorAccent = lipgloss.AdaptiveColor{Light: "#6D28D9", Dark: "#C084FC"}

	// Remote branches — periwinkle indigo (italic makes them secondary)
	ColorRemote = lipgloss.AdaptiveColor{Light: "#4338CA", Dark: "#818CF8"}

	// Short hash column — warm amber
	ColorHash = lipgloss.AdaptiveColor{Light: "#B45309", Dark: "#FCD34D"}

	// Relative time column — teal cyan
	ColorRelTime = lipgloss.AdaptiveColor{Light: "#0D9488", Dark: "#2DD4BF"}

	// Branch description column — soft lavender / cool slate
	ColorDesc = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#94A3B8"}

	// Parent status: all commits merged into parent
	ColorParentMerged = lipgloss.AdaptiveColor{Light: "#047857", Dark: "#00FF87"}

	// Parent status: ahead of parent (unmerged work, clean)
	ColorParentAhead = lipgloss.AdaptiveColor{Light: "#0891B2", Dark: "#22D3EE"}

	// Parent status: diverged — ahead AND parent has new commits
	ColorParentDiverged = lipgloss.AdaptiveColor{Light: "#EA580C", Dark: "#FB923C"}

	// Errors and conflict warnings
	ColorError = lipgloss.AdaptiveColor{Light: "#DC2626", Dark: "#FF6B6B"}

	// Header / body text at full brightness
	ColorHeader = lipgloss.AdaptiveColor{Light: "#111827", Dark: "#F1F5F9"}

	// Dim labels, tree connectors, punctuation — slate, NOT dark grey
	ColorDim = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#64748B"}

	// Key hint labels in footer — amber / gold, pops against any bg
	ColorKeyHint = lipgloss.AdaptiveColor{Light: "#B45309", Dark: "#FBBF24"}

	// Count badge (e.g. "4 local · 2 remote")
	ColorCountBadge = lipgloss.AdaptiveColor{Light: "#5B21B6", Dark: "#C4B5FD"}
	ColorBadgeBg    = lipgloss.AdaptiveColor{Light: "#EDE9FE", Dark: "#1E1B4B"}

	// Divider lines — vivid purple, not grey
	ColorDivider = lipgloss.AdaptiveColor{Light: "#7C3AED", Dark: "#6D28D9"}

	// Tree connector chars (├─ └─ │) — indigo
	ColorTreeConnector = lipgloss.AdaptiveColor{Light: "#4338CA", Dark: "#6366F1"}
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

	StyleRelTime = lipgloss.NewStyle().
			Foreground(ColorRelTime)

	StyleDesc = lipgloss.NewStyle().
			Foreground(ColorDesc)

	StyleError = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorError)

	// "  Branches" title — bold, full brightness
	StyleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorHeader)

	// Repo name next to header
	StyleAccent = lipgloss.NewStyle().
			Bold(true).
			Italic(true).
			Foreground(ColorAccent)

	// Dim labels and punctuation
	StyleDim = lipgloss.NewStyle().
			Foreground(ColorDim)

	// Key names in footer hints — amber, bold
	StyleKeyHint = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorKeyHint)

	// "N local · M remote" badge
	StyleCountBadge = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorCountBadge).
				Background(ColorBadgeBg).
				PaddingLeft(1).PaddingRight(1)

	// Horizontal divider lines
	StyleDivider = lipgloss.NewStyle().
			Foreground(ColorDivider)

	// Tree connector characters
	StyleTreeConnector = lipgloss.NewStyle().
				Foreground(ColorTreeConnector)
)
