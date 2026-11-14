#compdef arjax

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
