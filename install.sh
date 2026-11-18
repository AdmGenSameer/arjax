#!/bin/bash
set -euo pipefail

APP_NAME="arjax"
REPO_ARCHIVE_URL="https://github.com/AdmGenSameer/archpkg-helper/archive/refs/heads/main.zip"
INSTALL_ROOT="${XDG_DATA_HOME:-$HOME/.local/share}/arjax"
VENV_DIR="$INSTALL_ROOT/venv"
BIN_DIR="${XDG_BIN_HOME:-$HOME/.local/bin}"
DESKTOP_DIR="${XDG_DATA_HOME:-$HOME/.local/share}/applications"
SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
SOURCE_SPEC="$REPO_ARCHIVE_URL"
SUDO=""
UNINSTALL_MODE=false
PURGE_CONFIG=false
INSTALL_DEPS=false
YES_TO_ALL=false

for arg in "$@"; do
    case "$arg" in
        --uninstall|uninstall)
            UNINSTALL_MODE=true
            ;;
        --purge)
            PURGE_CONFIG=true
            ;;
        --install-deps)
            INSTALL_DEPS=true
            ;;
        --yes|-y)
            YES_TO_ALL=true
            INSTALL_DEPS=true
            ;;
    esac
done

# Tiny logging helpers keep the installer output readable.
log() {
    echo "[*] $*"
}

die() {
    echo "[x] $*"
    exit 1
}

detect_distro_family() {
    if [ ! -f /etc/os-release ]; then
        echo "unknown"
        return
    fi

    . /etc/os-release
    case "${ID:-unknown}" in
        arch|manjaro|endeavouros|archcraft|garuda)
            echo "arch"
            ;;
        ubuntu|debian|kali|linuxmint|mint|pop|elementary)
            echo "debian"
            ;;
        fedora|rhel|centos|rocky|alma|nobara|bluefin|silverblue|ultramarine)
            echo "fedora"
            ;;
        opensuse*|suse*|sles*)
            echo "suse"
            ;;
        *)
            echo "unknown"
            ;;
    esac
}

get_os_info() {
    local distro_family
    distro_family="$(detect_distro_family)"
    local os_name="Unknown Linux"
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        os_name="${NAME:-${ID:-Unknown OS}}"
        if [ -n "${VERSION_ID:-}" ]; then
            os_name="$os_name $VERSION_ID"
        fi
    fi
    echo "$os_name"
}

check_python_version() {
    python3 -c "import sys; sys.exit(0 if sys.version_info >= (3, 10) else 1)" >/dev/null 2>&1
}

install_python_system_package() {
    local family
    family="$(detect_distro_family)"
    init_sudo
    case "$family" in
        arch)
            $SUDO pacman -Sy --noconfirm python python-pip
            ;;
        debian)
            DEBIAN_FRONTEND=noninteractive $SUDO apt-get update -y || true
            DEBIAN_FRONTEND=noninteractive $SUDO apt-get install -y python3 python3-venv python3-pip
            ;;
        fedora)
            $SUDO dnf install -y python3 python3-pip python3-virtualenv
            ;;
        suse)
            $SUDO zypper install -y python3 python3-pip python3-virtualenv
            ;;
        *)
            return 1
            ;;
    esac
}

install_git_package() {
    local family="$1"
    init_sudo
    case "$family" in
        arch)
            $SUDO pacman -Sy --noconfirm git
            ;;
        debian)
            DEBIAN_FRONTEND=noninteractive $SUDO apt-get update -y || true
            DEBIAN_FRONTEND=noninteractive $SUDO apt-get install -y git
            ;;
        fedora)
            $SUDO dnf install -y git
            ;;
        suse)
            $SUDO zypper install -y git
            ;;
        *)
            return 1
            ;;
    esac
}

install_go_package() {
    local family="$1"
    init_sudo
    case "$family" in
        arch)
            $SUDO pacman -Sy --noconfirm go
            ;;
        debian)
            DEBIAN_FRONTEND=noninteractive $SUDO apt-get update -y || true
            DEBIAN_FRONTEND=noninteractive $SUDO apt-get install -y golang-go || DEBIAN_FRONTEND=noninteractive $SUDO apt-get install -y golang
            ;;
        fedora)
            $SUDO dnf install -y golang
            ;;
        suse)
            $SUDO zypper install -y go
            ;;
        *)
            return 1
            ;;
    esac
}

