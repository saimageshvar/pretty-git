# pretty-git — developer README

Small CLI to visualize git branch parent→child trees and record parent metadata on branch creation.

Status
- MVP implemented: `checkout` (wrapper), `branches` (static renderer), and `browse` (interactive TUI).
- Metadata recorded in repository-local git config keys under `pretty-git.parent.<branch>`.
- Branch names with slashes (e.g., `feature/login`) are encoded to `.` in git config keys for compatibility.
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
- Render recorded parent→child tree. Current branch is highlighted (green + bullet). Example output:

```
main
├── feature-1
│   ├── task-1
│   └── task-2
│       └── • task-2-subtask-1
└── bugfix-1
```

Branches flags
- `--compact` : use a compact layout with narrower indents and connectors.
- `--verbose` : show parent metadata inline for each branch (e.g., "feature (main)").
- `--no-color` : disable colored output.
- `--no-marker` : hide the current-branch marker.

Examples:

```bash
# default
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

browse
- Launch an interactive terminal UI (TUI) for navigating and managing branches. Provides a dynamic tree view with keyboard navigation, expand/collapse, and quick actions.

Controls:
- `↑/k`, `↓/j` : Navigate up/down through branches
- `Space` : Toggle expand/collapse on parent nodes
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
- Current branch highlighted in green with a bullet marker (•)
- Tree structure with expand/collapse indicators (▼/►)
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
- Renderer implemented in `internal/ui/render.go` for static display and `internal/ui/tui.go` for interactive TUI using bubbletea.
- Coloring in `internal/ui/style.go` (uses `github.com/fatih/color`).
- Interactive TUI in `cmd/pretty-git/browse.go` uses bubbletea (`github.com/charmbracelet/bubbletea`).

Future work / contributions
- Add `goreleaser.yml` and native packaging (nfpm) for releases.
- Add tests for `internal/git` and `internal/ui` functions.
- Add option to update parent metadata when switching to an existing branch.
- Support worktrees (currently out of scope).

Development workflow

1. Make edits, run `go build` and run `./pretty-git branches` to verify.
2. If adding new dependencies run `go mod tidy` and commit `go.sum`.

Contact / Maintainer
 - Project scaffolded locally. See git history for commits.
