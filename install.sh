#!/bin/bash
set -euEo pipefail

APP_NAME="arjax"
REPO_ARCHIVE_URL="https://github.com/AdmGenSameer/archpkg-helper/archive/refs/heads/main.zip"
FALLBACK_ARCHIVE_URL="https://github.com/AdmGenSameer/arjax/archive/refs/heads/main.zip"
INSTALL_ROOT="${XDG_DATA_HOME:-$HOME/.local/share}/arjax"
VENV_DIR="$INSTALL_ROOT/venv"
BIN_DIR="${XDG_BIN_HOME:-$HOME/.local/bin}"
GLOBAL_BIN_DIR="/usr/local/bin"
DESKTOP_DIR="${XDG_DATA_HOME:-$HOME/.local/share}/applications"
SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
INITIAL_PATH="${PATH:-}"
SOURCE_SPEC="$REPO_ARCHIVE_URL"
SOURCE_DIR=""
ARCHIVE_TMP_DIR=""
SUDO=""
UNINSTALL_MODE=false
PURGE_CONFIG=false
INSTALL_DEPS=false
YES_TO_ALL=false
WITH_GUI=false
UPGRADE_MODE=false
LOG_FILE="${TMPDIR:-/tmp}/arjax-install-${UID:-$(id -u 2>/dev/null || echo 0)}.log"
touch "$LOG_FILE" 2>/dev/null || LOG_FILE="$(mktemp /tmp/arjax-install-XXXXXX.log 2>/dev/null || echo "/tmp/arjax-install.log")"
INSTALL_SUCCESS=false
ERROR_REPORTED=false
FAILED_LINE=""
FAILED_CMD=""

for arg in "$@"; do
    case "$arg" in
        --uninstall|uninstall)
            UNINSTALL_MODE=true
            ;;
        --upgrade|upgrade)
            UPGRADE_MODE=true
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
        --with-gui)
            WITH_GUI=true
            ;;
        --log-file=*)
            LOG_FILE="${arg#*=}"
            ;;
    esac
done

# Initialize log file header
{
    echo "================================================================="
    echo "Arjax Installation Log"
    echo "Started at: $(date)"
    echo "System:     $(uname -s -m -r 2>/dev/null || uname -a)"
    echo "User:       $(whoami 2>/dev/null || id -un 2>/dev/null || echo "uid $UID")"
    echo "Arguments:  ${*:-none}"
    echo "================================================================="
} >> "$LOG_FILE" 2>/dev/null || true

trap 'FAILED_LINE="$LINENO"; FAILED_CMD="$BASH_COMMAND"' ERR

cleanup_tmp() {
    if [ -n "$ARCHIVE_TMP_DIR" ] && [ -d "$ARCHIVE_TMP_DIR" ]; then
        rm -rf "$ARCHIVE_TMP_DIR" 2>/dev/null || true
    fi
}

on_exit() {
    local exit_code="$1"
    tput cnorm 2>/dev/null || true
    cleanup_tmp
    if [ "$INSTALL_SUCCESS" = true ] || [ "$exit_code" -eq 0 ] || [ "$UNINSTALL_MODE" = true ]; then
        return
    fi
    if [ "$ERROR_REPORTED" = true ]; then
        return
    fi
    ERROR_REPORTED=true

    echo "" >&2
    echo "=================================================================" >&2
    echo "❌ Arjax Installation Failed!" >&2
    echo "=================================================================" >&2
    echo "Failure details:" >&2
    echo "  • Exit code:    $exit_code" >&2
    if [ -n "$FAILED_LINE" ]; then
        echo "  • Failed line:  Line $FAILED_LINE" >&2
        echo "  • Command:      $FAILED_CMD" >&2
    fi
    echo "  • Log file:     $LOG_FILE" >&2
    echo "=================================================================" >&2
    if [ -f "$LOG_FILE" ] && [ -s "$LOG_FILE" ]; then
        echo "Recent installer log output (last 15 lines):" >&2
        echo "-----------------------------------------------------------------" >&2
        tail -n 15 "$LOG_FILE" | sed 's/^/  /' >&2
        echo "-----------------------------------------------------------------" >&2
    fi
    echo "For assistance or debugging, inspect the full log at:" >&2
    echo "  $LOG_FILE" >&2
    echo "=================================================================" >&2
}
trap 'on_exit "$?"' EXIT

