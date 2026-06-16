# woossh

A terminal UI for managing and connecting to SSH hosts. Fuzzy-search your host list, pick a server, and connect — all from the keyboard.

```
      /\_/\         ██╗    ██╗ ██████╗  ██████╗ ███████╗███████╗██╗  ██╗
     ( o.o )        ██║    ██║██╔═══██╗██╔═══██╗██╔════╝██╔════╝██║  ██║
      > ^ <         ██║ █╗ ██║██║   ██║██║   ██║███████╗███████╗███████║
                    ██║███╗██║██║   ██║██║   ██║╚════██║╚════██║██╔══██║
                    ╚███╔███╔╝╚██████╔╝╚██████╔╝███████║███████║██║  ██║
                     ╚══╝╚══╝  ╚═════╝  ╚═════╝ ╚══════╝╚══════╝╚═╝  ╚═╝
```

## Requirements

- Go 1.24+
- Linux / macOS

## Installation

### Quick install (from source)

```bash
curl -fsSL https://raw.githubusercontent.com/silveX89/woossh/main/install.sh | bash
```

### Manual install

```bash
git clone https://github.com/silveX89/woossh
cd woossh
go build -o woossh .
sudo mv woossh /usr/local/bin/
```

### go install

```bash
go install github.com/silveX89/woossh@latest
```

### Shell completion

**Bash:**
Add to `~/.bashrc`:

```bash
complete -C "woossh --list-hosts" woossh
```

**Zsh:**
```bash
source completions/_woossh
# or copy to fpath:
cp completions/_woossh /usr/share/zsh/site-functions/
```

**Fish:**
```bash
cp completions/woossh.fish ~/.config/fish/completions/
```

## Configuration

woossh looks for config files in `./` first, then `~/.config/woossh/`:

| File | Purpose |
|------|---------|
| `hosts.csv` | Your host list |
| `config.ini` | SSH options (jump host, user, port, etc.) |

### hosts.csv formats

woossh auto-detects the CSV format:

```csv
hostname,ip,description
firewall,192.168.1.1,Edge firewall
loadbalancer,192.168.1.10,HAProxy LB
```

Also supports `name`/`ip address` column format and plain host lists (one hostname per line).

## Usage

### Interactive TUI

```bash
woossh
```

### TUI keybindings

| Key | Action |
|-----|--------|
| `Type` | Fuzzy-search hosts |
| `↑` / `↓` | Scroll host table |
| `Tab` | Accept suggestion |
| `Enter` | Connect |
| `Ctrl+C` | Quit |
| `Ctrl+F` | Toggle favorite (★) |
| `Ctrl+Y` | Toggle copy mode (/c) |
| `Ctrl+T` | Toggle tmux mode (/t) |
| `Ctrl+O` | Open tmux session overview |
| `Ctrl+S` | Save settings |
| `/s` | Open settings menu |

### Tmux session overview (Ctrl+O)

View active tmux sessions with host, PID, uptime and status:

| Key | Action |
|-----|--------|
| `Enter` | Attach to session |
| `k` | Kill selected session |
| `d` | Detach all sessions |
| `Esc` / `q` | Back to host list |
| `↑` / `↓` | Scroll session list |

### Direct connect

```bash
woossh <hostname>
```

### CLI commands

| Command | Description |
|---------|-------------|
| `--list-hosts` | Print all hostnames (for shell completion) |
| `--version` / `-v` | Show version ("woossh v0.2.0") |
| `--cleanup` | Kill stale detached tmux sessions (>24h) |
| `--import-ssh-config [path]` | Import hosts from `~/.ssh/config` |

### Flags

Flags are slash-prefixed and stackable (e.g. `/o/v`). Use them as CLI prefixes or type them interactively in the TUI.

| Flag | Effect |
|------|--------|
| `/d` | Dry-run — print the ssh command without connecting |
| `/o` | Bypass jump host |
| `/v` | Verbose ssh output |
| `/c` | Copy mode — copy ssh command to clipboard instead of connecting |
| `/t` | Tmux mode — wrap SSH in a named tmux session |
| `/l` | Legacy `ssh-rsa` key support |

```bash
woossh /d firewall          # print ssh command for "firewall"
woossh /o/v server          # connect bypassing jump, verbose
woossh /d/o loadbalancer    # dry-run + bypass jump
```

### List all hosts

```bash
woossh --list-hosts
```
