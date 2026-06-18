# woossh — Projektplanung & Roadmap

> **Stand:** 15. Juni 2026 · Version v0.2.0  
> **Zweck:** Vollständige Planungsübersicht aller offenen Aufgaben für das woossh-Projekt.

---

## Aktueller Stand — Was ist bereits implementiert?

Bevor die offenen Punkte besprochen werden, eine kurze Übersicht über den fertigen Kern:

| Bereich | Status | Anmerkung |
|---|---|---|
| `config.Config` (20+ Felder) | ✅ fertig | Load, Save, 3-faches Backup-Rotation |
| `model.HostEntry` + CSV-Parser | ✅ fertig | Unterstützt 4 CSV-Formate |
| `ssh.Flags` + ParseSlashPrefixes | ✅ fertig | Alle /o /v /d /l /c /t Flags |
| `tmux.*` Paket | ✅ fertig | Create, List, Attach, Kill, Cleanup |
| TUI — modeHosts | ✅ fertig | Fuzzy-Suche, Favorites, History, Scroll |
| TUI — modeSettings | ✅ fertig | Bool/Int/String/Enum-Editing, Kategorien |
| TUI — modeTmuxOverview | ✅ fertig | k=kill, d=detach, Enter=attach |
| `applyColorScheme` (8 Themes) | ✅ fertig | default, light, monokai, nord, gruvbox, dracula, solarized, catppuccin |
| Shell Completions (zsh, fish) | ✅ fertig | `completions/_woossh`, `completions/woossh.fish` |
| `--import-ssh-config` | ✅ fertig | ImportSSHConfig + MergeHosts |
| `--cleanup` + `--version` | ✅ fertig | CLI-Flags vollständig |
| Tests: config, ssh, model | ✅ vorhanden | Je ~200–330 Zeilen |
| Tests: tui, tmux | ❌ fehlen | applyColorScheme, Settings-Navigation, etc. |
| Plugin-System | ❌ nicht begonnen | Hook-Architektur fehlt komplett |
| Theme Customizer (Live-Editor) | ❌ nicht begonnen | applyColorScheme nur statisch |
| Kosten-Delegation | ❌ konzeptionell | Kein Code, rein strukturell |
| Task-Queue | ❌ nicht begonnen | Kein Ansatz vorhanden |
| Anonymisierung + GitHub Push | ❌ fehlt | .gitignore evtl. unvollständig |
| README — Settings-Abschnitt | ❌ fehlt | Neue Features undokumentiert |
| Theme-Nachschärfung | ⚠️ teilweise | styleDim="7" für alle Dunkel-Themes identisch |

---

## 1. Plugin-System (Phase A)

### Kontext / Why

woossh ist aktuell ein monolithisches Binary. Alle Logik sitzt fest in `config/`, `model/`, `ssh/`, `tui/` und `tmux/`. Es gibt keine definierten Erweiterungspunkte. Das Plugin-System soll externe Erweiterbarkeit ermöglichen, ohne die Kernarchitektur aufzubrechen — insbesondere für Passwortmanager-Integration (Bitwarden, KeePass), benutzerdefinierte Hooks vor/nach SSH-Verbindungen und zukünftige Drittanbieter-Integrationen.

**Wichtige Randbedingung:** Go unterstützt kein dynamisches Plugin-Laden (`.so`-Dateien) auf allen Plattformen. Deshalb wird kein `plugin.Open()`-Ansatz gewählt, sondern ein **Hook-System mit statisch eingebundenen Plugins** — Plugins werden nicht als separate Binaries geladen, sondern als Go-Pakete kompiliert und über eine Interface-Liste in `main.go` registriert.

### Umsetzungsplan

#### Schritt 1 — Plugin-Interface definieren (`plugin/plugin.go`, neu)

```go
package plugin

type Context struct {
    Hostname string
    User     string
    Port     int
    Flags    map[string]bool // UseTmux, BypassJumphost, etc.
}

type Hook interface {
    Name() string
    BeforeConnect(ctx *Context) error   // Kann Verbindung abbrechen (err != nil)
    AfterConnect(ctx *Context, exitCode int)
    BeforeTUIStart()
    AfterTUIStart()
    BeforeSettingsSave(cfg interface{})
    AfterSettingsSave(cfg interface{})
}

// Noop ist eine leere Basisimplementierung — Plugins embedden diese.
type Noop struct{}
func (Noop) Name() string { return "" }
func (Noop) BeforeConnect(*Context) error { return nil }
func (Noop) AfterConnect(*Context, int) {}
func (Noop) BeforeTUIStart() {}
func (Noop) AfterTUIStart() {}
func (Noop) BeforeSettingsSave(interface{}) {}
func (Noop) AfterSettingsSave(interface{}) {}
```

#### Schritt 2 — Plugin-Registry (`plugin/registry.go`)

```go
var registered []Hook

func Register(h Hook) { registered = append(registered, h) }
func All() []Hook     { return registered }

func RunBeforeConnect(ctx *Context) error {
    for _, h := range registered {
        if err := h.BeforeConnect(ctx); err != nil {
            return fmt.Errorf("plugin %q: %w", h.Name(), err)
        }
    }
    return nil
}
// ... entsprechend für alle anderen Hook-Punkte
```

#### Schritt 3 — Hooks in `main.go` einbinden