# Tiny logging helpers keep the installer output readable and persisted to disk.
log() {
    echo "[*] $*"
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [*] $*" >> "$LOG_FILE" 2>/dev/null || true
}

log_error() {
    echo "[x] $*" >&2
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [ERROR] $*" >> "$LOG_FILE" 2>/dev/null || true
}

die() {
    log_error "$*"
    exit 1
}

# Render a deterministic progress bar instead of an infinite spinner.
render_progress_bar() {
    local pct="$1"
    local label="$2"
    local width=24

    [ "$pct" -lt 0 ] 2>/dev/null && pct=0
    [ "$pct" -gt 100 ] 2>/dev/null && pct=100

    local filled=$(( pct * width / 100 ))
    local empty=$(( width - filled ))
    local bar_fill=""
    local bar_empty=""

    if [ "$filled" -gt 0 ]; then
        bar_fill="$(printf '%*s' "$filled" '' | tr ' ' '#')"
    fi
    if [ "$empty" -gt 0 ]; then
        bar_empty="$(printf '%*s' "$empty" '' | tr ' ' '-')"
    fi

    # Truncate overly long labels so the bar never wraps on narrow terminals.
    if [ "${#label}" -gt 52 ]; then
        label="${label:0:49}..."
    fi

    if [ -t 1 ]; then
        printf "\r\033[K  [%s%s] %3d%%  %s" "$bar_fill" "$bar_empty" "$pct" "$label"
    fi
}

