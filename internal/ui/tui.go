package ui

import (
	"fmt"
	"sort"
	"strings"

	"pretty-git/internal/git"

	tea "github.com/charmbracelet/bubbletea"
)

// BranchNode represents a branch in the interactive tree
type BranchNode struct {
	Name     string
	Parent   string
	Children []*BranchNode
	Expanded bool
}

// Model holds the state of the TUI
type Model struct {
	Root         *BranchNode
	All          []*BranchNode // flat list of all nodes in render order
	Current      string        // current branch
	Selection    int           // index in All list
	Parents      map[string]string
	ErrorMsg     string
	ActionMode   string // "" (normal), "checkout", "set-parent", "inspect"
	ActionBranch string
	Quitting     bool
}

// NewTUIModel creates a new TUI model
func NewTUIModel() (*Model, error) {
	parents, err := git.AllParents()
	if err != nil {
		return nil, err
	}

	branches, err := git.ListBranches()
	if err != nil {
		return nil, err
	}

	current, _ := git.GetCurrentBranch() // allow if detached

	// Build tree structure
	root := &BranchNode{
		Name:     "[root]",
		Children: []*BranchNode{},
	}

	nodeMap := map[string]*BranchNode{}
	for _, b := range branches {
		nodeMap[b] = &BranchNode{
			Name:     b,
			Parent:   parents[b],
			Children: []*BranchNode{},
			Expanded: true,
		}
	}

	// Link children to parents
	orphans := []*BranchNode{}
	for _, b := range branches {
		node := nodeMap[b]
		if p, ok := parents[b]; ok && p != "" {
			if parent, exists := nodeMap[p]; exists {
				parent.Children = append(parent.Children, node)
			} else {
				// parent not in repo, treat as orphan
				orphans = append(orphans, node)
			}
		} else {
			// no parent metadata, treat as root
			orphans = append(orphans, node)
		}
	}

	root.Children = orphans
	sortChildren(root)

	m := &Model{
		Root:       root,
		Current:    current,
		Parents:    parents,
		Selection:  0,
		ActionMode: "",
	}

	m.rebuildFlatList()
	return m, nil
}

func sortChildren(node *BranchNode) {
	sort.Slice(node.Children, func(i, j int) bool {
		return node.Children[i].Name < node.Children[j].Name
	})
	for _, child := range node.Children {
		sortChildren(child)
	}
}

func (m *Model) rebuildFlatList() {
	m.All = []*BranchNode{}
	m.visitForRender(m.Root)
	// Clamp selection
	if m.Selection >= len(m.All) {
		m.Selection = len(m.All) - 1
	}
	if m.Selection < 0 {
		m.Selection = 0
	}
}

func (m *Model) visitForRender(node *BranchNode) {
	for _, child := range node.Children {
		m.All = append(m.All, child)
		if child.Expanded {
			m.visitForRender(child)
		}
	}
}

// Init initializes the model
func (m *Model) Init() tea.Cmd {
	return nil
}

// Update handles user input
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Note: Error messages are persistent until next action.
		// Future: Could add a timer if we need auto-clearing.
		keyStr := msg.String()

		switch keyStr {
		case "q", "ctrl+c":
			m.Quitting = true
			return m, tea.Quit

		case "up", "k":
			if m.Selection > 0 {
				m.Selection--
			}

		case "down", "j":
			if m.Selection < len(m.All)-1 {
				m.Selection++
			}

		case " ":
			// Toggle expand/collapse only if node has children
			if m.Selection < len(m.All) && m.All[m.Selection] != nil {
				node := m.All[m.Selection]
				if len(node.Children) > 0 {
					node.Expanded = !node.Expanded
					m.rebuildFlatList()
					if node.Expanded {
						m.ErrorMsg = fmt.Sprintf("Expanded: %s", node.Name)
					} else {
						m.ErrorMsg = fmt.Sprintf("Collapsed: %s", node.Name)
					}
				} else {
					m.ErrorMsg = "This branch has no children to expand/collapse"
				}
			}

		case "enter":
			// Checkout selected branch
			if m.Selection < len(m.All) && m.All[m.Selection] != nil {
				branch := m.All[m.Selection].Name
				m.ActionMode = "checkout"
				m.ActionBranch = branch
				if err := git.CheckoutBranch(branch, false, ""); err != nil {
					m.ErrorMsg = fmt.Sprintf("✗ Checkout failed: %v", err)
				} else {
					m.Current = branch
					m.ErrorMsg = fmt.Sprintf("✓ Checked out: %s", branch)
				}
			}

		case "p":
			// Set parent for selected branch (placeholder)
			if m.Selection < len(m.All) && m.All[m.Selection] != nil {
				branch := m.All[m.Selection].Name
				m.ActionMode = "set-parent"
				m.ActionBranch = branch
				m.ErrorMsg = fmt.Sprintf("Set parent: not yet interactive (edit with: git config --local pretty-git.parent.<branch> <parent>)")
			}

		case "i":
			// Inspect branch metadata
			if m.Selection < len(m.All) && m.All[m.Selection] != nil {
				branch := m.All[m.Selection].Name
				parent := m.Parents[branch]
				if parent == "" {
					parent = "(no parent)"
				}
				m.ErrorMsg = fmt.Sprintf("Info: %s → parent: %s", branch, parent)
			}

		case "home":
			m.Selection = 0

		case "end":
			m.Selection = len(m.All) - 1
		}
	}

	return m, nil
}

