"""Shell autocompletion generator and installer for Arjax (bash, zsh, fish)."""

import os
from pathlib import Path
from typing import Dict, Optional

BASH_COMPLETION_TEMPLATE = """# arjax bash completion script
_arjax_completion() {
    local cur prev words cword
    _init_completion || return

    local commands="search install remove info update cleanup history stats tui completion"
    
    case "${prev}" in
        arjax)
            COMPREPLY=( $(compgen -W "${commands}" -- "${cur}") )
            return 0
            ;;
        history)
            COMPREPLY=( $(compgen -W "list rollback" -- "${cur}") )
            return 0
            ;;
        cleanup)
            COMPREPLY=( $(compgen -W "orphans cache" -- "${cur}") )
            return 0
            ;;
        completion)
            COMPREPLY=( $(compgen -W "show install" -- "${cur}") )
            return 0
            ;;
        bash|zsh|fish)
            return 0
            ;;
    esac

    if [[ "${cur}" == -* ]]; then
        COMPREPLY=( $(compgen -W "--help --debug --json --dry-run" -- "${cur}") )
    fi
}

complete -F _arjax_completion arjax
"""

ZSH_COMPLETION_TEMPLATE = """#compdef arjax

_arjax() {
    local -a commands
    commands=(
        'search:Search packages across providers'
        'install:Install package'
        'remove:Remove package'
        'info:View package metadata'
        'update:Update package databases'
        'cleanup:Clean up orphans or cache'
        'history:View transaction history and rollbacks'
        'stats:View system statistics dashboard'
        'tui:Launch interactive TUI'
        'completion:Shell autocompletion helper'
    )

    _arguments -s \
        '--help[Show help message]' \
        '--debug[Enable debug mode]' \
        '1: :->command' \
        '*:: :->args'

    case $state in
        command)
            _describe -t commands 'arjax command' commands
            ;;
        args)
            case $words[1] in
                cleanup)
                    _values 'target' 'orphans[Remove unreferenced dependency packages]' 'cache[Clean package cache]'
                    ;;
                history)
                    _values 'action' 'list[List past transactions]' 'rollback[Rollback transaction]'
                    ;;
                completion)
                    _values 'action' 'show[Output completion script]' 'install[Install completion script]'
                    ;;
            esac
            ;;
    esac
}

_arjax "$@"
"""

FISH_COMPLETION_TEMPLATE = """# arjax fish completion script

complete -c arjax -f -n "__fish_use_subcommand" -a search -d "Search packages across providers"
complete -c arjax -f -n "__fish_use_subcommand" -a install -d "Install package"
complete -c arjax -f -n "__fish_use_subcommand" -a remove -d "Remove package"
complete -c arjax -f -n "__fish_use_subcommand" -a info -d "View package metadata"
complete -c arjax -f -n "__fish_use_subcommand" -a update -d "Update package databases"
complete -c arjax -f -n "__fish_use_subcommand" -a cleanup -d "Clean up orphans or cache"
complete -c arjax -f -n "__fish_use_subcommand" -a history -d "View transaction history & rollbacks"
complete -c arjax -f -n "__fish_use_subcommand" -a stats -d "View system statistics dashboard"
complete -c arjax -f -n "__fish_use_subcommand" -a tui -d "Launch interactive TUI"
complete -c arjax -f -n "__fish_use_subcommand" -a completion -d "Shell autocompletion helper"

# Subcommands for cleanup
complete -c arjax -f -n "__fish_seen_subcommand_from cleanup" -a orphans -d "Remove unreferenced packages"
complete -c arjax -f -n "__fish_seen_subcommand_from cleanup" -a cache -d "Clean package cache"

# Subcommands for history
complete -c arjax -f -n "__fish_seen_subcommand_from history" -a list -d "List transaction logs"
complete -c arjax -f -n "__fish_seen_subcommand_from history" -a rollback -d "Rollback a transaction"
"""


def generate_completion_script(shell: str) -> str:
    """Generate completion script string for specified shell (bash, zsh, fish)."""
    sh = shell.lower().strip()
    if sh == "bash":
        return BASH_COMPLETION_TEMPLATE
    elif sh == "zsh":
        return ZSH_COMPLETION_TEMPLATE
    elif sh == "fish":
        return FISH_COMPLETION_TEMPLATE
    else:
        raise ValueError(f"Unsupported shell '{shell}'. Supported: bash, zsh, fish.")


def install_completion(shell: str) -> str:
    """Install completion script for specified shell into default user config path."""
    sh = shell.lower().strip()
    script = generate_completion_script(sh)
    home = Path(Path.home())

    if sh == "bash":
        target_dir = home / ".local" / "share" / "bash-completion" / "completions"
        target_dir.mkdir(parents=True, exist_ok=True)
        target_file = target_dir / "arjax"
    elif sh == "zsh":
        target_dir = home / ".zsh" / "completion"
        target_dir.mkdir(parents=True, exist_ok=True)
        target_file = target_dir / "_arjax"
    elif sh == "fish":
        target_dir = home / ".config" / "fish" / "completions"
        target_dir.mkdir(parents=True, exist_ok=True)
        target_file = target_dir / "arjax.fish"
    else:
        raise ValueError(f"Unsupported shell '{shell}'. Supported: bash, zsh, fish.")

    target_file.write_text(script, encoding="utf-8")
    return str(target_file)