prompt_yes_no() {
    local prompt_text="$1"
    local default_val="${2:-Y}"

    if [ "$YES_TO_ALL" = true ]; then
        return 0
    fi

    local choice=""
    if [ -c /dev/tty ]; then
        read -r -p "$prompt_text " choice < /dev/tty || choice=""
    elif [ -t 0 ]; then
        read -r -p "$prompt_text " choice || choice=""
    else
        case "$default_val" in
            [yY]*) return 0 ;;
            *) return 1 ;;
        esac
    fi

    choice="${choice:-$default_val}"
    case "$choice" in
        [yY]|[yY][eE][sS]) return 0 ;;
        *) return 1 ;;
    esac
}

run_sudo_auth() {
    local dep_name="$1"
    init_sudo
    if [ -z "$SUDO" ]; then
        return 0
    fi

    echo ""
    log "Authentication required to install $dep_name."
    if [ -c /dev/tty ]; then
        if ! $SUDO -v < /dev/tty; then
            log "Authentication failed or was cancelled."
            return 1
        fi
    elif [ -t 0 ]; then
        if ! $SUDO -v; then
            log "Authentication failed or was cancelled."
            return 1
        fi
    else
        if ! $SUDO -n -v 2>/dev/null; then
            log "Authentication required but non-interactive sudo is unavailable."
            return 1
        fi
    fi
    return 0
}

show_spinner() {
    local pid=$1
    local message="$2"
    local delay=0.1
    local spin=('⠋' '⠙' '⠹' '⠸' '⠼' '⠴' '⠦' '⠧' '⠇' '⠏')
    # Hide cursor
    tput civis 2>/dev/null || true
    while kill -0 "$pid" 2>/dev/null; do
        for c in "${spin[@]}"; do
            if ! kill -0 "$pid" 2>/dev/null; then
                break
            fi
            printf "\r\033[K %s %s" "$c" "$message"
            sleep $delay
        done
    done
    # Restore cursor and clear line
    tput cnorm 2>/dev/null || true
    printf "\r\033[K"
}

request_dep_install() {
    local dep_name="$1"
    local install_cmds="$2"
    local install_action="$3"
    local extra_msg="$4"

    local os_name
    os_name="$(get_os_info)"

    local do_install=false
    if [ "$INSTALL_DEPS" = true ] || [ "$YES_TO_ALL" = true ]; then
        do_install=true
    else
        echo ""
        echo "$dep_name is required but is not installed."
        echo "Detected OS: $os_name"
        echo ""
        if prompt_yes_no "Install $dep_name automatically? [Y/n]:" "Y"; then
            do_install=true
        else
            do_install=false
        fi
    fi

    if [ "$do_install" = true ]; then
        if run_sudo_auth "$dep_name"; then
            local log_file
            log_file="$(mktemp /tmp/arjax-install-XXXXXX.log 2>/dev/null || mktemp)"

            # Run action in background, isolated from stdin
            eval "$install_action" < /dev/null > "$log_file" 2>&1 &
            local action_pid=$!

            # Show spinner while running
            show_spinner "$action_pid" "Installing $dep_name..."

            wait "$action_pid"
            local exit_code=$?

            if [ "$exit_code" -eq 0 ]; then
                log "$dep_name installed successfully"
                rm -f "$log_file"
                return 0
            else
                log "Failed to install $dep_name automatically"
                if [ -s "$log_file" ]; then
                    echo "Installation output:"
                    tail -n 10 "$log_file" | sed 's/^/    /'
                fi
                rm -f "$log_file"
            fi
        fi
    fi

    # If we didn't install or installation failed:
    echo ""
    echo "[x] $dep_name was not found."
    echo ""
    if [ -n "$extra_msg" ]; then
        echo -e "$extra_msg"
        echo ""
    fi
    echo "Detected OS: $os_name"
    echo ""
    echo "You can install it manually with:"
    echo -e "    $install_cmds"
    echo ""
    echo "Or rerun this installer with --install-deps to install missing dependencies automatically."
    exit 1
}

