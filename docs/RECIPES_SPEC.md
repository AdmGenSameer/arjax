# 📖 Arjax Declarative Recipe Specification

Arjax allows anyone to add support for Linux applications without writing Python or Go code. This is achieved through **Declarative YAML Recipes**.

---

## 🎯 Design Principles

1. **Pure Data, Zero Code**: Recipes are static YAML files. They declare what providers are capable of supplying the software and any format-specific metadata.
2. **Deterministic Priority**: The installation engine respects provider priority:
   `Repository` $\rightarrow$ `Vendor` $\rightarrow$ `GitHub Release` $\rightarrow$ `Flatpak` $\rightarrow$ `Snap` $\rightarrow$ `AppImage`.
3. **Cross-Distro Portability**: A single recipe file can service Debian, Ubuntu, Pop!_OS, Arch Linux, Fedora, and openSUSE simultaneously.

---

## 📁 Recipe Locations

Arjax loads recipes from two directories:
1. **Core Recipes**: Distributed with Arjax under `arjax/recipes/*.yaml`.
2. **User Recipes**: Custom user overrides stored in `~/.config/arjax/recipes/*.yaml` (or `$XDG_CONFIG_HOME/arjax/recipes/`).

---

## 📝 Schema Reference

```yaml
# Top-Level Metadata
name: <string>              # Required: Unique canonical identifier (lowercase)
display_name: <string>      # Optional: Human-readable display title
description: <string>       # Optional: Short summary of what the software does
aliases:                    # Optional: Alternative search terms and keywords
  - <string>

# Provider Configurations
providers:
  repository:               # Native distribution package managers
    package: <string>       # Name of the package in the repository
    package_manager: <str>  # Optional override (e.g. pacman, paru, apt)

  vendor:                   # Third-party vendor repositories
    package: <string>
    package_manager: <str>
    repo_url: <string>      # Optional custom repository URL
    key_url: <string>       # Optional GPG key URL

  github:                   # Automated GitHub release resolution
    github_repo: <owner/repo> # GitHub repository slug (e.g. crmne/zapfast)
    install_type: <string>  # Optional: 'appimage', 'deb', 'binary', or 'source'
    download_url: <string>  # Optional: Direct asset URL override

  flatpak:                  # Flatpak package specification
    flatpak_id: <string>    # Flathub reverse-DNS ID (e.g. org.signal.Signal)

  snap:                     # Canonical Snap specification
    snap_id: <string>       # Snap Store identifier (e.g. signal-desktop)

  appimage:                 # Standalone AppImage download
    appimage_url: <url>     # Direct URL to download the .AppImage binary
    appimage_name: <string> # Binary name to install into ~/.local/bin/
```

---

## 💡 Practical Examples

### 1. Multi-Provider Fallback: WhatsApp Client (`zapfast.yaml`)

```yaml
name: zapfast
display_name: ZapFast
description: Fast native WhatsApp client built with Rust and egui
aliases:
  - zapfast
  - whatsapp

providers:
  repository:
    package: zapfast
    package_manager: paru
  github:
    github_repo: crmne/zapfast
  appimage:
    appimage_url: https://github.com/crmne/zapfast/releases/download/v0.21.1/zapfast-0.21.1-x86_64.AppImage
    appimage_name: zapfast
```

* **Behavior on Arch**: Installs via `paru` / AUR repository.
* **Behavior on Pop!_OS / Ubuntu / Fedora**: Since `zapfast` is not in APT/DNF, the engine falls back to `appimage`, downloads the binary, sets executable permissions in `~/.local/bin/zapfast`, and registers `zapfast.desktop`.

---

### 2. Browser with Vendor & Flatpak Support (`brave.yaml`)

```yaml
name: brave
display_name: Brave Browser
description: Privacy-focused web browser

providers:
  repository:
    package: brave-bin
    package_manager: pacman
  vendor:
    package: brave-browser
    package_manager: apt
  flatpak:
    flatpak_id: com.brave.Browser
  github:
    github_repo: brave/brave-browser
    github_install_type: source
```

---

### 3. Container Tooling (`docker.yaml`)

```yaml
name: docker
display_name: Docker
description: Pack, ship and run applications as containers

providers:
  repository:
    package: docker
```

---

## 🚀 How to Contribute a Recipe

1. Create your recipe file: `arjax/recipes/<my-app>.yaml`.
2. Validate recipe mapping:
   ```bash
   uv run python scripts/validate_mapping.py
   ```
3. Test installation:
   ```bash
   arjax install <my-app>
   ```
4. Verify tests pass:
   ```bash
   uv run --with pytest-asyncio pytest
   ```
5. Submit a Pull Request on GitHub.

