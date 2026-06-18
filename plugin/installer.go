package plugin

import (
	"fmt"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// PluginEntry describes a plugin for the 'plugin list' output.
type PluginEntry struct {
	ID      string
	Name    string
	Version string
	Enabled bool
	Source  string       // "builtin" | repo URL
	Trust   TrustLevel
	Dir     string       // local directory (empty for builtins)
	Status  PluginStatus // lifecycle state
}

// ImportMeta stores info about an installed external plugin.
type ImportMeta struct {
	ModulePath  string
	URL         string
	InstalledAt time.Time
	Trust       string
}

// pluginsDir returns ~/.local/share/woossh/plugins/.
func pluginsDir() string {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "woossh", "plugins")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "woossh", "plugins")
}

// sourceDir resolves the woossh source directory.
func sourceDir() (string, error) {
	if d := os.Getenv("WOOSSH_SOURCE_DIR"); d != "" {
		return d, nil
	}
	// Try relative to executable (assumes build from source dir)
	exe, err := os.Executable()
	if err == nil {
		// Check if go.mod exists in the parent hierarchy
		dir := filepath.Dir(exe)
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
	}
	return "", fmt.Errorf("WOOSSH_SOURCE_DIR not set and go.mod not found near executable. Set WOOSSH_SOURCE_DIR or run from the project directory")
}

// normalizeURL parses a GitHub URL into a module path.
func normalizeURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	rawURL = strings.TrimSuffix(rawURL, ".git")
	rawURL = strings.TrimSuffix(rawURL, "/")

	// Strip https:// prefix
	rawURL = strings.TrimPrefix(rawURL, "https://")
	rawURL = strings.TrimPrefix(rawURL, "http://")
	rawURL = strings.TrimPrefix(rawURL, "git@")
	rawURL = strings.Replace(rawURL, ":", "/", 1)

	return rawURL
}

// NameFromURL extracts the last path segment as the plugin name.
func NameFromURL(rawURL string) string {
	url := normalizeURL(rawURL)
	parts := strings.Split(url, "/")
	return parts[len(parts)-1]
}

// nameFromURL is the unexported alias for internal use.
func nameFromURL(rawURL string) string { return NameFromURL(rawURL) }

// gitClone runs git clone --depth 1 for a URL into pluginsDir/name.
func gitClone(rawURL, pluginsDir, name string) error {
	target := filepath.Join(pluginsDir, name)
	// Check if already exists
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("plugin directory %q already exists", target)
	}

	cmd := exec.Command("git", "clone", "--depth", "1", rawURL, target)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone %q: %w", rawURL, err)
	}
	return nil
}

// splitMonorepoURL detects GitHub monorepo URLs of the form
// github.com/<user>/<repo>/<subdir...> and returns the git clone URL
// and subdirectory separately.
//
//	github.com/silveX89/woossh-plugins/tmux
//	  → cloneURL: "https://github.com/silveX89/woossh-plugins"
//	  → subdir:   "tmux"
//
//	github.com/silveX89/woossh-plugin-tmux  (no subdir)
//	  → cloneURL: "https://github.com/silveX89/woossh-plugin-tmux"
//	  → subdir:   ""
func splitMonorepoURL(modulePath string) (cloneURL, subdir string) {
	parts := strings.Split(strings.TrimSuffix(modulePath, "/"), "/")
	// github.com/<user>/<repo> = 3 Segmente
	if len(parts) <= 3 {
		return "https://" + modulePath, ""
	}
	cloneURL = "https://" + strings.Join(parts[:3], "/")
	subdir = strings.Join(parts[3:], "/")
	return
}

