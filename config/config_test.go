package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaults(t *testing.T) {
	d := defaults()

	tests := []struct {
		key string
		got interface{}
		want interface{}
	}{
		{"DefaultUser", d.DefaultUser, "root"},
		{"SSHPort", d.SSHPort, 22},
		{"SessionPrefix", d.SessionPrefix, "woossh_"},
		{"Timeout", d.Timeout, 10},
		{"ColorScheme", d.ColorScheme, "default"},
		{"ShowIP", d.ShowIP, true},
		{"ShowPort", d.ShowPort, false},
		{"ShowFavorites", d.ShowFavorites, true},
		{"FavoriteSort", d.FavoriteSort, "name"},
		{"IdentityFile", d.IdentityFile, "~/.ssh/id_rsa"},
		{"GlobalJumphost", d.GlobalJumphost, false},
	}

	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("defaults().%s = %v, want %v", tt.key, tt.got, tt.want)
		}
	}
}

func TestLoadWithEnvConfig(t *testing.T) {
	tmpDir := t.TempDir()
	woosshDir := filepath.Join(tmpDir, "woossh")
	os.MkdirAll(woosshDir, 0o700)

	iniContent := `
global_jumphost = yes
jumpserver = bastion.example.com
jumpuser = jumpuser
ssh_user = admin
default_user = testuser
ssh_port = 2222
`
	os.WriteFile(filepath.Join(woosshDir, "config.ini"), []byte(iniContent), 0o600)

	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if !cfg.GlobalJumphost {
		t.Error("GlobalJumphost = false, want true")
	}
	if cfg.JumpServer != "bastion.example.com" {
		t.Errorf("JumpServer = %q, want bastion.example.com", cfg.JumpServer)
	}
	if cfg.JumpUser != "jumpuser" {
		t.Errorf("JumpUser = %q, want jumpuser", cfg.JumpUser)
	}
	if cfg.SSHUser != "admin" {
		t.Errorf("SSHUser = %q, want admin", cfg.SSHUser)
	}
	if cfg.DefaultUser != "testuser" {
	        t.Errorf("DefaultUser = %q, want testuser", cfg.DefaultUser)
	}
	if cfg.SSHPort != 2222 {
		t.Errorf("SSHPort = %d, want 2222", cfg.SSHPort)
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	woosshDir := filepath.Join(tmpDir, "woossh")
	os.MkdirAll(woosshDir, 0o700)

	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	original := Config{
		GlobalJumphost: true,
		JumpServer:     "jump.example.com",
		JumpUser:       "juser",
		SSHUser:        "op",
		DefaultUser:    "testuser",
		SSHPort:        2222,
		IdentityFile:   "~/.ssh/ed25519",
		ForwardAgent:   true,
		Timeout:        30,
		UseTmux:        true,
		SessionPrefix:  "ssh:",
		ShowPort:       true,
		PagerLines:     20,
		FavoriteSort:   "manual",
	}

	if err := Save(original); err != nil {
		t.Fatalf("Save() returned error: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if loaded.GlobalJumphost != original.GlobalJumphost {
		t.Errorf("GlobalJumphost = %v, want %v", loaded.GlobalJumphost, original.GlobalJumphost)
	}
	if loaded.JumpServer != original.JumpServer {
		t.Errorf("JumpServer = %q, want %q", loaded.JumpServer, original.JumpServer)
	}
	if loaded.JumpUser != original.JumpUser {
		t.Errorf("JumpUser = %q, want %q", loaded.JumpUser, original.JumpUser)
	}
	if loaded.SSHUser != original.SSHUser {
		t.Errorf("SSHUser = %q, want %q", loaded.SSHUser, original.SSHUser)
	}
	if loaded.DefaultUser != original.DefaultUser {
		t.Errorf("DefaultUser = %q, want %q", loaded.DefaultUser, original.DefaultUser)
	}
	if loaded.SSHPort != original.SSHPort {
		t.Errorf("SSHPort = %d, want %d", loaded.SSHPort, original.SSHPort)
	}
	if loaded.IdentityFile != original.IdentityFile {
		t.Errorf("IdentityFile = %q, want %q", loaded.IdentityFile, original.IdentityFile)
	}
	if loaded.ForwardAgent != original.ForwardAgent {
		t.Errorf("ForwardAgent = %v, want %v", loaded.ForwardAgent, original.ForwardAgent)
	}
	if loaded.Timeout != original.Timeout {
		t.Errorf("Timeout = %d, want %d", loaded.Timeout, original.Timeout)
	}
	if loaded.UseTmux != original.UseTmux {
		t.Errorf("UseTmux = %v, want %v", loaded.UseTmux, original.UseTmux)
	}
	if loaded.SessionPrefix != original.SessionPrefix {
		t.Errorf("SessionPrefix = %q, want %q", loaded.SessionPrefix, original.SessionPrefix)
	}
	if loaded.ShowPort != original.ShowPort {
		t.Errorf("ShowPort = %v, want %v", loaded.ShowPort, original.ShowPort)
	}
	if loaded.PagerLines != original.PagerLines {
		t.Errorf("PagerLines = %d, want %d", loaded.PagerLines, original.PagerLines)
	}
	if loaded.FavoriteSort != original.FavoriteSort {
		t.Errorf("FavoriteSort = %q, want %q", loaded.FavoriteSort, original.FavoriteSort)
	}
}

func TestSaveCreatesBackup(t *testing.T) {
	tmpDir := t.TempDir()
	woosshDir := filepath.Join(tmpDir, "woossh")
	os.MkdirAll(woosshDir, 0o700)

	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cfg1 := defaults()
	cfg1.DefaultUser = "first"
	Save(cfg1)

	cfg2 := defaults()
	cfg2.DefaultUser = "second"
	Save(cfg2)

	cfg3 := defaults()
	cfg3.DefaultUser = "third"
	Save(cfg3)

	backupDir := filepath.Join(woosshDir, "backups")
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatalf("backup dir not found: %v", err)
	}

	if len(entries) < 1 {
		t.Errorf("expected at least 1 backup, got %d", len(entries))
	}
}

func TestLoadMissingFileReturnsDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// No config.ini exists — Load should return defaults without error
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error when no config exists: %v", err)
	}
	if cfg.DefaultUser != "root" {
		t.Errorf("DefaultUser = %q, want root (default)", cfg.DefaultUser)
	}
}
