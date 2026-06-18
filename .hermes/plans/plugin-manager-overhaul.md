# Plugin Manager Overhaul & TUI Help Cleanup

## Goals

1. **Clean up TUI help text** — Remove tmux-specific entries from bottom help lines.
   Structure: `/` commands = prompt flags, `Ctrl` commands = views/menus.

2. **Plugin Manager rewrite** — Full state model: Available → Downloaded → Enabled.
   Repo-based plugin discovery. GitHub repo URLs in settings. `silveX89/woossh-plugins` default.
   Delete plugin (code) from TUI.

3. **English-only code** — All comments, strings, variables → English.

---

## Phase 1: German → English translation

Files with German comments needing translation:
- `plugin/plugin.go` — comments: "Kern-Interface", "eindeutige Plugin-ID", "wird einmalig", etc.
- `plugin/manager.go` — "wird persistiert", "alte Implementierung", "verwaltet den Plugin-Lebenszyklus", etc.
- `plugin/context.go` — "was ein Plugin sehen und tun darf", "verhindert Zyklen", "leere Basisimplementierung"
- `plugin/installer.go` — "Monorepo-URLs", "erkennt", "gibt", "injiziert"
- `tui/tui.go` — "Favoriten", "Toggle", etc.
- `tui/plugin_view.go` — "wendet den Edit-Buffer an"
- `config/config.go` — "Allgemein", SSH, Tmux, TUI, Favoriten categories
- `model/host.go` — comments

**Action:** Read each file, replace all German comments with English.

---

## Phase 2: TUI Help Text

Current (lines 1199-1201 in tui/tui.go):
```
Tab / type to autocomplete  ·  Enter to connect  ·  Ctrl+C to quit
/o direct  ·  /v verbose  ·  /d dry-run  ·  /l legacy  ·  /c copy  ·  /t tmux (stackable, e.g. /o/v)
Ctrl+S settings  ·  Ctrl+F favorite  ·  Ctrl+Y toggle /c  ·  Ctrl+T tmux  ·  Ctrl+O overview  ·  Ctrl+P plugins
```

Target:
```
Tab / type to autocomplete  ·  Enter to connect  ·  Ctrl+C to quit
/o direct  ·  /v verbose  ·  /d dry-run  ·  /l legacy  ·  /c copy  (stackable, e.g. /o/v)
Ctrl+S settings  ·  Ctrl+F favorite  ·  Ctrl+Y toggle /c  ·  Ctrl+P plugins
```

Remove: `/t tmux`, `Ctrl+T tmux`, `Ctrl+O overview` (tmux-specific).
Keep the layout consistent.

Also review `tui/plugin_view.go` line 86 for any outdated help text.

---

## Phase 3: Plugin Manager Overhaul

### 3.1 State Model (plugin/state.go — NEW FILE? or extend manager.go)

Add to `plugin.Manager.State`:
```go
type PluginRepo struct {
    URL     string `yaml:"url"`
    Name    string `yaml:"name"`
    Enabled bool   `yaml:"enabled"`
}

type PluginStatus string
const (
    PluginAvailable   PluginStatus = "available"    // known from repo, not cloned
    PluginDownloaded  PluginStatus = "downloaded"   // cloned to local disk
    PluginEnabled     PluginStatus = "enabled"      // downloaded + activated
)
```

Extend `State`:
```go
type State struct {
    Enabled  map[string]bool              `yaml:"enabled"`
    Versions map[string]string            `yaml:"versions"`
    Sources  map[string]string            `yaml:"sources"`
    Settings map[string]map[string]string `yaml:"settings,omitempty"`
    Repos    []PluginRepo                 `yaml:"repos"`          // NEW
    RepoCache map[string][]PluginEntry    `yaml:"repo_cache,omitempty"` // NEW: cached repo listings
}
```

On first load, if `Repos` is empty, seed with:
```go
Repos: []PluginRepo{
    {URL: "github.com/silveX89/woossh-plugins", Name: "woossh-plugins", Enabled: true},
},
```

### 3.2 Plugin Discovery