// gitCloneMonorepo clones a repo and returns the plugin directory.
// For monorepo URLs (subdir != "") the whole repo is cloned but
// only the subdirectory is used as the plugin target.
func gitCloneMonorepo(rawURL, pDir string) (target string, err error) {
	modPath := normalizeURL(rawURL)
	cloneURL, subdir := splitMonorepoURL(modPath)

	repoParts := strings.Split(strings.TrimSuffix(cloneURL, "/"), "/")
	repoName := repoParts[len(repoParts)-1]
	cloneDir := filepath.Join(pDir, repoName)

	if _, err := os.Stat(cloneDir); err == nil {
		if subdir != "" {
			target = filepath.Join(cloneDir, subdir)
			if _, err := os.Stat(target); err != nil {
				return "", fmt.Errorf("subdir %q not found in already-cloned repo %s", subdir, cloneDir)
			}
			return target, nil
		}
		return "", fmt.Errorf("plugin directory %q already exists", cloneDir)
	}

	cmd := exec.Command("git", "clone", "--depth", "1", cloneURL, cloneDir)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git clone %q: %w", cloneURL, err)
	}

	target = cloneDir
	if subdir != "" {
		target = filepath.Join(cloneDir, subdir)
	}
	return target, nil
}

// injectWoosshReplace checks the plugin go.mod for a require of
// github.com/silveX89/woossh and injects a replace directive pointing at the
// local woossh source tree, then runs 'go mod tidy' inside the plugin dir.
func injectWoosshReplace(target, srcDir string) error {
	gomodPath := filepath.Join(target, "go.mod")
	data, err := os.ReadFile(gomodPath)
	if err != nil {
		// Plugin ohne go.mod ist ok (wird nicht injecten)
		return nil
	}

	content := string(data)
	if !strings.Contains(content, `require github.com/silveX89/woossh`) {
		return nil // Plugin importiert woossh nicht, kein inject nötig
	}

	// Check if replace already exists
	if strings.Contains(content, "github.com/silveX89/woossh =>") {
		return nil // already has a replace
	}

	replaceLine := fmt.Sprintf("\nreplace github.com/silveX89/woossh => %s\n", srcDir)
	content += replaceLine
	if err := os.WriteFile(gomodPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write plugin go.mod: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Running go mod tidy in plugin directory ...\n")
	if err := runGoModTidy(target); err != nil {
		return fmt.Errorf("plugin go mod tidy: %w", err)
	}

	return nil
}

// pluginLocalDir returns the plugin directory for a URL,
// consistent with gitCloneMonorepo.
func pluginLocalDir(rawURL, pDir string) string {
	modPath := normalizeURL(rawURL)
	cloneURL, subdir := splitMonorepoURL(modPath)
	repoParts := strings.Split(strings.TrimSuffix(cloneURL, "/"), "/")
	repoName := repoParts[len(repoParts)-1]
	if subdir != "" {
		return filepath.Join(pDir, repoName, subdir)
	}
	return filepath.Join(pDir, repoName)
}

// readModulePath reads the module path from a directory's go.mod.
func readModulePath(dir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return "", fmt.Errorf("read go.mod in %s: %w", dir, err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}
	return "", fmt.Errorf("no module directive found in %s/go.mod", dir)
}

// readPluginManifest tries to read plugin.yaml or woossh-plugin.yaml from the plugin dir.
func readPluginManifest(dir string) (id, name, version string, err error) {
	for _, fn := range []string{"plugin.yaml", "woossh-plugin.yaml"} {
		data, err := os.ReadFile(filepath.Join(dir, fn))
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "id:") {
				id = strings.TrimSpace(strings.TrimPrefix(line, "id:"))
			} else if strings.HasPrefix(line, "name:") {
				name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
			} else if strings.HasPrefix(line, "version:") {
				version = strings.TrimSpace(strings.TrimPrefix(line, "version:"))
			}
		}
		if id != "" {
			return id, name, version, nil
		}
	}
	return "", "", "", fmt.Errorf("no plugin.yaml found in %s", dir)
}

// ─── registry_gen.go management ──────────────────────────────────────────

