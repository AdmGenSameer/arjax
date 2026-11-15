# 📦 Installation Guide

This document details all supported methods for installing, setting up, and verifying Arjax on Linux systems.

---

## Prerequisites

* **Operating System**: Any modern Linux distribution (Kernel 4.18+)
  * Fully tested on: Arch Linux, Pop!_OS 22.04, Ubuntu 22.04/24.04, Debian 12, Fedora 39+, openSUSE Tumbleweed
* **Python**: Python 3.10 or higher
* **Package Managers (optional)**:
  * Arch: `pacman`, `paru` / `yay`
  * Debian/Ubuntu/Pop!_OS: `apt-get`
  * Fedora: `dnf`
  * openSUSE: `zypper`
  * Universal: `flatpak`, `snap`

---

## Method 1: Automated Installer (Recommended)

The automated script configures everything required without altering your system Python:

```bash
curl -fsSL https://raw.githubusercontent.com/AdmGenSameer/archpkg-helper/main/install.sh | bash
```

### What `install.sh` Does:
1. Detects your distribution and verifies Python 3.10+.
2. Creates an isolated environment at `~/.local/share/arjax/venv`.
3. Installs Arjax and all required dependencies (`textual`, `rich`, `typer`, `pyyaml`, `platformdirs`, etc.).
4. Places the executable wrapper at `~/.local/bin/arjax`.
5. Installs the desktop application launcher at `~/.local/share/applications/arjax.desktop`.

---

## Method 2: Manual Git & Virtual Environment

```bash
# 1. Clone the repository
git clone https://github.com/AdmGenSameer/archpkg-helper.git
cd archpkg-helper

# 2. Create and activate a virtual environment
python3 -m venv .venv
source .venv/bin/activate

# 3. Upgrade pip and install package
pip install --upgrade pip
pip install -e .

# 4. Optional PyQt5 GUI support
pip install PyQt5
```

---

## Method 3: Using `uv` (Fastest Python Tooling)

If you use [astral-sh/uv](https://github.com/astral-sh/uv):

```bash
git clone https://github.com/AdmGenSameer/archpkg-helper.git
cd archpkg-helper

# Run directly via uv
uv run arjax

# Or install tool globally
uv tool install .
```

---

## Testing in Docker (Pop!_OS / Ubuntu / Arch)

You can verify Arjax inside an isolated container:

```bash
# Pull and run in Pop!_OS / Ubuntu
docker run -it --rm ubuntu:22.04 bash -c "
  apt-get update && apt-get install -y git python3 python3-pip curl &&
  git clone https://github.com/AdmGenSameer/archpkg-helper.git /opt/arjax &&
  cd /opt/arjax && bash install.sh &&
  ~/.local/bin/arjax stats
"
```

---

## Uninstallation

To completely remove Arjax and its desktop entries:

```bash
arjax uninstall --purge
```

