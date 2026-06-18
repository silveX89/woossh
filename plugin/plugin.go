package plugin

// Plugin ist das Kern-Interface, das jedes Plugin implementieren muss.
type Plugin interface {
	// ID gibt die eindeutige Plugin-ID zurück (z.B. "woossh-tmux").
	ID() string

	// Manifest gibt die statischen Metadaten zurück.
	Manifest() Manifest

	// Init wird einmalig beim Start aufgerufen. Der Context
	// erlaubt Zugriff auf Config, Host-Liste, Hook-Registry.
	Init(ctx *Context) error

	// Enable wird aufgerufen, wenn der Nutzer das Plugin aktiviert.
	// Hier können Ressourcen allokiert werden (z.B. Goroutinen starten).
	Enable() error

	// Disable räumt auf. Nach dem Aufruf feuern keine Hooks mehr.
	Disable() error
}

// Manifest enthält statische Plugin-Metadaten.
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

// SettingDef beschreibt ein einzelnes Plugin-Setting (für dynamische Settings-UI).
type SettingDef struct {
	Key       string   `yaml:"key"`
	Label     string   `yaml:"label"`
	Kind      string   `yaml:"kind"` // "bool", "string", "int", "enum"
	Default   string   `yaml:"default"`
	Category  string   `yaml:"category,omitempty"`
	EnumOpts  []string `yaml:"enum_opts,omitempty"`
}