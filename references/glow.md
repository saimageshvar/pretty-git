# Glow — Complete Reference

**Repository:** https://github.com/charmbracelet/glow
**Install:** `go install github.com/charmbracelet/glow/v2@latest`

Glow is a terminal-based Markdown reader and browser. It offers both a CLI for quick rendering and a full TUI for browsing local Markdown files.

---

## Installation

### Package Managers

```bash
# macOS / Linux (Homebrew)
brew install glow

# macOS (MacPorts)
sudo port install glow

# Arch Linux
pacman -S glow

# Void Linux
xbps-install -S glow

# NixOS
nix-shell -p glow --command glow

# FreeBSD
pkg install glow

# Solus
eopkg install glow

# Android (Termux)
pkg install glow

# Windows (Chocolatey)
choco install glow

# Windows (Scoop)
scoop install glow

# Windows (Winget)
winget install charmbracelet.glow

# Ubuntu (Snap)
sudo snap install glow

# Debian/Ubuntu (APT)
sudo mkdir -p /etc/apt/keyrings
curl -fsSL https://repo.charm.sh/apt/gpg.key | sudo gpg --dearmor -o /etc/apt/keyrings/charm.gpg
echo "deb [signed-by=/etc/apt/keyrings/charm.gpg] https://repo.charm.sh/apt/ * *" | sudo tee /etc/apt/sources.list.d/charm.list
sudo apt update && sudo apt install glow

# Fedora/RHEL (YUM)
echo '[charm]
name=Charm
baseurl=https://repo.charm.sh/yum/
enabled=1
gpgcheck=1
gpgkey=https://repo.charm.sh/yum/gpg.key' | sudo tee /etc/yum.repos.d/charm.repo
sudo yum install glow
```

### Go

```bash
go install github.com/charmbracelet/glow/v2@latest
```

### Build from Source (requires Go 1.21+)

```bash
git clone https://github.com/charmbracelet/glow.git
cd glow
go build
```

---

## TUI Mode

Run without arguments to launch the interactive browser:

```bash
glow
```

Glow will discover all Markdown files in:
- The current directory (and subdirectories)
- If inside a Git repository: all Markdown files in the repo

The TUI provides a high-performance Markdown pager. Most `less` keybindings apply. Press `?` to see all hotkeys.

---

## CLI Mode

Render Markdown directly to the terminal:

```bash
# Read from a local file
glow README.md

# Read from stdin
echo "# Hello" | glow -
cat file.md | glow -

# Fetch from GitHub / GitLab (auto-detects README)
glow github.com/charmbracelet/glow
glow gitlab.com/user/repo

# Fetch from HTTP
glow https://host.tld/file.md
```

---

## CLI Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--style` | `-s` | Style: `dark`, `light`, `notty`, `dracula`, or path to custom JSON |
| `--width` | `-w` | Word-wrap width (default: terminal width) |
| `--pager` | `-p` | Display output in pager (defaults to `less -r` or `$PAGER`) |
| `--all` | `-a` | Show all files including hidden and git-ignored |
| `--help` | `-h` | Show help |
| `--version` | `-v` | Show version |

### Examples

```bash
# Render with dark style
glow -s dark README.md

# Render with custom stylesheet
glow -s mystyle.json README.md

# Render with 60-char wrap width
glow -w 60 README.md

# Render to pager
glow -p README.md

# Start TUI showing all files (including hidden)
glow --all
```

---

## Style Selection

Glow auto-detects your terminal background:
- Dark terminal → `dark` style
- Light terminal → `light` style

To override:

```bash
glow -s dark
glow -s light
glow -s notty       # no color / plain text
glow -s dracula     # dracula theme
glow -s ./my.json   # custom JSON stylesheet
```

