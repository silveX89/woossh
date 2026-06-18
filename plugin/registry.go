package plugin

import "sync"

var (
	mu       sync.RWMutex
	registry []Plugin
)

// Register wird von Plugin-Packages in ihrer init()-Funktion aufgerufen.
func Register(p Plugin) {
	mu.Lock()
	defer mu.Unlock()
	registry = append(registry, p)
}

// All gibt eine Kopie der Plugin-Liste zurück.
func All() []Plugin {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]Plugin, len(registry))
	copy(out, registry)
	return out
}

// FindByID sucht ein Plugin anhand seiner ID.
func FindByID(id string) Plugin {
	mu.RLock()
	defer mu.RUnlock()
	for _, p := range registry {
		if p.ID() == id {
			return p
		}
	}
	return nil
}

// IsRegistered prüft ob ein Plugin mit der gegebenen ID registriert ist.
func IsRegistered(id string) bool {
	return FindByID(id) != nil
}