New function in `plugin/` (e.g., `repo.go`):
```go
// DiscoverRepoPlugins uses GitHub API or git to list plugin dirs in a repo.
// For now: use `curl https://api.github.com/repos/<path>/contents` to get dir listing.
// Returns []PluginEntry with status=available.
func DiscoverRepoPlugins(repo PluginRepo) ([]PluginEntry, error)

// RefreshRepoCache fetches all enabled repos and populates RepoCache in State.
func (m *Manager) RefreshRepoCache() error
```

For each subdirectory found in the repo:
- Create a `PluginEntry` with `Status = PluginAvailable`
- If already downloaded (dir exists): `Status = PluginDownloaded`
- If enabled in state: `Status = PluginEnabled`

### 3.3 PluginEntry Changes

Current `PluginEntry` in `installer.go`:
```go
type PluginEntry struct {
    ID      string
    Name    string
    Version string
    Enabled bool
    Source  string
    Trust   TrustLevel
    Dir     string
}
```

New fields:
```go
type PluginEntry struct {
    ID      string
    Name    string
    Version string
    Status  PluginStatus  // available | downloaded | enabled
    Source  string        // repo URL or "builtin"
    Trust   TrustLevel
    Dir     string
    Repo    string        // which repo this came from
}
```

### 3.4 TUI Changes (plugin_view.go + tui.go)

**Plugin Manager View** (`modePluginManager`):
- Show ALL plugins: builtins + repo-discovered + downloaded + enabled
- Header columns: Plugin / Version / Status / Source
- Status values: "available" / "downloaded" / "enabled"
- Key handling additions:
  - `d` — download currently selected available plugin
  - `r` — remove/delete selected downloaded plugin (delete code, keep state)
  - `Enter` — toggle enable/disable
  - `s` — open repo settings (new sub-view)
- Remove old `s` → plugin settings (too deep; settings still accessible via other path)

**Repo Settings View** (`modePluginRepoSettings` — NEW mode):
- Shows list of configured repos
- Each row: URL, Name, Enabled toggle
- Keys: `Enter` toggle enabled, `a` add new repo URL (opens inline editor), `d` delete repo
- Default repo `silveX89/woossh-plugins` pre-seeded
- On close: refresh repo cache → update plugin list

**Plugin Removal**:
- `r` on a downloaded plugin:
  1. Call `m.pluginMgr.RemovePlugin(pluginID)` (delete cloned directory)
  2. Remove from Go module (remove replace + require)
  3. Regenerate registry_gen.go
  4. Run go mod tidy
  5. Optionally rebuild (or prompt)
  6. Refresh view

### 3.5 RemovePlugin Method (plugin/installer.go)

NEW method:
```go
// RemovePlugin removes a downloaded external plugin.
// Steps: remove replace directive → remove require → rm -rf clone dir →
// regenerate registry_gen.go → go mod tidy.
func (m *Manager) RemovePlugin(pluginID string, sourceDir string) error
```

### 3.6 InstallPlugin Improvements

Current `InstallPlugin` already works. It needs:
- On success: update state with `RepoCache[repoURL]` entries for this plugin
- Refresh the TUI view after install

---

## Phase 4: Testing

After implementation:
```bash
cd /home/silvex/woossh && go build -o woossh . && go vet ./...
woossh --version
woossh --list-hosts
```

---

## Implementation Order

1. Translate German → English comments across all files
2. Update TUI help text (tui.go lines 1199-1201)
3. Add PluginRepo struct + state fields (manager.go)
4. Add repo seed on first load (manager.go LoadState)
5. Add DiscoverRepoPlugins + RefreshRepoCache (new repo.go)
6. Update PluginEntry with Status field (installer.go)
7. Add RemovePlugin method (installer.go)
8. Update PluginManager view (plugin_view.go) — new columns, new keys
9. Add RepoSettings view mode (tui.go + plugin_view.go)
10. Wire Ctrl+R or similar shortcut for repo settings

**Constraints:**
- ALL code, comments, strings → English
- Keep backward compatibility with existing registry_gen.go
- Default repo: github.com/silveX89/woossh-plugins
- TUI must handle no-internet gracefully (show "scan failed" for repo discovery)