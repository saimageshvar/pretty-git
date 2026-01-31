package ui

import (
	"fmt"
	"sort"
	"strings"

	"pretty-git/internal/git"
)

// RenderBranchesTree builds an ASCII tree of branches from parent metadata and highlights the current branch.
// compact: uses narrower indents and connectors.
// verbose: include parent metadata inline for each branch.
// showStatus: include branch status indicators (merged, ahead/behind, stale, etc).
func RenderBranchesTree(parents map[string]string, current string, compact bool, verbose bool) (string, error) {
	return RenderBranchesTreeWithStatus(parents, current, compact, verbose, true)
}

// RenderBranchesTreeWithStatus is the full implementation with status marker support
func RenderBranchesTreeWithStatus(parents map[string]string, current string, compact bool, verbose bool, showStatus bool) (string, error) {
	// get full branch list
	branches, err := git.ListBranches()
	if err != nil {
		return "", err
	}

	// load descriptions
	descriptions, err := git.AllDescriptions()
	if err != nil {
		return "", err
	}

	// build children map
	children := map[string][]string{}
	have := map[string]bool{}
	for _, b := range branches {
		have[b] = true
	}

	for _, b := range branches {
		p := parents[b]
		if p == "" {
			// no parent metadata
			children[""] = append(children[""], b)
			continue
		}
		children[p] = append(children[p], b)
	}

	// ensure deterministic order
	for k := range children {
		sort.Strings(children[k])
	}

	// roots are children[""] (no recorded parent)
	roots := children[""]
	sort.Strings(roots)

	var lines []string

	// choose connectors and indents - compact by default
	connectorMid := "├─ "
	connectorLast := "└─ "
	indentVert := "│  "
	indentBlank := "   "
	if compact {
		// extra compact mode for narrow terminals
		connectorMid = "├─"
		connectorLast = "└─"
		indentVert = "│ "
		indentBlank = "  "
	}

	var printNode func(name string, prefix string, last bool, depth int)
	printNode = func(name string, prefix string, last bool, depth int) {
		marker := ""
		display := name

		// Get status if enabled
		var status *git.BranchStatus
		if showStatus {
			// Get the parent of this branch from the metadata
			parentBranch := parents[name]
			status = git.GetBranchStatus(name, parentBranch)
		}

		// Apply coloring based on status
		display = GetBranchDisplay(name, name == current, status)

		if name == current {
			marker = MarkerForCurrent()
		}

		// Append status markers
		statusStr := ""
		if showStatus && status != nil {
			statusStr = GetStatusMarkers(status)
		}

		// verbose: append parent info inline if requested
		if verbose {
			if p := parents[name]; p != "" {
				statusStr = statusStr + fmt.Sprintf(" (%s)", p)
			}
		}

		connector := connectorMid
		if last {
			connector = connectorLast
		}

		if depth == 0 {
			// root
			if marker != "" {
				lines = append(lines, fmt.Sprintf("%s%s%s", marker, display, statusStr))
			} else {
				lines = append(lines, fmt.Sprintf("%s%s", display, statusStr))
			}
		} else {
			if marker != "" {
				lines = append(lines, fmt.Sprintf("%s%s%s%s%s", prefix, connector, marker, display, statusStr))
			} else {
				lines = append(lines, fmt.Sprintf("%s%s%s%s", prefix, connector, display, statusStr))
			}
		}

		// Add description line if present
		if desc, ok := descriptions[name]; ok && desc != "" {
			descLine := ColorDescription(desc)
			if depth == 0 {
				// Root level: check if it has children to show connector
				hasChildren := len(children[name]) > 0
				if hasChildren {
					lines = append(lines, "│ "+descLine)
				} else {
					lines = append(lines, "  "+descLine)
				}
			} else {
				// Continue the tree connectors through the description line
				// Check if this node has children to determine connector style
				hasChildren := len(children[name]) > 0
				var descConnector string

				if hasChildren {
					// Has children: use vertical line to continue
					if last {
						descConnector = prefix + indentBlank + "│  "
					} else {
						descConnector = prefix + indentVert + "│  "
					}
				} else {
					// No children: just maintain parent's connector
					if last {
						descConnector = prefix + indentBlank + "   "
					} else {
						descConnector = prefix + indentVert + "   "
					}
				}
				lines = append(lines, descConnector+descLine)
			}
		}

		ch := children[name]
		for i, c := range ch {
			isLast := i == len(ch)-1
			var newPrefix string
			if depth == 0 {
				// Children of root: no prefix before connector
				newPrefix = ""
			} else if last {
				newPrefix = prefix + indentBlank
			} else {
				newPrefix = prefix + indentVert
			}
			printNode(c, newPrefix, isLast, depth+1)
		}
	}

	// If there are no roots found (all branches have parents), find possible roots by scanning parents values
	if len(roots) == 0 {
		rootset := map[string]bool{}
		for _, b := range branches {
			if parents[b] == "" {
				rootset[b] = true
			}
		}
		for r := range rootset {
			roots = append(roots, r)
		}
		sort.Strings(roots)
	}

	// fallback: if still no roots, use all branches
	if len(roots) == 0 {
		roots = branches
	}

	for i, r := range roots {
		last := i == len(roots)-1
		printNode(r, "", last, 0)
	}

	return strings.Join(lines, "\n") + "\n", nil
}

