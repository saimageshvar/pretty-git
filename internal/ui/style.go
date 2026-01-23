package ui

import "github.com/fatih/color"

var (
    CurrentMarker = "• "
)

// ColorCurrent returns a colored version of the provided string (green, bold).
func ColorCurrent(s string) string {
    return color.New(color.FgGreen, color.Bold).Sprint(s)
}