An den vier Hook-Punkten in `main.go` je einen `plugin.Run*`-Aufruf einfügen:
- Vor `tui.Run()`: `plugin.RunBeforeTUIStart()`
- Nach `tui.Run()`: `plugin.RunAfterTUIStart()`
- Vor `sshpkg.Connect()` / `tmux.Create()`: `plugin.RunBeforeConnect(&ctx)` — bei Fehler: `os.Exit(1)`
- Nach SSH (exitCode bekannt): `plugin.RunAfterConnect(&ctx, exitCode)`

In `tui/tui.go` um die `saveSettings()`-Funktion:
- `plugin.RunBeforeSettingsSave(&m.cfg)` und `plugin.RunAfterSettingsSave(&m.cfg)`

#### Schritt 4 — Erstes Plugin: Password Manager (`plugin/pwmanager/pwmanager.go`)

Das Passwortmanager-Plugin implementiert `BeforeConnect`:
1. Prüft ob Bitwarden CLI (`bw`) im PATH → versucht Secret zu holen
2. Fallback: KeePass KDBX via `github.com/tobischo/gokeepasslib/v3` öffnen
3. Passwort wird als `SSH_ASKPASS`-Umgebungsvariable injiziert (kein Speichern in Klartext)

**Konfiguration** per neuer Config-Sektion `[plugin_pwmanager]`:
```ini
; plugin_pwmanager
pwmanager_enabled = yes
pwmanager_backend = bitwarden  ; oder: keepass
pwmanager_kdbx_path = ~/.config/woossh/keys.kdbx
```

Config-Struct um `PwManagerEnabled bool`, `PwManagerBackend string`, `PwManagerKDBXPath string` erweitern.

#### Schritt 5 — Plugin-Registrierung in `main.go`

```go
// Statische Plugin-Registrierung — kein dynamic loading
import _ "github.com/silveX89/woossh/plugin/pwmanager"

func init() {
    plugin.Register(pwmanager.New())
}
```

### Vorteile

- Klare Erweiterungspunkte ohne Eingriff in Kernpakete
- Jedes Plugin kann isoliert getestet werden
- Kein externer Plugin-Manager notwendig (Go ist statisch)
- Password-Manager-Integration löst echten Schmerzpunkt (kein `ssh-agent` für alle Szenarien nötig)
- `Noop`-Basis macht Plugins schreibfaul — man implementiert nur was man braucht

### Risiken

| Risiko | Wahrscheinlichkeit | Mitigation |
|---|---|---|
| `BeforeConnect`-Fehler blockiert normalen SSH | mittel | Klarer Fehlertext + `--no-plugins`-Flag als Escape-Hatch |
| Bitwarden CLI muss eingeloggt sein | hoch | Plugin prüft `bw status` zuerst, fällt graceful zurück |
| KeePass-Lib hat eigene Abhängigkeiten | niedrig | `go.sum` wird größer; akzeptabel |
| Plugin-Interface bricht bei Refactoring | niedrig | Einmal sorgfältig designen, dann stabil halten |
| `SSH_ASKPASS` funktioniert nicht in allen Terminals | mittel | Dokumentieren, Test auf gängigen Terminals |

### Entscheidungen

1. **Wo wird der Plugin-Hook-Aufruf für Settings platziert?** In `tui.go:saveSettings()` oder in `config.Save()`? → Empfehlung: in `tui.go`, weil `config.Save()` auch direkt aufgerufen werden könnte ohne User-Interaction.
2. **Soll `--no-plugins` immer vorhanden sein?** → Ja, als globales Flag für Debugging.
3. **Bitwarden: Session-Token cachen?** In RAM via Package-Variable, nie auf Disk.
4. **KeePass: Passwort-Abfrage für KDBX?** Terminal-Prompt via `term.ReadPassword()` oder Settings-Feld? → Terminal-Prompt bevorzugt (kein Plaintext in Config).

---

## 2. Theme Customizer (Plugin-Idee / TUI-Feature)

### Kontext / Why

Die acht statischen Farbschemata in `applyColorScheme()` (tui/tui.go:50–116) decken viele Bedürfnisse ab. Es fehlt aber die Möglichkeit, einzelne Variablen anzupassen — zum Beispiel die Prompt-Farbe oder den Dim-Stil — ohne den Quellcode zu ändern. Der Theme Customizer erweitert das Settings-Menü um eine Live-Farbbearbeitung.

**Aktueller Code-Anker:** `applyColorScheme()` setzt 8 package-level `lipgloss.Style`-Variablen. Ein Customizer muss diese selektiv überschreiben können.

### Umsetzungsplan

#### Schritt 1 — Config-Felder für Custom-Theme-Overrides

In `config.Config` neue optionale Felder:
```go
// Custom theme overrides (leerer String = Fallback auf ColorScheme)
CustomCyan       string // z.B. "#FF8800" oder ANSI "214"
CustomYellow     string
CustomDim        string
CustomHeader     string
CustomPromptBase string
CustomPromptFlag string
CustomRule       string
CustomColHead    string
```

In `config.Save()` / `config.Load()` entsprechende Keys: `custom_cyan`, `custom_yellow`, etc.

#### Schritt 2 — Overlay-Funktion nach `applyColorScheme`

In `tui.go` nach dem bestehenden `applyColorScheme()`-Aufruf eine neue Funktion `applyCustomOverrides(cfg config.Config)`:

```go
func applyCustomOverrides(cfg config.Config) {
    if cfg.CustomCyan != "" {
        styleCyan = lipgloss.NewStyle().Foreground(lipgloss.Color(cfg.CustomCyan))
    }
    // ... für alle 8 Variablen
}
```

`applyScheme()` ruft dann beide auf: erst `applyColorScheme`, dann `applyCustomOverrides`.