# Execute a background command while tracking real progress from its output log.
run_with_progress() {
    local label="$1"
    local mode="$2"
    local cmd_str="$3"
    local task_log
    task_log="$(mktemp /tmp/arjax-task-XXXXXX.log 2>/dev/null || mktemp)"

    eval "$cmd_str" < /dev/null > "$task_log" 2>&1 &
    local pid=$!

    local pct=5
    local last_pct=-1
    local status="$label"
    local tick=0

    if [ -t 1 ]; then
        tput civis 2>/dev/null || true
        render_progress_bar "$pct" "$status"
    else
        echo "[*] $label..."
    fi

    while kill -0 "$pid" 2>/dev/null; do
        tick=$((tick + 1))
        if [ -f "$task_log" ]; then
            case "$mode" in
                pkgmgr)
                    local dl_hits inst_hits
                    dl_hits="$(grep -cE '(^(Get:|Err:|Ign:|Retrieving )|Downloading|downloading|getting packages)' "$task_log" 2>/dev/null || true)"
                    inst_hits="$(grep -cE '(^(Unpacking|Setting up|Preparing to unpack|Installing|Upgrading|Verifying|Cleanup|Running hooks)|installing packages)' "$task_log" 2>/dev/null || true)"
                    dl_hits="${dl_hits:-0}"
                    inst_hits="${inst_hits:-0}"
                    if grep -qE '(Processing triggers|Running post-transaction|ldconfig)' "$task_log" 2>/dev/null; then
                        pct=93
                        status="$label (finalizing)"
                    elif [ "$inst_hits" -gt 0 ]; then
                        pct=$((62 + inst_hits * 3))
                        [ "$pct" -gt 92 ] && pct=92
                        status="$label (installing packages)"
                    elif [ "$dl_hits" -gt 0 ]; then
                        pct=$((22 + dl_hits * 4))
                        [ "$pct" -gt 60 ] && pct=60
                        status="$label (downloading packages)"
                    elif grep -qiE '(Reading package lists|Synchronizing|Metadata|Building dependency tree|Resolving dependencies)' "$task_log" 2>/dev/null; then
                        pct=18
                        status="$label (resolving dependencies)"
                    else
                        pct=$((8 + (tick % 10)))
                    fi
                    ;;
                pip)
                    local last_line col_hits dl_hits
                    col_hits="$(grep -c '^Collecting ' "$task_log" 2>/dev/null || true)"
                    dl_hits="$(grep -c '^  Downloading ' "$task_log" 2>/dev/null || true)"
                    col_hits="${col_hits:-0}"
                    dl_hits="${dl_hits:-0}"
                    if grep -q '^Successfully installed' "$task_log" 2>/dev/null; then
                        pct=98
                        status="$label (complete)"
                    elif grep -q '^Installing collected packages' "$task_log" 2>/dev/null; then
                        pct=$((78 + (tick % 16)))
                        status="$label (installing packages)"
                    elif grep -q 'Building wheel' "$task_log" 2>/dev/null; then
                        pct=72
                        status="$label (building wheel)"
                    elif [ "$dl_hits" -gt 0 ]; then
                        last_line="$(grep -E '^  Downloading ' "$task_log" 2>/dev/null | tail -n 1 | awk '{print $2}' || true)"
                        last_line="${last_line##*/}"
                        pct=$((35 + dl_hits * 3))
                        [ "$pct" -gt 70 ] && pct=70
                        status="$label (downloading ${last_line:-packages})"
                    elif [ "$col_hits" -gt 0 ]; then
                        last_line="$(grep '^Collecting ' "$task_log" 2>/dev/null | tail -n 1 | sed 's/^Collecting //' | awk '{print $1}' || true)"
                        pct=$((12 + col_hits * 2))
                        [ "$pct" -gt 35 ] && pct=35
                        status="$label (collecting ${last_line:-deps})"
                    else
                        pct=$((6 + (tick % 8)))
                    fi
                    ;;
                go)
                    local mod_hits pkg_hits last_pkg
                    mod_hits="$(grep -c '^go: downloading ' "$task_log" 2>/dev/null || true)"
                    pkg_hits="$(grep -cE '^[a-zA-Z0-9._-]+/' "$task_log" 2>/dev/null || true)"
                    mod_hits="${mod_hits:-0}"
                    pkg_hits="${pkg_hits:-0}"
                    if [ "$pkg_hits" -gt 0 ]; then
                        last_pkg="$(grep -E '^[a-zA-Z0-9._-]+/' "$task_log" 2>/dev/null | tail -n 1 | awk -F'/' '{print $NF}' || true)"
                        pct=$((42 + pkg_hits))
                        [ "$pct" -gt 96 ] && pct=96
                        status="$label (compiling ${last_pkg:-packages})"
                    elif [ "$mod_hits" -gt 0 ]; then
                        last_pkg="$(grep '^go: downloading ' "$task_log" 2>/dev/null | tail -n 1 | awk '{print $3}' | awk -F'/' '{print $NF}' || true)"
                        pct=$((12 + mod_hits * 2))
                        [ "$pct" -gt 40 ] && pct=40
                        status="$label (downloading ${last_pkg:-modules})"
                    else
                        pct=$((8 + (tick % 10)))
                    fi
                    ;;
                *)
                    pct=$((10 + (tick * 3)))
                    [ "$pct" -gt 92 ] && pct=92
                    ;;
            esac
        fi

        if [ -t 1 ]; then
            render_progress_bar "$pct" "$status"
        elif [ "$pct" -ge $((last_pct + 20)) ]; then
            echo "  [$pct%] $status"
            last_pct="$pct"
        fi
        sleep 0.2
    done

    wait "$pid"
    local exit_code=$?

    if [ -s "$task_log" ]; then
        cat "$task_log" >> "$LOG_FILE" 2>/dev/null || true
    fi

    if [ -t 1 ]; then
        tput cnorm 2>/dev/null || true
        if [ "$exit_code" -eq 0 ]; then
            render_progress_bar 100 "$label - Done"
            echo ""
        else
            render_progress_bar "$pct" "$label - Failed"
            echo ""
        fi
    fi

    if [ "$exit_code" -ne 0 ] && [ -s "$task_log" ]; then
        echo "Recent task output:" >&2
        tail -n 12 "$task_log" | sed 's/^/    /' >&2
    fi

    rm -f "$task_log"
    return "$exit_code"
}

