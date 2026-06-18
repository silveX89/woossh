package tui

import (
	"testing"

	"github.com/silveX89/woossh/config"
	"github.com/silveX89/woossh/model"
	"github.com/silveX89/woossh/plugin"
)

// initPluginRegistry registers a test plugin for TUI plugin manager tests.
func init() {
	// Only register if not already registered (tests run in same process)
	if !plugin.IsRegistered("test-plugin") {
		plugin.Register(&dummyPlugin{})
	}
}

type dummyPlugin struct {
	plugin.NoopPlugin
}

func (d *dummyPlugin) ID() string        { return "test-plugin" }
func (d *dummyPlugin) Manifest() plugin.Manifest {
	return plugin.Manifest{
		ID:      "test-plugin",
		Name:    "Test Plugin",
		Version: "1.0.0",
		RepoURL: "github.com/test/test-plugin",
	}
}

func TestInitialModelWithPluginMgr(t *testing.T) {
	cfg, _ := config.Load()
	hosts, _ := model.LoadHosts(cfg.HostsPath)

	// Create a minimal manager
	mgr := plugin.NewManager(&cfg, &hosts)
	_ = mgr.LoadState()
	mgr.InitAll()

	m := initialModel(cfg, hosts, "v0.3.1", mgr)
	if m.pluginMgr == nil {
		t.Fatal("pluginMgr should not be nil")
	}
	if m.mode != modeHosts {
		t.Fatal("initial mode should be modeHosts")
	}
}

func TestPluginViewRenders(t *testing.T) {
	cfg, _ := config.Load()
	hosts, _ := model.LoadHosts(cfg.HostsPath)
	mgr := plugin.NewManager(&cfg, &hosts)

	m := initialModel(cfg, hosts, "v0.3.1", mgr)
	m.mode = modePluginManager
	m.width = 80
	m.height = 40

	// Render should work without panic
	view := m.pluginView()
	if len(view) == 0 {
		t.Fatal("pluginView() returned empty string")
	}

	// Should show the test plugin
	if !plugin.IsRegistered("test-plugin") {
		t.Fatal("test-plugin should be registered")
	}
}

func TestPluginManagerToggle(t *testing.T) {
	cfg, _ := config.Load()
	hosts, _ := model.LoadHosts(cfg.HostsPath)
	mgr := plugin.NewManager(&cfg, &hosts)
	_ = mgr.LoadState()

	// Register via init() from other test — already done
	if !plugin.IsRegistered("test-plugin") {
		t.Fatal("test-plugin should be registered")
	}

	// Enable the plugin
	err := mgr.Enable("test-plugin")
	if err != nil {
		t.Fatalf("Enable failed: %v", err)
	}
	if !mgr.IsEnabled("test-plugin") {
		t.Fatal("plugin should be enabled after Enable()")
	}

	// Disable the plugin
	err = mgr.Disable("test-plugin")
	if err != nil {
		t.Fatalf("Disable failed: %v", err)
	}
	if mgr.IsEnabled("test-plugin") {
		t.Fatal("plugin should be disabled after Disable()")
	}
}