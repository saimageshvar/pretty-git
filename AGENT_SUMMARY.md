Crisp context summary

- Purpose: provide a developer-focused CLI that "prettifies" git branch output by rendering parent→child trees and recording parent metadata per branch in repository-local git config under `branch.<branch>.pretty-git-parent`.
- Current state: **Complete with clean refactor**. All commands working: `pretty-git checkout` (wraps branch creation with parent metadata), `pretty-git branches` (renders tree with status indicators), `pretty-git browse` (interactive TUI). All output aesthetically refined with status markers, coloring, and visual indicators.
- **Recent refactor**: Changed from `pretty-git.parent.<branch>` to `branch.<branch>.pretty-git-parent` format:
  - ✅ Eliminates encoding/decoding complexity (/, _, -, ~ work transparently)
  - ✅ Follows git conventions (`[branch "name"]` is standard)
  - ✅ Cleaner `.git/config` - actual branch names visible
  - ✅ No performance impact (same git queries)
  - ✅ Simpler codebase - removed `encodeKey()` and `decodeKey()` functions
- Important details:
  - `internal/cmdutil.RunGit` returns (stdout, stderr, exitCode, error) so callers can distinguish non-zero git exits.
  - `internal/git.AllParents()` treats git exit code 1 as "no entries".
  - **Merge detection**: checks if branch commits are merged into its **direct parent** (not main). Differentiates between truly merged branches vs newly-checked-out children.
  - Worktrees are intentionally out of scope.
  - Test repository available at `/tmp/pg-test` for verification.

Completed items

1) **Parent-update UX** (done)
	- Implemented `--update-parent` on `checkout` to allow explicitly updating parent metadata when switching to an existing branch.
	- Behavior: explicit flag required to overwrite existing parent; prompts for confirmation unless `--yes` provided; previous values are backed up under `pretty-git.parent.backup.<branch>`.
	- Acceptance: implemented in `cmd/pretty-git/checkout.go`, `internal/git.SetParent` (creates backups), added `pretty-git set-parent` command, and verification docs updated.

2) **Renderer polish** (done)
	- Improved visual output by refining the existing renderer in `internal/ui/render.go` rather than adding a new dependency.
	- Implemented deterministic ordering (branches/children sorted) to ensure stable output.
	- Added compact and verbose display modes (`--compact`, `--verbose`) to `pretty-git branches`.
	- Added style hooks/toggles in `internal/ui/style.go` to control coloring and the current-branch marker (`--no-color`, `--no-marker`).
	- Acceptance: renderer now produces cleaner Unicode box drawing with stable ordering, supports compact/verbose modes, and style toggles; documented in README.

3) **Interactive TUI** (done)
	- Implemented `pretty-git browse` TUI using bubbletea for interactive navigation, expand/collapse, and quick branch actions.
	- Features: keyboard navigation (↑/↓/k/j), Space toggle for expand/collapse (+ for collapsed, − for expanded), Enter to checkout, 'i' to inspect metadata, 'p' to set parent, 'q' to quit.
	- Tree display with visual indicators (− expanded, + collapsed), current branch highlight (green + ● marker).
	- Full error handling: checkout failures show error message, TUI remains responsive for retry/alternative actions.
	- Character support: branch names with `/`, `_`, `-` all work via dot encoding in git config keys.
	- Acceptance: `pretty-git browse` command fully functional, tested and working correctly.

4) **Aesthetic enhancements & status indicators** (done)
	- **Triangle indicators**: Using ▼ for expanded and ▶ for collapsed - clearly distinct from tree connector lines (├─, └─, │).
	- **Status markers**: Integrated into both `branches` command and TUI:
		- `✓` = branch merged into its parent
		- `↑ N` = N commits ahead of upstream
		- `↓ N` = N commits behind upstream
		- `↔ N↑M↓` = diverged (N ahead, M behind)
		- `◇` = tracking upstream
		- `⚡ Nd` = stale (no activity for >30 days)
	- **Simplified coloring**: Terminal-compatible colors - current branch is bright green, merged branches are dim/gray, stale branches are yellow, everything else uses default terminal color (works in all color schemes).
	- **Parent-aware merge detection**: Changed from checking merge into main branch to checking merge into direct parent branch, correctly distinguishing truly merged branches from newly-checked-out children.
	- **Tree guide lines in browse**: Added box-drawing characters (├─, └─, │) to interactive TUI matching the static `branches` output, with consistent 3-character indentation width.
	- Acceptance: All status markers working in both `branches` and `browse` commands; tree structure clarity improved with guide lines; output remains clean and terminal-agnostic.

Next items (for future agents)

5) **Branch filtering, grouping & config** (medium priority)
	- Add flags to `branches` for prefix/regex filtering, subtree focus, and grouping by remote/upstream.
	- Support repository/user config (`.pretty-git.yaml` or git-config fallbacks) to persist view preferences.

6) **Enhanced metadata & integrations** (lower priority)
	- Optionally record richer metadata at branch creation (timestamp, author, base SHA) and add `pretty-git inspect <branch>` to view it.
	- Provide export options (JSON, DOT) for external visualization or IDE integrations.

7) **Performance & large-repo handling** (lower priority)
	- Optimize metadata reads (batched git commands), add caching, and support streaming/pagination for very large branch sets.

8) **Tests, CI and releases** (deprioritized)
	- Tests, CI and packaging are important but intentionally deprioritized to favor rapid feature delivery. Add unit/integration tests and CI once core features stabilize.

Technical notes for next agent

- **Status marker integration**: Located in `internal/git/git.go` (`GetBranchStatus()` and `BranchStatus` type) and `internal/ui/style.go` (`GetStatusMarkers()` and coloring functions).
- **Merge detection fix**: The key insight is checking if branch is ancestor of **its parent** (from metadata) not the main branch. This is called in `internal/ui/render.go` and `internal/ui/tui.go` by passing `parents[branchName]` to `GetBranchStatus()`.
- **Stale detection**: Checks commit timestamp via `git log` and compares with current time; threshold is 30 days (constant `StaleThreshold` in `internal/git/git.go`).
- Coloring functions in `style.go`: `ColorCurrent()` (bright green), `ColorMergedBranch()` (dim), `ColorStaleBranch()` (yellow), `ColorDefault()` (terminal default). The key change was removing per-branch-type coloring that wasn't terminal-compatible.
- **Plus/Minus indicators**: Hard-coded in `internal/ui/tui.go` lines ~254-260; easily configurable if needed.

Notes for next agent

- Prioritize testing with real repositories to ensure status markers work correctly across different upstream configurations.
- Keep changes small and maintain the git-config namespace `pretty-git.parent.*` to avoid conflicts.
- The project is now "pretty" by design — further work should focus on usability features (filtering, export) rather than visual polish.
- Use `/tmp/pg-test` or create fresh test repos to verify branch status detection works correctly with various scenarios (ahead/behind/diverged).

