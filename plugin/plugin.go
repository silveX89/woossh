package plugin

// Plugin is the core interface that every plugin must implement.
type Plugin interface {
	// ID returns the unique plugin ID (e.g. "woossh-tmux").
	ID() string

	// Manifest returns static plugin metadata.
	Manifest() Manifest

	// Init is called once at startup. The Context provides access to
	// Config, host list, and hook registry.
	Init(ctx *Context) error

	// Enable is called when the user activates the plugin.
	// Resources can be allocated here (e.g. start goroutines).
	Enable() error

	// Disable cleans up. After this call no hooks fire.
	Disable() error
}

// Manifest holds static plugin metadata.
type Manifest struct {
	ID          string       `yaml:"id"`
	Name        string       `yaml:"name"`
	Version     string       `yaml:"version"`
	Description string       `yaml:"description"`
	Author      string       `yaml:"author"`
	APIVersion  string       `yaml:"api_version"`
	MinWoossh   string       `yaml:"min_woossh"`
	RepoURL     string       `yaml:"repo_url"`
	License     string       `yaml:"license,omitempty"`
	Hooks       []string     `yaml:"hooks,omitempty"`
	Settings    []SettingDef `yaml:"settings,omitempty"`
	Permissions []string     `yaml:"permissions,omitempty"`
}

// SettingDef describes a single plugin setting (for the dynamic settings UI).
type SettingDef struct {
	Key       string   `yaml:"key"`
	Label     string   `yaml:"label"`
	Kind      string   `yaml:"kind"` // "bool", "string", "int", "enum"
	Default   string   `yaml:"default"`
	Category  string   `yaml:"category,omitempty"`
	EnumOpts  []string `yaml:"enum_opts,omitempty"`
}