// readInstalledMetas parses registry_gen.go and returns ImportMeta entries.
func readInstalledMetas(sourceDir string) ([]ImportMeta, []string, error) {
	path := filepath.Join(sourceDir, "registry_gen.go")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("read registry_gen.go: %w", err)
	}

	var metas []ImportMeta
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "// INSTALLED:") {
			rest := strings.TrimSpace(strings.TrimPrefix(line, "// INSTALLED:"))
			parts := strings.Split(rest, " | ")
			meta := ImportMeta{URL: "", InstalledAt: time.Now(), Trust: "unknown"}
			if len(parts) > 0 {
				meta.ModulePath = strings.TrimSpace(parts[0])
			}
			if len(parts) > 1 {
				meta.URL = strings.TrimSpace(parts[1])
			}
			if len(parts) > 2 {
				meta.Trust = strings.TrimSpace(parts[2])
			}
			// Skip builtin entries
			if meta.URL == "builtin" {
				continue
			}
			metas = append(metas, meta)
		}
	}

	// Also extract import paths
	var imports []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "_ ") {
			// Parse: _ "github.com/..."
			start := strings.Index(line, `"`)
			end := strings.LastIndex(line, `"`)
			if start >= 0 && end > start {
				imp := line[start+1 : end]
				if imp != "" {
					imports = append(imports, imp)
				}
			}
		}
	}

	return metas, imports, nil
}

// writeRegistryGen writes registry_gen.go with given imports and metas.
func writeRegistryGen(sourceDir string, metas []ImportMeta, imports []string) error {
	var buf strings.Builder
	buf.WriteString("// AUTO-GENERATED by woossh plugin manager.\n")
	buf.WriteString("// Run 'woossh plugin install <url>' to add external plugins.\n")

	// Sort metas by module path for stable output
	sort.Slice(metas, func(i, j int) bool {
		return metas[i].ModulePath < metas[j].ModulePath
	})

	for _, m := range metas {
		fmt.Fprintf(&buf, "// INSTALLED: %s | %s | %s\n", m.ModulePath, m.URL, m.Trust)
	}
	buf.WriteString("\npackage main\n\n")

	if len(imports) > 0 {
		buf.WriteString("import (\n")
		for _, imp := range imports {
			fmt.Fprintf(&buf, "\t_ %q\n", imp)
		}
		buf.WriteString(")\n\n")
	}

	buf.WriteString("func init() {\n")
	buf.WriteString("\t// External plugins register themselves via their init() functions.\n")
	buf.WriteString("\t// Builtin plugins are imported in main.go via blank imports.\n")
	buf.WriteString("}\n")

	// Format with go/format
	formatted, err := format.Source([]byte(buf.String()))
	if err != nil {
		return fmt.Errorf("format registry_gen.go: %w", err)
	}

	path := filepath.Join(sourceDir, "registry_gen.go")
	return os.WriteFile(path, formatted, 0o644)
}

// ─── go.mod management ──────────────────────────────────────────────────

// addGoModReplace adds a replace directive to go.mod.
// Idempotent: replaces existing directive for the same module.
func addGoModReplace(sourceDir, modulePath, localDir string) error {
	gomodPath := filepath.Join(sourceDir, "go.mod")
	data, err := os.ReadFile(gomodPath)
	if err != nil {
		return fmt.Errorf("read go.mod: %w", err)
	}
	replaceLine := fmt.Sprintf("\nreplace %s => %s\n", modulePath, localDir)

	// Check if a replace for this module already exists
	lines := strings.Split(string(data), "\n")
	found := false
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "replace") &&
			strings.Contains(line, modulePath+" =>") {
			// Update existing replace directive
			lines[i] = strings.TrimSpace(replaceLine)
			found = true
			break
		}
	}

	if !found {
		lines = append(lines, replaceLine)
	}

	result := strings.Join(lines, "\n")
	return os.WriteFile(gomodPath, []byte(result), 0o644)
}

// removeGoModReplace removes a replace directive for the given module.
func removeGoModReplace(sourceDir, modulePath string) error {
	gomodPath := filepath.Join(sourceDir, "go.mod")
	data, err := os.ReadFile(gomodPath)
	if err != nil {
		return fmt.Errorf("read go.mod: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	var newLines []string
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "replace") &&
			strings.Contains(line, modulePath+" =>") {
			continue // skip this line
		}
		newLines = append(newLines, line)
	}

	result := strings.Join(newLines, "\n")
	return os.WriteFile(gomodPath, []byte(result), 0o644)
}

