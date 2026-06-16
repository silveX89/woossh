package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/ini.v1"
)

type Config struct {
	// Original fields
	GlobalJumphost bool
	JumpServer     string
	JumpUser       string
	SSHUser        string

	// Settings: Allgemein
	HostsPath   string
	ConfigPath  string
	DefaultUser string
	SSHPort     int

	// Settings: SSH
	IdentityFile  string
	JumpHost      string
	ForwardAgent  bool
	Timeout       int

	// Settings: Tmux
	UseTmux        bool
	SocketPath     string
	SessionPrefix  string

	// Settings: TUI
	ColorScheme   string
	ShowIP        bool
	ShowPort      bool
	ShowFavorites bool
	PagerLines    int

	// Settings: Favoriten
	FavoritesFile string
	FavoriteSort  string
}

func configDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "woossh")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "woossh")
}

func findConfigFile(paths ...string) string {
	if len(paths) > 0 && paths[0] != "" {
		if _, err := os.Stat(paths[0]); err == nil {
			return paths[0]
		}
	}
	if _, err := os.Stat("config.ini"); err == nil {
		return "config.ini"
	}
	p := filepath.Join(configDir(), "config.ini")
	if _, err := os.Stat(p); err == nil {
		return p
	}
	return ""
}

// ConfigPath returns the resolved path to config.ini (write target).
func ConfigPath() string {
	if p := findConfigFile(); p != "" {
		return p
	}
	// Default: write to config dir
	p := filepath.Join(configDir(), "config.ini")
	_ = os.MkdirAll(configDir(), 0o700)
	return p
}

func defaults() Config {
	return Config{
		GlobalJumphost: false,
		JumpServer:     "",
		JumpUser:       "",
		SSHUser:        "",

		HostsPath:   "./hosts.csv",
		ConfigPath:  "./config.ini",
		DefaultUser: "root",
		SSHPort:     22,

		IdentityFile: "~/.ssh/id_rsa",
		JumpHost:     "",
		ForwardAgent: false,
		Timeout:      10,

		UseTmux:       false,
		SocketPath:    "",
		SessionPrefix: "woossh_",

		ColorScheme:   "default",
		ShowIP:        true,
		ShowPort:      false,
		ShowFavorites: true,
		PagerLines:    0,

		FavoritesFile: "",
		FavoriteSort:  "name",
	}
}

