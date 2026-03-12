# pgit stash: command recommendations (agent-facing)

Use these command forms:

- `staged only`: `git stash push --staged -m "<msg>"`
- `unstaged only`: `git stash push --keep-index -m "<msg>"`
- `all files`: `git stash push --include-untracked -m "<msg>"`
- `custom files`: `git stash push --include-untracked -m "<msg>" -- <path1> <path2> ...`

Pitfalls to avoid:

- `--staged` can fail on `MM` files (both staged + unstaged) with cleanup errors after creating a stash object.
- `--keep-index` does not include untracked unless `--include-untracked` is added.
- `custom files` should fail fast if the selected path list is empty.
- `git stash show` is misleading for pathspec stashes (it can display unrelated staged files).
- Always pass pathspecs after `--`.
- Use `exec.Command("git", ...)` args, not shell-built strings.

Fallback for `staged only` when `--staged` fails on mixed (`MM`) files:

1. `git stash push --keep-index --include-untracked -m "<temp>"`
2. `git stash push --staged -m "<final>"`
3. `git stash apply <temp-ref>` (without `--index`)
4. `git stash drop <temp-ref>`

Verification:

- Capture `git status --porcelain` before/after; ensure only intended paths changed and non-selected paths kept their staged/unstaged/untracked state.