# Download a URL displaying a live byte-accurate progress bar.
download_with_progress() {
    local url="$1"
    local dest="$2"
    local label="$3"

    log "$label..."
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] Downloading $url -> $dest" >> "$LOG_FILE" 2>/dev/null || true

    if command -v curl >/dev/null 2>&1; then
        if [ -t 2 ] || [ -t 1 ]; then
            curl -fL --progress-bar "$url" -o "$dest"
        else
            curl -fsSL "$url" -o "$dest"
        fi
    elif command -v wget >/dev/null 2>&1; then
        if [ -t 2 ] || [ -t 1 ]; then
            wget -q --show-progress "$url" -O "$dest"
        else
            wget -q "$url" -O "$dest"
        fi
    elif command -v python3 >/dev/null 2>&1; then
        python3 - "$url" "$dest" <<'PY'
import sys, urllib.request
url, dest = sys.argv[1], sys.argv[2]
def report(block_num, block_size, total_size):
    if total_size > 0 and sys.stdout.isatty():
        pct = min(100, int(block_num * block_size * 100 / total_size))
        filled = pct * 24 // 100
        sys.stdout.write(f"\r  [{'#' * filled}{'-' * (24 - filled)}] {pct:3d}%  Downloading")
        sys.stdout.flush()
urllib.request.urlretrieve(url, dest, reporthook=report)
if sys.stdout.isatty():
    sys.stdout.write("\n")
PY
    else
        die "Neither curl, wget, nor python3 is available to download $url"
    fi
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

check_go_version() {
    command -v go >/dev/null 2>&1 || return 1
    local raw_ver
    raw_ver="$(go version 2>/dev/null | sed -n 's/.*go\([0-9][0-9]*\.[0-9][0-9]*\).*/\1/p' | head -n 1)"
    [ -n "$raw_ver" ] || return 0
    local major minor
    major="${raw_ver%%.*}"
    minor="${raw_ver#*.}"
    # Go >= 1.21 is required so GOTOOLCHAIN=auto can build go1.24+ modules reliably.
    if [ "$major" -gt 1 ] || { [ "$major" -eq 1 ] && [ "$minor" -ge 21 ]; }; then
        return 0
    fi
    return 1
}

# Use sudo only when we are not already root.
init_sudo() {
    if [ "$(id -u)" -eq 0 ]; then
        SUDO=""
    else
        SUDO="sudo"
    fi
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
            $SUDO zypper --non-interactive install -y python3 python3-pip python3-virtualenv
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
            $SUDO zypper --non-interactive install -y git
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
            $SUDO zypper --non-interactive install -y go
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
            if run_with_progress "Installing $dep_name" "pkgmgr" "$install_action"; then
                log "$dep_name installed successfully"
                return 0
            else
                log_error "Failed to install $dep_name automatically"
            fi
        fi
    fi

    echo ""
    echo "[x] $dep_name is required and was not installed."
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
            $SUDO zypper --non-interactive install -y python3-pip python3-virtualenv
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

# Go is compulsory: Arjax requires the compiled arjax-provider Go engine & TUI.
ensure_go() {
    if command -v go >/dev/null 2>&1; then
        local go_ver
        go_ver="$(go version | awk '{print $3}')"
        if check_go_version; then
            log "Go toolchain detected ($go_ver)"
            return 0
        else
            die "Detected Go $go_ver is too old. Arjax requires Go 1.21+ (go.mod targets go 1.24.2). Upgrade Go via your package manager or https://go.dev/dl/ and rerun."
        fi
    fi

    local distro_family
    distro_family="$(detect_distro_family)"

    local display_cmds=""
    case "$distro_family" in
        arch) display_cmds="sudo pacman -S go" ;;
        debian) display_cmds="sudo apt update\n    sudo apt install -y golang-go" ;;
        fedora) display_cmds="sudo dnf install -y golang" ;;
        suse) display_cmds="sudo zypper install -y go" ;;
        *) display_cmds="Install Go 1.21+ from https://go.dev/dl/ or your package manager" ;;
    esac

    log "Go compiler is compulsory to build the Arjax Go Provider Engine & TUI."
    request_dep_install "Go compiler" "$display_cmds" "install_go_package $distro_family" "Arjax requires Go (1.21+ / 1.24+) to compile the compulsory arjax-provider binary."

    if ! command -v go >/dev/null 2>&1; then
        die "Go compiler was not found on PATH after installation."
    fi
    if ! check_go_version; then
        die "Installed Go version ($(go version | awk '{print $3}')) is older than Go 1.21. Please install Go 1.21+ from https://go.dev/dl/."
    fi
    log "Go compiler ready ($(go version | awk '{print $3}'))"
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

