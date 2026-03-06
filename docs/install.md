# Installation

## Quick install (Linux & macOS)

The installer script detects your OS and architecture, downloads the latest release, and installs `pgit` to `/usr/local/bin`.

```bash
curl -sSL https://raw.githubusercontent.com/saimageshvar/pretty-git/main/install.sh | sudo bash
```

Supported platforms:
- Linux amd64 / arm64
- macOS amd64 (Intel) / arm64 (Apple Silicon)

After installation, verify it works:
```bash
pgit --version
```
---

## Manual — download binary

1. Go to the [Releases](https://github.com/saimageshvar/pretty-git/releases) page
2. Download the archive for your platform, e.g. `pgit_1.2.0_linux_amd64.tar.gz`
3. Extract and move the binary:

```bash
tar -xzf pgit_1.2.0_linux_amd64.tar.gz
sudo mv pgit /usr/local/bin/
```

---

## Build from source

Requires Go 1.21 or later.

```bash
git clone https://github.com/saimageshvar/pretty-git.git
cd pretty-git
go build -o pgit ./cmd/pretty-git
sudo mv pgit /usr/local/bin/
```

---

## Updating

Re-run the installer to get the latest version — it replaces the existing binary:

```bash
curl -sSL https://raw.githubusercontent.com/saimageshvar/pretty-git/main/install.sh | sudo bash
```
---

## Uninstalling

```bash
sudo rm /usr/local/bin/pgit
sudo rm -f /usr/local/bin/pretty-git   # symlink, if present
```

If you set up the shell prompt integration, remove the line you added to `~/.zshrc` or `~/.bashrc`.

---

## Install location

By default the installer places the binary in `/usr/local/bin`. To change this, set `INSTALL_DIR` before running:

```bash
INSTALL_DIR=~/.local/bin curl -sSL https://raw.githubusercontent.com/saimageshvar/pretty-git/main/install.sh | bash
```

(No `sudo` needed if the directory is user-writable.)
