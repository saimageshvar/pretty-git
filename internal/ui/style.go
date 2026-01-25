package ui

import "github.com/fatih/color"

var (
    // CurrentMarker is the marker shown next to the current branch when enabled
    CurrentMarker = "• "

    // EnableColor controls whether coloring is applied
    EnableColor = true

    // ShowCurrentMarker controls whether the current-branch marker is shown
    ShowCurrentMarker = true
)

// ColorCurrent returns a colored version of the provided string (green, bold) when EnableColor is true.
func ColorCurrent(s string) string {
    if !EnableColor {
        return s
    }
    return color.New(color.FgGreen, color.Bold).Sprint(s)
}

// MarkerForCurrent returns the marker string if markers are enabled, otherwise empty string.
func MarkerForCurrent() string {
    if ShowCurrentMarker {
        return CurrentMarker
    }
    return ""
}