# Use sudo only when we are not already root.
init_sudo() {
    if [ "$(id -u)" -eq 0 ]; then
        SUDO=""
    else
        SUDO="sudo"
    fi
}

# Install Python packaging support using the system package manager when ensurepip is not available.
install_python_support() {
    local distro_family="$1"

    log "Attempting to install Python packaging support for $distro_family..."
    init_sudo
    case "$distro_family" in
        arch)
            $SUDO pacman -Sy --noconfirm python python-pip
            ;;
        debian)
            DEBIAN_FRONTEND=noninteractive $SUDO apt-get update -y || true
            DEBIAN_FRONTEND=noninteractive $SUDO apt-get install -y python3-venv python3-pip
            ;;
        fedora)
            $SUDO dnf install -y python3-pip python3-virtualenv
            ;;
        suse)
            $SUDO zypper install -y python3-pip python3-virtualenv
            ;;
        *)
            return 1
            ;;
    esac

    return 0
}

# Make sure git exists because users may want to clone, update, or inspect the repo locally.
ensure_git() {
    if command -v git >/dev/null 2>&1; then
        log "git is already installed"
        return 0
    fi

    local distro_family
    distro_family="$(detect_distro_family)"
    
    local display_cmds=""
    case "$distro_family" in
        arch) display_cmds="sudo pacman -S git" ;;
        debian) display_cmds="sudo apt update\n    sudo apt install -y git" ;;
        fedora) display_cmds="sudo dnf install -y git" ;;
        suse) display_cmds="sudo zypper install -y git" ;;
        *) display_cmds="Install git using your package manager" ;;
    esac

    request_dep_install "git" "$display_cmds" "install_git_package $distro_family" "Arjax requires git to download/update recipes."
}

ensure_go() {
    if command -v go >/dev/null 2>&1; then
        log "Go toolchain is already installed ($(go version | awk '{print $3}'))"
        return 0
    fi

    local distro_family
    distro_family="$(detect_distro_family)"
    
    local display_cmds=""
    case "$distro_family" in
        arch) display_cmds="sudo pacman -S go" ;;
        debian) display_cmds="sudo apt update\n    sudo apt install -y golang-go" ;;
        fedora) display_cmds="sudo dnf install -y golang" ;;
        suse) display_cmds="sudo zypper install -y go" ;;
        *) display_cmds="Install Go using your package manager" ;;
    esac

    log "Go compiler was not detected on this system."
    log "Arjax includes an optional Go Provider Engine binary for ultra-fast parallel searches and TUI."
    log "Note: Go is OPTIONAL. If skipped, Arjax runs smoothly using built-in Python providers."
    echo ""

    local should_install=false
    if [ "$INSTALL_DEPS" = true ]; then
        should_install=true
    elif [ "$YES_TO_ALL" = true ]; then
        should_install=false
    else
        if prompt_yes_no "Install Go compiler automatically to compile provider engine? [y/N]:" "N"; then
            should_install=true
        else
            log "Skipping Go installation. Arjax will use Python fallback providers."
            return 0
        fi
    fi

    if [ "$should_install" = true ]; then
        if run_sudo_auth "Go compiler"; then
            local log_file
            log_file="$(mktemp /tmp/arjax-go-install-XXXXXX.log 2>/dev/null || mktemp)"
            eval "install_go_package $distro_family" < /dev/null > "$log_file" 2>&1 &
            local action_pid=$!
            show_spinner "$action_pid" "Installing Go compiler..."
            wait "$action_pid"
            local exit_code=$?
            if [ "$exit_code" -eq 0 ] && command -v go >/dev/null 2>&1; then
                log "Go compiler installed successfully ($(go version | awk '{print $3}'))"
                rm -f "$log_file"
                return 0
            else
                log "Notice: Go compiler installation did not complete."
                if [ -s "$log_file" ]; then
                    echo "Installer output:"
                    tail -n 10 "$log_file" | sed 's/^/    /'
                fi
                rm -f "$log_file"
            fi
        else
            log "Notice: Sudo authentication not completed. Skipping Go installation."
        fi
        log "Arjax will continue installation and use high-performance Python providers."
    else
        log "Skipping Go installation. Arjax will use Python fallback providers."
    fi
    return 0
}