# Resolve local checkout or download the repository archive once for both Python & Go builds.
ensure_source() {
    if [ -f "$SCRIPT_DIR/pyproject.toml" ] && [ -d "$SCRIPT_DIR/arjax-provider-engine" ]; then
        SOURCE_DIR="$SCRIPT_DIR"
        SOURCE_SPEC="$SCRIPT_DIR"
        log "Using local checkout at $SOURCE_DIR"
        return 0
    fi

    ARCHIVE_TMP_DIR="$(mktemp -d /tmp/arjax-source-XXXXXX 2>/dev/null || mktemp -d)"
    local zip_file="$ARCHIVE_TMP_DIR/arjax-source.zip"

    if ! download_with_progress "$REPO_ARCHIVE_URL" "$zip_file" "Downloading Arjax source archive"; then
        log "Primary archive failed; downloading from fallback archive URL..."
        download_with_progress "$FALLBACK_ARCHIVE_URL" "$zip_file" "Downloading fallback Arjax archive" \
            || die "Failed to download Arjax source archive from $REPO_ARCHIVE_URL or $FALLBACK_ARCHIVE_URL."
    fi

    [ -s "$zip_file" ] || die "Downloaded source archive is empty."

    run_with_progress "Extracting source archive" "generic" \
        "python3 -c \"import zipfile; zipfile.ZipFile('$zip_file').extractall('$ARCHIVE_TMP_DIR')\"" \
        || die "Failed to extract Arjax source archive."

    local extracted_root
    extracted_root="$(find "$ARCHIVE_TMP_DIR" -mindepth 1 -maxdepth 2 -name "pyproject.toml" -exec dirname {} \; 2>/dev/null | head -n 1)"
    if [ -z "$extracted_root" ] || [ ! -d "$extracted_root" ]; then
        die "Could not locate extracted Arjax project directory."
    fi

    SOURCE_DIR="$extracted_root"
    SOURCE_SPEC="$extracted_root"
    log "Source ready at $SOURCE_DIR"
}

# Create an isolated Python environment so the install does not affect system packages.
create_venv() {
    mkdir -p "$INSTALL_ROOT"
    if [ ! -x "$VENV_DIR/bin/python" ]; then
        log "Creating virtual environment in $VENV_DIR"
        if ! python3 -m venv "$VENV_DIR" >> "$LOG_FILE" 2>&1; then
            log "Standard venv creation failed; attempting minimal venv creation..."
            python3 -m venv --without-pip "$VENV_DIR" >> "$LOG_FILE" 2>&1 || die "Could not create virtual environment in $VENV_DIR."
        fi
    fi

    if [ ! -x "$VENV_DIR/bin/python" ]; then
        die "Could not find python binary in virtual environment: $VENV_DIR"
    fi

    # Ensure pip is available inside the virtual environment
    if [ ! -x "$VENV_DIR/bin/pip" ]; then
        log "Bootstrapping pip inside the virtual environment..."
        "$VENV_DIR/bin/python" -m ensurepip --upgrade >> "$LOG_FILE" 2>&1 || true
    fi

    if [ ! -x "$VENV_DIR/bin/pip" ]; then
        local get_pip_file="${ARCHIVE_TMP_DIR:-/tmp}/get-pip.py"
        download_with_progress "https://bootstrap.pypa.io/get-pip.py" "$get_pip_file" "Downloading get-pip.py" || true
        if [ -s "$get_pip_file" ]; then
            run_with_progress "Bootstrapping pip" "pip" "\"$VENV_DIR/bin/python\" \"$get_pip_file\"" || true
        fi
    fi

    if [ ! -x "$VENV_DIR/bin/pip" ]; then
        die "Could not prepare pip inside the virtual environment. Install python3-pip and python3-venv, then rerun the installer."
    fi
}