// runGoModTidy runs 'go mod tidy' in the source directory.
func runGoModTidy(sourceDir string) error {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = sourceDir
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go mod tidy: %w", err)
	}
	return nil
}

// runGoBuild builds woossh in the source directory.
func runGoBuild(sourceDir, output string) error {
	output = strings.TrimSpace(output)
	args := []string{"build"}
	if output != "" {
		args = append(args, "-o", output)
	}
	args = append(args, ".")

	cmd := exec.Command("go", args...)
	cmd.Dir = sourceDir
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go build: %w", err)
	}
	return nil
}

// ─── Public API ──────────────────────────────────────────────────────────

// InstallPlugin installs a plugin from a URL.
// Steps: normalize URL → git clone → read go.mod → update go.mod replace →
// go mod tidy → update registry_gen.go → [--rebuild] go build.
func (m *Manager) InstallPlugin(rawURL string, rebuild bool) error {
	srcDir, err := sourceDir()
	if err != nil {
		return err
	}

	name := nameFromURL(rawURL)
	pDir := pluginsDir()
	var target string

	// Create plugins dir
	if err := os.MkdirAll(pDir, 0o755); err != nil {
		return fmt.Errorf("create plugins dir: %w", err)
	}

	// Check if already in registry
	_, imports, err := readInstalledMetas(srcDir)
	if err != nil {
		return fmt.Errorf("read registry: %w", err)
	}
	for _, imp := range imports {
		if strings.HasSuffix(imp, "/"+name) || strings.HasSuffix(imp, name) {
			return fmt.Errorf("plugin %q already in registry imports (%s)", name, imp)
		}
	}

	// Trust level
	trust := TrustLevelForURL(rawURL)
	if trust == TrustUnknown {
		fmt.Fprintf(os.Stderr, "Warning: Installing from unknown source %q\n", rawURL)
	}

	// Git clone (mit Monorepo-Support)
	fmt.Fprintf(os.Stderr, "Cloning %s ...\n", rawURL)
	target, err = gitCloneMonorepo(rawURL, pDir)
	if err != nil {
		return err
	}
	name = filepath.Base(target)

	// Read module path from cloned repo
	modulePath, err := readModulePath(target)
	if err != nil {
		// Clean up on failure
		os.RemoveAll(target)
		return fmt.Errorf("read plugin module path: %w", err)
	}

	// Inject woossh replace into plugin go.mod (if plugin imports woossh)
	if err := injectWoosshReplace(target, srcDir); err != nil {
		os.RemoveAll(target)
		return fmt.Errorf("inject plugin dependency: %w", err)
	}

	// Read plugin manifest (optional)
	id, pName, version, _ := readPluginManifest(target)
	if id == "" {
		id = name
	}
	if pName == "" {
		pName = name
	}
	if version == "" {
		version = "0.0.0"
	}

	// Update go.mod with replace directive
	fmt.Fprintf(os.Stderr, "Adding replace directive for %s -> %s\n", modulePath, target)
	if err := addGoModReplace(srcDir, modulePath, target); err != nil {
		os.RemoveAll(target)
		return fmt.Errorf("update go.mod: %w", err)
	}

	// go mod tidy
	fmt.Fprintf(os.Stderr, "Running go mod tidy ...\n")
	if err := runGoModTidy(srcDir); err != nil {
		os.RemoveAll(target)
		removeGoModReplace(srcDir, modulePath)
		return err
	}

	// Update registry_gen.go
	trustStr := "unknown"
	switch trust {
	case TrustOfficial:
		trustStr = "official"
	case TrustCommunity:
		trustStr = "community"
	}

	newMeta := ImportMeta{
		ModulePath:  modulePath,
		URL:         rawURL,
		InstalledAt: time.Now(),
		Trust:       trustStr,
	}
	metas, imports, err := readInstalledMetas(srcDir)
	if err != nil {
		os.RemoveAll(target)
		removeGoModReplace(srcDir, modulePath)
		return err
	}
	metas = append(metas, newMeta)
	imports = append(imports, modulePath)
	if err := writeRegistryGen(srcDir, metas, imports); err != nil {
		os.RemoveAll(target)
		removeGoModReplace(srcDir, modulePath)
		return fmt.Errorf("write registry_gen.go: %w", err)
	}

	// Run go mod tidy again so the new blank-import lands as a require in go.mod.
	fmt.Fprintf(os.Stderr, "Running go mod tidy (with registry update)...\n")
	if err := runGoModTidy(srcDir); err != nil {
		os.RemoveAll(target)
		removeGoModReplace(srcDir, modulePath)
		return err
	}

	// Update plugins.yaml state
	m.state.Sources[id] = rawURL
	m.state.Versions[id] = version
	m.state.Enabled[id] = false // not auto-enabled
	_ = m.SaveState()

	// Rebuild if requested
	if rebuild {
		fmt.Fprintf(os.Stderr, "Rebuilding woossh ...\n")
		exe, _ := os.Executable()
		if err := runGoBuild(srcDir, exe); err != nil {
			return fmt.Errorf("rebuild failed, run manually: go build -o %s . (in %s): %w", exe, srcDir, err)
		}
		fmt.Fprintf(os.Stderr, "woossh rebuilt successfully: %s\n", exe)
	} else {
		fmt.Fprintf(os.Stderr, "\nPlugin %q installed. Rebuild manually: cd %s && go build -o $(which woossh) .\n", name, srcDir)
	}

	fmt.Fprintf(os.Stderr, "Installed plugin: %s (%s v%s)\n", pName, id, version)
	return nil
}

