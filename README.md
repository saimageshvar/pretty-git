# pretty-git — developer README

Small CLI to visualize git branch parent→child trees and record parent metadata and descriptions on branch creation. All output is aesthetically refined with status indicators, coloring, and interactive features.

Status
- **Complete**: `checkout` (wrapper with parent metadata and descriptions), `branches` (static renderer with status indicators and descriptions), `browse` (interactive TUI), `set` (set parent and/or description for branches), `log` (parent-aware commit log with smart scoping), and `snapshot` (save and restore working directory checkpoints without committing).
- Metadata recorded in repository-local git config under `branch.<branch>.pretty-git-parent` and `branch.<branch>.pretty-git-description`.
- Branch names with special characters (/, _, -, ~) work transparently - no encoding needed.
- **Status indicators** integrated throughout: merged, ahead/behind, diverged, tracking, stale detection.
- **Branch descriptions** displayed as subtle subtitle text beneath branch names in both static and interactive views.
- **Commit log** shows commits with parent-awareness: branch-unique commits by default, with `--all` flag to show full history with visual distinction.
- **Snapshots** allow saving periodic checkpoints of uncommitted changes, stored using git stash without affecting the working directory.
- Core files: `cmd/pretty-git/*`, `internal/{git,ui,cmdutil}/*`.


Build

Prerequisites: Go toolchain (1.20+ recommended) and git installed.

From the module root (recommended):

```bash
go mod tidy
go build -o pretty-git ./cmd/pretty-git
```

Alternatives:

- Build from any directory by specifying package path:

```bash
go build -o /tmp/pretty-git /absolute/path/to/pretty-git/cmd/pretty-git
```

Usage

Run the binary to see help and commands:

```bash
./pretty-git --help
./pretty-git checkout --help
./pretty-git branches --help
./pretty-git browse --help
./pretty-git set --help
./pretty-git log --help
./pretty-git snapshot --help
```

checkout
- Create new branch and record parent and description:

```bash
# create from current branch and record its parent
./pretty-git checkout -b feature/foo

# create from an explicit parent with description
./pretty-git checkout -b feature/foo --parent main --desc "New feature implementation"
./pretty-git checkout -b feature/foo --parent main

# switch to existing branch (does not modify metadata)
./pretty-git checkout feature/x
```

Updating existing parent metadata

By default `pretty-git checkout` will not overwrite an existing recorded parent for a branch to avoid accidental metadata loss. Use `--update-parent` when you intentionally want to change the recorded parent for an existing branch. When overwriting, the previous value is saved automatically under the repo-local git config key `pretty-git.parent.backup.<branch>` so you can inspect or restore it if needed.

Examples:

```bash
# Overwrite recorded parent for an existing branch (interactive confirmation)
./pretty-git checkout feature --update-parent

# Overwrite and skip confirmation
./pretty-git checkout feature --update-parent --yes

# Inspect backup value saved when overwriting
git config --get pretty-git.parent.backup.feature
```

Flags
- `-b`, `--create` : create a new branch (wrapper for `git checkout -b`).
- `--parent` : explicitly specify parent branch when creating.
- `--desc` : set description for the new branch.
- `--update-parent` : when switching to an existing branch, update its recorded parent metadata (must be explicit to overwrite).
- `-y`, `--yes` : assume yes for confirmations when updating parent metadata.

set
- Set or update the recorded parent and/or description for a branch (defaults to current branch).

The `set` command provides a unified way to manage branch metadata after branch creation. You can set parent, description, or both. When updating existing values, you'll be prompted for confirmation unless `--yes` is provided.

Examples:

```bash
# Set parent for current branch
./pretty-git set --parent main

# Set description for current branch
./pretty-git set --desc "Feature implementation"

# Set both parent and description for current branch
./pretty-git set --parent main --desc "Feature implementation"

# Set parent and description for a named branch
./pretty-git set feature-x --parent develop --desc "New feature X"

# Skip confirmation when overwriting existing values
./pretty-git set --parent main --yes
```