func Load(paths ...string) (Config, error) {
	path := findConfigFile(paths...)
	if path == "" {
		return defaults(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return defaults(), err
	}

	// Inject [main] section header as required by the spec
	content := "[main]\n" + string(data)

	cfg, err := ini.Load([]byte(content))
	if err != nil {
		return defaults(), err
	}

	section := cfg.Section("main")
	d := defaults()

	globalJumphost := strings.ToLower(section.Key("global_jumphost").String()) == "yes"

	return Config{
		// Original
		GlobalJumphost: globalJumphost,
		JumpServer:     section.Key("jumpserver").String(),
		JumpUser:       section.Key("jumpuser").String(),
		SSHUser:        section.Key("ssh_user").String(),

		// Allgemein
		HostsPath:   strVal(section.Key("hosts_path").String(), d.HostsPath),
		ConfigPath:  strVal(section.Key("config_path").String(), d.ConfigPath),
		DefaultUser: strVal(section.Key("default_user").String(), d.DefaultUser),
		SSHPort:     intVal(section.Key("ssh_port").String(), d.SSHPort),

		// SSH
		IdentityFile: strVal(section.Key("identity_file").String(), d.IdentityFile),
		JumpHost:     strVal(section.Key("jump_host").String(), d.JumpHost),
		ForwardAgent: boolVal(section.Key("forward_agent").String(), d.ForwardAgent),
		Timeout:      intVal(section.Key("timeout").String(), d.Timeout),

		// Tmux
		UseTmux:       boolVal(section.Key("use_tmux").String(), d.UseTmux),
		SocketPath:    strVal(section.Key("socket_path").String(), d.SocketPath),
		SessionPrefix: strVal(section.Key("session_prefix").String(), d.SessionPrefix),

		// TUI
		ColorScheme:   strVal(section.Key("color_scheme").String(), d.ColorScheme),
		ShowIP:        boolVal(section.Key("show_ip").String(), d.ShowIP),
		ShowPort:      boolVal(section.Key("show_port").String(), d.ShowPort),
		ShowFavorites: boolVal(section.Key("show_favorites").String(), d.ShowFavorites),
		PagerLines:    intVal(section.Key("pager_lines").String(), d.PagerLines),

		// Favoriten
		FavoritesFile: strVal(section.Key("favorites_file").String(), d.FavoritesFile),
		FavoriteSort:  strVal(section.Key("favorite_sort").String(), d.FavoriteSort),
	}, nil
}

// Save writes config to config.ini, creating a backup first.
func Save(cfg Config) error {
	path := ConfigPath()
	_ = os.MkdirAll(filepath.Dir(path), 0o700)

	// Backup existing
	if _, err := os.Stat(path); err == nil {
		backupDir := filepath.Join(configDir(), "backups")
		_ = os.MkdirAll(backupDir, 0o700)

		// Rotate backups: keep 3 versions
		for i := 3; i >= 1; i-- {
			old := filepath.Join(backupDir, "config.ini."+strconv.Itoa(i)+".bak")
			if i == 3 {
				os.Remove(old)
				continue
			}
			src := filepath.Join(backupDir, "config.ini."+strconv.Itoa(i+1)+".bak")
			if _, err := os.Stat(src); err == nil {
				os.Rename(src, old)
			}
		}
		// Shift existing to backup.1
		os.Rename(path, filepath.Join(backupDir, "config.ini.1.bak"))
	}

	var sb strings.Builder
	sb.WriteString("; woossh configuration\n")
	sb.WriteString("; Generated by woossh settings menu\n\n")

	writeBool(&sb, "global_jumphost", cfg.GlobalJumphost)
	writeStr(&sb, "jumpserver", cfg.JumpServer)
	writeStr(&sb, "jumpuser", cfg.JumpUser)
	writeStr(&sb, "ssh_user", cfg.SSHUser)
	sb.WriteString("\n; Allgemein\n")
	writeStr(&sb, "hosts_path", cfg.HostsPath)
	writeStr(&sb, "config_path", cfg.ConfigPath)
	writeStr(&sb, "default_user", cfg.DefaultUser)
	writeInt(&sb, "ssh_port", cfg.SSHPort)
	sb.WriteString("\n; SSH\n")
	writeStr(&sb, "identity_file", cfg.IdentityFile)
	writeStr(&sb, "jump_host", cfg.JumpHost)
	writeBool(&sb, "forward_agent", cfg.ForwardAgent)
	writeInt(&sb, "timeout", cfg.Timeout)
	sb.WriteString("\n; Tmux\n")
	writeBool(&sb, "use_tmux", cfg.UseTmux)
	writeStr(&sb, "socket_path", cfg.SocketPath)
	writeStr(&sb, "session_prefix", cfg.SessionPrefix)
	sb.WriteString("\n; TUI\n")
	writeStr(&sb, "color_scheme", cfg.ColorScheme)
	writeBool(&sb, "show_ip", cfg.ShowIP)
	writeBool(&sb, "show_port", cfg.ShowPort)
	writeBool(&sb, "show_favorites", cfg.ShowFavorites)
	writeInt(&sb, "pager_lines", cfg.PagerLines)
	sb.WriteString("\n; Favoriten\n")
	writeStr(&sb, "favorites_file", cfg.FavoritesFile)
	writeStr(&sb, "favorite_sort", cfg.FavoriteSort)

	return os.WriteFile(path, []byte(sb.String()), 0o600)
}

// ─── helpers ────────────────────────────────────────────────────────────────

func strVal(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func boolVal(s string, def bool) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return def
	}
	return s == "yes" || s == "true" || s == "1"
}

func intVal(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

func writeStr(sb *strings.Builder, key, val string) {
	if val != "" {
		sb.WriteString(key + " = " + val + "\n")
	}
}

func writeBool(sb *strings.Builder, key string, val bool) {
	if val {
		sb.WriteString(key + " = yes\n")
	} else {
		sb.WriteString(key + " = no\n")
	}
}

func writeInt(sb *strings.Builder, key string, val int) {
	if val != 0 {
		sb.WriteString(key + " = " + strconv.Itoa(val) + "\n")
	}
}
