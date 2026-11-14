# arjax bash completion script
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