Flags for `set`:
- `--parent` : parent branch to set (optional).
- `--desc` : description to set for the branch (optional).
- `-y`, `--yes` : assume yes for confirmations when updating existing metadata.

Note: At least one of `--parent` or `--desc` must be provided.

set-parent (deprecated)
- Legacy command for setting parent only. Use `set --parent` instead.

By design you can set the parent at branch creation time, or later using the `set` command. When called with a single argument the command sets the parent for the current branch; when called with two arguments it sets the parent for the named branch.

Examples:

```bash
# Set parent for the current branch to 'base' (interactive confirmation if a parent already exists)
./pretty-git set-parent base

# Set parent for a named branch and skip confirmation
./pretty-git set-parent feature base --yes

# Set parent for a named branch explicitly
./pretty-git set-parent feature base

# Inspect backup (previous value saved when overwriting):
git config --get pretty-git.parent.backup.feature
```

Flags for `set-parent`:
- `-y`, `--yes` : assume yes for confirmations when updating an existing parent.

branches
- Render recorded parent→child tree with status indicators and descriptions. Current branch is highlighted (green + ● marker). Shows merge status relative to parent branch. Descriptions are displayed as subtle subtitle text beneath each branch name.

Status indicators:
- `✓` = branch merged into its parent
- `↑ N` = N commits ahead of upstream
- `↓ N` = N commits behind upstream
- `↔ N↑M↓` = diverged (N ahead, M behind)
- `◇` = tracking upstream
- `⚡ Nd` = stale (no activity for >30 days)

Example output (default with descriptions):

```
● master
  Main development branch
└─ parent1 [✓]
   Updated feature branch description
   └─ parent1-child1 [✓]
      Implementation of X feature
      └─ parent1-child2 [✓]
         Bugfixes and improvements
```

Example output (default without descriptions):

```
● master
└─ parent1 [✓]
   └─ parent1-child1 [✓]
      └─ parent1-child2 [✓]
```

Example output (verbose):

```
● master
└─ parent1 [✓] (master)
   └─ parent1-child1 [✓] (parent1)
      └─ parent1-child2 [✓] (parent1-child1)
```

Branches flags
- `--compact` : use a compact layout with narrower indents and connectors.
- `--verbose` : show parent metadata inline for each branch (e.g., "feature (main)").
- `--no-color` : disable colored output.
- `--no-marker` : hide the current-branch marker.

Examples:

```bash
# default (with status markers)
./pretty-git branches

# compact layout
./pretty-git branches --compact

# verbose (show parent inline)
./pretty-git branches --verbose

# compact + verbose
./pretty-git branches --compact --verbose

# disable color
./pretty-git branches --no-color

# hide current marker
./pretty-git branches --no-marker
```

Important: Merge status (`✓`) checks if the branch is merged into its **direct parent branch** (from metadata), not into main/master. This correctly differentiates between:
- A branch newly checked out from its parent (no `✓`)
- A branch whose commits have been merged into its parent (`✓`)

browse
- Launch an interactive terminal UI (TUI) for navigating and managing branches. Provides a dynamic tree view with keyboard navigation, expand/collapse, and quick actions. Displays status indicators and branch descriptions inline with tree guide lines matching the static `branches` output.

Controls:
- `↑/k`, `↓/j` : Navigate up/down through branches
- `Space` : Toggle expand/collapse on parent nodes (▶ = collapsed, ▼ = expanded)
- `Enter` : Checkout the selected branch
- `p` : Set parent for the selected branch
- `i` : Inspect branch metadata (parent, description, backup info)
- `q`, `Ctrl+C` : Quit the TUI

Example:

```bash
# Launch the interactive TUI
./pretty-git browse
```

The TUI displays:
- Current branch highlighted in green with a ● marker
- Tree structure with box-drawing guide lines (├─, └─, │) showing nesting depth and sibling relationships
- Branch descriptions displayed as subtle subtitle text beneath each branch name
- Expand/collapse indicators (▼ = expanded, ▶ = collapsed) with clear visual distinction from tree connectors
- Status markers inline: merged (✓), ahead/behind (↑/↓/↔), tracking (◇), stale (⚡)
- Status bar showing selected branch and its parent metadata
- Keyboard-driven navigation for efficient branch management

