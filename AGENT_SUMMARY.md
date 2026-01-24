Crisp context summary

- Purpose: provide a developer-focused CLI that "prettifies" git branch output by rendering parent→child trees and recording parent metadata per branch in repository-local git config under `pretty-git.parent.<branch>`.
- Current state: MVP implemented. `pretty-git checkout` wraps branch creation and records parent; `pretty-git branches` renders an ASCII tree and highlights the current branch. Core helpers live in `internal/git` and `internal/cmdutil`.
- Important details:
  - `internal/cmdutil.RunGit` returns (stdout, stderr, exitCode, error) so callers can distinguish non-zero git exits (e.g., `git config --get-regexp` returns exit 1 when no matches).
  - `internal/git.AllParents()` treats git exit code 1 as "no entries".
  - Worktrees are intentionally out of scope for the MVP.

Refined next steps (concrete, prioritized)

1) Parent-update UX (top priority)
	- Implement `--update-parent` (or `--force-parent`) on `checkout` to allow explicitly updating parent metadata when switching to an existing branch.
	- Behavior: explicit flag required to overwrite existing parent; prompt for confirmation unless `--yes` provided; create a simple backup of modified config entries to allow undo.
	- Acceptance: flag implemented in `cmd/pretty-git/checkout.go`, `internal/git.SetParent` updated to support backups, and manual verification steps documented.

2) Renderer polish (high priority)
	- Improve visual output: adopt `treeprint` or refine current renderer for cleaner Unicode box drawing, deterministic ordering, and compact/verbose display modes.
	- Add theme/config hooks in `internal/ui/style.go` for markers and color toggles.

3) Interactive TUI (medium priority)
	- Add `pretty-git browse` TUI (bubbletea/tcell) for interactive navigation, expand/collapse, and quick branch actions (checkout, set parent, inspect).

4) Branch filtering, grouping & config (medium priority)
	- Add flags to `branches` for prefix/regex filtering, subtree focus, and grouping by remote/upstream.
	- Support repository/user config (`.pretty-git.yaml` or git-config fallbacks) to persist view preferences.

5) Enhanced metadata & integrations (lower priority)
	- Optionally record richer metadata at branch creation (timestamp, author, base SHA) and add `pretty-git inspect <branch>` to view it.
	- Provide export options (JSON, DOT) for external visualization or IDE integrations.

6) Performance & large-repo handling (lower priority)
	- Optimize metadata reads (batched git commands), add caching, and support streaming/pagination for very large branch sets.

7) Tests, CI and releases (deprioritized)
	- Tests, CI and packaging are important but intentionally deprioritized to favor rapid feature delivery. Add unit/integration tests and CI once core features stabilize.

Notes for next agent
 - Prioritize implementing parent-update UX and renderer improvements first; these deliver the most visible developer value.
 - Keep changes small, document manual verification steps, and avoid heavy upfront test investment—tests can follow after features land.
 - Preserve git-config namespace `pretty-git.parent.*` and avoid altering unrelated git config keys.
 - Use temporary repositories for manual or automated verification when possible.
