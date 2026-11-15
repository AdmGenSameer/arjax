# 🖥️ Arjax Terminal User Interface (TUI) Guide

Arjax features a responsive, keyboard- and mouse-accessible Terminal User Interface built using **Textual** (Python) with native **Bubbletea** (Go) engine bridges.

---

## 🚀 Launching the TUI

Arjax opens the interactive TUI **by default** when invoked without subcommands:

```bash
# Default launch
arjax

# Explicit launch via flag or subcommand
arjax --tui
arjax tui
arjax search --tui
```

---

## 🔍 Search & Feedback Mechanics

One of the key improvements in Arjax v2 is explicit, real-time feedback during package exploration.

### 1. Dual-Search Operation (Enter vs. Auto-Debounce)

You never have to guess whether you need to click Enter or if Arjax is searching:

* **Typing Feedback**: As you type into the search bar, the bottom status bar immediately acknowledges input:
  ```
  ⌨ Typing: 'zapfast' — Press Enter to search now, or pause for auto-search...
  ```
* **Instant Enter Search**: Pressing `Enter` (`Return`) immediately executes the search without waiting.
* **0.5-Second Auto-Debounce**: If you pause typing for **500 milliseconds**, Arjax automatically fires the search query in the background.

### 2. Active Search Progress

While queries run across declarative recipes, Go engines, and local package managers (APT, Pacman, Flatpak), visual indicators display the active state:
* The status bar updates to: `⏳ Searching for '<query>' across recipes and packages...`
* The search results pane displays: `⏳ Searching for '<query>' across providers...`

### 3. Resolution Results

* **Matches Found**:
  ```
  ✔ Found 3 package(s) for 'zapfast'. Press [i] to Install, [r] to Remove.
  ```
  The first result is automatically selected and its metadata loaded into the details pane.
* **No Matches**:
  ```
  ✖ No packages found for 'query'. Try another search query.
  ```

---

## 🎛️ Dual-Pane Layout & Widgets

```
┌───────────────────────────────────────────────────────────── arjax ─────────────────────────────────────────────────────────────┐
│ 🔍 [ Search Input Bar (Press '/' to focus)                                                                 ]                     │
├──────────────────────────────────────────────────────────────┬──────────────────────────────────────────────────────────────────┤
│ SEARCH RESULTS PANE                                          │ PACKAGE DETAILS & DEPENDENCY TREE                                │
│ Displays packages matching the query across all providers:   │ Shows metadata for the highlighted package:                      │
│                                                              │  • Name & Version                                                │
│ 📦 zapfast (recipe) latest                                   │  • Provider source (APT, Pacman, Flatpak, Recipe)                │
│ 📦 zapfast-bin (pacman)                                      │  • Repository / Upstream                                         │
│ 📦 zapfast-git (pacman)                                      │  • Detailed package description                                  │
│                                                              │                                                                  │
│ (Navigate with ↑ / ↓ arrow keys)                             │ DEPENDENCY & STRATEGY TREE                                       │
│                                                              │ Hierarchical tree showing direct dependencies or                 │
│                                                              │ recipe resolution fallback tiers.                                │
│                                                              │                                                                  │
│                                                              │ ACTION BUTTONS                                                   │
│                                                              │ [ Install (i) ]       [ Remove (r) ]                             │
├──────────────────────────────────────────────────────────────┴──────────────────────────────────────────────────────────────────┤
│ STATUS BAR: Real-time action messages, benchmark timers, and operation outcomes                                                 │
└─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## ⌨️ Keybindings & Controls

All actions are fully accessible via mouse clicks and keyboard shortcuts:

| Key | Binding ID | Action | Description |
| :---: | :--- | :--- | :--- |
| **`/`** | `focus_search` | **Focus Search Bar** | Jump cursor immediately to the search input field. |
| **`i`** | `install_selected` | **Install Package** | Triggers orchestrator installation for the selected package. |
| **`r`** | `remove_selected` | **Remove Package** | Triggers uninstallation and cleanup of desktop shortcuts. |
| **`↑` / `↓`** | `navigate` | **Select Package** | Move highlight up/down; updates details and dependency tree in real time. |
| **`d`** | `focus_deps` | **Focus Tree** | Shifts keyboard navigation focus to the dependency tree. |
| **`h`** | `view_history` | **Transaction History** | Displays chronological log of past installations/removals with execution time in ms. |
| **`o`** | `scan_orphans` | **Orphan Scan** | Analyzes system dependency DAG and displays orphaned packages. |
| **`s`** | `view_stats` | **System Dashboard** | Opens system resource usage, package counts, and health checks. |
| **`u`** | `sync_recipes` | **Sync Recipes** | Fetches and syncs declarative recipes from the central repository. |
| **`q`** | `quit` | **Quit** | Exits the TUI session cleanly. |

---

## ⚡ Wired Action Buttons

In addition to keyboard bindings, the UI provides clickable buttons in the details pane:

### 1. `[ Install (i) ]` (`#btn_install`)
1. Verifies a package is currently selected from the search list.
2. Changes the status bar to: `⏳ Installing <package> via <provider>... Please wait.`
3. Executes installation via `InstallationOrchestrator`:
   * Matches declarative recipe strategies (e.g. AppImage / GitHub release / repository).
   * Downloads, validates, and installs the package.
   * If an AppImage is downloaded, marks it executable in `~/.local/bin/<package>` and generates `~/.local/share/applications/<package>.desktop`.
4. Records the transaction in `TransactionManager`.
5. Updates status bar with outcome: `✔ Successfully installed <package> via <provider>!`.

### 2. `[ Remove (r) ]` (`#btn_remove`)
1. Checks for local AppImage installations and unlinks binary and desktop files.
2. Dispatches uninstallation to the detected system package manager (`apt-get remove -y`, `pacman -Rns --noconfirm`, `flatpak uninstall -y`, etc.).
3. Removes tracking from `installed_apps_manager`.
4. Records removal transaction in `TransactionManager`.
5. Updates status bar: `✔ Successfully removed <package>!`.