# Install the Python CLI/TUI package into the private venv (no PyQt5).
install_package() {
    local pip_bin="$VENV_DIR/bin/pip"

    if [ ! -x "$pip_bin" ]; then
        die "pip is unavailable inside the virtual environment."
    fi

    run_with_progress "Upgrading packaging tools" "pip" \
        "\"$pip_bin\" install --upgrade pip setuptools wheel" || true

    run_with_progress "Installing Arjax CLI & Python core" "pip" \
        "\"$pip_bin\" install --upgrade \"$SOURCE_SPEC\"" \
        || die "Failed to install Arjax from $SOURCE_SPEC."

    [ -x "$VENV_DIR/bin/arjax" ] || die "Installation finished but $VENV_DIR/bin/arjax entry point is missing."
}

# Compile the compulsory Go Provider Engine binary.
build_go_engine() {
    command -v go >/dev/null 2>&1 || die "Go compiler is compulsory but missing."

    local engine_dir="$SOURCE_DIR/arjax-provider-engine"
    if [ ! -d "$engine_dir" ]; then
        die "Compulsory Go engine source directory not found at $engine_dir."
    fi

    mkdir -p "$BIN_DIR"
    local temp_bin="$BIN_DIR/arjax-provider.tmp"

    run_with_progress "Compiling Arjax Go Provider Engine" "go" \
        "cd \"$engine_dir\" && GOTOOLCHAIN=auto go build -v -o \"$temp_bin\" ." \
        || die "Failed to compile compulsory Arjax Go Provider Engine."

    install -m 755 "$temp_bin" "$BIN_DIR/arjax-provider" 2>/dev/null || mv -f "$temp_bin" "$BIN_DIR/arjax-provider"
    rm -f "$temp_bin"

    if [ ! -x "$BIN_DIR/arjax-provider" ]; then
        die "Compiled binary missing at $BIN_DIR/arjax-provider."
    fi
    log "Arjax Go Provider Engine installed to $BIN_DIR/arjax-provider"
}

# Build the native Wails desktop GUI when graphical support (or --with-gui) is present.
build_wails_gui() {
    if ! has_graphical_environment && [ "$WITH_GUI" != true ]; then
        log "CLI-only environment detected; skipping native Wails GUI build (pass --with-gui to force build)."
        return 0
    fi

    local gui_dir="$SOURCE_DIR/arjax-gui"
    if [ ! -d "$gui_dir" ]; then
        return 0
    fi

    # Build frontend bundle if dist is not already compiled
    if [ ! -d "$gui_dir/frontend/dist" ]; then
        if command -v npm >/dev/null 2>&1; then
            run_with_progress "Installing Wails frontend packages" "generic" \
                "cd \"$gui_dir/frontend\" && npm install --no-audit --no-fund" || {
                    log "Notice: Frontend npm install failed; skipping arjax-gui binary build."
                    return 0
                }
            run_with_progress "Building Wails frontend bundle" "generic" \
                "cd \"$gui_dir/frontend\" && npm run build" || {
                    log "Notice: Frontend build failed; skipping arjax-gui binary build."
                    return 0
                }
        else
            log "Notice: Node.js/npm not found to build arjax-gui frontend/dist; skipping native GUI build."
            return 0
        fi
    fi

    local build_tags=""
    if command -v pkg-config >/dev/null 2>&1; then
        if pkg-config --exists webkit2gtk-4.1 2>/dev/null && ! pkg-config --exists webkit2gtk-4.0 2>/dev/null; then
            build_tags="-tags webkit2_41"
        elif ! pkg-config --exists webkit2gtk-4.1 2>/dev/null && ! pkg-config --exists webkit2gtk-4.0 2>/dev/null; then
            log "Notice: WebKit2GTK dev headers not found; skipping native Wails GUI binary build (CLI and Go TUI work without it)."
            return 0
        fi
    fi

    mkdir -p "$BIN_DIR"
    local temp_gui="$BIN_DIR/arjax-gui.tmp"
    if run_with_progress "Compiling Native Wails GUI (arjax-gui)" "go" \
        "cd \"$gui_dir\" && GOTOOLCHAIN=auto go build $build_tags -v -o \"$temp_gui\" ."; then
        install -m 755 "$temp_gui" "$BIN_DIR/arjax-gui" 2>/dev/null || mv -f "$temp_gui" "$BIN_DIR/arjax-gui"
        rm -f "$temp_gui"
        if [ -x "$BIN_DIR/arjax-gui" ]; then
            log "Native Wails GUI compiled and installed to $BIN_DIR/arjax-gui"
        fi
    else
        rm -f "$temp_gui"
        log "Notice: Native Wails GUI build did not complete cleanly (CLI and Go TUI remain ready). See $LOG_FILE for details."
    fi
}