ensure_python() {
    local family
    family="$(detect_distro_family)"
    local display_cmds=""
    case "$family" in
        arch) display_cmds="sudo pacman -Sy python python-pip" ;;
        debian) display_cmds="sudo apt update\n    sudo apt install -y python3 python3-venv python3-pip" ;;
        fedora) display_cmds="sudo dnf install -y python3 python3-pip python3-virtualenv" ;;
        suse) display_cmds="sudo zypper install -y python3 python3-pip python3-virtualenv" ;;
        *) display_cmds="Install Python 3 using your package manager" ;;
    esac

    if command -v python3 >/dev/null 2>&1; then
        if ! check_python_version; then
            local current_ver
            current_ver=$(python3 -c "import sys; print(f'{sys.version_info.major}.{sys.version_info.minor}.{sys.version_info.micro}')" 2>/dev/null || echo "unknown")
            local os_name
            os_name="$(get_os_info)"
            echo ""
            echo "[x] Python version is too old: $current_ver"
            echo ""
            echo "Arjax requires Python 3.10 or newer."
            echo ""
            echo "Detected OS: $os_name"
            echo ""
            echo "You can update Python with:"
            echo -e "    $display_cmds"
            exit 1
        fi
    else
        request_dep_install "Python 3" "$display_cmds" "install_python_system_package" "Arjax requires Python 3.10 or newer."
    fi

    # Check if python3-venv is available to create isolated environments
    if ! python3 -m venv --help >/dev/null 2>&1; then
        local venv_cmds=""
        case "$family" in
            arch) venv_cmds="sudo pacman -Sy python" ;;
            debian) venv_cmds="sudo apt update\n    sudo apt install -y python3-venv python3-pip" ;;
            fedora) venv_cmds="sudo dnf install -y python3-virtualenv" ;;
            suse) venv_cmds="sudo zypper install -y python3-virtualenv" ;;
            *) venv_cmds="Install python3-venv using your package manager" ;;
        esac
        request_dep_install "Python venv support" "$venv_cmds" "install_python_support $family" "Arjax requires venv to create an isolated environment."
    fi
}

ensure_source() {
    if [ -f "$SCRIPT_DIR/pyproject.toml" ]; then
        SOURCE_SPEC="$SCRIPT_DIR"
        log "Using local checkout at $SOURCE_SPEC"
    else
        # Fallback for web-based installs where the repo is not present locally.
        log "Using remote archive source"
    fi
}

# Create an isolated Python environment so the install does not affect system packages.
create_venv() {
    mkdir -p "$INSTALL_ROOT"
    if [ ! -x "$VENV_DIR/bin/python" ]; then
        log "Creating virtual environment in $VENV_DIR"
        if ! python3 -m venv "$VENV_DIR" 2>/dev/null; then
            log "Standard venv creation failed; attempting minimal venv creation..."
            python3 -m venv --without-pip "$VENV_DIR" || die "Could not create virtual environment in $VENV_DIR."
        fi
    fi

    if [ ! -x "$VENV_DIR/bin/python" ]; then
        die "Could not find python binary in virtual environment: $VENV_DIR"
    fi

    # Ensure pip is available inside the virtual environment
    if [ ! -x "$VENV_DIR/bin/pip" ]; then
        log "Bootstrapping pip inside the virtual environment..."
        "$VENV_DIR/bin/python" -m ensurepip --upgrade >/dev/null 2>&1 || true
    fi

    if [ ! -x "$VENV_DIR/bin/pip" ]; then
        log "Bootstrapping pip using standalone get-pip..."
        if command -v curl >/dev/null 2>&1; then
            curl -fsSL https://bootstrap.pypa.io/get-pip.py | "$VENV_DIR/bin/python" - >/dev/null 2>&1 || true
        elif command -v wget >/dev/null 2>&1; then
            wget -qO- https://bootstrap.pypa.io/get-pip.py | "$VENV_DIR/bin/python" - >/dev/null 2>&1 || true
        fi
    fi

    if [ ! -x "$VENV_DIR/bin/pip" ]; then
        die "Could not prepare pip inside the virtual environment. Install python3-pip and python3-venv, then rerun the installer."
    fi
}