#### Schritt 3 — Neue Settings-Kategorie "Theme"

`initSettings()` (tui.go:1018) um eine sechste Kategorie **"Theme"** erweitern mit 8 String-Items (je eine Farbe). Diese werden dann wie bestehende String-Items inline editiert (Enter → Eingabemodus).

Live-Preview: Nach Bestätigung jedes Eingabefelds (Enter) wird `m.applyScheme()` sofort aufgerufen → der Nutzer sieht die Änderung in Echtzeit auf dem Banner und in der Tabelle.

#### Schritt 4 — Farbvalidierung

Beim Speichern eines Custom-Color-Wertes validieren:
- Akzeptiert: ANSI-Zahl (0–255), Hex (#RRGGBB), leer (kein Override)
- Abgelehnt: Alles andere → kurze Inline-Fehlermeldung statt Absturz

#### Schritt 5 — "Reset Theme"-Option

Letzter Eintrag in der Theme-Kategorie: `[Reset alle Overrides]` — setzt alle `Custom*`-Felder auf `""` und ruft `applyScheme()` auf.

### Vorteile

- Kein Neustart nötig — Änderung sofort sichtbar
- Rückwärtskompatibel: fehlende Keys = bestehende Theme-Defaults
- Einfach testbar: Overlay-Funktion kann isoliert getestet werden

### Risiken

| Risiko | Mitigation |
|---|---|
| 16-Farben-Terminal ignoriert Hex-Werte | Dokumentieren; ANSI-Zahlen bevorzugen |
| Config.go wird sehr breit (8 neue Felder) | Akzeptabel, alle im Abschnitt `; Theme Overrides` gebündelt |
| Live-Preview kann bei vielen Hosts ruckeln | `applyScheme()` ist O(1), kein Performance-Problem |
| Nutzer bricht `custom_dim = ""` durch Tippfehler | Validierungsfunktion mit explizitem Leer-String-Reset |

### Entscheidungen

1. **Werden Bold-Flags ebenfalls überschreibbar?** → In Phase 1: Nein. Nur Farbe. Bold-Support in Phase 2.
2. **Soll ein "Custom Theme" als Named Theme gespeichert werden?** → Vorerst nein, nur ein Custom-Slot.
3. **Hex-Farb-Unterstützung:** lipgloss unterstützt `#RRGGBB` nativ via `lipgloss.Color()` — kein zusätzlicher Parser nötig.

---

## 3. Kosten-Delegation

### Kontext / Why

Dieser Punkt ist kein klassisches Code-Feature, sondern eine **Workflow-Entscheidung**: Wie werden teure oder zeitaufwändige Aufgaben (große Codeanalysen, komplexes Refactoring, Dokumentations-Generierung) auf günstigere Modelle oder Tools verteilt, um Anthropic-API-Kosten zu senken?

woossh selbst ist ein Go-CLI-Tool ohne LLM-Integration — die "Kosten-Delegation" bezieht sich auf den **Entwicklungsworkflow mit Claude Code**, nicht auf eine Produktfunktion.

### Umsetzungsplan

#### Delegation-Strategie für woossh-Entwicklung

| Aufgabe | Empfohlenes Modell / Tool | Begründung |
|---|---|---|
| Boilerplate (Tests schreiben nach Muster) | `claude-haiku-4-5` | Repetitiv, keine Kreativität nötig |
| Einzelne Funktionen debuggen | `claude-sonnet-4-6` (aktuell) | Ausgewogenes Kosten/Nutzen-Verhältnis |
| Architektur-Entscheidungen (Plugin-System, Theme-API) | `claude-opus-4-8` oder `/code-review ultra` | Komplexe Abwägungen brauchen tiefes Reasoning |
| README / Dokumentation | `claude-haiku-4-5` | Schreiben nach vorgegebener Struktur |
| Sicherheitsreview (SSH-Flag-Parsing, KeePass-Integration) | `/security-review` Skill | Spezialisierter Agent, nicht by hand |
| Codebase-Exploration | Explore-Subagent | Kein teures Hauptkontext-Fenster belegen |

#### Konkreter Trigger: Wann delegieren?

- Aufgabe ist **rein mechanisch** (z.B. "Schreibe Tests für alle 8 Color Schemes"): → Haiku
- Aufgabe **überschreitet 3 Dateien** und erfordert konsistente Änderungen über Pakete hinweg: → Sonnet mit Subagent
- Aufgabe erfordert **Architektur-Urteil** mit trade-offs: → Opus oder Plan-Mode

#### Integration in den Workflow

Kein Code in woossh selbst. Stattdessen: `CONTRIBUTING.md`-Abschnitt "AI-Assisted Development" dokumentiert die Strategie.

Ein `Makefile`-Target `make review` ruft `/code-review` auf — das ist bereits als Skill verfügbar.

### Vorteile

- Spart Kosten bei repetitiven Aufgaben (Tests, Docs)
- Hochwertige Entscheidungen bleiben bei leistungsstarken Modellen
- Dokumentierte Strategie verhindert willkürliche Modellwahl

### Risiken

| Risiko | Mitigation |
|---|---|
| Haiku-generierter Code ist oberflächlich | Code Review vor Merge ist Pflicht |
| Modell-IDs ändern sich | Immer mit `claude-api`-Skill aktuelle IDs prüfen |

### Entscheidungen

1. **Wird `claude-haiku` direkt in CI eingebunden?** → Nein, manueller Trigger.
2. **Cost-Tracking:** Anthropic Console → Usage-Dashboard. Kein eigenes Tracking nötig.

---

## 4. Task-Queue

### Kontext / Why

Beim Arbeiten mit woossh entstehen Kontextaufgaben — "Teste den neuen Nord-Theme", "Prüfe ob tmux-Session nach 24h aufgeräumt wird" — die zwischen Sessions verloren gehen. Eine leichtgewichtige, persistente Task-Queue direkt in der TUI oder als CLI-Output würde helfen, offene Punkte nicht zu vergessen.

**Wichtig:** Das ist kein Issue-Tracker-Ersatz, sondern ein persönliches "Scratchpad", das beim TUI-Start angezeigt wird.

### Umsetzungsplan

#### Schritt 1 — Task-Datei definieren

```
~/.config/woossh/tasks.json
```

Format:
```json
[
  {"id": 1, "text": "Nord-Theme testen", "done": false, "created": "2026-06-15"},
  {"id": 2, "text": "Backup-Rotation prüfen", "done": true, "created": "2026-06-10"}
]
```

#### Schritt 2 — Neues Paket `tasks/tasks.go`

```go
type Task struct {
    ID      int       `json:"id"`
    Text    string    `json:"text"`
    Done    bool      `json:"done"`
    Created time.Time `json:"created"`
}

func Load() ([]Task, error)
func Save(tasks []Task) error
func Add(text string) error
func Toggle(id int) error
func Purge() error  // löscht erledigte Tasks
```

#### Schritt 3 — Startup-Anzeige in TUI

In `initialModel()` nach dem Banner: Falls offene Tasks vorhanden, eine Zeile:
```
⏳ 3 offene Tasks — Ctrl+Q für Übersicht
```

#### Schritt 4 — Neue TUI-View `modeTasks` (optional, kann in Phase 2)

`Ctrl+Q` öffnet eine einfache Liste:
- `Space` = Task-Status togggeln
- `n` = neuen Task eingeben (Inline-Editor wie in Settings)
- `x` = erledigte Tasks löschen
- `Esc` / `q` = zurück

#### Schritt 5 — CLI-Befehle

```bash
woossh --tasks          # listet alle offene Tasks
woossh --task-add "..."  # fügt Task hinzu
woossh --task-done 3    # markiert Task 3 als erledigt
```

### Vorteile

- Vollständig offline, keine externe Abhängigkeit
- JSON ist leicht zu diffensen und zu versionieren (wenn gewünscht)
- Startup-Hinweis hält Aufgaben sichtbar ohne aufdringlich zu sein

### Risiken

| Risiko | Mitigation |
|---|---|
| Nutzer ignoriert Tasks → Feature ist nutzlos | Optional; lässt sich in Settings deaktivieren |
| Vierte TUI-View erhöht Komplexität | modeTasks erst in Phase 2; Startup-Hint reicht für Phase 1 |
| JSON-Datei wird korrumpiert | Load() mit Fehlerbehandlung; bei Fehler leere Liste zurückgeben |

### Entscheidungen

1. **JSON oder einfaches Text-Format?** → JSON, weil strukturierter und erweiterbar (z.B. Priorität hinzufügen).
2. **Soll die Task-Queue versioniert werden (git)?** → Optionaler Sync; vorerst lokal.
3. **Limit für Task-Anzahl?** → 100 Tasks max, älteste automatisch archivieren.

---

## 5. Anonymisierung & GitHub Push

### Kontext / Why

Das Projekt soll auf GitHub veröffentlicht werden (`github.com/silveX89/woossh`), aber `hosts.csv` enthält reale IP-Adressen, Hostnamen, Benutzernamen und ggf. Jump-Host-Informationen. Diese dürfen nicht im öffentlichen Repository erscheinen — weder in der aktuellen Version noch in der Git-History.

### Umsetzungsplan

#### Schritt 1 — .gitignore härten

Prüfen, ob folgende Einträge in `.gitignore` vorhanden sind:

```gitignore
# Sensible Konfigurationsdateien
hosts.csv
config.ini
*.bak

# Bitwarden / KeePass
*.kdbx
*.key

# Test-Output
*.test
coverage.out

# Build-Artefakt
woossh

# Persönliche Entwicklungsnotizen
.hermes/
```

#### Schritt 2 — Beispiel-Dateien erstellen

Neue Dateien, die ins Repo kommen:

```
hosts.csv.example   — mit anonymisierten Platzhaltern
config.ini.example  — mit allen Feldern als Kommentar/Default
```

`hosts.csv.example`:
```csv
hostname,host,port,user,jumphost,jumpuser,notes
webserver-01,192.168.1.10,22,admin,,,"Web Application Server"
db-primary,10.0.0.5,5432,dbadmin,bastion.example.com,jumpuser,"Primary Database"
legacy-device,172.16.0.100,22,root,,,"Legacy: OpenSSH 6.x"
```

#### Schritt 3 — Git-History auf sensible Daten prüfen

```bash
# Prüfen ob hosts.csv jemals committed wurde
git log --all --full-history -- hosts.csv
git log --all --full-history -- config.ini
```

Falls ja: Mit `git filter-repo` (bevorzugt gegenüber `git filter-branch`) aus der History entfernen:
```bash
pip install git-filter-repo
git filter-repo --path hosts.csv --invert-paths
git filter-repo --path config.ini --invert-paths
```

**Achtung:** Dies rewritten die History. Nur sinnvoll vor dem ersten Push. Bei bereits gepushten Commits: Force-Push erforderlich (mit User-Bestätigung).

#### Schritt 4 — GitHub-Repository einrichten

```bash
gh repo create silveX89/woossh --public --description "TUI SSH Manager" --source=. --remote=origin
git push -u origin main
```

#### Schritt 5 — GitHub Actions CI (optional, Phase 2)

`.github/workflows/ci.yml`:
```yaml
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.24' }
      - run: go test ./...
      - run: go vet ./...
```

### Vorteile

- Öffentliches Repo ohne Sicherheitsrisiko
- `hosts.csv.example` + `config.ini.example` erleichtern Neuinstallationen
- CI sichert Codequalität auf jedem Push

### Risiken

| Risiko | Mitigation |
|---|---|
| `git filter-repo` löscht versehentlich zu viel | Vorher `git clone` als Backup anlegen |
| Host-Muster in Commit-Nachrichten | `git log --all` nach IP-Mustern durchsuchen |
| Force-Push zerstört Remote-History | Nur vor erstem Public-Push; danach branch protection |
| `.hermes/`-Notizen enthalten Klartext | Bereits im Gitignore-Entwurf berücksichtigt |

### Entscheidungen

1. **Soll `.hermes/` jemals versioniert werden?** → Nein, bleibt lokal.
2. **Public oder Private Repo?** → Public (Projektabsicht), aber erst nach History-Bereinigung.
3. **Wann wird gepusht?** → Nach Abschluss von Phase 1 (Testing + Docs fertig).

---

## 6. Testing & Qualität

### Kontext / Why

Tests für `config`, `model/host` und `ssh` sind vorhanden (insgesamt ~640 Zeilen über drei Dateien). Vollständig **fehlend** sind Tests für:

1. `tui/tui.go` — `applyColorScheme()` und `applyScheme()` (visuelle Regression)
2. Settings-Navigation und Persistenz (Bool-Toggle, Int-Eingabe, String-Edit, Enum-Cycle)
3. `tmux/tmux.go` — alle Funktionen sind untested

### Umsetzungsplan

#### Schritt 1 — Tests für `applyColorScheme` / `applyScheme`

Neue Datei `tui/tui_test.go`. Da `applyColorScheme` package-level Variablen setzt, müssen Tests sequenziell laufen (`t.Parallel()` ist hier falsch):

```go
func TestApplyColorSchemeDoesNotPanic(t *testing.T) {
    schemes := []string{
        "default", "light", "monokai", "nord",
        "gruvbox", "dracula", "solarized", "catppuccin", "unknown",
    }
    for _, s := range schemes {
        t.Run(s, func(t *testing.T) {
            // Darf nicht paniken; kein Rückgabewert zu prüfen
            applyColorScheme(s)
        })
    }
}

func TestApplyColorSchemeSetsNonNilStyles(t *testing.T) {
    applyColorScheme("nord")
    // lipgloss.Style ist ein Value-Typ; Prüfung via Render
    if got := styleCyan.Render("x"); got == "" {
        t.Error("styleCyan nach nord-Theme ist leer")
    }
}
```

#### Schritt 2 — Settings-Logik testen (ohne TUI)

Die Funktionen `getCfgVal`, `setCfgVal`, `toggleCfgBool`, `setCfgInt`, `cycleSettingsEnum` operieren auf `tuiModel`. Sie können mit einem initialen `tuiModel` getestet werden:

```go
func TestToggleCfgBool(t *testing.T) {
    m := initialModel(config.Config{ForwardAgent: false}, nil, "test")
    m.toggleCfgBool("forward_agent")
    if !m.cfg.ForwardAgent {
        t.Error("ForwardAgent sollte nach Toggle true sein")
    }
    m.toggleCfgBool("forward_agent")
    if m.cfg.ForwardAgent {
        t.Error("ForwardAgent sollte nach zweitem Toggle false sein")
    }
}

func TestSetCfgInt_Validation(t *testing.T) {
    m := initialModel(config.Config{SSHPort: 22}, nil, "test")
    m.setCfgInt("ssh_port", 2222)
    if m.cfg.SSHPort != 2222 {
        t.Errorf("SSHPort = %d, want 2222", m.cfg.SSHPort)
    }
}

func TestCycleEnum(t *testing.T) {
    m := initialModel(config.Config{FavoriteSort: "name"}, nil, "test")
    item := &settingsNavItem{
        key:      "favorite_sort",
        kind:     "enum",
        enumOpts: []string{"name", "manual", "last_used"},
    }
    m.cycleSettingsEnum(item)
    if m.cfg.FavoriteSort != "manual" {
        t.Errorf("FavoriteSort nach Cycle = %q, want manual", m.cfg.FavoriteSort)
    }
}
```

#### Schritt 3 — Tests für `applySettingsEdit` (String-Commit)

```go
func TestApplySettingsEdit_String(t *testing.T) {
    m := initialModel(config.Config{}, nil, "test")
    // Simuliere: Nutzer editiert default_user
    m.settingsItemIdx = 0 // müsste auf default_user zeigen
    m.settingsEditBuf = "timo"
    // direkter setCfgVal-Aufruf als Proxy
    m.setCfgVal("default_user", "timo")
    if m.cfg.DefaultUser != "timo" {
        t.Errorf("DefaultUser = %q, want timo", m.cfg.DefaultUser)
    }
}
```

#### Schritt 4 — Tests für `tmux` Paket

`tmux/tmux_test.go` — da echte tmux-Prozesse in CI nicht verfügbar sind, wird die Logik in testbare Hilfsfunktionen extrahiert:

```go
// Testbar ohne tmux-Prozess:
func TestSessionNameUniqueness(t *testing.T) {
    // sessionName() ist package-privat; via Wrapper oder Export testen
}

func TestListParsing(t *testing.T) {
    // listRaw() mocken via Dependency-Injection (io.Reader statt exec.Command)
}
```

**Empfehlung:** `listRaw()` zu `parseListOutput(data []byte) ([]string, error)` refaktorieren — dann ist die Parse-Logik ohne tmux testbar.

#### Schritt 5 — go vet + Linting einrichten

```bash
# In Makefile:
lint:
	go vet ./...
	staticcheck ./...   # go install honnef.co/go/tools/cmd/staticcheck@latest

test:
	go test -race ./...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out
```

### Vorteile

- Regressionssicherheit beim Hinzufügen neuer Themes / Settings-Felder
- `go test -race` deckt Data-Races in der TUI auf (parallel Updates)
- Coverage-Report zeigt Lücken

### Risiken

| Risiko | Mitigation |
|---|---|
| TUI-Tests sind schwer zu isolieren (globale Styles) | `t.Cleanup()` um Styles nach jedem Test zurückzusetzen |
| `tmux`-Tests schlagen in CI fehl | Build-Tag `//go:build integration` für tmux-Tests |
| `initialModel` ist zu schwer für Unit-Tests | Helper-Funktion `testModel()` mit Minimal-Config |

### Entscheidungen

1. **Soll `tui_test.go` im selben Package liegen (`package tui`) oder extern (`package tui_test`)?** → `package tui` (black-box geht nicht, da private Felder getestet werden).
2. **Mindest-Coverage-Ziel?** → 60% als Einstieg, 80% nach Phase 1 komplett.
3. **staticcheck oder golangci-lint?** → `staticcheck` ist leichtgewichtiger; golangci-lint erst wenn CI eingerichtet ist.

---

## 7. Dokumentation

### Kontext / Why

`README.md` existiert, wurde aber vor Implementation des Settings-Menüs, des Tmux-Overviews, der 8 Farbschemata und der Shell-Completions geschrieben. Neue Nutzer haben keinen Überblick über diese Features.

### Umsetzungsplan

#### Schritt 1 — README-Abschnitt: Settings-Menü

Neuer Abschnitt nach "Usage":

```markdown
## Settings (Ctrl+S)

`Ctrl+S` öffnet das Settings-Menü. Navigation:
- `↑` / `↓` — zwischen Einstellungen wechseln
- `Tab` / `Shift+Tab` — zwischen Kategorien wechseln
- `Enter` — Bool-Wert umschalten / Inline-Editor öffnen / Enum weiterschalten
- `Esc` oder `Ctrl+S` — speichern und schließen

### Kategorien

| Kategorie | Schlüssel | Typ | Standard |
|---|---|---|---|
| Allgemein | hosts_path | String | ./hosts.csv |
| Allgemein | default_user | String | root |
| SSH | identity_file | String | ~/.ssh/id_rsa |
| SSH | forward_agent | Bool | no |
| Tmux | use_tmux | Bool | no |
| Tmux | session_prefix | String | woossh: |
| TUI | color_scheme | Enum | default |
| TUI | show_ip | Bool | yes |
| Favoriten | favorite_sort | Enum | name |
```

#### Schritt 2 — README-Abschnitt: Farbschemata

```markdown
## Farbschemata

Verfügbare Themes (einstellbar unter TUI → color_scheme):

| Schema | Optimiert für |
|---|---|
| `default` | Dunkle Terminals (Standard) |
| `light` | Helle Hintergründe |
| `monokai` | Monokai-Farbpalette |
| `nord` | Nord-Farbpalette |
| `gruvbox` | Gruvbox-Farbpalette |
| `dracula` | Dracula-Farbpalette |
| `solarized` | Solarized Dark |
| `catppuccin` | Catppuccin Mocha |
```

#### Schritt 3 — Screenshots erstellen

Screenshots mit `woossh` im Terminalmodus via `vhs` (VHS TUI-Recording-Tool):

```bash
# Installation: go install github.com/charmbracelet/vhs@latest
# Demo-Datei: docs/demo.tape
```

Szenen für `demo.tape`:
1. woossh-Start mit Banner + Host-Liste
2. Fuzzy-Suche (tippen von "web")
3. Settings-Menü öffnen (Ctrl+S) + Farbe wechseln
4. Tmux-Overview (Ctrl+O)

Ergebnis: `docs/demo.gif` für README-Header.

#### Schritt 4 — CHANGELOG.md anlegen

Format: [Keep a Changelog](https://keepachangelog.com/de/1.0.0/)

```markdown
## [Unreleased]
### Added
- Settings-Menü (Ctrl+S) mit 20 konfigurierbaren Einstellungen
- 8 Farbschemata
- Tmux-Session-Verwaltung (/t, Ctrl+O)
- ~/.ssh/config Import (--import-ssh-config)
- Shell-Completions für zsh und fish
```

### Vorteile

- Neue Nutzer verstehen das Feature-Set sofort
- Screenshots erhöhen Akzeptanz auf GitHub erheblich
- CHANGELOG ist Standard für Open-Source-Projekte

### Risiken

| Risiko | Mitigation |
|---|---|
| Screenshots veralten schnell | VHS-Tape-Datei mit im Repo → leicht neu zu rendern |
| README wird zu lang | Separate `docs/`-Seite für vollständige Referenz |

### Entscheidungen

1. **VHS oder manuelle Screenshots?** → VHS, weil reproduzierbar und automatisierbar.
2. **Deutsche oder englische README?** → Englisch (GitHub-Standard für Open-Source).
3. **Wiki oder `docs/`?** → `docs/` Verzeichnis, kein GitHub-Wiki (bleibt im Repo).

---

## 8. Theme-Nachschärfung

### Kontext / Why

Beim Betrachten von `applyColorScheme()` (tui.go:50–116) fällt auf:

- **`styleDim`** wird in allen Dunkel-Themes auf ANSI `"7"` gesetzt (monokai, nord, gruvbox, dracula, solarized). Farbe `"7"` ist terminal-abhängig: meistens hellgrau bis weiß. Auf manchen Terminals ist das **zu hell** (zu wenig Kontrast gegen weiße Elemente) oder **unsichtbar** auf hellem Hintergrund. Nur `light` verwendet `"235"` (dunkelgrau), was semantisch korrekt ist.
- **`styleRule`** (Trennlinie) variiert von `"8"` (default/light) über `"237"` (monokai) bis `"96"` (gruvbox) — aber z.B. gruvbox `"96"` (dunkles Lila) passt nicht gut zur grünen Farbpalette.
- **`styleHeader`** beim `light`-Theme: `"235"` ist dunkelgrau — gut. Bei nord `"188"` (hellgrau) fehlt genug Kontrast auf weißem Hintergrund, falls jemand nord auf hellem Terminal nutzt.

### Umsetzungsplan

#### Schritt 1 — Theme-Audit-Tabelle erstellen

Alle 8 Themes × 8 Style-Variablen in einer Tabelle erfassen und Kontrast-Probleme markieren:

| Theme | styleDim | styleRule | Probleme |
|---|---|---|---|
| default | `"7"` | `"239"` | styleDim = ANSI white, ok für reines Dark |
| light | `"235"` | `"8"` | ✅ korrekt invertiert |
| monokai | `"7"` | `"237"` | styleDim zu hell, styleRule zu dunkel |
| nord | `"7"` | `"240"` | styleDim zu hell auf Arctic BG |
| gruvbox | `"7"` | `"96"` | styleRule passt nicht zur Palette |
| dracula | `"7"` | `"245"` | styleRule zu hell (gleich wie Header) |
| solarized | `"7"` | `"244"` | styleDim passt nicht zu Solarized-Semantik |
| catppuccin | *(prüfen)* | *(prüfen)* | *(noch im Detail anschauen)* |

#### Schritt 2 — Korrekturen je Theme

**Faustregel:**
- `styleDim` soll subtil aber **sichtbar** sein: 1–2 Helligkeitsstufen unter dem normalen Text.
- `styleRule` soll eine Linie sein, die **klar trennend** wirkt, aber nicht dominiert.

Vorgeschlagene Korrekturen:

```go
case "monokai":
    styleDim = lipgloss.NewStyle().Foreground(lipgloss.Color("243")) // statt "7"
    styleRule = lipgloss.NewStyle().Foreground(lipgloss.Color("59"))  // dunkleres Lila

case "nord":
    styleDim = lipgloss.NewStyle().Foreground(lipgloss.Color("67"))  // North Blue, statt "7"
    // styleRule "240" ist ok

case "gruvbox":
    styleDim = lipgloss.NewStyle().Foreground(lipgloss.Color("243")) // statt "7"
    styleRule = lipgloss.NewStyle().Foreground(lipgloss.Color("130")) // Gruvbox orange-dark

case "dracula":
    styleDim = lipgloss.NewStyle().Foreground(lipgloss.Color("61"))  // Dracula Comment, statt "7"
    styleRule = lipgloss.NewStyle().Foreground(lipgloss.Color("240")) // statt "245"

case "solarized":
    styleDim = lipgloss.NewStyle().Foreground(lipgloss.Color("33"))  // Solarized Blue dim, statt "7"
    // styleRule "244" ist ok
```

#### Schritt 3 — Visuelle Verifizierung

Nach den Änderungen muss `woossh` in einem echten Terminal mit jedem Theme manuell geprüft werden (kein Test kann das automatisch validieren). Dabei prüfen:
- Ist die Trennlinie zwischen Header und Tabellenzeilen klar erkennbar?
- Sind Dim-Texte (Notizen, leere Felder) sichtbar aber zurückhaltend?
- Passt `stylePromptFlag` (aktive Flags) gut zum jeweiligen Hintergrund?

#### Schritt 4 — Tests aktualisieren

Die neuen Farben werden in den Tests aus Schritt 6.1 automatisch abgedeckt (kein Panic, nicht leer).

### Vorteile

- Professionelleres Erscheinungsbild über alle Themes
- Solarized- und Nord-Nutzer bekommen ein "richtiges" Theme-Feeling
- Einmalige Arbeit mit langfristigem Nutzen

### Risiken

| Risiko | Mitigation |
|---|---|
| ANSI-Farben variieren zwischen Terminalemulatorenen | Testen in: Alacritty, Kitty, GNOME Terminal, macOS Terminal |
| Subjektive Ästhetik — keine "richtige" Antwort | Screenshots als Referenz, Feedback einholen |
| Catppuccin-Theme noch nicht vollständig auditiert | Separater Sub-Punkt in Phase 2 |

### Entscheidungen

1. **Werden Farbänderungen als Breaking Change gewertet?** → Nein, rein kosmetisch.
2. **Wer gibt finales OK für die Farben?** → Entwickler (du) nach manuellem Test.
3. **Wird `styleBold` je Theme differenziert?** → Vorerst nein, einheitlich Bold.

---

## Abhängigkeiten zwischen den Items

```
Plugin-System (1)
    └── benötigt: config.go (erweiterbar, ✅ vorhanden)
    └── ermöglicht: Password Manager Plugin
    └── ermöglicht (später): Theme als Plugin

Theme Customizer (2)
    └── benötigt: applyColorScheme (✅ vorhanden)
    └── benötigt: config.Config (neue Custom*-Felder)
    └── benötigt: Settings-Menü (✅ vorhanden)
    └── sinnvoll nach: Theme-Nachschärfung (8) — damit Basis stimmt

Kosten-Delegation (3)
    └── keine Code-Abhängigkeiten; reine Workflow-Dokumentation

Task-Queue (4)
    └── benötigt: neues `tasks/` Paket (unabhängig)
    └── optional: Plugin-Hook "AfterTUIStart" (aus Item 1)

Anonymisierung + GitHub (5)
    └── benötigt: .gitignore vollständig (einfach)
    └── sinnvoll nach: Testing (6) und Dokumentation (7) fertig
    └── blockiert: öffentlicher GitHub-Push

Testing (6)
    └── benötigt: alle bestehenden Pakete (✅ vorhanden)
    └── sinnvoll nach: Theme-Nachschärfung (8) — Tests sollen finale Farben abdecken
    └── muss vor: GitHub-Push (5)

Dokumentation (7)
    └── benötigt: Features fertig (Settings ✅, Tmux ✅, Themes nach 8)
    └── benötigt: Screenshots → VHS-Tool
    └── muss vor: GitHub-Push (5)

Theme-Nachschärfung (8)
    └── keine Abhängigkeiten — kann jederzeit gemacht werden
    └── sinnvoll vor: Dokumentation (Screenshots) und Testing (finale Farben)
```

---

## Empfohlene Reihenfolge

### Phase 1 — Qualität & Veröffentlichung (ca. 1–2 Wochen)

**Ziel:** Das bestehende v0.2.0 in einen veröffentlichungsreifen Zustand bringen.

| Reihenfolge | Item | Begründung |
|---|---|---|
| 1 | Theme-Nachschärfung (8) | Basis für Screenshots und Tests — kurze, isolierte Arbeit |
| 2 | Testing / Qualität (6) | Stellt sicher, dass alles funktioniert bevor es öffentlich wird |
| 3 | Dokumentation (7) | Screenshots erst nach finalen Themes sinnvoll |
| 4 | Anonymisierung + GitHub (5) | Erst nach Tests + Docs pushen |

### Phase 2 — Neue Features (ca. 2–4 Wochen nach Phase 1)

**Ziel:** Erweiterbarkeit und Komfort ausbauen.

| Reihenfolge | Item | Begründung |
|---|---|---|
| 5 | Theme Customizer (2) | Eigenständiges, in sich geschlossenes TUI-Feature; kein Plugin-System nötig |
| 6 | Task-Queue (4) | Nützliches persönliches Feature; unabhängig von allem anderen |
| 7 | Kosten-Delegation (3) | Workflow-Dokumentation; passt gut nach Phase 1 Abschluss |

### Phase 3 — Architektur (nach Phase 2)

**Ziel:** Externe Erweiterbarkeit.

| Reihenfolge | Item | Begründung |
|---|---|---|
| 8 | Plugin-System (1) | Größte architektonische Änderung; braucht stabilen Kern |

---

## Zeitliche Aufteilung

```
Juni 2026
  Woche 1:  Theme-Nachschärfung → manuelle Tests → Fertig
  Woche 2:  tui_test.go + tmux_test.go schreiben; go vet + staticcheck

Juli 2026
  Woche 1:  README aktualisieren; VHS-Demo aufnehmen; CHANGELOG anlegen
  Woche 2:  .gitignore + History-Bereinigung + GitHub Push
             → v0.3.0 Release Tag setzen

August 2026
  Woche 1:  Theme Customizer (neue Config-Felder + Settings-Kategorie "Theme")
  Woche 2:  Task-Queue (tasks/ Paket + Startup-Hint + CLI-Befehle)

September 2026
  Woche 1:  Kosten-Delegations-Dokumentation + Makefile-Targets
  Woche 2:  Plugin-System Phase A (Interface + Registry + Noop)

Oktober 2026
  Woche 1–2: Password Manager Plugin (Bitwarden + KeePass)
              → v0.4.0 Release
```

---

## Zusammenfassung

| Item | Aufwand | Priorität | Phase |
|---|---|---|---|
| Theme-Nachschärfung (8) | Klein (2–4h) | Hoch | 1 |
| Testing / Qualität (6) | Mittel (1–2 Tage) | Hoch | 1 |
| Dokumentation (7) | Mittel (1 Tag) | Hoch | 1 |
| Anonymisierung + GitHub (5) | Klein (2–3h) | Hoch | 1 |
| Theme Customizer (2) | Mittel (1–2 Tage) | Mittel | 2 |
| Task-Queue (4) | Mittel (1 Tag) | Mittel | 2 |
| Kosten-Delegation (3) | Klein (2h Doku) | Niedrig | 2 |
| Plugin-System (1) | Groß (3–5 Tage) | Niedrig | 3 |

> **Empfehlung:** Starte mit Phase 1 komplett — es gibt kaum Risiko und liefert einen sauberen öffentlichen Stand. Phase 2 und 3 können jederzeit unterbrochen oder umsortiert werden.
