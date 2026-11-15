# 💻 CLI Command Reference

Arjax provides a comprehensive, structured command-line interface powered by Typer and Rich.

---

## Command Overview

```
arjax [OPTIONS] COMMAND [ARGS]...
```

* Running `arjax` without arguments opens the **interactive TUI** by default.
* Passing `--tui` / `-t` to `search` or running `arjax tui` explicitly opens the TUI.

---

## 1. Package Discovery & Installation

### `arjax search [QUERY]...`
Search for software across all repositories and declarative recipes. (Read-only operation).

```bash
# Basic search
arjax search firefox

# Multi-word search
arjax search "video editor"

# ML-based smart recommendations & Linux alternatives (TF-IDF)
arjax search photoshop --smart
arjax search zapfast -s

# Limit returned results
arjax search git --limit 10

# Sort AUR results (votes, popularity, modified)
arjax search neovim --aur-sortby votes

# Bypass local search cache
arjax search brave --no-cache
```

### `arjax install <PACKAGES>...`
Install one or more packages using prioritized provider fallback.

```bash
# Standard installation
arjax install brave

# Multi-package parallel installation
arjax install vlc gimp obs-studio

# Target a specific provider
arjax install signal --provider flatpak
arjax install zapfast --provider appimage
arjax install vscode --provider vendor

# Verbose debug mode (shows exact commands executed)
arjax install brave --debug
```

### `arjax suggest <PURPOSE>`
Purpose-based software recommendations based on user intention or ML TF-IDF vectorization.

```bash
# Natural language / intent suggestions
arjax suggest "video editing"
arjax suggest coding
arjax suggest --list

# ML TF-IDF alternatives for proprietary apps
arjax suggest photoshop --smart
arjax suggest "office suite" -s
```

---

## 2. Maintenance & Diagnostics

### `arjax stats`
Displays disk usage, cache statistics, installed package counts, and system health alerts.

```bash
arjax stats
```

### `arjax cleanup <TARGET>`
Cleans orphaned packages or temporary build caches.

```bash
# Preview orphaned packages without removing
arjax cleanup orphans --dry-run

# Remove orphaned packages via DAG dependency analysis
arjax cleanup orphans

# Clean package cache
arjax cleanup cache
```

### `arjax history <SUBCOMMAND>`
Audit trail of past operations.

```bash
# View last 20 transactions with duration in ms
arjax history list

# Rollback a previous transaction
arjax history rollback <transaction_id>
```

---

## 3. Recipes & System Management

### `arjax recipes <SUBCOMMAND>`
Manage local and upstream declarative installation recipes.

```bash
# List local vs upstream recipe versions
arjax recipes list

# Pull and synchronize recipes from central registry
arjax recipes sync
```

### `arjax completion <SHELL>`
Install or output tab completion scripts for bash, zsh, or fish:

```bash
arjax completion install bash
arjax completion show zsh
```

### `arjax uninstall`
Remove Arjax and clean up its launcher files:

```bash
arjax uninstall
arjax uninstall --purge   # Also deletes saved configuration
```

