package ui

import (
    "fmt"
    "sort"
    "strings"

    "pretty-git/internal/git"
)

// RenderBranchesTree builds an ASCII tree of branches from parent metadata and highlights the current branch.
func RenderBranchesTree(parents map[string]string, current string) (string, error) {
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
            children[""] = append(children[""] , b)
            continue
        }
        children[p] = append(children[p], b)
    }

    // ensure deterministic order
    for k := range children {
        sort.Strings(children[k])
    }

    // roots are children[""] (no recorded parent) plus any parent names that exist but not children of others
    roots := children[""]
    sort.Strings(roots)

    var lines []string

    // recursive printer
    var printNode func(name string, prefix string, last bool)
    printNode = func(name string, prefix string, last bool) {
        marker := ""
        display := name
        if name == current {
            marker = CurrentMarker
            display = ColorCurrent(name)
        }

        connector := "├── "
        if last {
            connector = "└── "
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
                    newPrefix = "    "
                } else {
                    newPrefix = "│   "
                }
            } else {
                if last {
                    newPrefix = prefix + "    "
                } else {
                    newPrefix = prefix + "│   "
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
