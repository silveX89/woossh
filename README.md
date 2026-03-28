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

### Shell completion (bash)

Add to `~/.bashrc`:

```bash
complete -C "woossh --list-hosts" woossh
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

Also supports XIQ-SE exports (`name`/`ip address` columns) and plain host lists (one hostname per line).

## Usage

### Interactive TUI

```bash
woossh
```

- Type to fuzzy-search hosts
- `↑` / `↓` — scroll the host table
- `Tab` — accept fuzzy suggestion
- `Enter` — connect
- `Ctrl+C` — quit

### Direct connect

```bash
woossh <hostname>
```

### Flags

Flags are slash-prefixed and stackable (e.g. `/o/v`). Use them as CLI prefixes or type them interactively in the TUI.

| Flag | Effect |
|------|--------|
| `/d` | Dry-run — print the ssh command without connecting |
| `/o` | Bypass jump host |
| `/v` | Verbose ssh output |
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
