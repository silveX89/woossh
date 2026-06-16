package ssh

import (
	"strings"
	"testing"

	"github.com/silveX89/woossh/config"
	"github.com/silveX89/woossh/model"
)

func TestParseSlashPrefixes_NoFlags(t *testing.T) {
	target, flags := ParseSlashPrefixes("server1")
	if target != "server1" {
		t.Errorf("ParseSlashPrefixes(\"server1\") target = %q, want server1", target)
	}
	if flags.Any() {
		t.Error("expected no flags set")
	}
}

func TestParseSlashPrefixes_SingleFlag(t *testing.T) {
	tests := []struct {
		input  string
		target string
		check  func(Flags) bool
	}{
		{"/o server1", "server1", func(f Flags) bool { return f.BypassJumphost }},
		{"/v server1", "server1", func(f Flags) bool { return f.Verbose }},
		{"/d server1", "server1", func(f Flags) bool { return f.DryRun }},
		{"/l server1", "server1", func(f Flags) bool { return f.Legacy }},
		{"/c server1", "server1", func(f Flags) bool { return f.CopyCmd }},
		{"/t server1", "server1", func(f Flags) bool { return f.UseTmux }},
	}

	for _, tt := range tests {
		target, flags := ParseSlashPrefixes(tt.input)
		if target != tt.target {
			t.Errorf("ParseSlashPrefixes(%q) target = %q, want %q", tt.input, target, tt.target)
		}
		if !tt.check(flags) {
			t.Errorf("ParseSlashPrefixes(%q) flag not set", tt.input)
		}
	}
}

func TestParseSlashPrefixes_StackedFlags(t *testing.T) {
	target, flags := ParseSlashPrefixes("/o/v/d server1")
	if target != "server1" {
		t.Errorf("target = %q, want server1", target)
	}
	if !flags.BypassJumphost {
		t.Error("BypassJumphost not set")
	}
	if !flags.Verbose {
		t.Error("Verbose not set")
	}
	if !flags.DryRun {
		t.Error("DryRun not set")
	}
	if flags.Legacy || flags.CopyCmd || flags.UseTmux {
		t.Error("unexpected flags set")
	}
}

func TestParseSlashPrefixes_AllFlags(t *testing.T) {
	target, flags := ParseSlashPrefixes("/o/v/d/l/c/t server1")
	if target != "server1" {
		t.Errorf("target = %q, want server1", target)
	}
	if !flags.BypassJumphost || !flags.Verbose || !flags.DryRun || !flags.Legacy || !flags.CopyCmd || !flags.UseTmux {
		t.Error("not all flags were set")
	}
}

func TestParseSlashPrefixes_EmptyInput(t *testing.T) {
	target, flags := ParseSlashPrefixes("")
	if target != "" {
		t.Errorf("target = %q, want empty", target)
	}
	if flags.Any() {
		t.Error("expected no flags for empty input")
	}
}

func TestParseSlashPrefixes_OnlyFlagsNoTarget(t *testing.T) {
	target, flags := ParseSlashPrefixes("/o/v")
	if target != "" {
		t.Errorf("target = %q, want empty", target)
	}
	if !flags.BypassJumphost || !flags.Verbose {
		t.Error("expected flags to be set")
	}
}

func TestParseSlashPrefixes_SpacesHandling(t *testing.T) {
	target, flags := ParseSlashPrefixes("  /o /v   myserver  ")
	if target != "myserver" {
		t.Errorf("target = %q, want myserver", target)
	}
	if !flags.BypassJumphost || !flags.Verbose {
		t.Error("expected flags to be set")
	}
}

func TestParseSlashPrefixes_NoFlagPrefix(t *testing.T) {
	// /s is not a flag, should stay as part of target
	target, flags := ParseSlashPrefixes("/something")
	if target != "/something" {
		t.Errorf("target = %q, want /something", target)
	}
	if flags.Any() {
		t.Error("expected no flags for /something")
	}
}

