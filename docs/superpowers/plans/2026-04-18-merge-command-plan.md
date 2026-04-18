# `pgit merge` Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `pgit merge` command with interactive branch picker TUI, then run `git merge` with full output passthrough.

**Architecture:** Add a `PickerMode` to the existing branch UI model that controls Enter behavior: `ModeSwitch` (current checkout behavior) vs `ModeSelect` (returns branch name with no git action). The `merge` command uses `ModeSelect`, then runs `git merge` as a regular exec command with stdout/stderr piped to the terminal.

**Tech Stack:** Go, bubbletea (charmbracelet), lipgloss, standard library exec

---

### Task 1: Add `PickerMode` to branchui

**Files:**
- Modify: `internal/ui/branch/keymap.go`
- Modify: `internal/ui/branch/model.go`

- [ ] **Step 1: Add PickerMode type to keymap.go**

Add at the top of `keymap.go`, after the imports and before `keyMap`:

```go
// PickerMode controls the Enter-key behavior in the branch list.
type PickerMode int

const (
	ModeSwitch PickerMode = iota // default: checkout/switch to selected branch
	ModeSelect                    // return selected branch name, no git action
)

// String returns a human-readable label for the mode, used in key hints.
func (m PickerMode) String() string {
	switch m {
	case ModeSelect:
		return "select"
	default:
		return "switch"
	}
}
```

Update `defaultKeyMap()` to accept a `PickerMode` parameter and use `mode.String()` for the Switch key help text:

```go
// defaultKeyMap returns the keybindings used by the branch view.
func defaultKeyMap(mode PickerMode) keyMap {
	return keyMap{
		Up: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("↑", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("↓", "down"),
		),
		Switch: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", mode.String()),
		),
		Edit: key.NewBinding(
			key.WithKeys("ctrl+e"),
			key.WithHelp("ctrl+e", "edit"),
		),
		EscBack: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "clear/quit"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
		),
	}
}
```

- [ ] **Step 2: Add mode field to Model struct in model.go**

Add `mode PickerMode` to the Model struct, after the `spinner` field:

```go
type Model struct {
	// ... existing fields ...
	spinner spinner.Model
	mode    PickerMode // controls Enter behavior: ModeSwitch or ModeSelect
}
```

- [ ] **Step 3: Update New() to accept variadic mode parameter**

Replace the `New` function signature and its return statement:

```go
func New(branches []git.Branch, repoName string, termWidth, termHeight int, modes ...PickerMode) Model {
```

At the top of the function body, add mode resolution:

```go
	mode := ModeSwitch // default
	if len(modes) > 0 {
		mode = modes[0]
	}
```

Change the `keys` field in the return struct from:
```go
		keys:             defaultKeyMap(),
```
to:
```go
		keys:             defaultKeyMap(mode),
```

Add the mode to the returned struct:
```go
		mode:             mode,
```

- [ ] **Step 4: Modify the Switch key handler in updateKeys for ModeSelect**

Replace the entire `key.Matches(msg, m.keys.Switch)` case in `updateKeys` with:

```go
	case key.Matches(msg, m.keys.Switch):
		if m.switching || len(m.filtered) == 0 {
			return m, nil
		}
		b := m.filtered[m.cursor].branch
		if b.IsCurrent {
			if m.mode == ModeSelect {
				m.err = "cannot merge a branch into itself"
			}
			return m, nil
		}
		if m.mode == ModeSelect {
			m.switchedTo = b.Name
			m.done = true
			return m, tea.Quit
		}
		m.switching = true
		m.err = ""
		name := b.Name
		if b.IsRemote {
			parts := strings.SplitN(name, "/", 3)
			if len(parts) == 3 {
				name = parts[2]
			}
		}
		return m, doSwitch(name)
```

This keeps `ModeSwitch` behavior unchanged and adds `ModeSelect` behavior: quit the TUI with `switchedTo` set to the selected branch name.

- [ ] **Step 5: Build to verify no compilation errors**

```bash
cd /home/sai/projects/pretty-git && go build ./...
```

Expected: clean build, no errors.

- [ ] **Step 6: Commit**

```bash
git add internal/ui/branch/keymap.go internal/ui/branch/model.go
git commit -m "feat(branchui): add PickerMode for reusable branch selection"
```