// RemovePlugin removes an installed plugin.
func (m *Manager) RemovePlugin(idOrName string) error {
	srcDir, err := sourceDir()
	if err != nil {
		return err
	}

	// Find the plugin in registry
	metas, imports, err := readInstalledMetas(srcDir)
	if err != nil {
		return err
	}

	var foundMeta *ImportMeta
	var foundImport string
	var foundIdx int

	// Search by module path suffix (matches ID or name)
	for i, m := range metas {
		if strings.HasSuffix(m.ModulePath, "/"+idOrName) || strings.HasSuffix(m.ModulePath, idOrName) || m.ModulePath == idOrName {
			foundMeta = &metas[i]
			foundIdx = i
			break
		}
	}
	if foundMeta == nil {
		return fmt.Errorf("plugin %q not found in registry", idOrName)
	}

	// Find matching import
	for _, imp := range imports {
		if imp == foundMeta.ModulePath {
			foundImport = imp
			break
		}
	}

	// Remove from plugins directory
	name := nameFromURL(foundMeta.URL)
	pluginDir := pluginLocalDir(foundMeta.URL, pluginsDir())
	if _, err := os.Stat(pluginDir); err == nil {
		// Path traversal protection
		cleanDir := filepath.Clean(pluginDir)
		cleanBase := filepath.Clean(pluginsDir())
		if !strings.HasPrefix(cleanDir, cleanBase) {
			return fmt.Errorf("suspicious plugin dir %q — aborting", pluginDir)
		}
		if err := os.RemoveAll(pluginDir); err != nil {
			return fmt.Errorf("remove plugin dir %s: %w", pluginDir, err)
		}
	}

	// Remove from registry_gen.go
	newMetas := append(metas[:foundIdx], metas[foundIdx+1:]...)
	var newImports []string
	for _, imp := range imports {
		if imp != foundImport {
			newImports = append(newImports, imp)
		}
	}
	if err := writeRegistryGen(srcDir, newMetas, newImports); err != nil {
		return fmt.Errorf("update registry_gen.go: %w", err)
	}

	// Remove go.mod replace directive
	if foundImport != "" {
		if err := removeGoModReplace(srcDir, foundImport); err != nil {
			return fmt.Errorf("clean go.mod: %w", err)
		}
	}

	// Run go mod tidy
	fmt.Fprintf(os.Stderr, "Running go mod tidy ...\n")
	_ = runGoModTidy(srcDir)

	// Clean up state
	for _, id := range []string{idOrName, name} {
		delete(m.state.Enabled, id)
		delete(m.state.Sources, id)
		delete(m.state.Versions, id)
		delete(m.state.Settings, id)
	}
	_ = m.SaveState()

	fmt.Fprintf(os.Stderr, "Plugin %q removed. Rebuild: cd %s && go build -o $(which woossh) .\n", idOrName, srcDir)
	return nil
}

