package plugin

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/silveX89/woossh/model"
	sshpkg "github.com/silveX89/woossh/ssh"
)

type testPlugin struct {
	NoopPlugin
}

func (p *testPlugin) ID() string        { return "test-plugin" }
func (p *testPlugin) Manifest() Manifest { return Manifest{ID: "test-plugin", Name: "Test Plugin", Version: "1.0.0"} }

func TestRegisterAndFind(t *testing.T) {
	// Reset registry for test
	registry = nil

	p := &testPlugin{}
	Register(p)

	if !IsRegistered("test-plugin") {
		t.Fatal("expected test-plugin to be registered")
	}

	found := FindByID("test-plugin")
	if found == nil {
		t.Fatal("expected to find test-plugin")
	}

	all := All()
	if len(all) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(all))
	}
}

func TestHookDispatcher(t *testing.T) {
	d := newHookDispatcher()

	// BeforeConnect
	called := false
	d.OnBeforeConnect(func(entry model.HostEntry, flags sshpkg.Flags) (model.HostEntry, sshpkg.Flags, error) {
		called = true
		return entry, flags, nil
	})

	entry, flags, err := d.FireBeforeConnect(model.HostEntry{Hostname: "test"}, sshpkg.Flags{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("BeforeConnect handler was not called")
	}
	if entry.Hostname != "test" {
		t.Fatalf("expected hostname 'test', got %q", entry.Hostname)
	}

	// BeforeConnect error propagation
	d.OnBeforeConnect(func(entry model.HostEntry, flags sshpkg.Flags) (model.HostEntry, sshpkg.Flags, error) {
		return entry, flags, nil
	})
	_, _, err = d.FireBeforeConnect(entry, flags)
	if err != nil {
		t.Fatal("expected no error from pass-through handler")
	}

	// TUIKey
	keyHandled := false
	d.OnTUIKeyMsg(func(key string) (bool, tea.Cmd) {
		if key == "x" {
			keyHandled = true
			return true, nil
		}
		return false, nil
	})

	handled, _ := d.FireTUIKey("x")
	if !handled {
		t.Fatal("expected TUIKey 'x' to be handled")
	}
	if !keyHandled {
		t.Fatal("TUIKey handler was not called")
	}

	handled, _ = d.FireTUIKey("y")
	if handled {
		t.Fatal("expected TUIKey 'y' to not be handled")
	}

	// AfterConnect
	afterCalled := false
	d.OnAfterConnect(func(entry model.HostEntry, exitCode int) {
		if exitCode != 42 {
			t.Fatalf("expected exitCode 42, got %d", exitCode)
		}
		afterCalled = true
	})
	d.FireAfterConnect(entry, 42)
	if !afterCalled {
		t.Fatal("AfterConnect handler was not called")
	}

	// CLICommand
	d.OnCLICommand(func(args []string) (bool, int) {
		if len(args) > 0 && args[0] == "mycmd" {
			return true, 0
		}
		return false, 0
	})

	handled, code := d.FireCLICommand([]string{"mycmd"})
	if !handled {
		t.Fatal("expected CLICommand to be handled")
	}
	if code != 0 {
		t.Fatalf("expected exitCode 0, got %d", code)
	}

	// HostLoaded
	d.OnHostLoaded(func(hosts []model.HostEntry) []model.HostEntry {
		return append(hosts, model.HostEntry{Hostname: "injected"})
	})

	hosts := d.FireHostLoaded([]model.HostEntry{{Hostname: "original"}})
	if len(hosts) != 2 {
		t.Fatalf("expected 2 hosts, got %d", len(hosts))
	}
	if hosts[1].Hostname != "injected" {
		t.Fatalf("expected second host to be 'injected', got %q", hosts[1].Hostname)
	}
}

func TestTrustLevel(t *testing.T) {
	tests := []struct {
		url  string
		want TrustLevel
	}{
		{"github.com/silveX89/woossh-tmux", TrustOfficial},
		{"https://github.com/silveX89/woossh-bitwarden", TrustOfficial},
		{"github.com/other/woossh-plugin", TrustCommunity},
		{"gitlab.com/some/project", TrustUnknown},
		{"bitbucket.org/user/repo", TrustUnknown},
	}

	for _, tt := range tests {
		got := TrustLevelForURL(tt.url)
		if got != tt.want {
			t.Errorf("TrustLevelForURL(%q) = %v, want %v", tt.url, got, tt.want)
		}
	}
}

func TestConcurrentRegistry(t *testing.T) {
	registry = nil
	done := make(chan bool)

	go func() {
		Register(&testPlugin{})
		done <- true
	}()
	go func() {
		All()
		done <- true
	}()
	go func() {
		FindByID("test-plugin")
		done <- true
	}()

	for i := 0; i < 3; i++ {
		<-done
	}
}