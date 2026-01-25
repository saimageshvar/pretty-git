Crisp context summary

- Purpose: provide a developer-focused CLI that "prettifies" git branch output by rendering parent→child trees and recording parent metadata per branch in repository-local git config under `pretty-git.parent.<branch>`.
- Current state: MVP implemented. `pretty-git checkout` wraps branch creation and records parent; `pretty-git branches` renders an ASCII tree and highlights the current branch. Core helpers live in `internal/git` and `internal/cmdutil`.
- Important details:
  - `internal/cmdutil.RunGit` returns (stdout, stderr, exitCode, error) so callers can distinguish non-zero git exits (e.g., `git config --get-regexp` returns exit 1 when no matches).
  - `internal/git.AllParents()` treats git exit code 1 as "no entries".
  - Worktrees are intentionally out of scope for the MVP.

Refined next steps (concrete, prioritized)

1) Parent-update UX (done)
	- Implemented `--update-parent` on `checkout` to allow explicitly updating parent metadata when switching to an existing branch.
	- Behavior: explicit flag required to overwrite existing parent; prompts for confirmation unless `--yes` provided; previous values are backed up under `pretty-git.parent.backup.<branch>`.
	- Acceptance: implemented in `cmd/pretty-git/checkout.go`, `internal/git.SetParent` (creates backups), added `pretty-git set-parent` command, and verification docs (`docs/VERIFY_UPDATE_PARENT.md` and `README.md`) updated.

2) Renderer polish (done)
	- Improved visual output by refining the existing renderer in `internal/ui/render.go` rather than adding a new dependency.
	- Implemented deterministic ordering (branches/children sorted) to ensure stable output.
	- Added compact and verbose display modes (`--compact`, `--verbose`) to `pretty-git branches`.
	- Added style hooks/toggles in `internal/ui/style.go` to control coloring and the current-branch marker (`--no-color`, `--no-marker`).
	- Acceptance: renderer now produces cleaner Unicode box drawing with stable ordering, supports compact/verbose modes, and style toggles; documented in `README.md`.

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
