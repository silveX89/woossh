package plugin

import (
	"github.com/silveX89/woossh/config"
	"github.com/silveX89/woossh/model"
)

// Context defines what a plugin can see and do.
// No direct dependency on tui/ or main/ to prevent import cycles.
type Context struct {
	Config   *config.Config
	Hosts    *[]model.HostEntry
	Hooks    HookRegistry
	Settings PluginSettings
}

// PluginSettings is a simple key-value interface for plugin configuration.
type PluginSettings interface {
	Get(key string) string
	Set(key, value string)
	GetBool(key string) bool
	GetInt(key string) int
}

// NoopPlugin is an empty base implementation. Plugins can embed it and
// override only the methods they need.
type NoopPlugin struct{}

func (NoopPlugin) ID() string                 { return "" }
func (NoopPlugin) Manifest() Manifest         { return Manifest{} }
func (NoopPlugin) Init(*Context) error        { return nil }
func (NoopPlugin) Enable() error              { return nil }
func (NoopPlugin) Disable() error             { return nil }