# Install Arjax and the GUI dependency into the private venv.
install_package() {
    local pip_bin="$VENV_DIR/bin/pip"

    if [ ! -x "$pip_bin" ]; then
        die "pip is unavailable inside the virtual environment."
    fi

    log "Upgrading packaging tools..."
    "$pip_bin" install --upgrade pip setuptools wheel >/dev/null 2>&1 || true

    log "Installing Arjax from $SOURCE_SPEC..."
    if ! "$pip_bin" install --upgrade "$SOURCE_SPEC"; then
        local fallback_url="https://github.com/AdmGenSameer/arjax/archive/refs/heads/main.zip"
        if [ "$SOURCE_SPEC" != "$fallback_url" ]; then
            log "Retrying installation with fallback archive URL..."
            "$pip_bin" install --upgrade "$fallback_url"
        else
            die "Failed to install Arjax from $SOURCE_SPEC."
        fi
    fi

    log "Installing GUI dependency (PyQt5)..."
    "$pip_bin" install --upgrade PyQt5 >/dev/null 2>&1 || log "Notice: PyQt5 could not be installed automatically. Arjax CLI and TUI are fully functional; run 'arjax gui-install' to install GUI support later."
}

build_go_engine() {
    local engine_dir="$SCRIPT_DIR/arjax-provider-engine"
    if [ ! -d "$engine_dir" ]; then
        if [ -d "$INSTALL_ROOT/arjax-provider-engine" ]; then
            engine_dir="$INSTALL_ROOT/arjax-provider-engine"
        fi
    fi

    local tmp_engine=""
    if [ ! -d "$engine_dir" ] && command -v go >/dev/null 2>&1; then
        tmp_engine="$(mktemp -d /tmp/arjax-engine-XXXXXX 2>/dev/null || mktemp -d)"
        local zip_file="$tmp_engine/repo.zip"
        if command -v curl >/dev/null 2>&1; then
            curl -fsSL "$REPO_ARCHIVE_URL" -o "$zip_file" 2>/dev/null || true
        elif command -v wget >/dev/null 2>&1; then
            wget -qO "$zip_file" "$REPO_ARCHIVE_URL" 2>/dev/null || true
        fi
        if [ -f "$zip_file" ] && command -v python3 >/dev/null 2>&1; then
            python3 -c "import zipfile; zipfile.ZipFile('$zip_file').extractall('$tmp_engine')" 2>/dev/null || true
            local extracted_dir
            extracted_dir="$(find "$tmp_engine" -maxdepth 2 -type d -name "arjax-provider-engine" 2>/dev/null | head -n 1)"
            if [ -n "$extracted_dir" ] && [ -d "$extracted_dir" ]; then
                engine_dir="$extracted_dir"
            fi
        fi
    fi

    if [ -d "$engine_dir" ] && command -v go >/dev/null 2>&1; then
        log "Compiling Arjax Go Provider Engine..."
        (
            cd "$engine_dir"
            go build -o "$BIN_DIR/arjax-provider" main.go || true
        )
        if [ -x "$BIN_DIR/arjax-provider" ]; then
            log "Arjax Go Provider Engine successfully compiled and installed to $BIN_DIR/arjax-provider"
        else
            log "Notice: Go compilation was not completed. Arjax will seamlessly run using built-in Python providers."
        fi
    else
        log "Go compiler or engine source not present. Arjax will use Python fallback provider system."
    fi

    if [ -n "$tmp_engine" ] && [ -d "$tmp_engine" ]; then
        rm -rf "$tmp_engine"
    fi
}

# Add a small launcher so the app is easy to start from the terminal.
create_launcher() {
    mkdir -p "$BIN_DIR"

    cat > "$BIN_DIR/$APP_NAME" <<EOF
#!/bin/bash
exec "$VENV_DIR/bin/arjax" "\$@"
EOF

    chmod +x "$BIN_DIR/$APP_NAME"

    case ":${PATH:-}:" in
        *:"$BIN_DIR":*) ;;
        *) export PATH="$BIN_DIR:$PATH" ;;
    esac
}

