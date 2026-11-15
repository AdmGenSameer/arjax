# ⚡ Quick Start Guide

Get up and running with Arjax in less than 2 minutes.

---

## 1. Installation

Install Arjax using the official one-line bootstrap script:

```bash
curl -fsSL https://raw.githubusercontent.com/AdmGenSameer/archpkg-helper/main/install.sh | bash
```

The script sets up a isolated virtual environment, installs Arjax, links the executable to `~/.local/bin/arjax`, and registers application desktop launchers.

Make sure `~/.local/bin` is in your `$PATH`:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

---

## 2. Launching the Interactive TUI

Simply type `arjax`:

```bash
arjax
```

* Type a query (e.g. `zapfast`, `firefox`, or `vlc`).
* Pause for **0.5 seconds** or press `Enter` to search.
* Use `↑` / `↓` arrow keys to highlight a result.
* Press `i` or click **`[ Install (i) ]`** to install.
* Press `r` or click **`[ Remove (r) ]`** to remove.
* Press `q` to quit.

---

## 3. Basic CLI Workflows

If you prefer direct CLI commands without entering the TUI:

### Search for Packages (Read-Only)

```bash
# Query across all available repositories and recipes
arjax search zapfast

# Multi-word query
arjax search "video player" --limit 5
```

### Install Packages

```bash
# Install a package with automatic provider fallback
arjax install zapfast

# Install multiple packages in parallel
arjax install vlc htop neovim

# Force a specific provider
arjax install signal --provider flatpak
```

### Check System Status

```bash
arjax stats
```

---

## Next Steps

* 🖥️ Learn about the [Interactive Terminal UI](TUI_GUIDE.md).
* 💻 Read the full [CLI Command Reference](cli.md).
* 📖 Understand how [Declarative Recipes](RECIPES_SPEC.md) work.

