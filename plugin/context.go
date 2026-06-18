package plugin

import (
	"github.com/silveX89/woossh/config"
	"github.com/silveX89/woossh/model"
)

// Context ist das "was ein Plugin sehen und tun darf".
// Keine direkte Abhängigkeit auf tui/ oder main/ — verhindert Zyklen.
type Context struct {
	Config   *config.Config
	Hosts    *[]model.HostEntry
	Hooks    HookRegistry
	Settings PluginSettings
}

// PluginSettings ist ein einfaches KV-Interface über config.ini (Plugin-Sektion).
type PluginSettings interface {
	Get(key string) string
	Set(key, value string)
	GetBool(key string) bool
	GetInt(key string) int
}

// NoopPlugin ist eine leere Basisimplementierung — Plugins können diese embedden,
// müssen dann nur die Methoden überschreiben die sie brauchen.
type NoopPlugin struct{}

func (NoopPlugin) ID() string                 { return "" }
func (NoopPlugin) Manifest() Manifest         { return Manifest{} }
func (NoopPlugin) Init(*Context) error        { return nil }
func (NoopPlugin) Enable() error              { return nil }
func (NoopPlugin) Disable() error             { return nil }