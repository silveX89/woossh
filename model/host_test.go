package model

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFullFormat(t *testing.T) {
	csv := `hostname,host,port,user,jumphost,jumpuser,notes,legacy
firewall,192.168.1.1,22,admin,,,Edge Firewall,
loadbalancer,192.168.1.10,8022,ubuntu,bastion,jumpuser,HAProxy,
alteKiste,10.0.0.5,2222,root,,,Veralteter Server,yes
`
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	woosshDir := filepath.Join(tmpDir, "woossh")
	os.MkdirAll(woosshDir, 0o700)
	os.WriteFile(filepath.Join(woosshDir, "hosts.csv"), []byte(csv), 0o600)

	hosts, err := LoadHosts("")
	if err != nil {
		t.Fatalf("LoadHosts() error: %v", err)
	}
	if len(hosts) != 3 {
		t.Fatalf("expected 3 hosts, got %d", len(hosts))
	}

	h := hosts[0]
	if h.Hostname != "alteKiste" {
		t.Errorf("hosts[0].Hostname = %q, want alteKiste (alphabetical)", h.Hostname)
	}
	if h.Host != "10.0.0.5" {
		t.Errorf("alteKiste.Host = %q, want 10.0.0.5", h.Host)
	}
	if h.Port != 2222 {
		t.Errorf("alteKiste.Port = %d, want 2222", h.Port)
	}
	if h.User != "root" {
		t.Errorf("alteKiste.User = %q, want root", h.User)
	}
	if !h.Legacy {
		t.Error("alteKiste.Legacy = false, want true")
	}

	h2 := hosts[1]
	if h2.Hostname != "firewall" {
		t.Errorf("hosts[1].Hostname = %q, want firewall", h2.Hostname)
	}
	if h2.Host != "192.168.1.1" {
		t.Errorf("firewall.Host = %q, want 192.168.1.1", h2.Host)
	}
	if h2.Port != 22 {
		t.Errorf("firewall.Port = %d, want 22", h2.Port)
	}
	if h2.User != "admin" {
		t.Errorf("firewall.User = %q, want admin", h2.User)
	}
	if h2.Notes != "Edge Firewall" {
		t.Errorf("firewall.Notes = %q, want Edge Firewall", h2.Notes)
	}

	h3 := hosts[2]
	if h3.Hostname != "loadbalancer" {
		t.Errorf("hosts[2].Hostname = %q, want loadbalancer", h3.Hostname)
	}
	if h3.Port != 8022 {
		t.Errorf("loadbalancer.Port = %d, want 8022", h3.Port)
	}
	if h3.User != "ubuntu" {
		t.Errorf("loadbalancer.User = %q, want ubuntu", h3.User)
	}
	if h3.JumpHost != "bastion" {
		t.Errorf("loadbalancer.JumpHost = %q, want bastion", h3.JumpHost)
	}
	if h3.JumpUser != "jumpuser" {
		t.Errorf("loadbalancer.JumpUser = %q, want jumpuser", h3.JumpUser)
	}
	if h3.Notes != "HAProxy" {
		t.Errorf("loadbalancer.Notes = %q, want HAProxy", h3.Notes)
	}
}

func TestParseNameIPFormat(t *testing.T) {
	csv := `name,ip address
Mein-Switch,192.168.10.5
Core-Switch,192.168.10.1
`
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	woosshDir := filepath.Join(tmpDir, "woossh")
	os.MkdirAll(woosshDir, 0o700)
	os.WriteFile(filepath.Join(woosshDir, "hosts.csv"), []byte(csv), 0o600)

	hosts, err := LoadHosts("")
	if err != nil {
		t.Fatalf("LoadHosts() error: %v", err)
	}
	if len(hosts) != 2 {
		t.Fatalf("expected 2 hosts, got %d", len(hosts))
	}

	if hosts[0].Hostname != "Core-Switch" {
		t.Errorf("hosts[0].Hostname = %q, want Core-Switch (alphabetical)", hosts[0].Hostname)
	}
	if hosts[0].Host != "192.168.10.1" {
		t.Errorf("Core-Switch.Host = %q, want 192.168.10.1", hosts[0].Host)
	}
	if hosts[1].Hostname != "Mein-Switch" {
		t.Errorf("hosts[1].Hostname = %q, want Mein-Switch", hosts[1].Hostname)
	}
	if hosts[1].Host != "192.168.10.5" {
		t.Errorf("Mein-Switch.Host = %q, want 192.168.10.5", hosts[1].Host)
	}
}