// RenderCommitLog renders commits with parent-awareness
// branch: current branch name
// parent: parent branch name (may be empty)
// hasParent: whether a parent is configured
// bySection: if true, group by section; if false, show chronologically (default)
// multiline: if true, use detailed format; if false, use oneline format
// ascii: if true, use ASCII symbols; if false, use Unicode symbols
// maxCommits: maximum commits per section (0 for unlimited)
func RenderCommitLog(branch string, parent string, hasParent bool, bySection bool, multiline bool, ascii bool, maxCommits int) (string, error) {
	var output strings.Builder

	// Show header with ahead/behind status
	if hasParent {
		ahead, behind, err := git.GetAheadBehindCounts(branch, parent)
		if err != nil {
			return "", err
		}

		header := fmt.Sprintf("Branch: %s (parent: %s)\n", ColorCurrent(branch), parent)
		if ahead > 0 || behind > 0 {
			status := fmt.Sprintf("Status: %d ahead, %d behind\n", ahead, behind)
			header += status
		}
		output.WriteString(header)
		output.WriteString("\n")
	} else {
		header := fmt.Sprintf("Branch: %s (no parent configured)\n\n", ColorCurrent(branch))
		output.WriteString(header)
	}

	if hasParent {
		// Add legend at the top for better visibility when paging
		if bySection {
			output.WriteString(renderLegend(ascii))
		} else {
			output.WriteString(renderLegendChronological(ascii))
		}
		output.WriteString("\n\n")

		// Show commits from current branch with distinction
		_, err := renderCommitsFromBranch(branch, parent, multiline, bySection, ascii, maxCommits, &output)
		if err != nil {
			return "", err
		}

		return output.String(), nil
	}

	// No parent: show all commits from branch
	commits, err := git.GetCommitLog(branch, "", false)
	if err != nil {
		return "", err
	}

	if len(commits) == 0 {
		output.WriteString("No commits found.\n")
		return output.String(), nil
	}

	// Render commits
	for _, commit := range commits {
		if multiline {
			output.WriteString(renderCommitMultiline(commit, true))
		} else {
			output.WriteString(renderCommitOneline(commit, true))
		}
	}

	return output.String(), nil
}

// renderCommitsFromBranch shows commits from the current branch only (no parent-only commits)
func renderCommitsFromBranch(branch string, parent string, multiline bool, bySection bool, ascii bool, maxCommits int, output *strings.Builder) (string, error) {
	if bySection {
		return renderCommitsGrouped(branch, parent, multiline, ascii, maxCommits, output)
	}
	return renderCommitsChronological(branch, parent, multiline, maxCommits, output)
}

// renderCommitsChronological shows commits from current branch in time order with visual distinction
func renderCommitsChronological(branch string, parent string, multiline bool, maxCommits int, output *strings.Builder) (string, error) {
	// Get branch-only commits to identify them
	branchCommits, err := git.GetCommitLog(branch, parent, true)
	if err != nil {
		return "", err
	}

	// Build set for quick lookup
	branchHashes := make(map[string]bool)
	for _, c := range branchCommits {
		branchHashes[c.Hash] = true
	}

	// Get all commits from current branch (includes common commits)
	branchAllCommits, err := git.GetCommitLog(branch, "", false)
	if err != nil {
		return "", err
	}

	// Apply limit if set
	totalCommits := len(branchAllCommits)
	truncated := 0
	if maxCommits > 0 && totalCommits > maxCommits {
		truncated = totalCommits - maxCommits
		branchAllCommits = branchAllCommits[:maxCommits]
	}

	// Render all commits with distinction
	for _, commit := range branchAllCommits {
		isBranchCommit := branchHashes[commit.Hash]

		if multiline {
			output.WriteString(renderCommitMultilineWithIndicator(commit, isBranchCommit))
		} else {
			output.WriteString(renderCommitOnelineWithIndicator(commit, isBranchCommit))
		}
	}

	// Show truncation message if commits were limited
	if truncated > 0 {
		truncationMsg := fmt.Sprintf("\n... %d more commits (use --max-commits=0 to see all)\n", truncated)
		if EnableColor {
			truncationMsg = ColorDescription(truncationMsg)
		}
		output.WriteString(truncationMsg)
	}

	return output.String(), nil
}

