# pretty-git — developer README

Small CLI to visualize git branch parent→child trees and record parent metadata on branch creation.

Status
- MVP implemented: `checkout` (wrapper) and `branches` (renderer).
- Metadata recorded in repository-local git config keys under `pretty-git.parent.<branch>`.
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

Flags
- `-b`, `--create` : create a new branch (wrapper for `git checkout -b`).
- `--parent` : explicitly specify parent branch when creating.

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

Implementation notes
- Git commands use the system `git` via `internal/cmdutil.RunGit`.
- Parent metadata stored with:

```bash
git config --local pretty-git.parent.<child> <parent>
```

- `internal/cmdutil.RunGit` now returns stdout, stderr, exit code, and error so callers can treat `git config --get-regexp` exit code 1 as "no matches" (empty metadata).
- Renderer implemented in `internal/ui/render.go` and coloring in `internal/ui/style.go` (uses `github.com/fatih/color`).

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
