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
// showAll: if true, show commits from parent; if false, only branch-unique commits
// multiline: if true, use detailed format; if false, use oneline format
// chronological: if true with showAll, show in time order; if false, group by relationship
// ascii: if true, use ASCII symbols; if false, use Unicode symbols
func RenderCommitLog(branch string, parent string, hasParent bool, showAll bool, multiline bool, chronological bool, ascii bool) (string, error) {
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

	if showAll && hasParent {
		// Show all commits with distinction between branch and parent commits
		_, err := renderAllCommitsWithParent(branch, parent, multiline, chronological, ascii, &output)
		if err != nil {
			return "", err
		}

		// Add legend at the bottom
		if chronological {
			output.WriteString(renderLegendChronological(ascii))
		} else {
			output.WriteString(renderLegend(ascii))
		}

		return output.String(), nil
	}

	// Default: show only commits unique to branch
	commits, err := git.GetCommitLog(branch, parent, true)
	if err != nil {
		return "", err
	}

	if len(commits) == 0 {
		if hasParent {
			output.WriteString("No commits unique to this branch.\n")
			output.WriteString("(All commits are shared with parent or branch is up to date)\n")
		} else {
			output.WriteString("No commits found.\n")
		}
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

// renderAllCommitsWithParent shows full history with visual distinction
func renderAllCommitsWithParent(branch string, parent string, multiline bool, chronological bool, ascii bool, output *strings.Builder) (string, error) {
	if chronological {
		return renderAllCommitsChronological(branch, parent, multiline, output)
	}
	return renderAllCommitsGrouped(branch, parent, multiline, ascii, output)
}

// renderAllCommitsChronological shows commits in time order with visual distinction
func renderAllCommitsChronological(branch string, parent string, multiline bool, output *strings.Builder) (string, error) {
	// Get branch-only commits to identify them
	branchCommits, err := git.GetCommitLog(branch, parent, true)
	if err != nil {
		return "", err
	}

	// Get parent-only commits
	parentCommits, err := git.GetCommitLog(parent, branch, true)
	if err != nil {
		return "", err
	}

	// Build sets for quick lookup
	branchHashes := make(map[string]bool)
	for _, c := range branchCommits {
		branchHashes[c.Hash] = true
	}

	parentHashes := make(map[string]bool)
	for _, c := range parentCommits {
		parentHashes[c.Hash] = true
	}

	// Get all commits in chronological order
	allCommits, err := git.GetCommitLog(branch, "", false)
	if err != nil {
		return "", err
	}

	// Render all commits with distinction
	for _, commit := range allCommits {
		isBranchCommit := branchHashes[commit.Hash]
		isParentCommit := parentHashes[commit.Hash]

		if multiline {
			output.WriteString(renderCommitMultilineWithIndicator(commit, isBranchCommit, isParentCommit))
		} else {
			output.WriteString(renderCommitOnelineWithIndicator(commit, isBranchCommit, isParentCommit))
		}
	}

	return output.String(), nil
}

// renderAllCommitsGrouped shows commits grouped by relationship
func renderAllCommitsGrouped(branch string, parent string, multiline bool, ascii bool, output *strings.Builder) (string, error) {
	// Get branch-only commits
	branchCommits, err := git.GetCommitLog(branch, parent, true)
	if err != nil {
		return "", err
	}

	// Determine symbols based on ascii flag
	currentSymbol := "▲"
	commonSymbol := "●"
	parentSymbol := "▼"
	if ascii {
		currentSymbol = "^"
		commonSymbol = "o"
		parentSymbol = "v"
	}

	if len(branchCommits) > 0 {
		// Show section header for branch commits
		sectionHeader := fmt.Sprintf("%s Current branch only (%d)\n", currentSymbol, len(branchCommits))
		output.WriteString(sectionHeader)

		// Render branch-unique commits
		for _, commit := range branchCommits {
			if multiline {
				output.WriteString(renderCommitMultiline(commit, true))
			} else {
				output.WriteString(renderCommitOneline(commit, true))
			}
		}
		output.WriteString("\n")
	}

	// Get parent-only commits (commits in parent that aren't in branch yet)
	parentCommits, err := git.GetCommitLog(parent, branch, true)
	if err != nil {
		return "", err
	}

	// Get shared/common commits (commits in both branches)
	sharedCommits, err := git.GetCommitLog(parent, "", false)
	if err != nil {
		return "", err
	}

	// Build set of branch and parent commit hashes to identify shared commits
	branchSet := make(map[string]bool)
	for _, c := range branchCommits {
		branchSet[c.Hash] = true
	}
	parentOnlySet := make(map[string]bool)
	for _, c := range parentCommits {
		parentOnlySet[c.Hash] = true
	}

	// Filter to only shared commits
	var actualShared []*git.Commit
	for _, c := range sharedCommits {
		if !branchSet[c.Hash] && !parentOnlySet[c.Hash] {
			actualShared = append(actualShared, c)
		}
	}

	if len(actualShared) > 0 {
		// Show section header for shared commits
		sectionHeader := fmt.Sprintf("%s Common commits (%d)\n", commonSymbol, len(actualShared))
		output.WriteString(sectionHeader)

		for _, commit := range actualShared {
			if multiline {
				output.WriteString(renderCommitMultiline(commit, false))
			} else {
				output.WriteString(renderCommitOneline(commit, false))
			}
		}
		output.WriteString("\n")
	}

	if len(parentCommits) > 0 {
		// Show section header for parent commits
		sectionHeader := fmt.Sprintf("%s Parent branch only (%d)\n", parentSymbol, len(parentCommits))
		output.WriteString(sectionHeader)

		for _, commit := range parentCommits {
			if multiline {
				output.WriteString(renderCommitMultiline(commit, false))
			} else {
				output.WriteString(renderCommitOneline(commit, false))
			}
		}
	}

	return output.String(), nil
}

// renderLegend shows the legend for grouped mode
func renderLegend(ascii bool) string {
	var legend string

	if ascii {
		legend = "\nLegend: ^ Current branch only | o Common commits | v Parent branch only\n"
	} else {
		legend = "\nLegend: \u25b2 Current branch only | \u25cf Common commits | \u25bc Parent branch only\n"
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
		legend = "\nLegend: o Common commit | v Parent branch only\n"
	} else {
		legend = "\nLegend: \u25cf Common commit | \u25bc Parent branch only\n"
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
func renderCommitOnelineWithIndicator(commit *git.Commit, isBranch bool, isParent bool) string {
	hash := commit.ShortHash
	message := commit.Message
	indicator := "  "

	if isBranch {
		// Branch-unique commit: bright, no indicator
		if EnableColor {
			hash = ColorCurrent(hash)
		}
		return fmt.Sprintf("%s%s  %s\n", indicator, hash, message)
	} else if isParent {
		// Parent-only commit: dimmed with ▼
		indicator = "▼ "
		if EnableColor {
			hash = ColorDescription(hash)
			message = ColorDescription(message)
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
func renderCommitMultilineWithIndicator(commit *git.Commit, isBranch bool, isParent bool) string {
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
	} else if isParent {
		indicator = "[▼ Parent] "
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
