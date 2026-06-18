package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/silveX89/woossh/config"
	"github.com/silveX89/woossh/model"
	"gopkg.in/yaml.v3"
)

// PluginRepo holds the configuration for a remote plugin repository.
type PluginRepo struct {
	URL     string `yaml:"url"`
	Name    string `yaml:"name"`
	Enabled bool   `yaml:"enabled"`
}

// PluginStatus represents the lifecycle state of a plugin.
type PluginStatus string

const (
	PluginAvailable  PluginStatus = "available"  // known from repo, not yet cloned
	PluginDownloaded PluginStatus = "downloaded" // cloned to local disk
	PluginStatusEnabled PluginStatus = "enabled" // downloaded + activated
)

// State is persisted in ~/.config/woossh/plugins.yaml.
type State struct {
	Enabled   map[string]bool              `yaml:"enabled"`
	Versions  map[string]string            `yaml:"versions"`
	Sources   map[string]string            `yaml:"sources"`
	Settings  map[string]map[string]string `yaml:"settings,omitempty"`
	Repos     []PluginRepo                 `yaml:"repos"`
}

// managerSettings is a PluginSettings implementation backed by Manager state
// with automatic persistence.
type managerSettings struct {
	state  *State
	id     string
	saveFn func() error
}

func (ms *managerSettings) Get(key string) string {
	if ms.state.Settings == nil || ms.state.Settings[ms.id] == nil {
		return ""
	}
	return ms.state.Settings[ms.id][key]
}

func (ms *managerSettings) Set(key, value string) {
	if ms.state.Settings == nil {
		ms.state.Settings = make(map[string]map[string]string)
	}
	if ms.state.Settings[ms.id] == nil {
		ms.state.Settings[ms.id] = make(map[string]string)
	}
	ms.state.Settings[ms.id][key] = value
	_ = ms.saveFn()
}

func (ms *managerSettings) GetBool(key string) bool {
	v := ms.Get(key)
	return v == "yes" || v == "true" || v == "1"
}

func (ms *managerSettings) GetInt(key string) int {
	v := ms.Get(key)
	if v == "" {
		return 0
	}
	n := 0
	fmt.Sscanf(v, "%d", &n)
	return n
}

// pluginSettings is a standalone in-memory PluginSettings implementation.
type pluginSettings struct {
	mu     sync.RWMutex
	values map[string]string
}

func newPluginSettings() *pluginSettings {
	return &pluginSettings{values: make(map[string]string)}
}

func (ps *pluginSettings) Get(key string) string {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.values[key]
}

func (ps *pluginSettings) Set(key, value string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.values[key] = value
}

func (ps *pluginSettings) GetBool(key string) bool {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	v := ps.values[key]
	return v == "yes" || v == "true" || v == "1"
}

func (ps *pluginSettings) GetInt(key string) int {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	v := ps.values[key]
	if v == "" {
		return 0
	}
	n := 0
	fmt.Sscanf(v, "%d", &n)
	return n
}

// Manager handles the plugin lifecycle.
type Manager struct {
	ctx       *Context
	state     State
	statePath string
	hooks     *hookDispatcher
}

// NewManager creates a new plugin manager.
func NewManager(cfg *config.Config, hosts *[]model.HostEntry) *Manager {
	hooks := newHookDispatcher()
	ctx := &Context{
		Config:   cfg,
		Hosts:    hosts,
		Hooks:    hooks,
		Settings: newPluginSettings(),
	}
	return &Manager{
		ctx:       ctx,
		statePath: stateFilePath(),
		hooks:     hooks,
		state: State{
			Enabled:  make(map[string]bool),
			Versions: make(map[string]string),
			Sources:  make(map[string]string),
			Settings: make(map[string]map[string]string),
			Repos:    defaultRepos(),
		},
	}
}

// defaultRepos returns the default plugin repository list.
func defaultRepos() []PluginRepo {
	return []PluginRepo{
		{URL: "github.com/silveX89/woossh-plugins", Name: "woossh-plugins", Enabled: true},
	}
}

// Hooks returns the HookDispatcher.
func (m *Manager) Hooks() *hookDispatcher {
	return m.hooks
}

// Context returns the plugin context.
func (m *Manager) Context() *Context {
	return m.ctx
}

