# woossh completions for fish
# Source this file: source completions/woossh.fish
# Or install: cp completions/woossh.fish ~/.config/fish/completions/

complete -c woossh -f

# List hosts as completions
complete -c woossh -n "not __fish_seen_subcommand_from --list-hosts --version --import-ssh-config --cleanup" \
  -a "(woossh --list-hosts 2>/dev/null)" -d "SSH host"

# Long options
complete -c woossh -l list-hosts -d "List all known hosts"
complete -c woossh -l version -d "Show version"
complete -c woossh -s v -d "Show version"
complete -c woossh -l import-ssh-config -d "Import hosts from ~/.ssh/config" \
  -r -F
complete -c woossh -l cleanup -d "Clean up stale tmux sessions"

# Slash flags (only valid after woossh before a hostname)
complete -c woossh -n "not __fish_seen_subcommand_from --list-hosts --version --import-ssh-config --cleanup" \
  -a "/d" -d "Dry-run (print ssh command)"
complete -c woossh -n "not __fish_seen_subcommand_from --list-hosts --version --import-ssh-config --cleanup" \
  -a "/o" -d "Bypass jump host"
complete -c woossh -n "not __fish_seen_subcommand_from --list-hosts --version --import-ssh-config --cleanup" \
  -a "/v" -d "Verbose SSH output"
complete -c woossh -n "not __fish_seen_subcommand_from --list-hosts --version --import-ssh-config --cleanup" \
  -a "/l" -d "Legacy ssh-rsa key support"
complete -c woossh -n "not __fish_seen_subcommand_from --list-hosts --version --import-ssh-config --cleanup" \
  -a "/c" -d "Copy SSH command to clipboard"
complete -c woossh -n "not __fish_seen_subcommand_from --list-hosts --version --import-ssh-config --cleanup" \
  -a "/t" -d "Use tmux session"