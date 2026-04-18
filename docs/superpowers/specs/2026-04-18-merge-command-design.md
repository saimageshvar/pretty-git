# Design: `pgit merge` with Reusable Branch Picker

## Problem

`pgit` lacks a merge command. Users want an interactive TUI to pick a branch to merge into the current branch, then let regular `git merge` handle the actual merge (including conflicts).

The branch picker should be reusable by any future command that needs to select a local branch.

## Architecture

### 1. Reusable Branch Picker Mode

The existing `branchui` model (`internal/ui/branch/model.go`) gains a **picker mode** that controls what happens when the user presses Enter.

```go
// PickerMode controls the Enter-key behavior in the branch list.
type PickerMode int

const (
    ModeSwitch PickerMode = iota // default: checkout/switch to selected branch
    ModeSelect                    // return selected branch name, no git action
)
```

**Changes to `New()`:**
- Add variadic `modes ...PickerMode` parameter — zero args means `ModeSwitch`
- Existing callers (`runBranch`, `runCheckout`) pass no mode and continue unchanged

**Changes to Enter handler (`updateKeys`):**
- `ModeSwitch` → existing async `doSwitch()` flow (unchanged)
- `ModeSelect` → if selected branch is the current branch (`IsCurrent == true`), set `err` message and stay open. Otherwise set `switchedTo` to branch name, set `done = true`, return `tea.Quit`. No git operation runs.

**Changes to keymap:**
- Help text for Enter key shows "switch" in `ModeSwitch`, "select" in `ModeSelect`

### 2. `pgit merge` Command (`cmd/pretty-git/merge.go`)

**Flow:**
1. Validate we're inside a git repo with a valid HEAD
2. Load local branches via `git.ListLocalBranches()`
3. Exit early if fewer than 2 branches (nothing to merge from)
4. Open TUI in `ModeSelect` mode
5. User selects branch → TUI closes, returns branch name
6. User quits (esc/ctrl+c) → silent exit, no action
7. Run `git merge <branch>` with stdout/stderr piped directly to terminal
8. Capture exit code and handle:

| Exit code | Meaning | Action |
|-----------|---------|--------|
| 0 | Clean merge succeeded | Print `✓ Merged '<branch>'` |
| 1 | Merge conflicts | Stay silent — git's own output already shows conflict details and `git merge --continue` / `--abort` instructions |
| >1 | Fatal error | Print error message from stderr, exit with same code |

**Current branch guard:** If the user selects the current branch, the TUI shows an inline error: `✗ cannot merge a branch into itself` and stays open, letting them pick a different branch or quit.

**Key implementation detail:**
```go
cmd := exec.Command("git", "merge", selectedBranch)
cmd.Stdout = os.Stdout
cmd.Stderr = os.Stderr
err := cmd.Run()
```
This ensures git's full output (including post-merge hints) flows to the user naturally.

### 3. Git Layer (`internal/git/git.go`)

No new functions needed. `git.CurrentBranch()` already returns `""` when not in a git repo or on detached HEAD, which is sufficient to detect the "not in a git repo" case. `merge.go` uses `exec.Command("git", "merge", ...)` directly so it can access the raw exit code.

## Files Changed

| File | Change |
|------|--------|
| `internal/ui/branch/model.go` | Add `PickerMode` type, pass to `New()`, modify Enter handler for `ModeSelect` |
| `internal/ui/branch/keymap.go` | Accept `PickerMode` in `defaultKeyMap()` for dynamic help text |
| `internal/git/git.go` | No changes (existing `CurrentBranch()` covers repo detection) |
| `cmd/pretty-git/main.go` | Add `"merge"` case in command switch + usage text |
| `cmd/pretty-git/merge.go` | **New file**: `runMerge()` — TUI picker + passthrough `git merge` |

## Reusability Pattern

Any future command that needs a branch picker follows this template:

```go
branches, err := git.ListLocalBranches()
// handle err...

m := branchui.New(branches, repoName, width, height, branchui.ModeSelect)
p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
result, err := p.Run()
// handle err...

final, ok := result.(branchui.Model)
if !ok || final.SwitchedTo() == "" {
    return // user quit
}
selectedBranch := final.SwitchedTo()
// proceed with the selected branch
```

## Error Handling

- **Not a git repo** → print error, exit 1
- **Branch listing fails** → print error, exit 1
- **No local branches** → print "no branches found", exit 0
- **Only 1 branch or no other local branches** → print message, exit 0
- **User quits TUI** → silent exit, no action
- **TUI runtime error** → print error, exit 1
- **`git merge` exit 0** → print success message
- **`git merge` exit 1** (conflicts) → silent exit, let git's output guide the user
- **`git merge` exit >1** → print stderr, exit with same code

No pre-merge safety checks are added. `git merge` itself refuses to overwrite uncommitted changes and handles all conflict scenarios correctly.
