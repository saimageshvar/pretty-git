# Command Reference

---

## pgit branch

Browse local branches and switch between them.

```bash
pgit branch
```

### What you see

A split-pane view: branch list on the left, branch details on the right. The current branch is shown first; others are sorted by most-recent commit.

The detail pane shows:
- Branch name and description
- Parent branch and relationship (merged / N ahead / diverged)

### Keybindings

| Key | Action |
|-----|--------|
| `↑` / `↓` | Navigate list |
| `Enter` | Switch to selected branch |
| `Ctrl+E` | Edit description and parent branch |
| `Esc` | Clear filter / quit |
| `Ctrl+C` | Force quit |

### Branch metadata

`pgit` can store a **description** and a **parent branch** for each branch. These live in your local `.git/config` and are used across all commands.

```
branch.<name>.pgit-desc    # short description
branch.<name>.pgit-parent  # parent branch name
```

Edit them with `Ctrl+E` in the branch browser, or set them at creation time with `pgit checkout -b`.

---

## pgit checkout

Switch branches or create new ones.

### Usage

```bash
pgit checkout                            # same as pgit branch
pgit checkout <name>                     # switch if branch exists; open create form if not
pgit checkout -b                         # open create wizard on current branch
pgit checkout -b <name>                  # create wizard pre-filled with name
pgit checkout -b <name> -p <parent>      # pre-fill parent too
pgit checkout -b <name> -p <parent> -d <desc>  # create directly, no TUI
```

### Create wizard

Running `pgit checkout -b` opens a three-field form:

1. **Branch name** — text input
2. **Parent branch** — scrollable list of local branches (defaults to current branch)
3. **Description** — optional short text, stored in `.git/config`

Tab / Shift+Tab moves between fields. Enter on the last field creates the branch and switches to it.

### Flags

| Flag | Description |
|------|-------------|
| `-b` | Create mode — open the create wizard |
| `-p <branch>` | Set the parent branch |
| `-d <desc>` | Set the branch description |

When all three of `-b`, `-p`, and `-d` are provided the branch is created immediately without opening a TUI.

---

## pgit log

Browse commit history with an inline detail pane.

```bash
pgit log          # commits on HEAD
pgit log <ref>    # commits on any branch, tag, or commit hash
```

Shows up to 200 commits.

### What you see

Left pane: commit list — short hash, author, relative date, subject line.
Right pane (open with `→`): full commit message, author email, absolute date, file change stats.

### Keybindings

| Key | Action |
|-----|--------|
| `↑` / `↓` or `k` / `j` | Navigate commits |
| `PgUp` / `PgDn` or `Ctrl+U` / `Ctrl+D` | Page up / down |
| `→` or `l` | Open detail pane |
| `←` or `h` | Close detail pane |
| `f` | Toggle filter bar |
| `q` or `Ctrl+C` | Quit |

### Filters

Press `f` to reveal the filter bar. Toggle with Space:

- **My commits** — show only commits by your git user email
- **Skip merges** — hide merge commits

---

## pgit stash

Full stash lifecycle — create, browse, apply, pop, and drop.

### Create wizard

```bash
pgit stash
```

A three-phase wizard:

**Phase 1 — Choose what to stash:**

| Option | What gets stashed |
|--------|-------------------|
| Staged | Only staged changes (`git stash --staged`) |
| Unstaged | Only unstaged changes |
| All | Everything (staged + unstaged) |
| Custom | Select individual files from a list |

Use `↑/↓` to pick, `Space` to toggle files in Custom mode, `a` to select all, `n` to deselect all.

**Phase 2 — Enter a message.**
The stash is automatically prefixed with the short hash of the last commit.

**Phase 3 — Spinner while the stash is created.**

### Quick stash (no TUI)

When you already know what you want:

```bash
pgit stash "saving work"                              # stash all changes
pgit stash --staged "UI tweak"                        # staged only
pgit stash --unstaged "WIP"                           # unstaged only
pgit stash --custom "partial" -- src/a.go src/b.go   # specific files
```

### Browse modes

```bash
pgit stash apply    # browse and apply a stash (stash is kept)
pgit stash list     # same as apply
pgit stash pop      # browse and pop a stash (asks for confirmation)
pgit stash drop     # browse and drop a stash (asks for confirmation)
```

All browse modes show a list on the left and a file-change detail pane on the right.

**Keys in browse mode:**

| Key | Action |
|-----|--------|
| `↑` / `↓` | Navigate list |
| `→` / `←` | Open / close detail pane |
| `Enter` | Apply / pop / drop selected stash |
| `y` | Confirm destructive action (pop / drop) |
| `q` or `Ctrl+C` | Quit |

---

## pgit prompt

Print compact git context for embedding in your shell prompt.

```bash
pgit prompt [flags]
```

Outputs nothing when not in a git repo or on a detached HEAD.

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--shell bash` | — | Required for bash. Wraps ANSI escapes so line-length is counted correctly. |
| `--max-desc N` | `32` | Max characters of branch description to show. Set to `0` to hide. |
| `--newline` | off | Two-line layout — git info on line 1, arrow on line 2. |
| `--arrow CHAR` | `❯` | Arrow character. Only shown with `--newline`. |
| `--no-color` | off | Plain text, no ANSI colour codes. |

### Shell setup

See [docs/prompt.md](prompt.md) for setup instructions and examples.

---

## Environment variables

| Variable | Effect |
|----------|--------|
| `PGIT_NO_UPDATE_CHECK=1` | Disable the automatic update-available notification |

---

## Global notes

- All TUI views use **inline rendering** — they render inside the terminal's normal scrollback, not an alternate screen. After you quit, the output remains visible in your history.
- Mouse support is not enabled; all interaction is keyboard-driven.
- `pgit` requires `git` to be installed and available on your `$PATH`.