// renderCommitsGrouped shows commits from current branch grouped by type (no parent-only)
func renderCommitsGrouped(branch string, parent string, multiline bool, ascii bool, maxCommits int, output *strings.Builder) (string, error) {
	// Get branch-only commits
	branchCommits, err := git.GetCommitLog(branch, parent, true)
	if err != nil {
		return "", err
	}

	// Determine symbols based on ascii flag
	currentSymbol := "▲"
	commonSymbol := "●"
	if ascii {
		currentSymbol = "^"
		commonSymbol = "o"
	}

	if len(branchCommits) > 0 {
		totalBranch := len(branchCommits)
		displayCommits := branchCommits
		truncatedBranch := 0
		
		// Apply limit if set
		if maxCommits > 0 && totalBranch > maxCommits {
			truncatedBranch = totalBranch - maxCommits
			displayCommits = branchCommits[:maxCommits]
		}

		// Show section header for branch commits
		sectionHeader := fmt.Sprintf("%s Current branch only (%d)\n", currentSymbol, totalBranch)
		output.WriteString(sectionHeader)

		// Render branch-unique commits
		for _, commit := range displayCommits {
			if multiline {
				output.WriteString(renderCommitMultiline(commit, true))
			} else {
				output.WriteString(renderCommitOneline(commit, true))
			}
		}
		
		// Show truncation message if commits were limited
		if truncatedBranch > 0 {
			truncationMsg := fmt.Sprintf("... %d more commits\n", truncatedBranch)
			if EnableColor {
				truncationMsg = ColorDescription(truncationMsg)
			}
			output.WriteString(truncationMsg)
		}
		
		output.WriteString("\n")
	}

	// Get shared/common commits (commits in both branches)
	sharedCommits, err := git.GetCommitLog(parent, "", false)
	if err != nil {
		return "", err
	}

	// Build set of branch commit hashes to identify shared commits
	branchSet := make(map[string]bool)
	for _, c := range branchCommits {
		branchSet[c.Hash] = true
	}

	// Filter to only shared commits (commits in parent that are also in current branch)
	var actualShared []*git.Commit
	for _, c := range sharedCommits {
		if !branchSet[c.Hash] {
			actualShared = append(actualShared, c)
		}
	}

	if len(actualShared) > 0 {
		totalShared := len(actualShared)
		displayShared := actualShared
		truncatedShared := 0
		
		// Apply limit if set
		if maxCommits > 0 && totalShared > maxCommits {
			truncatedShared = totalShared - maxCommits
			displayShared = actualShared[:maxCommits]
		}

		// Show section header for shared commits
		sectionHeader := fmt.Sprintf("%s Common commits (%d)\n", commonSymbol, totalShared)
		output.WriteString(sectionHeader)

		for _, commit := range displayShared {
			if multiline {
				output.WriteString(renderCommitMultiline(commit, false))
			} else {
				output.WriteString(renderCommitOneline(commit, false))
			}
		}
		
		// Show truncation message if commits were limited
		if truncatedShared > 0 {
			truncationMsg := fmt.Sprintf("... %d more commits\n", truncatedShared)
			if EnableColor {
				truncationMsg = ColorDescription(truncationMsg)
			}
			output.WriteString(truncationMsg)
		}
	}

	return output.String(), nil
}

// renderLegend shows the legend for grouped mode
func renderLegend(ascii bool) string {
	var legend string

	if ascii {
		legend = "Legend: ^ Current branch only | o Common commits"
	} else {
		legend = "Legend: ▲ Current branch only | ● Common commits"
	}

	if EnableColor {
		return ColorDescription(legend)
	}
	return legend
}

// renderLegendChronological shows the legend for chronological mode
func renderLegendChronological(ascii bool) string {
	var legend string

	if ascii {
		legend = "Legend: o Common commit"
	} else {
		legend = "Legend: ● Common commit"
	}

	if EnableColor {
		return ColorDescription(legend)
	}
	return legend
}