func TestParseTwoColFormat(t *testing.T) {
	csv := `host,addr
webserver,10.0.1.100
db-server,10.0.1.101
`
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	woosshDir := filepath.Join(tmpDir, "woossh")
	os.MkdirAll(woosshDir, 0o700)
	os.WriteFile(filepath.Join(woosshDir, "hosts.csv"), []byte(csv), 0o600)

	hosts, err := LoadHosts("")
	if err != nil {
		t.Fatalf("LoadHosts() error: %v", err)
	}
	if len(hosts) != 2 {
		t.Fatalf("expected 2 hosts, got %d", len(hosts))
	}

	if hosts[0].Hostname != "db-server" {
		t.Errorf("hosts[0].Hostname = %q, want db-server", hosts[0].Hostname)
	}
	if hosts[0].Host != "10.0.1.101" {
		t.Errorf("db-server.Host = %q, want 10.0.1.101", hosts[0].Host)
	}
}

func TestParsePlainList(t *testing.T) {
	plain := `server1.example.com
router.lan
192.168.1.50
`
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	woosshDir := filepath.Join(tmpDir, "woossh")
	os.MkdirAll(woosshDir, 0o700)
	os.WriteFile(filepath.Join(woosshDir, "hosts.csv"), []byte(plain), 0o600)

	hosts, err := LoadHosts("")
	if err != nil {
		t.Fatalf("LoadHosts() error: %v", err)
	}
	if len(hosts) != 3 {
		t.Fatalf("expected 3 hosts, got %d", len(hosts))
	}

	// Plain list: first col = Hostname, no Host/IP field
	if hosts[0].Hostname != "192.168.1.50" {
		t.Errorf("hosts[0].Hostname = %q, want 192.168.1.50", hosts[0].Hostname)
	}
	if hosts[1].Hostname != "router.lan" {
		t.Errorf("hosts[1].Hostname = %q, want router.lan", hosts[1].Hostname)
	}
}

func TestLoadHostsCommentsIgnored(t *testing.T) {
	csv := `hostname,host
# this is a comment
server1,10.0.0.1


server2,10.0.0.2
`
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	woosshDir := filepath.Join(tmpDir, "woossh")
	os.MkdirAll(woosshDir, 0o700)
	os.WriteFile(filepath.Join(woosshDir, "hosts.csv"), []byte(csv), 0o600)

	hosts, err := LoadHosts("")
	if err != nil {
		t.Fatalf("LoadHosts() error: %v", err)
	}
	if len(hosts) != 2 {
		t.Fatalf("expected 2 hosts, got %d", len(hosts))
	}
}

func TestFindEntryExactMatch(t *testing.T) {
	hosts := []HostEntry{
		{Hostname: "server1", Host: "10.0.0.1"},
		{Hostname: "server2", Host: "10.0.0.2"},
	}
	e := FindEntry(hosts, "server1")
	if e.Hostname != "server1" {
		t.Errorf("FindEntry = %q, want server1", e.Hostname)
	}
}

func TestFindEntryUniquePrefix(t *testing.T) {
	hosts := []HostEntry{
		{Hostname: "firewall01", Host: "10.0.0.1"},
		{Hostname: "switch01", Host: "10.0.0.2"},
	}
	e := FindEntry(hosts, "fire")
	if e.Hostname != "firewall01" {
		t.Errorf("FindEntry(\"fire\") = %q, want firewall01", e.Hostname)
	}
}