log
- Display commit history with parent-awareness. Default shows only commits unique to the current branch. Use `--all` to show full history organized into three sections (▲ current only, ● common, ▼ parent only).

Flags:
- `--all` : show full history with sections (default: branch-unique only)
- `--multiline` : detailed format with file stats (default: compact oneline)
- `--chronological` : time-ordered view with inline indicators (only with `--all`)
- `--ascii` : use ASCII symbols (^, o, v) instead of Unicode (▲, ●, ▼)
- `--no-color` : disable colored output
- `--max-commits <N>` : limit commits per section to N (default: 300, use 0 for unlimited)
- `--no-pager` : disable automatic pagination (pager is enabled by default)
- `--force-pager` : force pager to stay open even for small outputs

Performance notes:
- By default, each section is limited to 300 commits to prevent performance issues with large repositories
- When a section is truncated, a message like "... 150 more commits" is displayed
- Use `--max-commits=0` to disable the limit and show all commits
- **Pager is enabled by default** and will automatically activate when output exceeds your terminal height
- Pager uses your system's `$PAGER` variable or defaults to `less` with options for color support and auto-quit
- Legend appears at the top for immediate visibility when paging through commits

Examples:

```bash
# Default: branch-unique commits only (pager auto-activates if needed)
./pretty-git log

# Full history organized by sections (with pager)
./pretty-git log --all

# Disable pager for piping or scripts
./pretty-git log --all --no-pager

# Full history with limited commits per section
./pretty-git log --all --max-commits=50

# Full history with no limit
./pretty-git log --all --max-commits=0

# Chronological order with indicators
./pretty-git log --all --chronological

# Detailed multiline format
./pretty-git log --multiline

# Force pager to stay open (useful for small outputs)
./pretty-git log --all --force-pager
```

snapshot
- Save and restore working directory snapshots without committing. Snapshots are stored using git stash under the hood but don't affect your working directory.

**Important**: You must stage the files you want to snapshot using `git add` before creating a snapshot. Only staged changes are included in snapshots.

Create a snapshot:

```bash
# Stage files first, then create a snapshot
git add file1.txt file2.txt
./pretty-git snapshot create checkpoint-1

# Stage all changes and create a snapshot with a message
git add .
./pretty-git snapshot create checkpoint-1 -m "Before refactoring"
```

List snapshots:

```bash
# List all saved snapshots (grouped by branch)
./pretty-git snapshot list
```

Output shows snapshots organized by branch, with the current branch highlighted:

```
Saved snapshots (4):

■ master (current)
  ● master-checkpoint
    Time: 2026-01-31 08:54:49
    Hash: 21c3381d3cff

■ feature/ui-updates
  ● ui-checkpoint-1
    Time: 2026-01-31 08:54:40
    Hash: c8bd240427c3
```

Restore to a snapshot:

```bash
# Restore working directory to a saved snapshot
./pretty-git snapshot restore checkpoint-1
```

**Note**: If restoring a snapshot from a different branch than the one you're currently on, you'll receive a warning and confirmation prompt to prevent accidental cross-branch restores.

Clear snapshots:

```bash
# Remove a specific snapshot
./pretty-git snapshot clear checkpoint-1

# Remove all snapshots (with confirmation)
./pretty-git snapshot clear --all

# Remove all snapshots without confirmation
./pretty-git snapshot clear --all --yes
```

Flags for `snapshot create`:
- `-m`, `--message` : Optional message for the snapshot (default: "Snapshot: <name>")

Flags for `snapshot clear`:
- `--all` : Clear all snapshots instead of a single one
- `-y`, `--yes` : Skip confirmation when clearing all snapshots

Use cases:
- Experiment with changes while preserving multiple checkpoint states
- Create named savepoints during a complex refactoring
- Quickly switch between different work-in-progress states
- Save periodic backups of staged work without cluttering commit history

