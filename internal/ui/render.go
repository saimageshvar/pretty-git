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