remove_shell_path_block() {
    local shell_file="$1"
    local block_file
    block_file="$(mktemp)"

    python3 - "$shell_file" "$block_file" <<'PY'
from pathlib import Path
import sys

shell_file = Path(sys.argv[1])
block_file = Path(sys.argv[2])
block1 = "\n# arjax-helper: make user-local commands available\nif [ -d \"$HOME/.local/bin\" ]; then\n    export PATH=\"$HOME/.local/bin:$PATH\"\nfi\n"
block2 = "\n# archpkg-helper: make user-local commands available\nif [ -d \"$HOME/.local/bin\" ]; then\n    export PATH=\"$HOME/.local/bin:$PATH\"\nfi\n"

if shell_file.exists():
    content = shell_file.read_text(encoding='utf-8')
    content = content.replace(block1, "\n").replace(block2, "\n")
    shell_file.write_text(content, encoding='utf-8')
PY

    rm -f "$block_file"
}

uninstall_installation() {
    log "Removing Arjax installation..."
    rm -f "$BIN_DIR/$APP_NAME" "$DESKTOP_DIR/arjax.desktop" "$DESKTOP_DIR/archpkg-helper.desktop"
    rm -rf "$INSTALL_ROOT"

    remove_shell_path_block "$HOME/.profile" || true
    remove_shell_path_block "$HOME/.bashrc" || true
    remove_shell_path_block "$HOME/.zshrc" || true

    if [ "$PURGE_CONFIG" = true ]; then
        rm -rf "$HOME/.arjax" "$HOME/.archpkg"
    fi

    log "Arjax uninstall complete."
    echo ""
    echo "If the app still appears in your desktop menu, log out and back in or refresh the application cache."
}

# Persist the launcher directory in common shell startup files so `arjax` is available in new shells.
ensure_shell_path() {
    local path_line='export PATH="$HOME/.local/bin:$PATH"'
    local shell_files=("$HOME/.profile" "$HOME/.bashrc" "$HOME/.zshrc")

    for shell_file in "${shell_files[@]}"; do
        if [ -e "$shell_file" ] || [ "$shell_file" = "$HOME/.profile" ]; then
            if ! grep -Fq "$path_line" "$shell_file" 2>/dev/null; then
                {
                    echo ""
                    echo "# arjax-helper: make user-local commands available"
                    echo "if [ -d \"\$HOME/.local/bin\" ]; then"
                    echo "    $path_line"
                    echo "fi"
                } >> "$shell_file"
                log "Updated $(basename "$shell_file") to include ~/.local/bin"
            fi
        fi
    done
}

# Only register a desktop app when the machine appears to have a graphical desktop.
has_graphical_environment() {
    if [ -n "${DISPLAY:-}" ] || [ -n "${WAYLAND_DISPLAY:-}" ]; then
        return 0
    fi

    if [ -n "${XDG_CURRENT_DESKTOP:-}" ] || [ -n "${DESKTOP_SESSION:-}" ]; then
        return 0
    fi

    if [ -d /usr/share/xsessions ] && [ -n "$(ls -A /usr/share/xsessions 2>/dev/null)" ]; then
        return 0
    fi

    if [ -d /usr/share/wayland-sessions ] && [ -n "$(ls -A /usr/share/wayland-sessions 2>/dev/null)" ]; then
        return 0
    fi

    if command -v systemctl >/dev/null 2>&1; then
        if [ "$(systemctl get-default 2>/dev/null || true)" = "graphical.target" ]; then
            return 0
        fi
    fi

    return 1
}

# Register the GUI in the desktop menu when a graphical session or desktop stack is present.
install_desktop_entry() {
    if ! has_graphical_environment; then
        log "No graphical desktop detected, skipping application menu integration"
        return 0
    fi

    mkdir -p "$DESKTOP_DIR"

    cat > "$DESKTOP_DIR/arjax-helper.desktop" <<EOF
[Desktop Entry]
Version=1.0
Type=Application
Name=Arjax
GenericName=Package Manager
Comment=Cross-distribution package manager with GUI
Exec=$BIN_DIR/$APP_NAME gui
TryExec=$BIN_DIR/$APP_NAME
Icon=system-software-install
Terminal=false
Categories=System;PackageManager;Settings;
Keywords=package;install;update;software;pacman;apt;dnf;aur;flatpak;snap;
StartupNotify=true
StartupWMClass=Arjax
EOF

    chmod +x "$DESKTOP_DIR/arjax-helper.desktop"

    if command -v update-desktop-database >/dev/null 2>&1; then
        update-desktop-database "$DESKTOP_DIR" >/dev/null 2>&1 || true
    fi

    log "Desktop launcher installed and should appear in the application menu"
}