Technical details:
- **Staging required**: Only staged changes are included in snapshots. Use `git add` to stage files before creating a snapshot
- **Branch scoping**: Each snapshot records the branch it was created on. Snapshots are grouped by branch in list view, and restoring a snapshot from a different branch triggers a warning
- Snapshots use `git stash create` internally, which creates a stash commit of staged changes without modifying your working directory
- Snapshot metadata is stored in `.git/pretty-git-snapshots.json`
- Restoring uses `git restore --source <hash>` to apply the snapshot state
- **Warning on restore**: If you have uncommitted changes, you'll be prompted before they're overwritten
- **Cross-branch warning**: Restoring a snapshot from a different branch shows a warning and requires confirmation
- Conflicts during restore are handled by git (conflict markers will be left for manual resolution)
- Creating a snapshot with an existing name overwrites the previous snapshot with that name
- **Snapshot names**: Can contain spaces and most special characters (except newlines/tabs). Names are validated before creation
- **Hash validation**: When restoring, the tool checks if the snapshot hash still exists (it may be cleaned up by `git gc`)
- **Independent snapshots**: Each snapshot is self-contained. Clearing one snapshot doesn't affect others

Future enhancements:
- See [LATER.md](LATER.md) for planned improvements


Implementation notes
- Git commands use the system `git` via `internal/cmdutil.RunGit`.
- Parent metadata stored with:

```bash
git config --local branch.<child>.pretty-git-parent <parent>
```

- Description metadata stored with:

```bash
git config --local branch.<branch>.pretty-git-description <description>
```

- Branch names with special characters (/, _, -, ~) work transparently in git config - no encoding needed as git handles quoted section names automatically.
- `internal/cmdutil.RunGit` returns stdout, stderr, exit code, and error so callers can treat `git config --get-regexp` exit code 1 as "no matches" (empty metadata).
- **Status detection** (`internal/git/git.go`): `GetBranchStatus()` checks merge status against **direct parent** (not main), detects ahead/behind via `git rev-list`, and checks staleness by comparing commit timestamp with current time (>30 days = stale).
- **Coloring** (`internal/ui/style.go`): Current branch is bright green; merged branches are dim/gray; stale branches are yellow; descriptions are rendered in dim italic; everything else uses terminal default color (works in all color schemes).
- **Commit display** (`internal/ui/render.go`): Branch-unique commits shown bright/bold; common/parent commits shown dimmed. Supports both grouped (by section) and chronological modes.
- **Description display**: Rendered as subtle subtitle text beneath branch names with proper indentation matching the tree structure.
- **Expand/collapse indicators** (`internal/ui/tui.go`): ▼ for expanded, ▶ for collapsed (triangles provide clear visual distinction from tree connectors).
- Renderer implemented in `internal/ui/render.go` for static display and `internal/ui/tui.go` for interactive TUI using bubbletea.
- Interactive TUI in `cmd/pretty-git/browse.go` uses bubbletea (`github.com/charmbracelet/bubbletea`).
- **Snapshots** (`cmd/pretty-git/snapshot.go`, `internal/git/git.go`): Use `git stash create` to snapshot all changes without modifying working directory; metadata stored in `.git/pretty-git-snapshots.json`; restore via `git restore --source <hash>`.

Future work / contributions
- Add `goreleaser.yml` and native packaging (nfpm) for releases.
- Add tests for `internal/git` and `internal/ui` functions.
- Implement branch filtering, grouping by remote/upstream, and configuration persistence.
- Support worktrees (currently out of scope).
- Optional: Add migration tool to convert existing `pretty-git.parent.*` entries to new `branch.<branch>.pretty-git-parent` format.

Development workflow

1. Make edits, run `go build` and run `./pretty-git branches` to verify output.
2. Test with `./pretty-git browse` for interactive verification.
3. Use `/tmp/pg-test` as a test repository or create fresh test repos to verify status detection.
4. If adding new dependencies run `go mod tidy` and commit `go.sum`.

Contact / Maintainer
 - Project scaffolded locally. See git history for commits.
