# arjax fish completion script

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
