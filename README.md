# pretty-git — developer README

Small CLI to visualize git branch parent→child trees and record parent metadata on branch creation. All output is aesthetically refined with status indicators, coloring, and interactive features.

Status
- **Complete**: `checkout` (wrapper with parent metadata), `branches` (static renderer with status indicators), and `browse` (interactive TUI).
- Metadata recorded in repository-local git config under `branch.<branch>.pretty-git-parent`.
- Branch names with special characters (/, _, -, ~) work transparently - no encoding needed.
- **Status indicators** integrated throughout: merged, ahead/behind, diverged, tracking, stale detection.
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
```

checkout
- Create new branch and record parent (default):

```bash
# create from current branch and record its parent
./pretty-git checkout -b feature/foo

# create from an explicit parent
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
- `--update-parent` : when switching to an existing branch, update its recorded parent metadata (must be explicit to overwrite).
- `-y`, `--yes` : assume yes for confirmations when updating parent metadata.

set-parent
- Set or update the recorded parent for the current branch (or a named branch).

By design you can set the parent at branch creation time, or later using the `set-parent` command. When called with a single argument the command sets the parent for the current branch; when called with two arguments it sets the parent for the named branch.

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
- Render recorded parent→child tree with status indicators. Current branch is highlighted (green + ● marker). Shows merge status relative to parent branch.

Status indicators:
- `✓` = branch merged into its parent
- `↑ N` = N commits ahead of upstream
- `↓ N` = N commits behind upstream
- `↔ N↑M↓` = diverged (N ahead, M behind)
- `◇` = tracking upstream
- `⚡ Nd` = stale (no activity for >30 days)

Example output (default):

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
- Launch an interactive terminal UI (TUI) for navigating and managing branches. Provides a dynamic tree view with keyboard navigation, expand/collapse, and quick actions. Displays status indicators inline with tree guide lines matching the static `branches` output.

Controls:
- `↑/k`, `↓/j` : Navigate up/down through branches
- `Space` : Toggle expand/collapse on parent nodes (▶ = collapsed, ▼ = expanded)
- `Enter` : Checkout the selected branch
- `p` : Set parent for the selected branch
- `i` : Inspect branch metadata (parent, backup info)
- `q`, `Ctrl+C` : Quit the TUI

Example:

```bash
# Launch the interactive TUI
./pretty-git browse
```

The TUI displays:
- Current branch highlighted in green with a ● marker
- Tree structure with box-drawing guide lines (├─, └─, │) showing nesting depth and sibling relationships
- Expand/collapse indicators (▼ = expanded, ▶ = collapsed) with clear visual distinction from tree connectors
- Status markers inline: merged (✓), ahead/behind (↑/↓/↔), tracking (◇), stale (⚡)
- Status bar showing selected branch and its parent metadata
- Keyboard-driven navigation for efficient branch management


Implementation notes
- Git commands use the system `git` via `internal/cmdutil.RunGit`.
- Parent metadata stored with:

```bash
git config --local pretty-git.parent.<child> <parent>
```

- Branch names containing `/` are encoded as `.` in git config keys (e.g., `feature/login` → `pretty-git.parent.feature.login`) for compatibility with git config key restrictions.
- `internal/cmdutil.RunGit` returns stdout, stderr, exit code, and error so callers can treat `git config --get-regexp` exit code 1 as "no matches" (empty metadata).
- **Status detection** (`internal/git/git.go`): `GetBranchStatus()` checks merge status against **direct parent** (not main), detects ahead/behind via `git rev-list`, and checks staleness by comparing commit timestamp with current time (>30 days = stale).
- **Coloring** (`internal/ui/style.go`): Current branch is bright green; merged branches are dim/gray; stale branches are yellow; everything else uses terminal default color (works in all color schemes).
- **Expand/collapse indicators** (`internal/ui/tui.go`): ▼ for expanded, ▶ for collapsed (triangles provide clear visual distinction from tree connectors).
- Renderer implemented in `internal/ui/render.go` for static display and `internal/ui/tui.go` for interactive TUI using bubbletea.
- Interactive TUI in `cmd/pretty-git/browse.go` uses bubbletea (`github.com/charmbracelet/bubbletea`).

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