// LoadState reads plugins.yaml from disk.
func (m *Manager) LoadState() error {
	data, err := os.ReadFile(m.statePath)
	if os.IsNotExist(err) {
		m.state = State{
			Enabled:  make(map[string]bool),
			Versions: make(map[string]string),
			Sources:  make(map[string]string),
			Repos:    defaultRepos(),
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s: %w", m.statePath, err)
	}
	if err := yaml.Unmarshal(data, &m.state); err != nil {
		return fmt.Errorf("parse %s: %w", m.statePath, err)
	}
	if m.state.Enabled == nil {
		m.state.Enabled = make(map[string]bool)
	}
	if m.state.Versions == nil {
		m.state.Versions = make(map[string]string)
	}
	if m.state.Sources == nil {
		m.state.Sources = make(map[string]string)
	}
	if m.state.Settings == nil {
		m.state.Settings = make(map[string]map[string]string)
	}
	if len(m.state.Repos) == 0 {
		m.state.Repos = defaultRepos()
	}
	return nil
}

// SaveState writes plugins.yaml to disk.
func (m *Manager) SaveState() error {
	_ = os.MkdirAll(filepath.Dir(m.statePath), 0o700)
	data, err := yaml.Marshal(m.state)
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	return os.WriteFile(m.statePath, data, 0o600)
}

// InitAll initializes all registered plugins and enables those marked in state.
func (m *Manager) InitAll() {
	for _, p := range All() {
		ctx := &Context{
			Config:   m.ctx.Config,
			Hosts:    m.ctx.Hosts,
			Hooks:    m.hooks,
			Settings: m.pluginSettingsFor(p.ID()),
		}
		if err := p.Init(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "woossh: plugin %q init: %v\n", p.ID(), err)
			continue
		}
		if m.state.Enabled[p.ID()] {
			if err := p.Enable(); err != nil {
				fmt.Fprintf(os.Stderr, "woossh: plugin %q enable: %v\n", p.ID(), err)
				m.state.Enabled[p.ID()] = false
			}
		}
	}
}

// IsEnabled reports whether a plugin is currently enabled.
func (m *Manager) IsEnabled(id string) bool {
	return m.state.Enabled[id]
}

// Enable activates a plugin at runtime.
func (m *Manager) Enable(id string) error {
	p := FindByID(id)
	if p == nil {
		return fmt.Errorf("plugin %q not found", id)
	}
	if err := p.Enable(); err != nil {
		return fmt.Errorf("enable %q: %w", id, err)
	}
	m.state.Enabled[id] = true
	return m.SaveState()
}

// Disable deactivates a plugin at runtime.
func (m *Manager) Disable(id string) error {
	p := FindByID(id)
	if p == nil {
		return fmt.Errorf("plugin %q not found", id)
	}
	if err := p.Disable(); err != nil {
		return fmt.Errorf("disable %q: %w", id, err)
	}
	m.state.Enabled[id] = false
	return m.SaveState()
}

// Shutdown disables all active plugins.
func (m *Manager) Shutdown() {
	for _, p := range All() {
		if m.state.Enabled[p.ID()] {
			_ = p.Disable()
		}
	}
}

// pluginSettingsFor returns a PluginSettings backed by Manager state with persistence.
func (m *Manager) pluginSettingsFor(id string) PluginSettings {
	return &managerSettings{
		state:  &m.state,
		id:     id,
		saveFn: m.SaveState,
	}
}

// GetPluginSetting returns a named setting value for a plugin.
func (m *Manager) GetPluginSetting(pluginID, key string) string {
	if m.state.Settings == nil || m.state.Settings[pluginID] == nil {
		return ""
	}
	return m.state.Settings[pluginID][key]
}

// SetPluginSetting sets a named setting value for a plugin and saves state.
func (m *Manager) SetPluginSetting(pluginID, key, value string) {
	if m.state.Settings == nil {
		m.state.Settings = make(map[string]map[string]string)
	}
	if m.state.Settings[pluginID] == nil {
		m.state.Settings[pluginID] = make(map[string]string)
	}
	m.state.Settings[pluginID][key] = value
	_ = m.SaveState()
}

// PluginManifest returns the manifest for a plugin by ID, or nil if not found.
func (m *Manager) PluginManifest(id string) *Manifest {
	p := FindByID(id)
	if p == nil {
		return nil
	}
	mf := p.Manifest()
	return &mf
}

// GetRepos returns the current list of plugin repositories.
func (m *Manager) GetRepos() []PluginRepo {
	return m.state.Repos
}

// AddRepo adds a new plugin repository to the list.
func (m *Manager) AddRepo(repo PluginRepo) error {
	for _, r := range m.state.Repos {
		if r.URL == repo.URL {
			return fmt.Errorf("repo %q already exists", repo.URL)
		}
	}
	m.state.Repos = append(m.state.Repos, repo)
	return m.SaveState()
}

// RemoveRepo removes a repo by URL.
func (m *Manager) RemoveRepo(url string) error {
	var newRepos []PluginRepo
	for _, r := range m.state.Repos {
		if r.URL != url {
			newRepos = append(newRepos, r)
		}
	}
	m.state.Repos = newRepos
	return m.SaveState()
}

// ToggleRepo enables or disables a repo by URL.
func (m *Manager) ToggleRepo(url string) error {
	for i, r := range m.state.Repos {
		if r.URL == url {
			m.state.Repos[i].Enabled = !m.state.Repos[i].Enabled
			return m.SaveState()
		}
	}
	return fmt.Errorf("repo %q not found", url)
}

func stateFilePath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "woossh", "plugins.yaml")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "woossh", "plugins.yaml")
}