func TestFlagString_Empty(t *testing.T) {
	var f Flags
	if s := f.FlagString(); s != "" {
		t.Errorf("FlagString() = %q, want empty", s)
	}
}

func TestFlagString_Single(t *testing.T) {
	f := Flags{BypassJumphost: true}
	if s := f.FlagString(); s != "[o]" {
		t.Errorf("FlagString({BypassJumphost}) = %q, want [o]", s)
	}
}

func TestFlagString_Multiple(t *testing.T) {
	f := Flags{DryRun: true, Verbose: true, UseTmux: true}
	s := f.FlagString()
	if !strings.Contains(s, "[d]") || !strings.Contains(s, "[v]") || !strings.Contains(s, "[t]") {
		t.Errorf("FlagString() = %q, should contain [d], [v], [t]", s)
	}
}

func TestAny_NoFlags(t *testing.T) {
	var f Flags
	if f.Any() {
		t.Error("Any() = true for empty flags")
	}
}

func TestAny_WithFlags(t *testing.T) {
	f := Flags{Verbose: true}
	if !f.Any() {
		t.Error("Any() = false when Verbose is set")
	}
}

func TestBuildArgs_Simple(t *testing.T) {
	// BuildArgs with no flags and no cfg SSHUser
	entry := model.HostEntry{Hostname: "server1", Host: "10.0.0.1", User: ""}
	cfg := config.Config{}
	var flags Flags

	args := BuildArgs(entry, cfg, flags)
	if len(args) != 1 || args[0] != "10.0.0.1" {
		t.Fatalf("BuildArgs simple = %v, want [10.0.0.1]", args)
	}
	
	// With User set on entry
	entry2 := model.HostEntry{Hostname: "server1", Host: "10.0.0.1", User: "admin"}
	args2 := BuildArgs(entry2, cfg, flags)
	if len(args2) != 1 || args2[0] != "admin@10.0.0.1" {
		t.Fatalf("BuildArgs with user = %v, want [admin@10.0.0.1]", args2)
	}
}

func TestBuildArgs_WithUser(t *testing.T) {
	entry := model.HostEntry{Hostname: "server1", Host: "10.0.0.1"}
	cfg := config.Config{SSHUser: "admin"}
	var flags Flags

	args := BuildArgs(entry, cfg, flags)
	expected := "admin@10.0.0.1"
	if len(args) != 1 || args[0] != expected {
		t.Fatalf("BuildArgs with user = %v, want [%s]", args, expected)
	}
}

func TestBuildArgs_Verbose(t *testing.T) {
	entry := model.HostEntry{Hostname: "server1", Host: "10.0.0.1"}
	flags := Flags{Verbose: true}

	args := BuildArgs(entry, config.Config{}, flags)
	if len(args) < 2 || args[0] != "-v" {
		t.Fatalf("BuildArgs verbose = %v, want first arg -v", args)
	}
}

func TestBuildArgs_CustomPort(t *testing.T) {
	entry := model.HostEntry{Hostname: "server1", Host: "10.0.0.1", Port: 2222}
	args := BuildArgs(entry, config.Config{}, Flags{})

	found := false
	for i, a := range args {
		if a == "-p" && i+1 < len(args) && args[i+1] == "2222" {
			found = true
		}
	}
	if !found {
		t.Fatalf("BuildArgs with port = %v, expected -p 2222", args)
	}
}

func TestBuildArgs_JumpHost(t *testing.T) {
	entry := model.HostEntry{Hostname: "server1", Host: "10.0.0.1", JumpHost: "bastion", JumpUser: "jump"}
	args := BuildArgs(entry, config.Config{}, Flags{})

	found := false
	for i, a := range args {
		if a == "-J" && i+1 < len(args) && args[i+1] == "jump@bastion" {
			found = true
		}
	}
	if !found {
		t.Fatalf("BuildArgs with jumphost = %v, expected -J jump@bastion", args)
	}
}