// renderCommitOneline renders a commit in oneline format
// bright: if true, use bright/bold styling; if false, use dimmed styling
func renderCommitOneline(commit *git.Commit, bright bool) string {
	hash := commit.ShortHash
	message := commit.Message

	if bright {
		// Bright/bold for branch-unique commits
		if EnableColor {
			hash = ColorCurrent(hash)
		}
		return fmt.Sprintf("%s  %s\n", hash, message)
	}

	// Dimmed for parent commits
	if EnableColor {
		hash = ColorDescription(hash)
		message = ColorDescription(message)
	}
	return fmt.Sprintf("%s  %s\n", hash, message)
}

// renderCommitOnelineWithIndicator renders a commit with type indicator for chronological mode
func renderCommitOnelineWithIndicator(commit *git.Commit, isBranch bool) string {
	hash := commit.ShortHash
	message := commit.Message
	indicator := "  "

	if isBranch {
		// Branch-unique commit: bright, no indicator
		if EnableColor {
			hash = ColorCurrent(hash)
		}
		return fmt.Sprintf("%s%s  %s\n", indicator, hash, message)
	} else {
		// Common commit: dimmed with ●
		indicator = "● "
		if EnableColor {
			hash = ColorDescription(hash)
			message = ColorDescription(message)
		}
		return fmt.Sprintf("%s%s  %s\n", indicator, hash, message)
	}
}

// renderCommitMultiline renders a commit in detailed multiline format
// bright: if true, use bright/bold styling; if false, use dimmed styling
func renderCommitMultiline(commit *git.Commit, bright bool) string {
	var output strings.Builder

	hash := commit.Hash
	author := commit.Author
	date := commit.Date
	message := commit.FullMessage
	if message == "" {
		message = commit.Message
	}

	// Apply styling
	if bright {
		if EnableColor {
			hash = ColorCurrent(hash)
		}
	} else {
		if EnableColor {
			hash = ColorDescription(hash)
			author = ColorDescription(author)
			date = ColorDescription(date)
			message = ColorDescription(message)
		}
	}

	// Format output
	output.WriteString(fmt.Sprintf("commit %s\n", hash))
	output.WriteString(fmt.Sprintf("Author: %s\n", author))
	output.WriteString(fmt.Sprintf("Date:   %s\n", date))
	output.WriteString("\n")

	// Indent message
	messageLines := strings.Split(strings.TrimSpace(message), "\n")
	for _, line := range messageLines {
		output.WriteString(fmt.Sprintf("    %s\n", line))
	}

	// Get file stats
	stats, err := git.GetCommitStats(commit.Hash)
	if err == nil && stats != "" {
		output.WriteString("\n")
		// Indent and style stats
		statsLines := strings.Split(stats, "\n")
		for _, line := range statsLines {
			if !bright && EnableColor {
				line = ColorDescription(line)
			}
			output.WriteString(fmt.Sprintf("    %s\n", line))
		}
	}

	output.WriteString("\n")
	return output.String()
}

// renderCommitMultilineWithIndicator renders a commit with type indicator for chronological mode
func renderCommitMultilineWithIndicator(commit *git.Commit, isBranch bool) string {
	var output strings.Builder

	hash := commit.Hash
	author := commit.Author
	date := commit.Date
	message := commit.FullMessage
	if message == "" {
		message = commit.Message
	}

	// Determine indicator and styling
	indicator := ""
	bright := isBranch

	if isBranch {
		indicator = "" // No indicator for branch commits
	} else {
		indicator = "[● Common] "
	}

	// Apply styling
	if bright {
		if EnableColor {
			hash = ColorCurrent(hash)
		}
	} else {
		if EnableColor {
			hash = ColorDescription(hash)
			author = ColorDescription(author)
			date = ColorDescription(date)
			message = ColorDescription(message)
			indicator = ColorDescription(indicator)
		}
	}

	// Format output
	output.WriteString(fmt.Sprintf("%scommit %s\n", indicator, hash))
	output.WriteString(fmt.Sprintf("Author: %s\n", author))
	output.WriteString(fmt.Sprintf("Date:   %s\n", date))
	output.WriteString("\n")

	// Indent message
	messageLines := strings.Split(strings.TrimSpace(message), "\n")
	for _, line := range messageLines {
		output.WriteString(fmt.Sprintf("    %s\n", line))
	}

	// Get file stats
	stats, err := git.GetCommitStats(commit.Hash)
	if err == nil && stats != "" {
		output.WriteString("\n")
		statsLines := strings.Split(stats, "\n")
		for _, line := range statsLines {
			if !bright && EnableColor {
				line = ColorDescription(line)
			}
			output.WriteString(fmt.Sprintf("    %s\n", line))
		}
	}

	output.WriteString("\n")
	return output.String()
}
