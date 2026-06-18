package tui

import (
	"os"
	"strings"
	"testing"

	"github.com/silveX89/woossh/config"
	"github.com/silveX89/woossh/model"
	"github.com/silveX89/woossh/plugin"
)

// testSettingsPlugin is a dummy plugin with settings for testing.
type testSettingsPlugin struct {
	plugin.NoopPlugin
}

func (p *testSettingsPlugin) ID() string { return "test-settings-plugin" }
func (p *testSettingsPlugin) Manifest() plugin.Manifest {
	return plugin.Manifest{
		ID:      "test-settings-plugin",
		Name:    "Test Settings Plugin",
		Version: "1.0.0",
		Settings: []plugin.SettingDef{
			{Key: "enable_feature", Label: "Feature aktivieren", Kind: "bool", Default: "false", Category: "General"},
			{Key: "server_url", Label: "Server URL", Kind: "string", Default: "https://example.com", Category: "General"},
			{Key: "refresh_interval", Label: "Refresh Intervall", Kind: "int", Default: "30", Category: "General"},
		},
	}
}

func init() {
	if !plugin.IsRegistered("test-settings-plugin") {
		plugin.Register(&testSettingsPlugin{})
	}
}

func TestPluginSettingsViewRenders(t *testing.T) {
	cfg, _ := config.Load()
	hosts, _ := model.LoadHosts(cfg.HostsPath)
	mgr := plugin.NewManager(&cfg, &hosts)
	_ = mgr.LoadState()
	mgr.InitAll()

	m := initialModel(cfg, hosts, "v0.3.1", mgr)
	m.mode = modePluginSettings
	m.width = 80
	m.height = 40

	// Set a plugin ID that has Settings defined
	m.pluginSettingsPluginID = "test-settings-plugin"

	// Render should work without panic
	view := m.pluginSettingsView()
	if len(view) == 0 {
		t.Fatal("pluginSettingsView() returned empty string")
	}
	if !strings.Contains(view, "Feature aktivieren") {
		t.Error("expected 'Feature aktivieren' in plugin settings view")
	}
	if !strings.Contains(view, "Server URL") {
		t.Error("expected 'Server URL' in plugin settings view")
	}
}

func TestPluginSettingsPersist(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cfg, _ := config.Load()
	hosts, _ := model.LoadHosts(cfg.HostsPath)
	mgr := plugin.NewManager(&cfg, &hosts)
	_ = mgr.LoadState()
	mgr.InitAll()

	// Set a setting
	mgr.SetPluginSetting("test-settings-plugin", "server_url", "https://internal.example.com")

	// Read it back
	val := mgr.GetPluginSetting("test-settings-plugin", "server_url")
	if val != "https://internal.example.com" {
		t.Fatalf("expected 'https://internal.example.com', got %q", val)
	}

	// Verify it was persisted to disk
	stateFile := tmpDir + "/woossh/plugins.yaml"
	data, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("failed to read state file: %v", err)
	}
	if !strings.Contains(string(data), "https://internal.example.com") {
		t.Error("expected 'https://internal.example.com' in persisted state file")
	}
}

func TestPluginSettingsEditFlow(t *testing.T) {
	cfg, _ := config.Load()
	hosts, _ := model.LoadHosts(cfg.HostsPath)
	mgr := plugin.NewManager(&cfg, &hosts)
	_ = mgr.LoadState()
	mgr.InitAll()

	m := initialModel(cfg, hosts, "v0.3.1", mgr)
	m.pluginSettingsPluginID = "test-settings-plugin"
	m.pluginMgr = mgr

	// Set edit buffer and apply
	m.pluginSettingsOffset = 1 // "server_url" is the second setting
	m.pluginSettingsEditBuf = "https://custom.example.com"
	m.applyPluginSettingEdit()

	// Verify
	val := mgr.GetPluginSetting("test-settings-plugin", "server_url")
	if val != "https://custom.example.com" {
		t.Fatalf("expected 'https://custom.example.com', got %q", val)
	}
}