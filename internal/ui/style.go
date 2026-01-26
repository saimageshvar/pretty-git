package ui

import (
	"fmt"

	"pretty-git/internal/git"

	"github.com/fatih/color"
)

var (
	// CurrentMarker is the marker shown next to the current branch when enabled
	CurrentMarker = "● "

	// EnableColor controls whether coloring is applied
	EnableColor = true

	// ShowCurrentMarker controls whether the current-branch marker is shown
	ShowCurrentMarker = true

	// ShowStatusMarkers controls whether status indicators are shown
	ShowStatusMarkers = true
)

// ColorCurrent returns a colored version of the provided string (bright green, bold) when EnableColor is true.
func ColorCurrent(s string) string {
	if !EnableColor {
		return s
	}
	return color.New(color.FgGreen, color.Bold).Sprint(s)
}

// ColorMergedBranch returns a colored version for merged branches (dim/gray)
func ColorMergedBranch(s string) string {
	if !EnableColor {
		return s
	}
	return color.New(color.FgHiBlack).Sprint(s)
}

// ColorStaleBranch returns a colored version for stale branches (yellow/warning)
func ColorStaleBranch(s string) string {
	if !EnableColor {
		return s
	}
	return color.New(color.FgYellow).Sprint(s)
}

// ColorDefault returns the string unchanged - uses terminal default coloring
func ColorDefault(s string) string {
	return s
}

// MarkerForCurrent returns the marker string if markers are enabled, otherwise empty string.
func MarkerForCurrent() string {
	if ShowCurrentMarker {
		return CurrentMarker
	}
	return ""
}

// GetBranchDisplay returns the display string with appropriate coloring based on status
func GetBranchDisplay(name string, isCurrent bool, status *git.BranchStatus) string {
	if isCurrent {
		return ColorCurrent(name)
	}
	if status != nil && status.Merged {
		return ColorMergedBranch(name)
	}
	if status != nil && status.IsStale {
		return ColorStaleBranch(name)
	}
	return ColorDefault(name)
}

// GetStatusMarkers returns visual indicators for branch status
func GetStatusMarkers(status *git.BranchStatus) string {
	if !ShowStatusMarkers || status == nil {
		return ""
	}

	var markers []string

	if status.Merged {
		markers = append(markers, "✓")
	} else if status.Ahead > 0 && status.Behind > 0 {
		markers = append(markers, fmt.Sprintf("↔ %d↑%d↓", status.Ahead, status.Behind))
	} else if status.Ahead > 0 {
		markers = append(markers, fmt.Sprintf("↑ %d", status.Ahead))
	} else if status.Behind > 0 {
		markers = append(markers, fmt.Sprintf("↓ %d", status.Behind))
	}

	if status.Tracking {
		markers = append(markers, "◇")
	}

	if status.IsStale {
		markers = append(markers, fmt.Sprintf("⚡ %dd", status.LastDays))
	}

	if len(markers) > 0 {
		return " [" + joinMarkers(markers) + "]"
	}
	return ""
}

// joinMarkers concatenates status markers with spacing
func joinMarkers(markers []string) string {
	if len(markers) == 0 {
		return ""
	}
	result := markers[0]
	for i := 1; i < len(markers); i++ {
		result += " " + markers[i]
	}
	return result
}
