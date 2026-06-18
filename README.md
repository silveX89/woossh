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

**Version:** v0.3.0 — Plugin-System mit external Plugin Repo

## Requirements

- Go 1.24+
- Linux / macOS
- `tmux` (optional, für Tmux-Plugin)

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
sudo cp woossh /usr/local/bin/
```

> **Hinweis:** `woossh plugin install` benötigt Zigriff auf die `go.mod`. Führe Plugin-Befehle
> aus dem repo-Verzeichnis aus oder setze `WOOSSH_SOURCE_DIR=/path/to/woossh`.

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
| `plugins.yaml` | Plugin state (enabled/disabled, settings) |

### Plugin-System

woossh v0.3.0+ hat ein compiled-in Plugin-System. Plugins werden via blank import
in `registry_gen.go` eingebunden und bei jedem Build neu compiliert.

**Plugin-Manager (Ctrl+P):**
```
  🔌  Plugin Manager — woossh v0.3.0
  ─────────────────────────────────────
  Plugin                    Version    Status     Source
  ─────────────────────────────────────
▸ Tmux Integration         0.3.0      enabled    github.com/silveX89/woossh-plugins/tmux

  [↑↓] Scroll  [Enter] Toggle enable/disable  [s] Settings  [Esc] or [q] Back
```

**Plugin CLI:**

| Command | Description |
|---------|-------------|
| `woossh plugin list` | List all installed plugins |
| `woossh plugin install <url> [--rebuild]` | Install plugin from Git URL + rebuild |
| `woossh plugin remove <id>` | Remove an installed plugin |
| `woossh plugin help` | Show plugin CLI help |

**Externes Plugin installieren:**
```bash
cd /path/to/woossh
woossh plugin install github.com/silveX89/woossh-plugins/tmux --rebuild
```

Plugins werden nach `~/.local/share/woossh/plugins/` geklont.
Trust-Level: `github.com/silveX89/*` = official, alles andere = unknown.

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
| `Ctrl+P` | Open plugin manager |
| `s` (in Plugin-Manager) | Open plugin settings |
| `Esc` / `q` | Back / Close |

### Settings (Ctrl+S)

Open settings menu with tabs: Allgemein, SSH, Tmux, TUI, Favoriten.

- `↑↓` items, `Tab`/`Shift+Tab` categories
- `Enter` toggles bool / opens editor for string / cycles enum
- `Esc` saves and closes

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
| `--version` / `-v` | Show version (`woossh v0.3.0`) |
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