# Link executables into /usr/local/bin when permitted so `arjax` is immediately available in active shells.
link_global_binaries() {
    if [ ! -d "$GLOBAL_BIN_DIR" ]; then
        return 0
    fi

    local link_prefix=""
    if [ -w "$GLOBAL_BIN_DIR" ]; then
        link_prefix=""
    else
        init_sudo
        if [ -z "$SUDO" ]; then
            link_prefix=""
        elif $SUDO -n true 2>/dev/null; then
            link_prefix="$SUDO"
        elif [ -c /dev/tty ] || [ -t 0 ]; then
            case ":${INITIAL_PATH}:" in
                *:"$BIN_DIR":*) return 0 ;;
            esac
            if prompt_yes_no "Link '$APP_NAME' into $GLOBAL_BIN_DIR so it is available immediately in this terminal? [Y/n]:" "Y"; then
                if run_sudo_auth "$GLOBAL_BIN_DIR symlinks"; then
                    link_prefix="$SUDO"
                else
                    return 0
                fi
            else
                return 0
            fi
        else
            return 0
        fi
    fi

    ${link_prefix:+$link_prefix }ln -sf "$BIN_DIR/$APP_NAME" "$GLOBAL_BIN_DIR/$APP_NAME" 2>/dev/null || true
    if [ -x "$BIN_DIR/arjax-provider" ]; then
        ${link_prefix:+$link_prefix }ln -sf "$BIN_DIR/arjax-provider" "$GLOBAL_BIN_DIR/arjax-provider" 2>/dev/null || true
    fi
    if [ -x "$BIN_DIR/arjax-gui" ]; then
        ${link_prefix:+$link_prefix }ln -sf "$BIN_DIR/arjax-gui" "$GLOBAL_BIN_DIR/arjax-gui" 2>/dev/null || true
    fi
    if [ -L "$GLOBAL_BIN_DIR/$APP_NAME" ] || [ -x "$GLOBAL_BIN_DIR/$APP_NAME" ]; then
        log "Linked $APP_NAME commands into $GLOBAL_BIN_DIR for immediate terminal access"
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

    link_global_binaries
}

