# pretty-git

**Prettier, more useful git commands — right in your terminal.**

`pgit` wraps common git operations in interactive TUI views: colourful, keyboard-driven, and information-dense. Think `fzf` for your git workflow.

```
⎇ feature/auth · add OAuth login flow ❯
```

---

## Why pgit?

Plain git output is terse. Finding the right branch, reading a stash list, or understanding commit history often means running several commands and mentally assembling the picture.

`pgit` gives you that picture directly — branch relationships, commit details, stash contents — all in one place, navigable with arrow keys.

- **Fewer commands** — one `pgit branch` replaces `git branch`, `git log --oneline`, `git switch`
- **More context** — see descriptions, parent branches, file diffs, commit stats alongside lists
- **Stays in scrollback** — output remains visible after you quit, like `fzf` (no alternate screen)

---

## Install

```bash
curl -sSL https://raw.githubusercontent.com/saimageshvar/pretty-git/main/install.sh | sudo bash
```

Supports Linux and macOS on amd64 and arm64. Installs `pgit` to `/usr/local/bin`.

> Ubuntu users can also install the `.deb` package from the [Releases](https://github.com/saimageshvar/pretty-git/releases) page.

Full installation options → [docs/install.md](docs/install.md)

---

## Commands

| Command | What it does |
|---------|-------------|
| `pgit branch` | Browse and switch branches |
| `pgit checkout` | Switch branch or create a new one with a wizard |
| `pgit log [ref]` | Browse commit history |
| `pgit stash` | Create, apply, pop, or drop stashes |
| `pgit prompt` | Output compact git context for your shell prompt |

### pgit branch

Browse all local branches, see their descriptions and parent relationships, and switch with Enter.

```
  main                      ┌─────────────────────────────┐
▸ feature/auth              │ feature/auth                │
  fix/login-redirect        │ parent: main  (2 ahead)     │
  chore/deps                │                             │
                            │ add OAuth login flow        │
                            └─────────────────────────────┘
```

**Keys:** `↑/↓` navigate · `Enter` switch · `Ctrl+E` edit description · `Esc` quit

### pgit checkout

Same as `pgit branch` when run alone. Pass a name to switch directly or trigger the create wizard:

```bash
pgit checkout                        # interactive branch browser
pgit checkout feature/auth           # switch (or open create form if it doesn't exist)
pgit checkout -b feature/auth        # create wizard — prompts for parent & description
pgit checkout -b feature/auth -p main -d "OAuth login flow"  # create without TUI
```

### pgit log

Browse up to 200 commits with an inline detail pane.

```bash
pgit log          # commits on HEAD
pgit log main     # commits on any ref
```

**Keys:** `↑/↓` navigate · `→/←` open/close detail pane · `f` toggle filters (mine / skip merges) · `q` quit

### pgit stash

Full stash workflow from one command.

```bash
pgit stash                      # wizard — choose what to stash, add a message
pgit stash apply                # browse stashes, apply one
pgit stash pop                  # browse stashes, pop one (with confirmation)
pgit stash drop                 # browse stashes, drop one (with confirmation)

# Quick stash (no TUI):
pgit stash "saving work"        # stash everything with a message
pgit stash --staged "UI tweak"  # staged changes only
pgit stash --unstaged "WIP"     # unstaged changes only
pgit stash --custom "partial" -- src/auth.go src/session.go  # specific files
```

### pgit prompt

Embed git context in your shell prompt. Outside a git repo it outputs nothing — your prompt stays exactly as-is.

**zsh** — add to `~/.zshrc`:
```zsh
PROMPT='$(pgit prompt) '$PROMPT
```

**bash** — add to `~/.bashrc`:
```bash
PS1='$(pgit prompt --shell bash) '$PS1
```

Full prompt options and two-line layout → [docs/prompt.md](docs/prompt.md)

---

## Branch descriptions & parent tracking

`pgit` lets you attach a short description and a parent branch to any branch. These are stored in your local `.git/config` and shown throughout the UI.

Set them when creating a branch:
```bash
pgit checkout -b feature/auth -p main -d "OAuth login flow"
```

Or edit them later in the branch browser with `Ctrl+E`.

The description also appears in your shell prompt: `⎇ feature/auth · add OAuth login flow ❯`

---

## Full documentation

- [Installation guide](docs/install.md) — all install methods, updating, uninstalling
- [Command reference](docs/commands.md) — every flag, keybinding, and option
- [Shell prompt setup](docs/prompt.md) — prompt integration and customisation

---

## Requirements

- Git installed and available on `$PATH`
- A terminal emulator with colour support (virtually all modern terminals)
- Linux or macOS (Windows not currently supported)