---

### Task 2: Add `merge` case to main.go

**Files:**
- Modify: `cmd/pretty-git/main.go`

- [ ] **Step 1: Add merge usage text and command case**

In the usage block (inside the `if len(os.Args) < 2` block), add this line after the stash lines:

```go
		fmt.Fprintln(os.Stderr, "  merge                     pick & merge a branch")
```

In the switch statement, add a case for `"merge"` before the `default` case:

```go
	case "merge":
		runWithUpdate("merge", runMerge)
```

- [ ] **Step 2: Build to verify**

```bash
cd /home/sai/projects/pretty-git && go build ./...
```

Expected: compilation error on undefined `runMerge` — this is correct, we create it next.

- [ ] **Step 3: Commit**

```bash
git add cmd/pretty-git/main.go
git commit -m "feat: add merge command entry point to main.go"
```

---

### Task 3: Create merge.go

**Files:**
- Create: `cmd/pretty-git/merge.go`

- [ ] **Step 1: Write merge.go**

Create `cmd/pretty-git/merge.go` with the following content:

```go
package main

import (
	"fmt"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"

	"github.com/sai/pretty-git/internal/git"
	branchui "github.com/sai/pretty-git/internal/ui/branch"
)

func runMerge() {
	// ── Pre-flight checks ──────────────────────────────────────────────
	currentBranch := git.CurrentBranch()
	if currentBranch == "" {
		fmt.Fprintln(os.Stderr, "pgit: not a git repository or detached HEAD")
		os.Exit(1)
	}

	branches, err := git.ListLocalBranches()
	if err != nil {
		fmt.Fprintf(os.Stderr, "pgit: %v\n", err)
		os.Exit(1)
	}

	// Count branches excluding the current one
	mergeable := 0
	for _, b := range branches {
		if !b.IsCurrent {
			mergeable++
		}
	}
	if mergeable == 0 {
		fmt.Fprintln(os.Stderr, "pgit: no other local branches to merge")
		os.Exit(0)
	}

	repoName := git.RepoName()

	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		width, height = 120, 40
	}

	// ── Open branch picker in select mode ──────────────────────────────
	m := branchui.New(branches, repoName, width, height, branchui.ModeSelect)

	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	result, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "pgit: %v\n", err)
		os.Exit(1)
	}

	final, ok := result.(branchui.Model)
	if !ok || final.SwitchedTo() == "" {
		return // user quit without selecting
	}

	selectedBranch := final.SwitchedTo()

	// ── Run git merge with passthrough output ──────────────────────────
	cmd := exec.Command("git", "merge", selectedBranch)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			// Non-exit error (e.g. command not found)
			fmt.Fprintf(os.Stderr, "pgit: %v\n", err)
			os.Exit(1)
		}
	}

	switch exitCode {
	case 0:
		fmt.Printf("✓ Merged '%s' into '%s'\n", selectedBranch, currentBranch)
	case 1:
		// Conflicts — git already printed instructions (git merge --continue, --abort)
		// Stay silent and let the user follow git's own guidance.
	default:
		os.Exit(exitCode)
	}
}
```

- [ ] **Step 2: Build and verify**

```bash
cd /home/sai/projects/pretty-git && go build ./...
```

Expected: clean build, no errors.

- [ ] **Step 3: Verify usage text**

```bash
cd /home/sai/projects/pretty-git && go build -o pgit-test ./cmd/pretty-git && ./pgit-test 2>&1 | grep merge
```

Expected output includes:
```
  merge                     pick & merge a branch
```

- [ ] **Step 4: Commit**

```bash
git add cmd/pretty-git/merge.go
git commit -m "feat: add pgit merge command with branch picker TUI"
```

---

### Task 4: Manual verification

- [ ] **Step 1: Test merge command opens TUI**

```bash
cd /home/sai/projects/pretty-git && ./pgit-test merge
```

Expected: Branch picker TUI opens with local branches, Enter shows "select" in help text, current branch shows ★ marker, pressing Enter on current branch shows error "cannot merge a branch into itself".

- [ ] **Step 2: Test quit behavior**

In the TUI, press `esc` without selecting anything.

Expected: TUI closes silently, no merge runs.

- [ ] **Step 3: Clean up test binary**

```bash
rm -f pgit-test
```
