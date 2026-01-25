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
func RenderBranchesTree(parents map[string]string, current string, compact bool, verbose bool) (string, error) {
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

    // choose connectors and indents for compact mode
    connectorMid := "├── "
    connectorLast := "└── "
    indentVert := "│   "
    indentBlank := "    "
    if compact {
        connectorMid = "├─ "
        connectorLast = "└─ "
        indentVert = "│  "
        indentBlank = "   "
    }

    // recursive printer
    var printNode func(name string, prefix string, last bool)
    printNode = func(name string, prefix string, last bool) {
        marker := ""
        display := name
        if name == current {
            marker = MarkerForCurrent()
            display = ColorCurrent(name)
        }

        // verbose: append parent info inline if requested
        if verbose {
            if p := parents[name]; p != "" {
                display = fmt.Sprintf("%s (%s)", display, p)
            }
        }

        connector := connectorMid
        if last {
            connector = connectorLast
        }

        if prefix == "" {
            // root
            if marker != "" {
                lines = append(lines, fmt.Sprintf("%s%s", marker, display))
            } else {
                lines = append(lines, display)
            }
        } else {
            if marker != "" {
                lines = append(lines, fmt.Sprintf("%s%s%s%s", prefix, connector, marker, display))
            } else {
                lines = append(lines, fmt.Sprintf("%s%s%s", prefix, connector, display))
            }
        }

        ch := children[name]
        for i, c := range ch {
            isLast := i == len(ch)-1
            newPrefix := prefix
            if prefix == "" {
                if last {
                    newPrefix = indentBlank
                } else {
                    newPrefix = indentVert
                }
            } else {
                if last {
                    newPrefix = prefix + indentBlank
                } else {
                    newPrefix = prefix + indentVert
                }
            }
            printNode(c, newPrefix, isLast)
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
        printNode(r, "", last)
    }

    return strings.Join(lines, "\n") + "\n", nil
}
