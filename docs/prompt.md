# Shell Prompt Integration

See your current branch and its description right in the prompt:

```
⎇ feature/auth · add OAuth login flow ❯
```

Outside a git repo your prompt looks exactly as it does today — nothing changes.

---

## Setup

Pick your shell and add **one line** to your config file.

### zsh — add to `~/.zshrc`

```zsh
PROMPT='$(pgit prompt) '$PROMPT
```

### bash — add to `~/.bashrc`

```bash
PS1='$(pgit prompt --shell bash) '$PS1
```

Then open a new terminal (or run `source ~/.zshrc` / `source ~/.bashrc`).

---

## Two-line prompt

Prefer more room to type? Git info on line 1, cursor on line 2:

```
⎇ feature/auth · add OAuth login flow
❯ _
```

### zsh

```zsh
PROMPT='$(pgit prompt --newline)'
```

### bash

```bash
PS1='$(pgit prompt --shell bash --newline)'
```

---

## Options

| Flag | What it does |
|------|-------------|
| `--shell bash` | Required for bash. Makes colours work without breaking line wrapping. |
| `--max-desc 32` | How many characters of the description to show. Default is 32. Set to `0` to hide the description. |
| `--newline` | Two-line layout — git info above, cursor below. |
| `--arrow ❯` | Change the arrow on the second line. Only used with `--newline`. |
| `--no-color` | Plain text, no colour. |

---

## How to set a branch description

When creating a branch:
```
pgit checkout -b
```

Or edit an existing branch's description from the branch browser:
```
pgit branch   →  navigate to branch  →  press e
```