// ListPlugins returns all plugins (builtin + external).
func (m *Manager) ListPlugins() ([]PluginEntry, error) {
	var entries []PluginEntry

	// Collect external module paths for cross-referencing
	externalModules := make(map[string]bool)
	srcDir, err := sourceDir()
	if err == nil {
		metas, _, _ := readInstalledMetas(srcDir)
		for _, m := range metas {
			externalModules[m.ModulePath] = true
		}
	}

	// 1. Builtin/external plugins from global registry
	for _, p := range All() {
		mf := p.Manifest()
		source := "builtin"
		trust := TrustOfficial
		// Check if this plugin matches an external install
		for modPath := range externalModules {
			base := filepath.Base(modPath)
			if base == p.ID() || strings.Contains(p.ID(), base) || strings.Contains(modPath, p.ID()) {
				source = modPath
				trust = TrustLevelForURL(modPath)
				break
			}
		}
		status := PluginDownloaded
		if m.IsEnabled(p.ID()) {
			status = PluginStatusEnabled
		}
		entries = append(entries, PluginEntry{
			ID:      p.ID(),
			Name:    mf.Name,
			Version: mf.Version,
			Enabled: m.IsEnabled(p.ID()),
			Source:  source,
			Trust:   trust,
			Dir:     "",
			Status:  status,
		})
	}

	// 2. External plugins from registry_gen.go INSTALLED comments (skip if already shown from All())
	srcDir2, err2 := sourceDir()
	if err2 == nil {
		metas, imports, err := readInstalledMetas(srcDir2)
		if err == nil {
			for _, meta := range metas {
				// Skip if already shown from All()
				alreadyShown := false
				base := filepath.Base(meta.ModulePath)
				for _, e := range entries {
					if e.ID == base || strings.Contains(e.ID, base) || e.Source == meta.URL {
						alreadyShown = true
						break
					}
				}
				if alreadyShown {
					continue
				}

				name := nameFromURL(meta.URL)
				pluginDir := pluginLocalDir(meta.URL, pluginsDir())
				dirExists := false
				if _, err := os.Stat(pluginDir); err == nil {
					dirExists = true
				}

				// Check if runtime-registered (rebuild was done)
				registered := false
				for _, imp := range imports {
					if imp == meta.ModulePath {
						registered = true
						break
					}
				}

				trust := TrustLevelForURL(meta.URL)
				id := name

				// If runtime-registered, use actual plugin data
				var pName, pVersion string
				if registered {
					// Find which runtime-registered plugin matches this import
					var matchedID string
					for _, rp := range All() {
						if strings.Contains(meta.ModulePath, rp.ID()) || strings.Contains(meta.ModulePath, name) {
							matchedID = rp.ID()
							mf := rp.Manifest()
							id = rp.ID()
							pName = mf.Name
							pVersion = mf.Version
							break
						}
					}
					if matchedID == "" {
						pName = name + " (imported)"
						pVersion = "?"
					}
				} else {
					pName = name
					pVersion = "?"
				}

				extEnabled := m.IsEnabled(id) && registered
				extStatus := PluginAvailable
				if dirExists {
					if extEnabled {
						extStatus = PluginStatusEnabled
					} else {
						extStatus = PluginDownloaded
					}
				}
				entries = append(entries, PluginEntry{
					ID:      id,
					Name:    pName,
					Version: pVersion,
					Enabled: extEnabled,
					Source:  meta.URL,
					Trust:   trust,
					Dir:     pluginDir,
					Status:  extStatus,
				})

				// Mark version label for broken/not-yet-rebuilt state.
				if !dirExists {
					entries[len(entries)-1].Version = "broken"
				} else if !registered {
					entries[len(entries)-1].Version = "needs-rebuild"
				}
			}
		}
	}

	return entries, nil
}