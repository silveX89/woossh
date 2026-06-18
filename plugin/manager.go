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

// State wird in ~/.config/woossh/plugins.yaml persistiert.
type State struct {
	Enabled  map[string]bool              `yaml:"enabled"`
	Versions map[string]string             `yaml:"versions"`
	Sources  map[string]string             `yaml:"sources"`
	Settings map[string]map[string]string  `yaml:"settings,omitempty"`
}

// managerSettings ist eine PluginSettings-Implementierung,
// die auf den Manager-State zugreift und persistiert wird.
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

// pluginSettings ist die alte Implementierung (wird nur noch in NewManager genutzt).
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

// Manager verwaltet den Plugin-Lebenszyklus.
type Manager struct {
	ctx       *Context
	state     State
	statePath string
	hooks     *hookDispatcher
}

// NewManager erstellt einen neuen Plugin-Manager.
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
		},
	}
}

// Hooks gibt den HookDispatcher zurück.
func (m *Manager) Hooks() *hookDispatcher {
	return m.hooks
}

// Context gibt den PluginContext zurück.
func (m *Manager) Context() *Context {
	return m.ctx
}

// LoadState liest plugins.yaml von Disk.
func (m *Manager) LoadState() error {
	data, err := os.ReadFile(m.statePath)
	if os.IsNotExist(err) {
		m.state = State{
			Enabled:  make(map[string]bool),
			Versions: make(map[string]string),
			Sources:  make(map[string]string),
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
	return nil
}

// SaveState schreibt plugins.yaml auf Disk.
func (m *Manager) SaveState() error {
	_ = os.MkdirAll(filepath.Dir(m.statePath), 0o700)
	data, err := yaml.Marshal(m.state)
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	return os.WriteFile(m.statePath, data, 0o600)
}

// InitAll initialisiert alle registrierten Plugins und aktiviert die laut State enabled.
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

// IsEnabled prüft ob ein Plugin aktiviert ist.
func (m *Manager) IsEnabled(id string) bool {
	return m.state.Enabled[id]
}

// Enable aktiviert ein Plugin zur Laufzeit.
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

// Disable deaktiviert ein Plugin zur Laufzeit.
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

// Shutdown deaktiviert alle aktiven Plugins.
func (m *Manager) Shutdown() {
	for _, p := range All() {
		if m.state.Enabled[p.ID()] {
			_ = p.Disable()
		}
	}
}

// pluginSettingsFor erstellt ein PluginSettings-Interface,
// das auf den State des Managers zugreift und persistiert wird.
func (m *Manager) pluginSettingsFor(id string) PluginSettings {
	return &managerSettings{
		state:  &m.state,
		id:     id,
		saveFn: m.SaveState,
	}
}

// GetPluginSetting gibt einen benannten Setting-Wert eines Plugins zurück.
func (m *Manager) GetPluginSetting(pluginID, key string) string {
	if m.state.Settings == nil || m.state.Settings[pluginID] == nil {
		return ""
	}
	return m.state.Settings[pluginID][key]
}

// SetPluginSetting setzt einen benannten Setting-Wert eines Plugins und persistiert.
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

// PluginManifest gibt das Manifest eines Plugins aus dem State zurück (oder nil).
func (m *Manager) PluginManifest(id string) *Manifest {
	p := FindByID(id)
	if p == nil {
		return nil
	}
	mf := p.Manifest()
	return &mf
}

func stateFilePath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "woossh", "plugins.yaml")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "woossh", "plugins.yaml")
}