package config

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/ini.v1"
)

type Config struct {
	GlobalJumphost bool
	JumpServer     string
	JumpUser       string
	SSHUser        string
}

func configDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "woossh")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "woossh")
}

func findConfigFile() string {
	if _, err := os.Stat("config.ini"); err == nil {
		return "config.ini"
	}
	p := filepath.Join(configDir(), "config.ini")
	if _, err := os.Stat(p); err == nil {
		return p
	}
	return ""
}

func Load() (Config, error) {
	path := findConfigFile()
	if path == "" {
		return Config{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	// Inject [main] section header as required by the spec
	content := "[main]\n" + string(data)

	cfg, err := ini.Load([]byte(content))
	if err != nil {
		return Config{}, err
	}

	section := cfg.Section("main")
	globalJumphost := strings.ToLower(section.Key("global_jumphost").String()) == "yes"

	return Config{
		GlobalJumphost: globalJumphost,
		JumpServer:     section.Key("jumpserver").String(),
		JumpUser:       section.Key("jumpuser").String(),
		SSHUser:        section.Key("ssh_user").String(),
	}, nil
}
