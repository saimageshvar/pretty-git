package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/sai/pretty-git/internal/git"
	ui "github.com/sai/pretty-git/internal/ui"
)

func runList() {
	branches, err := git.ListLocalBranches()
	if err != nil {
		fmt.Fprintf(os.Stderr, "pgit: %v\n", err)
		os.Exit(1)
	}
	if len(branches) == 0 {
		fmt.Fprintln(os.Stderr, "pgit: no branches found")
		os.Exit(0)
	}

	items := buildTreeItems(branches)
	repoName := git.RepoName()

	var sb strings.Builder

	sb.WriteString(ui.StyleHeader.Render("  Branches") + "  " +
		ui.StyleAccent.Render(repoName) + "\n")
	sb.WriteString(ui.StyleDivider.Render(strings.Repeat("─", 80)) + "\n")

	for _, item := range items {
		sb.WriteString(renderTreeLine(item))
		sb.WriteString("\n")
	}

	page(sb.String())
}

type treeItem struct {
	branch     git.Branch
	treePrefix string
	depth      int
}

func buildTreeItems(branches []git.Branch) []treeItem {
	nameToIdx := make(map[string]int)
	for i, b := range branches {
		if !b.IsRemote {
			nameToIdx[b.Name] = i
		}
	}

	children := make(map[string][]git.Branch)
	for _, b := range branches {
		if b.IsRemote {
			continue
		}
		if b.Parent != "" {
			if _, ok := nameToIdx[b.Parent]; ok {
				children[b.Parent] = append(children[b.Parent], b)
				continue
			}
		}
		children[""] = append(children[""], b)
	}

	var result []treeItem

	var dfs func(parentName string, isLastAtDepth []bool)
	dfs = func(parentName string, isLastAtDepth []bool) {
		kids := children[parentName]
		for i, b := range kids {
			isLast := i == len(kids)-1
			depth := len(isLastAtDepth)

			prefix := ""
			for d := 0; d < depth; d++ {
				if isLastAtDepth[d] {
					prefix += "   "
				} else {
					prefix += "│  "
				}
			}
			if isLast {
				prefix += "└─ "
			} else {
				prefix += "├─ "
			}

			result = append(result, treeItem{
				branch:     b,
				treePrefix: prefix,
				depth:      depth,
			})

			next := make([]bool, depth+1)
			copy(next, isLastAtDepth)
			next[depth] = isLast
			dfs(b.Name, next)
		}
	}

	dfs("", []bool{})
	return result
}

func renderTreeLine(item treeItem) string {
	b := item.branch

	prefix := ui.StyleTreeConnector.Render(item.treePrefix)

	var name string
	if b.IsCurrent {
		name = ui.StyleCurrentBranch.Render("★ " + b.Name)
	} else {
		name = lipgloss.NewStyle().Foreground(ui.ColorHeader).Render(b.Name)
	}

	var desc string
	if b.Description != "" {
		desc = "  " + ui.StyleDesc.Italic(true).Render(b.Description)
	}

	var status string
	if b.Parent != "" {
		switch {
		case b.ParentAhead == 0:
			status = "  " + lipgloss.NewStyle().Foreground(ui.ColorParentMerged).Render("✓ merged")
		case b.ParentBehind == 0:
			status = "  " + lipgloss.NewStyle().Foreground(ui.ColorParentAhead).Render(fmt.Sprintf("↑%d", b.ParentAhead))
		default:
			status = "  " + lipgloss.NewStyle().Foreground(ui.ColorParentDiverged).Render(fmt.Sprintf("↑%d ↓%d", b.ParentAhead, b.ParentBehind))
		}
	}

	return prefix + name + desc + status
}

func page(output string) {
	cmdName := os.Getenv("PAGER")
	if cmdName == "" {
		cmdName = "less"
	}

	var args []string
	if cmdName == "less" || strings.HasSuffix(cmdName, "/less") {
		args = []string{"-FR", "--tilde"}
	}

	cmd := exec.Command(cmdName, args...)
	cmd.Stdin = strings.NewReader(output)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Print(output)
	}
}
