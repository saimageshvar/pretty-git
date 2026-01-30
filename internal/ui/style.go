package ui

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

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

// ColorDescription returns a dimmed/subtle version for branch descriptions
func ColorDescription(s string) string {
	if !EnableColor {
		return s
	}
	return color.New(color.FgHiBlack, color.Italic).Sprint(s)
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

// DisplayWithPager pipes the output through a pager (less or more)
// Falls back to direct output if pager is not available or fails
// forcePager: if true, pager stays open even for small outputs (removes -F flag)
func DisplayWithPager(content string, forcePager bool) error {
	// Try to find a suitable pager
	pagerCmd := os.Getenv("PAGER")
	if pagerCmd == "" {
		// Default to less with common options
		pagerCmd = "less"
	}

	// Parse pager command (handle commands with arguments)
	parts := strings.Fields(pagerCmd)
	if len(parts) == 0 {
		return fmt.Errorf("empty pager command")
	}

	cmdName := parts[0]
	cmdArgs := parts[1:]

	// Add default options for less if it's the pager and no args provided
	if cmdName == "less" && len(cmdArgs) == 0 {
		if forcePager {
			// Force pager to stay open (no -F flag)
			cmdArgs = []string{"-R", "-X"}
		} else {
			cmdArgs = []string{"-R", "-F", "-X"}
		}
		// -R: allow ANSI color codes
		// -F: quit if content fits on one screen (omitted when forcePager=true)
		// -X: don't clear screen on exit
	}

	// Check if pager exists
	if _, err := exec.LookPath(cmdName); err != nil {
		return fmt.Errorf("pager '%s' not found: %w", cmdName, err)
	}

	// Create the pager command
	cmd := exec.Command(cmdName, cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Create a pipe to write content to pager
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	// Start the pager
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start pager: %w", err)
	}

	// Write content to pager
	if _, err := io.WriteString(stdin, content); err != nil {
		stdin.Close()
		cmd.Wait()
		return fmt.Errorf("failed to write to pager: %w", err)
	}

	// Close stdin to signal end of input
	stdin.Close()

	// Wait for pager to finish
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("pager exited with error: %w", err)
	}

	return nil
}