remove_shell_path_block() {
    local shell_file="$1"
    local block_file
    block_file="$(mktemp)"

    python3 - "$shell_file" "$block_file" <<'PY'
from pathlib import Path
import sys

shell_file = Path(sys.argv[1])
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
    rm -f "$BIN_DIR/$APP_NAME" "$BIN_DIR/arjax-provider" "$BIN_DIR/arjax-gui" \
          "$DESKTOP_DIR/arjax.desktop" "$DESKTOP_DIR/archpkg-helper.desktop" "$DESKTOP_DIR/arjax-helper.desktop"
    rm -rf "$INSTALL_ROOT"

    init_sudo
    for global_name in "$APP_NAME" "arjax-provider" "arjax-gui"; do
        if [ -L "$GLOBAL_BIN_DIR/$global_name" ] || [ -e "$GLOBAL_BIN_DIR/$global_name" ]; then
            if [ -w "$GLOBAL_BIN_DIR" ] || [ -z "$SUDO" ]; then
                rm -f "$GLOBAL_BIN_DIR/$global_name" 2>/dev/null || true
            elif $SUDO -n true 2>/dev/null; then
                $SUDO rm -f "$GLOBAL_BIN_DIR/$global_name" 2>/dev/null || true
            fi
        fi
    done

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
    local exec_cmd="$BIN_DIR/$APP_NAME gui"
    if [ -x "$BIN_DIR/arjax-gui" ]; then
        exec_cmd="$BIN_DIR/arjax-gui"
    fi

    cat > "$DESKTOP_DIR/arjax-helper.desktop" <<EOF
[Desktop Entry]
Version=1.0
Type=Application
Name=Arjax
GenericName=Package Manager
Comment=Cross-distribution package manager with native GUI
Exec=$exec_cmd
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
    if [ -f "$SOURCE_DIR/data/man/man1/arjax.1" ]; then
        cp "$SOURCE_DIR/data/man/man1/arjax.1" "$MAN_DIR/arjax.1"
        log "Man page installed to $MAN_DIR/arjax.1 (view with: man arjax)"
    fi
}

# Preserve the Arch-only profile setup, but keep it out of the universal path for other distros.
configure_arch_profile() {
    local distro_family="$1"
    if [ "$distro_family" != "arch" ]; then
        return
    fi

    # During upgrade or if config already exists, preserve existing user settings
    if [ -f "$HOME/.arjax/config.json" ]; then
        log "Existing Arjax configuration found at $HOME/.arjax/config.json; preserving user settings."
        return 0
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

    if [ "$UPGRADE_MODE" = true ]; then
        log "Starting upgrade of Arjax suite..."
    else
        log "Starting universal installation of Arjax..."
    fi

    ensure_python
    ensure_git
    ensure_go
    ensure_source
    create_venv
    install_package
    build_go_engine
    build_wails_gui
    create_launcher
    ensure_shell_path
    install_desktop_entry
    install_man_page
    configure_arch_profile "$(detect_distro_family)"

    INSTALL_SUCCESS=true
    if [ "$UPGRADE_MODE" = true ]; then
        log "Arjax suite upgrade complete!"
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] Arjax suite upgrade completed successfully." >> "$LOG_FILE" 2>/dev/null || true
    else
        log "Arjax installation complete!"
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] Arjax installation completed successfully." >> "$LOG_FILE" 2>/dev/null || true
    fi
    echo ""
    echo "Quick start:"
    echo "    arjax               # Launch interactive TUI (Default)"
    echo "    arjax search <pkg>  # Search for packages"
    echo "    arjax install <pkg> # Install packages"
    echo "    arjax gui           # Launch native Wails desktop GUI"
    echo "    arjax --help        # Show all CLI commands"
    echo ""

    local resolved_cmd=""
    resolved_cmd="$(PATH="$INITIAL_PATH" command -v "$APP_NAME" 2>/dev/null || true)"
    if [ -n "$resolved_cmd" ]; then
        echo "Ready: '$APP_NAME' is immediately available at $resolved_cmd."
    else
        echo "IMPORTANT: Reload your current shell PATH before running 'arjax' in this terminal:"
        echo "    source ~/.bashrc   # (or: export PATH=\"$BIN_DIR:\$PATH\")"
    fi
    echo ""
    if has_graphical_environment; then
        if [ -x "$BIN_DIR/arjax-gui" ]; then
            echo "Native Wails GUI is installed at $BIN_DIR/arjax-gui and in your application menu."
        else
            echo "Graphical environment detected; desktop launcher added (see log if GUI build was skipped)."
        fi
    else
        echo "CLI-only environment detected, so no application menu entry was created."
    fi
    echo ""
    echo "Full installation log is saved at: $LOG_FILE"
}

main "$@"