// View renders the TUI
func (m *Model) View() string {
	var sb strings.Builder

	sb.WriteString("pretty-git browse\n")
	sb.WriteString("─────────────────────────────────────────\n")
	sb.WriteString("Navigation: ↑/k, ↓/j | Space: expand/collapse | Enter: checkout\n")
	sb.WriteString("p: set parent | i: inspect | q: quit\n")
	sb.WriteString("─────────────────────────────────────────\n\n")

	// Render tree
	for i, node := range m.All {
		if node == nil {
			continue
		}

		prefix := m.getTreePrefix(node)
		marker := ""

		if i == m.Selection {
			marker = " > "
		} else {
			marker = "   "
		}

		// Get status for this branch
		parentBranch := m.Parents[node.Name]
		status := git.GetBranchStatus(node.Name, parentBranch)

		display := GetBranchDisplay(node.Name, node.Name == m.Current, status)
		statusStr := GetStatusMarkers(status)

		// Show expand indicator if has children
		expandChar := "  "
		if len(node.Children) > 0 {
			if node.Expanded {
				expandChar = "▼ "
			} else {
				expandChar = "▶ "
			}
		}

		sb.WriteString(fmt.Sprintf("%s%s%s%s%s\n", marker, prefix, expandChar, display, statusStr))
	}

	sb.WriteString("\n─────────────────────────────────────────\n")
	if m.ErrorMsg != "" {
		sb.WriteString(fmt.Sprintf("Status: %s\n", m.ErrorMsg))
	} else {
		if m.Selection < len(m.All) && m.All[m.Selection] != nil {
			selected := m.All[m.Selection]
			parentInfo := m.Parents[selected.Name]
			if parentInfo == "" {
				parentInfo = "(no parent)"
			}
			sb.WriteString(fmt.Sprintf("Selected: %s | Parent: %s\n", selected.Name, parentInfo))
		}
	}

	return sb.String()
}

// getTreePrefix calculates the tree prefix with connectors showing nesting structure
// Uses box-drawing characters (├─, └─, │) to match the static branches output
func (m *Model) getTreePrefix(node *BranchNode) string {
	path := m.getPathToNode(m.Root, node)
	if len(path) == 0 {
		return ""
	}

	var prefix string
	// path[0] is always Root, skip it
	for i := 1; i < len(path); i++ {
		parent := path[i-1]
		current := path[i]
		isLast := current == parent.Children[len(parent.Children)-1]

		if i == len(path)-1 {
			// Last node in path: use connector
			if isLast {
				prefix += "└─ "
			} else {
				prefix += "├─ "
			}
		} else {
			// Intermediate node: use vertical line or blank
			if isLast {
				prefix += "   "
			} else {
				prefix += "│  "
			}
		}
	}

	return prefix
}

// getPathToNode returns the path from root to the target node
func (m *Model) getPathToNode(parent *BranchNode, target *BranchNode) []*BranchNode {
	if parent == target {
		return []*BranchNode{parent}
	}

	for _, child := range parent.Children {
		if path := m.getPathToNode(child, target); len(path) > 0 {
			return append([]*BranchNode{parent}, path...)
		}
	}

	return nil
}
