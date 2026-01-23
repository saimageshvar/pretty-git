pretty-git — Branch-listing MVP Plan

Module: pretty-git

Overview
- Goal: small CLI that visualizes branch parent→child trees and records parent metadata on branch creation/checkouts. Produce a minimal MVP with two commands: `checkout` and `branches`.
- Constraints: use system `git` via `os/exec`; store metadata in repository-local git config keys under `pretty-git.parent.<branch>`; skip worktrees (future scope); skip tests for now.

Todo (current)
- [x] Research libraries & design
- [x] Define project layout (module name `pretty-git`; git-config namespace `pretty-git.*`; worktrees = future)
- [ ] Implement CLI skeleton (`cmd/pretty-git/*`)
- [ ] Implement checkout wrapper (`internal/git/*`)
- [ ] Implement branches visualization (`internal/ui/*`)
- [ ] Configure packaging (`goreleaser.yml` + nfpm)
- [ ] Future: worktree support

Recommended libraries (one line each)
- CLI framework: github.com/spf13/cobra — battle-tested, subcommand support.
- Git interaction: system git via os/exec — fidelity with user's git and easy access to config.
- Tree rendering: github.com/xlab/treeprint — lightweight ASCII tree rendering.
- Coloring: github.com/fatih/color — simple cross-platform terminal coloring.
- Packaging (.deb): goreleaser + nfpm — standard release and native package toolchain.

File skeleton (paths, purpose, and exported signatures)
- cmd/pretty-git/main.go
  Purpose: app entrypoint; initialize root Cobra command and add subcommands.
  Export: func Execute() error

- cmd/pretty-git/checkout.go
  Purpose: implement `pretty-git checkout` command behavior.
  Export: func NewCheckoutCmd() *cobra.Command

- cmd/pretty-git/branches.go
  Purpose: implement `pretty-git branches` command (tree visualization).
  Export: func NewBranchesCmd() *cobra.Command

- internal/git/git.go
  Purpose: wrapper helpers calling system `git` and parsing config.
  Exports:
    func ListBranches() ([]string, error)
    func CheckoutBranch(branch string, create bool) error
    func SetParent(child, parent string) error
    func AllParents() (map[string]string, error)
    func GetParent(child string) (string, bool, error)
    func GetCurrentBranch() (string, error)

- internal/ui/render.go
  Purpose: render branch tree and handle coloring/highlight of current branch.
  Exports:
    func RenderBranchesTree(parents map[string]string, current string) (string, error)
    func PrintBranchesTree(parents map[string]string, current string) error

- internal/ui/style.go
  Purpose: centralize color and marker selection for current branch.
  Exports/vars:
    var CurrentMarker string
    func ColorCurrent(s string) string

- internal/cmdutil/exec.go
  Purpose: helper to run git commands and return stdout/stderr wrapped errors.
  Export: func RunGit(args ...string) (string, string, error)

Git metadata (exact commands and helper signature)
- After creating a branch from `main`:
  git checkout -b feature/x main
  git config --local pretty-git.parent.feature/x main

- To set/update parent for a branch:
  git config --local pretty-git.parent.<child> <parent>

- Go helper to write metadata:
  func SetParent(child, parent string) error

- Error handling: run the git command via os/exec, return nil on success; on failure return wrapped error containing command, exit code and combined stderr for diagnosability (do not panic; propagate to CLI).

Reading metadata (signatures and notes)
- Helpers:
    func AllParents() (map[string]string, error)
    func GetParent(child string) (string, bool, error)
    func ListBranches() ([]string, error)
    func GetCurrentBranch() (string, error)

- Read current branch: use `git rev-parse --abbrev-ref HEAD` and trim newline; treat detached HEAD specially (return error or special marker).
- Read all parents: use `git config --local --get-regexp '^pretty-git\.parent\.'` and parse lines `pretty-git.parent.<child> <parent>`.

CLI UX examples
- Create-and-checkout (recording parent automatically):
  pretty-git checkout -b feature/foo
  Behavior: determine previous current branch (GetCurrentBranch), run `git checkout -b feature/foo`, on success run `git config --local pretty-git.parent.feature/foo <previous-branch>`.

- Create-from-parent:
  pretty-git checkout -b feature/foo --parent main
  Behavior: run `git checkout -b feature/foo main` and set metadata accordingly.

- Switch to existing branch (no parent change):
  pretty-git checkout feature/x
  Behavior: run `git checkout feature/x`; do not modify parent metadata unless user supplies flag to update.

- Branches display:
  pretty-git branches
  Output: ASCII parent→child tree (unicode box drawing) where the current branch is highlighted (green/bold) and marked with a bullet or asterisk.
  Example (conceptual):
    main
    ├── feature-1
    │   ├── task-1
    │   └── task-2
    │       └── task-2-subtask-1
    ├── feature-2
    └── bugfix-1
        └── bugfix-1-refined
  Current branch example: `feature-1/task-2` shown as `• feature-1/task-2` in green.

goreleaser (`goreleaser.yml`) minimal outline keys for .deb
- top-level keys to include:
  - project_name / name
  - builds (list: binary, main path, env, ldflags)
  - archive (optional)
  - nfpm (package metadata)
    - package
    - name
    - version (use {{ .Version }})
    - arch
    - maintainer
    - description
    - assets (map binary -> /usr/bin/pretty-git)
    - section, homepage, depends (optional)
  - release (optional)

Short next actions (4 steps)
1. Scaffold repository files (`cmd/pretty-git/*`, `internal/git/*`, `internal/ui/*`, `internal/cmdutil/*`) and initialize `go.mod` (module `pretty-git`).
2. Implement `internal/cmdutil.RunGit` and `internal/git` helpers: `ListBranches`, `GetCurrentBranch`, `AllParents`, `SetParent`.
3. Implement `checkout` command: support `-b` and `--parent`, record parent metadata on new branch creation.
4. Implement `branches` command: call `AllParents` + `GetCurrentBranch`, render tree with `treeprint`, highlight current branch with `fatih/color`.

Notes
- Worktrees are explicitly out of scope; add a TODO/future-scope note in README.
- Tests are deferred per current request