install_man_page() {
    log "Installing man page..."
    local MAN_DIR="$HOME/.local/share/man/man1"
    mkdir -p "$MAN_DIR"
    if [ -f "$SCRIPT_DIR/data/man/man1/arjax.1" ]; then
        cp "$SCRIPT_DIR/data/man/man1/arjax.1" "$MAN_DIR/arjax.1"
        log "Man page installed to $MAN_DIR/arjax.1 (view with: man arjax)"
    fi
}

# Preserve the Arch-only profile setup, but keep it out of the universal path for other distros.
configure_arch_profile() {
    local distro_family="$1"
    if [ "$distro_family" != "arch" ]; then
        return
    fi

    echo ""
    log "Choose your Arjax profile:"
    echo "    1) normal (recommended) - Arjax handles updates/news/trust checks automatically"
    echo "    2) advanced - full manual control"
    
    local ARJAX_PROFILE_CHOICE="1"
    if [ "$YES_TO_ALL" = true ]; then
        log "Non-interactive mode: selecting normal profile"
    else
        if [ -c /dev/tty ]; then
            read -r -p "Enter choice [1/2] (default: 1): " ARJAX_PROFILE_CHOICE < /dev/tty || ARJAX_PROFILE_CHOICE="1"
        elif [ -t 0 ]; then
            read -r -p "Enter choice [1/2] (default: 1): " ARJAX_PROFILE_CHOICE || ARJAX_PROFILE_CHOICE="1"
        else
            ARJAX_PROFILE_CHOICE="1"
        fi
    fi

    local arjax_user_mode="normal"
    if [ "${ARJAX_PROFILE_CHOICE:-1}" = "2" ]; then
        arjax_user_mode="advanced"
    fi

    mkdir -p "$HOME/.arjax"
    if [ "$arjax_user_mode" = "advanced" ]; then
        cat > "$HOME/.arjax/config.json" <<EOF
{
  "user_mode": "advanced",
  "default_interface": "tui",
  "auto_update_enabled": false,
  "auto_update_mode": "manual",
  "update_check_interval_hours": 24,
  "background_download_enabled": true,
  "notification_enabled": true,
  "auto_handle_arch_news": false,
  "auto_review_aur_trust": false,
  "auto_snapshot_before_update": false,
  "proactive_system_advice": false
}
EOF
    else
        cat > "$HOME/.arjax/config.json" <<EOF
{
  "user_mode": "normal",
  "default_interface": "tui",
  "auto_update_enabled": true,
  "auto_update_mode": "automatic",
  "update_check_interval_hours": 24,
  "background_download_enabled": true,
  "notification_enabled": true,
  "auto_handle_arch_news": true,
  "auto_review_aur_trust": true,
  "auto_snapshot_before_update": true,
  "proactive_system_advice": true
}
EOF
    fi

    log "Profile configured: $arjax_user_mode"
}
main() {
    if [ "$UNINSTALL_MODE" = true ]; then
        uninstall_installation
        return 0
    fi

    log "Starting universal installation of Arjax..."

    ensure_python
    ensure_git
    ensure_go
    ensure_source
    create_venv
    install_package
    build_go_engine
    create_launcher
    ensure_shell_path
    install_desktop_entry
    install_man_page
    configure_arch_profile "$(detect_distro_family)"

    log "Arjax installation complete!"
    echo ""
    echo "Quick start:"
    echo "    arjax               # Launch interactive TUI (Default)"
    echo "    arjax search <pkg>  # Search for packages"
    echo "    arjax install <pkg> # Install packages"
    echo "    arjax gui           # Launch native desktop GUI"
    echo "    arjax --help        # Show all CLI commands"
    echo ""
    echo "Your shell startup files were updated so ~/.local/bin is available in new sessions."
    echo ""
    if has_graphical_environment; then
        echo "The GUI is also available in your application menu."
    else
        echo "CLI-only environment detected, so no application menu entry was created."
    fi
}

main "$@"