func TestBuildArgs_BypassJumphost(t *testing.T) {
	entry := model.HostEntry{Hostname: "server1", Host: "10.0.0.1", JumpHost: "bastion"}
	flags := Flags{BypassJumphost: true}
	args := BuildArgs(entry, config.Config{}, flags)

	for i, a := range args {
		if a == "-J" {
			t.Fatalf("BypassJumphost should prevent -J arg, but found at index %d: %v", i, args)
		}
	}
}

func TestBuildArgs_GlobalJumphost(t *testing.T) {
	entry := model.HostEntry{Hostname: "server1", Host: "10.0.0.1"}
	cfg := config.Config{GlobalJumphost: true, JumpServer: "bastion.example.com", JumpUser: "jump"}
	args := BuildArgs(entry, cfg, Flags{})

	found := false
	for i, a := range args {
		if a == "-J" && i+1 < len(args) && args[i+1] == "jump@bastion.example.com" {
			found = true
		}
	}
	if !found {
		t.Fatalf("BuildArgs with global jumphost = %v, expected -J jump@bastion.example.com", args)
	}
}

func TestBuildArgs_LegacyFlag(t *testing.T) {
	entry := model.HostEntry{Hostname: "server1", Host: "10.0.0.1"}
	flags := Flags{Legacy: true}
	args := BuildArgs(entry, config.Config{}, flags)

	foundHostKey := false
	for i, a := range args {
		if a == "-o" && i+1 < len(args) && strings.Contains(args[i+1], "HostKeyAlgorithms") {
			foundHostKey = true
		}
	}
	if !foundHostKey {
		t.Fatalf("BuildArgs with legacy = %v, expected -o HostKeyAlgorithms=+ssh-rsa", args)
	}
}

func TestBuildArgs_LegacyEntry(t *testing.T) {
	entry := model.HostEntry{Hostname: "server1", Host: "10.0.0.1", Legacy: true}
	args := BuildArgs(entry, config.Config{}, Flags{})

	foundHostKey := false
	for i, a := range args {
		if a == "-o" && i+1 < len(args) && strings.Contains(args[i+1], "HostKeyAlgorithms") {
			foundHostKey = true
		}
	}
	if !foundHostKey {
		t.Fatalf("BuildArgs with legacy entry = %v, expected legacy SSH options", args)
	}
}

func TestBuildArgs_HostFallback(t *testing.T) {
	// When Host is empty, use Hostname
	entry := model.HostEntry{Hostname: "myserver", User: "admin"}
	args := BuildArgs(entry, config.Config{}, Flags{})
	if len(args) != 1 || args[0] != "admin@myserver" {
		t.Fatalf("BuildArgs with empty Host = %v, want [admin@myserver]", args)
	}
}

func TestCommandLine(t *testing.T) {
	entry := model.HostEntry{Hostname: "fw01", Host: "192.168.1.1"}
	line := CommandLine(entry, config.Config{}, Flags{})
	expected := "ssh 192.168.1.1"
	if line != expected {
		t.Errorf("CommandLine = %q, want %q", line, expected)
	}
}

func TestCommandLine_WithFlags(t *testing.T) {
	entry := model.HostEntry{Hostname: "fw01", Host: "192.168.1.1", Port: 2222, User: "admin"}
	flags := Flags{Verbose: true}
	line := CommandLine(entry, config.Config{}, flags)
	if !strings.Contains(line, "ssh -v") {
		t.Errorf("CommandLine = %q, should contain ssh -v", line)
	}
	if !strings.Contains(line, "-p 2222") {
		t.Errorf("CommandLine = %q, should contain -p 2222", line)
	}
	if !strings.Contains(line, "admin@192.168.1.1") {
		t.Errorf("CommandLine = %q, should contain admin@192.168.1.1", line)
	}
}