Community styles are available in the [Glamour style gallery](https://github.com/charmbracelet/glamour/blob/master/styles/gallery/README.md).

---

## Config File

Run `glow config` to open the config file in `$EDITOR`.

Default location depends on OS:
- **Linux/macOS:** `~/.config/glow/glow.yml`
- **Windows:** `%APPDATA%\glow\glow.yml`

Run `glow --help` to see the exact path for your system.

### Full Config Reference (`glow.yml`)

```yaml
# Style name or path to custom JSON stylesheet
# Options: "auto", "dark", "light", "notty", "dracula", or "/path/to/style.json"
# Default: "auto" (auto-detects terminal background)
style: "auto"

# Enable mouse wheel scrolling in TUI mode
# Default: false
mouse: false

# Open output in pager (CLI mode only)
# Default: false
pager: false

# Maximum column width for word wrapping (0 = terminal width)
# Default: 0
width: 80

# Show all files including hidden and git-ignored (TUI mode)
# Default: false
all: false

# Show line numbers in TUI pager
# Default: false
showLineNumbers: false

# Preserve newlines in the rendered output
# Default: false
preserveNewLines: false
```

---

## Custom Styles (JSON)

Glow uses [Glamour](https://github.com/charmbracelet/glamour) for Markdown rendering. You can create a custom JSON stylesheet.

Example minimal style:

```json
{
  "document": {
    "block_prefix": "\n",
    "block_suffix": "\n",
    "color": "252",
    "margin": 2
  },
  "heading": {
    "block_suffix": "\n",
    "bold": true,
    "color": "39"
  },
  "h1": {
    "prefix": "# ",
    "color": "205",
    "bold": true
  },
  "code": {
    "prefix": "  ",
    "suffix": "",
    "background_color": "236",
    "color": "203"
  },
  "code_block": {
    "indent": 1,
    "margin": 2,
    "color": "120",
    "background_color": "236"
  },
  "link": {
    "color": "30",
    "underline": true
  }
}
```

See the [full Glamour style spec](https://github.com/charmbracelet/glamour/tree/master/styles) for all configurable elements.

---

## TUI Keybindings

### File Browser

| Key | Action |
|-----|--------|
| `↑` / `k` | Move up |
| `↓` / `j` | Move down |
| `Enter` | Open file |
| `/` | Filter/search |
| `Esc` | Clear filter |
| `Tab` | Switch section |
| `q` / `Ctrl+C` | Quit |

### Pager (reading a file)

| Key | Action |
|-----|--------|
| `↑` / `k` | Scroll up |
| `↓` / `j` / `Space` | Scroll down |
| `PgUp` / `b` | Page up |
| `PgDn` / `f` | Page down |
| `g` / `Home` | Go to top |
| `G` / `End` | Go to bottom |
| `Esc` / `q` | Back to browser |
| `?` | Show help |

---

## Environment Variables

| Variable | Description |
|----------|-------------|
| `PAGER` | Pager program used with `--pager` flag (default: `less -r`) |
| `GLAMOUR_STYLE` | Set the Glamour/Glow style globally |
| `NO_COLOR` | Disable color output entirely |

---

## Usage as a Library (Glamour)

Glow itself is a CLI/TUI tool. For **programmatic Markdown rendering** in Go code, use [Glamour](https://github.com/charmbracelet/glamour) directly:

```go
import "github.com/charmbracelet/glamour"

// Auto-detect dark/light
out, err := glamour.Render(markdownString, "auto")

// With style
r, _ := glamour.NewTermRenderer(
    glamour.WithAutoStyle(),
    glamour.WithWordWrap(80),
)
out, err := r.Render(markdownString)
fmt.Fprint(os.Stdout, out)
```

---

## Source Code of Glow

Glow is built with:
- **Bubble Tea** — TUI framework
- **Bubbles** — List, viewport, textinput components
- **Lip Gloss** — Styling
- **Glamour** — Markdown rendering engine

It is a real-world example of a production Bubble Tea application. Studying its source code is a great way to learn advanced Bubble Tea patterns.

---

**License:** MIT
