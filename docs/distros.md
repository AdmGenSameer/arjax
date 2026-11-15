# 🐧 Distribution Support Matrix

Arjax automatically identifies your distribution family and tailors package management commands accordingly.

---

## Supported Linux Distributions

| Distribution | Family | Native PM | Secondary / AUR | Universal Formats |
| :--- | :--- | :--- | :--- | :--- |
| **Arch Linux** | `arch` | `pacman` | `paru` / `yay` | Flatpak, Snap, AppImage |
| **Pop!_OS** | `debian` | `apt` | PPA / Vendor | Flatpak, Snap, AppImage |
| **Ubuntu** | `debian` | `apt` | PPA / Vendor | Flatpak, Snap, AppImage |
| **Debian** | `debian` | `apt` | Native Repo | Flatpak, Snap, AppImage |
| **Linux Mint** | `debian` | `apt` | PPA / Vendor | Flatpak, Snap, AppImage |
| **Fedora** | `fedora` | `dnf` | RPM fallback | Flatpak, Snap, AppImage |
| **RHEL / CentOS** | `fedora` | `dnf` | RPM fallback | Flatpak, Snap, AppImage |
| **openSUSE** | `suse` | `zypper` | OBS | Flatpak, Snap, AppImage |

---

## Distro-Specific Behaviors

### 1. Debian / Ubuntu / Pop!_OS
* Uses `apt-cache search` for fast non-privileged package lookups.
* Executes installations using `sudo apt-get install -y <pkg>`.
* When a package is not found in APT (e.g. `zapfast`), Arjax automatically falls back to GitHub releases or AppImages, marks the binary executable in `~/.local/bin/`, and configures a `.desktop` application launcher.

### 2. Arch Linux / Manjaro / EndeavourOS
* Queries `pacman` repositories and AUR RPC API (v5) concurrently.
* Supports trust metrics: votes, popularity, and maintainer details.
* Invokes `paru` or `pacman` with `--noconfirm` for automated installation.

### 3. Fedora / RHEL
* Queries DNF metadata cache with fallback to RPM command utilities.
* Automatically uses `sudo dnf install -y <pkg>`.

### 4. openSUSE Tumbleweed / Leap
* Uses `zypper search` and `sudo zypper install -y <pkg>`.
* Fully supports vendor repository mappings and Flatpak.