func TestFindEntryByIP(t *testing.T) {
	hosts := []HostEntry{
		{Hostname: "server1", Host: "10.0.0.1"},
		{Hostname: "server2", Host: "10.0.0.2"},
	}
	e := FindEntry(hosts, "10.0.0.2")
	if e.Hostname != "server2" {
		t.Errorf("FindEntry by IP = %q, want server2", e.Hostname)
	}
}

func TestFindEntryLiteralFallback(t *testing.T) {
	hosts := []HostEntry{
		{Hostname: "server1", Host: "10.0.0.1"},
	}
	e := FindEntry(hosts, "unknown")
	if e.Hostname != "unknown" {
		t.Errorf("FindEntry(\"unknown\") = %q, want unknown (literal fallback)", e.Hostname)
	}
}

func TestMergeHostsNewOnly(t *testing.T) {
	existing := []HostEntry{
		{Hostname: "server1", Host: "10.0.0.1"},
	}
	newEntries := []HostEntry{
		{Hostname: "server2", Host: "10.0.0.2"},
	}
	merged := MergeHosts(existing, newEntries)
	if len(merged) != 2 {
		t.Fatalf("expected 2 merged hosts, got %d", len(merged))
	}
}

func TestMergeHostsNoOverwrite(t *testing.T) {
	existing := []HostEntry{
		{Hostname: "server1", Host: "10.0.0.1", User: "admin"},
	}
	newEntries := []HostEntry{
		{Hostname: "server1", Host: "10.0.0.99", User: "root"},
	}
	merged := MergeHosts(existing, newEntries)
	if len(merged) != 1 {
		t.Fatalf("expected 1 merged host (existing preserved), got %d", len(merged))
	}
	if merged[0].User != "admin" {
		t.Errorf("existing server1.User overwritten! got %q, want admin", merged[0].User)
	}
	if merged[0].Host != "10.0.0.1" {
		t.Errorf("existing server1.Host overwritten! got %q, want 10.0.0.1", merged[0].Host)
	}
}

func TestImportSSHConfig(t *testing.T) {
	content := `Host myserver
    HostName 192.168.1.100
    User admin
    Port 2222

Host *.example.com
    User ubuntu

Host bastion
    HostName bastion.example.com
    User jumpuser

Host behind-bastion
    HostName 10.0.0.5
    User app
    ProxyJump jumpuser@bastion.example.com

Host *
    User nobody
`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config")
	os.WriteFile(path, []byte(content), 0o600)

	entries, err := ImportSSHConfig(path)
	if err != nil {
		t.Fatalf("ImportSSHConfig error: %v", err)
	}

	// Should have 3 entries (myserver, bastion, behind-bastion)
	// *.example.com and * are skipped
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	if entries[0].Hostname != "myserver" {
		t.Errorf("entries[0].Hostname = %q, want myserver", entries[0].Hostname)
	}
	if entries[0].Host != "192.168.1.100" {
		t.Errorf("myserver.Host = %q, want 192.168.1.100", entries[0].Host)
	}
	if entries[0].User != "admin" {
		t.Errorf("myserver.User = %q, want admin", entries[0].User)
	}
	if entries[0].Port != 2222 {
		t.Errorf("myserver.Port = %d, want 2222", entries[0].Port)
	}

	if entries[2].Hostname != "behind-bastion" {
		t.Errorf("entries[2].Hostname = %q, want behind-bastion", entries[2].Hostname)
	}
	if entries[2].JumpHost != "bastion.example.com" {
		t.Errorf("behind-bastion.JumpHost = %q, want bastion.example.com", entries[2].JumpHost)
	}
	if entries[2].JumpUser != "jumpuser" {
		t.Errorf("behind-bastion.JumpUser = %q, want jumpuser", entries[2].JumpUser)
	}
}
