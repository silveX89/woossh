package tui

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"

	"github.com/silveX89/woossh/config"
	"github.com/silveX89/woossh/model"
	"github.com/silveX89/woossh/plugin"
	sshpkg "github.com/silveX89/woossh/ssh"
	"github.com/silveX89/woossh/tmux"
)

// resetPromptMsg is sent after a timed delay to restore the default prompt look.
type resetPromptMsg struct{}

// Result is returned from Run after the user makes a selection.
type Result struct {
	Target     string
	Flags      sshpkg.Flags
	TmuxAttach string // session name to attach to (from tmux overview)
}

// ─── Styles ──────────────────────────────────────────────────────────────────

var (
	styleCyan       = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	styleYellow     = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
	styleDim        = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleHeader     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	stylePromptBase = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	stylePromptFlag = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
	styleRule       = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleColHead    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
)

// applyColorScheme reassigns the package-level style vars based on the
// active color_scheme setting.
func applyColorScheme(scheme string) {
	switch scheme {
	case "light":
		styleCyan = lipgloss.NewStyle().Foreground(lipgloss.Color("30"))
		styleYellow = lipgloss.NewStyle().Foreground(lipgloss.Color("94")).Bold(true)
		styleDim = lipgloss.NewStyle().Foreground(lipgloss.Color("235"))
		styleHeader = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("235"))
		stylePromptBase = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("30"))
		stylePromptFlag = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("124"))
		styleRule = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
		styleColHead = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("235"))
	case "monokai":
		styleCyan = lipgloss.NewStyle().Foreground(lipgloss.Color("141"))
		styleYellow = lipgloss.NewStyle().Foreground(lipgloss.Color("213")).Bold(true)
		styleDim = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
		styleHeader = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("148"))
		stylePromptBase = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("141"))
		stylePromptFlag = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("213"))
		styleRule = lipgloss.NewStyle().Foreground(lipgloss.Color("59"))
		styleColHead = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("148"))
	case "nord":
		styleCyan = lipgloss.NewStyle().Foreground(lipgloss.Color("67"))
		styleYellow = lipgloss.NewStyle().Foreground(lipgloss.Color("179")).Bold(true)
		styleDim = lipgloss.NewStyle().Foreground(lipgloss.Color("67"))
		styleHeader = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("188"))
		stylePromptBase = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("67"))
		stylePromptFlag = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("179"))
		styleRule = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		styleColHead = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("188"))
	case "gruvbox":
		styleCyan = lipgloss.NewStyle().Foreground(lipgloss.Color("142"))
		styleYellow = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
		styleDim = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
		styleHeader = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("223"))
		stylePromptBase = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("142"))
		stylePromptFlag = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
		styleRule = lipgloss.NewStyle().Foreground(lipgloss.Color("130"))
		styleColHead = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("223"))
	case "dracula":
		styleCyan = lipgloss.NewStyle().Foreground(lipgloss.Color("141"))
		styleYellow = lipgloss.NewStyle().Foreground(lipgloss.Color("215")).Bold(true)
		styleDim = lipgloss.NewStyle().Foreground(lipgloss.Color("61"))
		styleHeader = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("231"))
		stylePromptBase = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("141"))
		stylePromptFlag = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
		styleRule = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		styleColHead = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("231"))
	case "solarized":
		styleCyan = lipgloss.NewStyle().Foreground(lipgloss.Color("37"))
		styleYellow = lipgloss.NewStyle().Foreground(lipgloss.Color("136")).Bold(true)
		styleDim = lipgloss.NewStyle().Foreground(lipgloss.Color("33"))
		styleHeader = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("245"))
		stylePromptBase = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("37"))
		stylePromptFlag = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("136"))
		styleRule = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
		styleColHead = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("245"))
	default: // "default" or "dark"
		styleCyan = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
		styleYellow = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
		styleDim = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
		styleHeader = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
		stylePromptBase = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
		stylePromptFlag = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
		styleRule = lipgloss.NewStyle().Foreground(lipgloss.Color("239"))
		styleColHead = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	}
}

// ─── Banner ───────────────────────────────────────────────────────────────────

// bannerText is 11 lines: cat creature (left, cols 0-24) + WOOSSH block art (right, cols 25+).
// The left column is padded to exactly 25 ASCII characters so the split is always clean.
const bannerText = `                         ⟡ ═══════════════════════════════════════════════════
         /\     /\
        /  \___/  \        ██╗    ██╗ ██████╗  ██████╗ ███████╗███████╗██╗  ██╗
       / .-. . .-. \       ██║    ██║██╔═══██╗██╔═══██╗██╔════╝██╔════╝██║  ██║
      | ( o ) ( o ) |      ██║ █╗ ██║██║   ██║██║   ██║███████╗███████╗███████║
      |   \ ~~~ /   |      ██║███╗██║██║   ██║██║   ██║╚════██║╚════██║██╔══██║
       \   ~~~~~   /       ╚███╔███╔╝╚██████╔╝╚██████╔╝███████║███████║██║  ██║
        \_________/         ╚══╝╚══╝  ╚═════╝  ╚═════╝ ╚══════╝╚══════╝╚═╝  ╚═╝
            |||||
          __||_||__         ⟡ ═══════════════════════════════════════════════════
                                                   ⚡`

const catSplitCol = 25 // left column width (pure ASCII, so bytes == runes)

func renderBanner() string {
	lines := strings.Split(bannerText, "\n")
	var sb strings.Builder
	for _, line := range lines {
		if len(line) == 0 {
			sb.WriteString("\n")
			continue
		}
		var catPart, woosshPart string
		if len(line) <= catSplitCol {
			catPart = line
		} else {
			catPart = line[:catSplitCol]
			woosshPart = line[catSplitCol:]
		}
		if strings.TrimSpace(catPart) != "" {
			sb.WriteString(styleCyan.Render(catPart))
		} else {
			sb.WriteString(catPart)
		}
		if woosshPart != "" {
			sb.WriteString(styleYellow.Render(woosshPart))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// ─── History ─────────────────────────────────────────────────────────────────

func historyPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "woossh", ".history")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "woossh", ".history")
}

func loadHistory() []string {
	f, err := os.Open(historyPath())
	if err != nil {
		return nil
	}
	defer f.Close()
	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if l := strings.TrimSpace(sc.Text()); l != "" {
			lines = append(lines, l)
		}
	}
	// Reverse so most-recent is first
	for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
		lines[i], lines[j] = lines[j], lines[i]
	}
	return lines
}

func AppendHistory(entry string) {
	path := historyPath()
	_ = os.MkdirAll(filepath.Dir(path), 0o700)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintln(f, entry)
}

// ─── Favorites ───────────────────────────────────────────────────────────────

func favoritePath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "woossh", ".favorites")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "woossh", ".favorites")
}

func loadFavorites() map[string]bool {
	favs := make(map[string]bool)
	f, err := os.Open(favoritePath())
	if err != nil {
		return favs
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if l := strings.TrimSpace(sc.Text()); l != "" {
			favs[l] = true
		}
	}
	return favs
}

func saveFavorites(favs map[string]bool) {
	path := favoritePath()
	_ = os.MkdirAll(filepath.Dir(path), 0o700)
	f, err := os.Create(path)
	if err != nil {
		return
	}
	defer f.Close()
	for name := range favs {
		fmt.Fprintln(f, name)
	}
}

func sortFavoritesFirst(hosts []model.HostEntry, favs map[string]bool) []model.HostEntry {
	sorted := make([]model.HostEntry, len(hosts))
	copy(sorted, hosts)
	// Stable sort: favorites first, then alphabetical within each group
	// Using simple insertion: build two lists and concatenate
	var favList, rest []model.HostEntry
	for _, h := range sorted {
		if favs[h.Hostname] {
			favList = append(favList, h)
		} else {
			rest = append(rest, h)
		}
	}
	// Within each group already alphabetically sorted (LoadHosts sorts)
	return append(favList, rest...)
}

// toggleFavorite flips the favorite status for a hostname and persists.
func toggleFavorite(favs map[string]bool, hostname string) {
	if favs[hostname] {
		delete(favs, hostname)
	} else {
		favs[hostname] = true
	}
	saveFavorites(favs)
}

// ─── Table rendering ─────────────────────────────────────────────────────────

type colWidths struct {
	hostname int
	host     int
	port     int
	user     int
	jump     int
	notes    int
}

func computeColWidths(hosts []model.HostEntry, cfg config.Config) colWidths {
	w := colWidths{
		hostname: len("Hostname"),
		host:     len("IP / Host"),
		port:     len("Port"),
		user:     len("User"),
		jump:     len("Via Jump"),
		notes:    len("Notes"),
	}
	for _, h := range hosts {
		if len(h.Hostname) > w.hostname {
			w.hostname = len(h.Hostname)
		}
		if len(h.Host) > w.host {
			w.host = len(h.Host)
		}
		portStr := ""
		if h.Port != 0 && h.Port != 22 {
			portStr = fmt.Sprintf("%d", h.Port)
		}
		if len(portStr) > w.port {
			w.port = len(portStr)
		}
		user := h.User
		if user == "" {
			user = cfg.SSHUser
		}
		if len(user) > w.user {
			w.user = len(user)
		}
		jump := effectiveJump(h, cfg)
		if len(jump) > w.jump {
			w.jump = len(jump)
		}
		if len(h.Notes) > w.notes {
			w.notes = len(h.Notes)
		}
	}
	return w
}

func effectiveJump(h model.HostEntry, cfg config.Config) string {
	if h.JumpHost != "" {
		juser := h.JumpUser
		if juser != "" {
			return juser + "@" + h.JumpHost
		}
		return h.JumpHost
	}
	if cfg.GlobalJumphost && cfg.JumpServer != "" {
		juser := cfg.JumpUser
		if juser != "" {
			return juser + "@" + cfg.JumpServer
		}
		return cfg.JumpServer
	}
	return ""
}

func pad(s string, w int) string {
	if len(s) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-len(s))
}

func renderTableHeader(w colWidths) string {
	row := "  " +
		pad("Hostname", w.hostname) + "  " +
		pad("IP / Host", w.host) + "  " +
		pad("Port", w.port) + "  " +
		pad("User", w.user) + "  " +
		pad("Via Jump", w.jump) + "  " +
		"Notes"
	sep := styleRule.Render(strings.Repeat("─", len(row)))
	return styleColHead.Render(row) + "\n" + sep
}

func renderTableRow(h model.HostEntry, cfg config.Config, w colWidths) string {
	portStr := ""
	if h.Port != 0 && h.Port != 22 {
		portStr = fmt.Sprintf("%d", h.Port)
	}
	user := h.User
	if user == "" {
		user = cfg.SSHUser
	}
	jump := effectiveJump(h, cfg)
	return "  " +
		pad(h.Hostname, w.hostname) + "  " +
		pad(h.Host, w.host) + "  " +
		pad(portStr, w.port) + "  " +
		pad(user, w.user) + "  " +
		pad(jump, w.jump) + "  " +
		h.Notes
}

// renderTableRowWithName is like renderTableRow but uses a custom hostname string
// (e.g. with a star prefix for favorites).
func renderTableRowWithName(h model.HostEntry, cfg config.Config, w colWidths, hostname string) string {
	portStr := ""
	if h.Port != 0 && h.Port != 22 {
		portStr = fmt.Sprintf("%d", h.Port)
	}
	user := h.User
	if user == "" {
		user = cfg.SSHUser
	}
	jump := effectiveJump(h, cfg)
	return "  " +
		pad(hostname, w.hostname) + "  " +
		pad(h.Host, w.host) + "  " +
		pad(portStr, w.port) + "  " +
		pad(user, w.user) + "  " +
		pad(jump, w.jump) + "  " +
		h.Notes
}

// ─── Bubbletea model ─────────────────────────────────────────────────────────

// appMode tracks which view the TUI is showing.
type appMode int

const (
	modeHosts         appMode = iota // Host list view (default)
	modeSettings                     // Settings view
	modeTmuxOverview                 // Tmux session overview
	modePluginManager                // Plugin manager view
	modePluginSettings               // Plugin settings view
)

// settingsNavItem is a single row in the settings view.
type settingsNavItem struct {
	category string // category name (Allgemein, SSH, Tmux, TUI, Favoriten)
	key      string // config key
	label    string // display label
	kind     string // "bool", "string", "int", "enum"
	enumOpts []string // options for "enum" type
}

// settingsEditState tracks inline editing.
type settingsEditState int

const (
	settingsBrowse settingsEditState = iota
	settingsEditing
)

// pluginSettingsEditState tracks inline editing for plugin settings.
type pluginSettingsEditState int

const (
	pluginSettingsBrowse  pluginSettingsEditState = iota
	pluginSettingsEditing
)

type tuiModel struct {
	hosts   []model.HostEntry
	cfg     config.Config
	version string
	history []string

	// filtered hosts for live search
	filteredHosts []model.HostEntry

	// favorites
	favorites map[string]bool

	// terminal
	width  int
	height int

	// scroll
	scrollOffset int

	// input
	input textinput.Model
	flags sshpkg.Flags

	// completion
	allHostnames []string

	// result / exit
	result   Result
	quitting bool
	err      error

	// settings mode
	mode              appMode
	settingsCatIdx    int            // selected category index
	settingsItemIdx   int            // selected item index within category
	settingsEditState settingsEditState
	settingsEditBuf   string         // buffer for inline editing
	settingsItems     []settingsNavItem // flat list of all settings items
	settingsCats      []string       // category list

	// tmux overview
	tmuxSessions     []tmux.SessionInfo
	tmuxScrollOffset int
	tmuxError        string // error message from tmux

	// plugin manager
	pluginMgr            *plugin.Manager
	pluginScrollOffset   int

	// plugin settings view
	pluginSettingsPluginID  string
	pluginSettingsOffset    int
	pluginSettingsEditState pluginSettingsEditState
	pluginSettingsEditBuf   string
}

func initialModel(cfg config.Config, hosts []model.HostEntry, version string, pluginMgr *plugin.Manager) tuiModel {
	ti := textinput.New()
	ti.Placeholder = "type hostname or IP…"
	ti.ShowSuggestions = true
	// Remove up/down from suggestion navigation so we can use them for scrolling
	ti.KeyMap.NextSuggestion = key.NewBinding(key.WithKeys("ctrl+n"))
	ti.KeyMap.PrevSuggestion = key.NewBinding(key.WithKeys("ctrl+p"))
	ti.Focus()

	var hostnames []string
	for _, h := range hosts {
		hostnames = append(hostnames, h.Hostname)
	}

	history := loadHistory()
	favs := loadFavorites()
	sortedHosts := sortFavoritesFirst(hosts, favs)

	m := tuiModel{
		hosts:         sortedHosts,
		filteredHosts: sortedHosts,
		cfg:           cfg,
		version:       version,
		history:      history,
		favorites:    favs,
		input:        ti,
		allHostnames: hostnames,
		width:        80,
		height:       24,
		mode:         modeHosts,
		tmuxSessions: nil,
		pluginMgr:    pluginMgr,
		pluginScrollOffset: 0,
	}
	m.initSettings()
	m.applyScheme()
	m.updatePromptStyle()
	m.updateSuggestions()
	return m
}

func (m *tuiModel) applyScheme() {
	applyColorScheme(m.cfg.ColorScheme)
}

func (m *tuiModel) visibleRows() int {
	rows := m.height - 23
	if rows < 3 {
		rows = 3
	}
	return rows
}

func (m *tuiModel) needsScroll() bool {
	return len(m.filteredHosts) > m.visibleRows()
}

func (m *tuiModel) updatePromptStyle() {
	if m.flags.Any() {
		flagStr := m.flags.FlagString()
		m.input.PromptStyle = stylePromptFlag
		m.input.Prompt = flagStr + " >  "
	} else {
		m.input.PromptStyle = stylePromptBase
		m.input.Prompt = ">  "
	}
}

func (m *tuiModel) updateSuggestions() {
	val := m.input.Value()
	// Strip any active slash prefixes from the search term
	searchTerm, _ := sshpkg.ParseSlashPrefixes(val)

	if searchTerm == "" {
		// Show all hosts + history suggestions when nothing typed
		m.filteredHosts = m.hosts
		m.input.SetSuggestions(m.history)
		return
	}

	// Fuzzy match against hostnames
	matches := fuzzy.Find(searchTerm, m.allHostnames)
	suggestions := make([]string, 0, len(matches))
	matchSet := make(map[string]bool, len(matches))
	for _, match := range matches {
		hostname := match.Str
		suggestions = append(suggestions, hostname)
		matchSet[hostname] = true
	}
	m.input.SetSuggestions(suggestions)

	// Filter the displayed host list
	m.filteredHosts = nil
	for _, h := range m.hosts {
		if matchSet[h.Hostname] {
			m.filteredHosts = append(m.filteredHosts, h)
		}
	}
	// Reset scroll offset if filtered list is now smaller
	if m.scrollOffset >= len(m.filteredHosts) && len(m.filteredHosts) > 0 {
		m.scrollOffset = len(m.filteredHosts) - 1
	} else if len(m.filteredHosts) == 0 {
		m.scrollOffset = 0
	}
}


func (m tuiModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case resetPromptMsg:
		m.updatePromptStyle()
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		// Settings mode keyboard handling
		if m.mode == modeSettings {
			switch {
			case msg.Type == tea.KeyCtrlC:
				m.err = errors.New("interrupted")
				m.quitting = true
				return m, tea.Quit

			case msg.Type == tea.KeyCtrlS:
				m.saveSettings()
				m.mode = modeHosts
				m.input.Focus()
				m.updatePromptStyle()
				m.updateSuggestions()
				return m, nil

			case msg.Type == tea.KeyEscape:
				m.saveSettings()
				m.mode = modeHosts
				m.input.Focus()
				m.updatePromptStyle()
				m.updateSuggestions()
				return m, nil

			case m.settingsEditState == settingsEditing:
				// Inline editing mode
				switch {
				case msg.Type == tea.KeyEnter:
					// Confirm edit
					m.applySettingsEdit()
					m.settingsEditState = settingsBrowse
					m.settingsEditBuf = ""
					return m, nil

				case msg.Type == tea.KeyEscape:
					// Cancel edit
					m.settingsEditState = settingsBrowse
					m.settingsEditBuf = ""
					return m, nil

				case msg.Type == tea.KeyBackspace:
					if len(m.settingsEditBuf) > 0 {
						m.settingsEditBuf = m.settingsEditBuf[:len(m.settingsEditBuf)-1]
					}
					return m, nil

				case msg.Type == tea.KeyRunes:
					m.settingsEditBuf += string(msg.Runes)
					return m, nil
				}

			case msg.Type == tea.KeyTab || msg.Type == tea.KeyShiftTab:
				// Switch category
				dir := 1
				if msg.Type == tea.KeyShiftTab {
					dir = -1
				}
				m.settingsCatIdx = (m.settingsCatIdx + dir + len(m.settingsCats)) % len(m.settingsCats)
				m.settingsItemIdx = 0
				return m, nil

			case msg.Type == tea.KeyUp:
				if m.settingsItemIdx > 0 {
					m.settingsItemIdx--
				}
				return m, nil

			case msg.Type == tea.KeyDown:
				cat := m.settingsCats[m.settingsCatIdx]
				itemCount := 0
				for _, item := range m.settingsItems {
					if item.category == cat {
						itemCount++
					}
				}
				if m.settingsItemIdx < itemCount-1 {
					m.settingsItemIdx++
				}
				return m, nil

			case msg.Type == tea.KeyEnter:
				// Start editing the selected item
				item := m.currentSettingsItem()
				if item != nil {
					switch item.kind {
					case "bool":
						m.toggleCfgBool(item.key)
						return m, nil
					case "enum":
						m.cycleSettingsEnum(item)
						return m, nil
					case "string", "int":
						m.settingsEditBuf = m.getCfgVal(item.key)
						m.settingsEditState = settingsEditing
						return m, nil
					}
				}
				return m, nil
			}
			return m, nil
		}

		// Tmux overview mode keyboard handling
		if m.mode == modeTmuxOverview {
			switch {
			case msg.Type == tea.KeyCtrlC:
				m.err = errors.New("interrupted")
				m.quitting = true
				return m, tea.Quit

			case msg.Type == tea.KeyEscape, msg.Runes != nil && string(msg.Runes) == "q":
				m.mode = modeHosts
				m.input.Focus()
				m.updatePromptStyle()
				m.updateSuggestions()
				return m, nil

			case msg.Type == tea.KeyUp:
				if m.tmuxScrollOffset > 0 {
					m.tmuxScrollOffset--
				}
				return m, nil

			case msg.Type == tea.KeyDown:
				if m.tmuxScrollOffset < len(m.tmuxSessions)-1 {
					m.tmuxScrollOffset++
				}
				return m, nil

			case msg.Runes != nil && string(msg.Runes) == "k":
				// Kill selected session
				if len(m.tmuxSessions) > 0 {
					idx := m.tmuxScrollOffset
					if idx < len(m.tmuxSessions) {
						_ = tmux.Kill(m.tmuxSessions[idx].Name)
						m.refreshTmuxSessions()
						if m.tmuxScrollOffset >= len(m.tmuxSessions) && m.tmuxScrollOffset > 0 {
							m.tmuxScrollOffset--
						}
					}
				}
				return m, nil

			case msg.Runes != nil && string(msg.Runes) == "d":
				// Detach all
				tmux.DetachAll(m.cfg.SessionPrefix)
				m.refreshTmuxSessions()
				return m, nil

			case msg.Type == tea.KeyEnter:
				// Attach to selected session
				if len(m.tmuxSessions) > 0 {
					idx := m.tmuxScrollOffset
					if idx < len(m.tmuxSessions) {
						m.result = Result{TmuxAttach: m.tmuxSessions[idx].Name}
						m.quitting = true
						return m, tea.Quit
					}
				}
				return m, nil
			}
			return m, nil
		}

		// Plugin settings mode keyboard handling
		if m.mode == modePluginSettings {
			switch {
			case msg.Type == tea.KeyCtrlC:
				m.err = errors.New("interrupted")
				m.quitting = true
				return m, tea.Quit

			case msg.Type == tea.KeyEscape, msg.Runes != nil && string(msg.Runes) == "q":
				m.mode = modePluginManager
				m.pluginScrollOffset = 0
				return m, nil

			case msg.Type == tea.KeyUp:
				if m.pluginSettingsEditState == pluginSettingsEditing {
					break // don't navigate while editing
				}
				if m.pluginSettingsOffset > 0 {
					m.pluginSettingsOffset--
				}
				return m, nil

			case msg.Type == tea.KeyDown:
				if m.pluginSettingsEditState == pluginSettingsEditing {
					break // don't navigate while editing
				}
				settings := m.pluginMgr.PluginManifest(m.pluginSettingsPluginID)
				if settings != nil && m.pluginSettingsOffset < len(settings.Settings)-1 {
					m.pluginSettingsOffset++
				}
				return m, nil

			case m.pluginSettingsEditState == pluginSettingsEditing:
				switch {
				case msg.Type == tea.KeyEnter:
					m.applyPluginSettingEdit()
					m.pluginSettingsEditState = pluginSettingsBrowse
					m.pluginSettingsEditBuf = ""
					return m, nil

				case msg.Type == tea.KeyEscape:
					m.pluginSettingsEditState = pluginSettingsBrowse
					m.pluginSettingsEditBuf = ""
					return m, nil

				case msg.Type == tea.KeyBackspace:
					if len(m.pluginSettingsEditBuf) > 0 {
						m.pluginSettingsEditBuf = m.pluginSettingsEditBuf[:len(m.pluginSettingsEditBuf)-1]
					}
					return m, nil

				case msg.Type == tea.KeyRunes:
					m.pluginSettingsEditBuf += string(msg.Runes)
					return m, nil
				}
				return m, nil

			case msg.Type == tea.KeyEnter:
				// Toggle bool or open editor for string/int
				settings := m.pluginMgr.PluginManifest(m.pluginSettingsPluginID)
				if settings != nil && m.pluginSettingsOffset < len(settings.Settings) {
					def := settings.Settings[m.pluginSettingsOffset]
					switch def.Kind {
					case "bool":
						curr := m.pluginMgr.GetPluginSetting(m.pluginSettingsPluginID, def.Key)
						newVal := "false"
						if curr == "false" || curr == "" {
							newVal = "true"
						}
						m.pluginMgr.SetPluginSetting(m.pluginSettingsPluginID, def.Key, newVal)
						return m, nil
					case "enum":
						curr := m.pluginMgr.GetPluginSetting(m.pluginSettingsPluginID, def.Key)
						next := false
						for _, opt := range def.EnumOpts {
							if next {
								m.pluginMgr.SetPluginSetting(m.pluginSettingsPluginID, def.Key, opt)
								return m, nil
							}
							if opt == curr {
								next = true
							}
						}
						if len(def.EnumOpts) > 0 {
							m.pluginMgr.SetPluginSetting(m.pluginSettingsPluginID, def.Key, def.EnumOpts[0])
						}
						return m, nil
					case "string", "int":
						curr := m.pluginMgr.GetPluginSetting(m.pluginSettingsPluginID, def.Key)
						if curr == "" {
							curr = def.Default
						}
						m.pluginSettingsEditBuf = curr
						m.pluginSettingsEditState = pluginSettingsEditing
						return m, nil
					}
				}
				return m, nil
			}
			return m, nil
		}

		// Plugin manager mode keyboard handling
		if m.mode == modePluginManager {
			switch {
			case msg.Type == tea.KeyCtrlC:
				m.err = errors.New("interrupted")
				m.quitting = true
				return m, tea.Quit

			case msg.Type == tea.KeyEscape, msg.Runes != nil && string(msg.Runes) == "q":
				m.mode = modeHosts
				m.input.Focus()
				m.updatePromptStyle()
				m.updateSuggestions()
				return m, nil

			case msg.Type == tea.KeyUp:
				if m.pluginScrollOffset > 0 {
					m.pluginScrollOffset--
				}
				return m, nil

			case msg.Type == tea.KeyDown:
				plugins := plugin.All()
				if m.pluginScrollOffset < len(plugins)-1 {
					m.pluginScrollOffset++
				}
				return m, nil

			case msg.Type == tea.KeyEnter:
				// Toggle plugin enable/disable
				plugins := plugin.All()
				if len(plugins) > 0 && m.pluginScrollOffset < len(plugins) {
					p := plugins[m.pluginScrollOffset]
					if m.pluginMgr.IsEnabled(p.ID()) {
						_ = m.pluginMgr.Disable(p.ID())
					} else {
						_ = m.pluginMgr.Enable(p.ID())
					}
				}
				return m, nil

			case msg.Runes != nil && string(msg.Runes) == "s":
				// Open plugin settings
				plugins := plugin.All()
				if len(plugins) > 0 && m.pluginScrollOffset < len(plugins) {
					p := plugins[m.pluginScrollOffset]
					m.pluginSettingsPluginID = p.ID()
					m.pluginSettingsOffset = 0
					m.pluginSettingsEditState = pluginSettingsBrowse
					m.pluginSettingsEditBuf = ""
					m.mode = modePluginSettings
				}
				return m, nil
			}
			return m, nil
		}

		// Host mode keyboard handling
		switch {
		case msg.Type == tea.KeyCtrlC:
			m.err = errors.New("interrupted")
			m.quitting = true
			return m, tea.Quit

		case msg.Type == tea.KeyCtrlS:
			m.mode = modeSettings
			m.input.SetValue("")
			m.input.Blur()
			return m, nil

		case msg.Type == tea.KeyCtrlY:
			// Toggle /c copy flag directly
			m.flags.CopyCmd = !m.flags.CopyCmd
			m.input.SetValue("")
			m.updatePromptStyle()
			m.updateSuggestions()
			return m, nil

		case msg.Type == tea.KeyCtrlT:
			// Toggle /t tmux flag directly
			m.flags.UseTmux = !m.flags.UseTmux
			m.input.SetValue("")
			m.updatePromptStyle()
			m.updateSuggestions()
			return m, nil

		case msg.Type == tea.KeyCtrlO:
			// Tmux session overview
			if !tmux.Available() {
				m.tmuxError = "tmux not found in PATH"
			} else {
				m.tmuxError = ""
				m.refreshTmuxSessions()
			}
			m.mode = modeTmuxOverview
			m.tmuxScrollOffset = 0
			m.input.Blur()
			return m, nil

		case msg.Type == tea.KeyCtrlP:
			// Plugin manager
			m.mode = modePluginManager
			m.pluginScrollOffset = 0
			m.input.Blur()
			return m, nil

		case msg.Type == tea.KeyCtrlF:
			// Favoriten: toggle favorite for the first filtered host
			if len(m.filteredHosts) > 0 {
				hName := m.filteredHosts[0].Hostname
				toggleFavorite(m.favorites, hName)
				// Re-sort hosts and filtered list
				m.hosts = sortFavoritesFirst(m.hosts, m.favorites)
				m.filteredHosts = sortFavoritesFirst(m.filteredHosts, m.favorites)
				// Flash feedback
				star := "☆"
				if m.favorites[hName] {
					star = "★"
				}
				m.input.Prompt = styleDim.Render(star+" "+hName) + " "
				m.input.PromptStyle = styleDim
				return m, tea.Tick(1200*time.Millisecond, func(t time.Time) tea.Msg {
					return resetPromptMsg{}
				})
			}
			return m, nil

		case msg.Type == tea.KeyEnter:
			raw := m.input.Value()



			// Parse any typed slash prefixes
			target, extraFlags := sshpkg.ParseSlashPrefixes(raw)
			// Merge interactively-set flags with typed-prefix flags (XOR = toggle)
			merged := sshpkg.Flags{
				BypassJumphost: m.flags.BypassJumphost != extraFlags.BypassJumphost,
				Verbose:        m.flags.Verbose != extraFlags.Verbose,
				DryRun:         m.flags.DryRun != extraFlags.DryRun,
				Legacy:         m.flags.Legacy != extraFlags.Legacy,
				CopyCmd:        m.flags.CopyCmd != extraFlags.CopyCmd,
				UseTmux:        m.flags.UseTmux != extraFlags.UseTmux,
			}
			target = strings.TrimSpace(target)
			if target == "" {
				return m, nil
			}

			// /c mode: copy SSH command to clipboard instead of connecting
			if merged.CopyCmd {
				entry := model.FindEntry(m.hosts, target)
				cmdLine := sshpkg.CommandLine(entry, m.cfg, merged)
				err := clipboard.WriteAll(cmdLine)
				if err == nil {
					m.input.Prompt = styleDim.Render("📋 Copied!") + " "
					m.input.PromptStyle = styleDim
					m.flags.CopyCmd = false
					m.input.SetValue("")
					m.updatePromptStyle()
					m.updateSuggestions()
					return m, tea.Tick(1200*time.Millisecond, func(t time.Time) tea.Msg {
						return resetPromptMsg{}
					})
				}
				// Clipboard not available — fall through to quit so main.go
				// can print it to stdout (like dry-run).
				m.result = Result{Target: target, Flags: merged}
				m.quitting = true
				return m, tea.Quit
			}

			m.result = Result{Target: target, Flags: merged}
			m.quitting = true
			return m, tea.Quit

		case msg.Type == tea.KeyUp:
			if m.scrollOffset > 0 {
				m.scrollOffset--
			}
			return m, nil

		case msg.Type == tea.KeyDown:
			maxScroll := len(m.hosts) - m.visibleRows()
			if maxScroll < 0 {
				maxScroll = 0
			}
			if m.scrollOffset < maxScroll {
				m.scrollOffset++
			}
			return m, nil

		case msg.Type == tea.KeySpace,
			msg.Type == tea.KeyRunes && string(msg.Runes) == " ":
			val := m.input.Value()
			hostname, parsedFlags := sshpkg.ParseSlashPrefixes(val)



			if hostname == "" && parsedFlags.Any() {
				// Toggle each active flag
				if parsedFlags.BypassJumphost {
					m.flags.BypassJumphost = !m.flags.BypassJumphost
				}
				if parsedFlags.Verbose {
					m.flags.Verbose = !m.flags.Verbose
				}
				if parsedFlags.DryRun {
					m.flags.DryRun = !m.flags.DryRun
				}
				if parsedFlags.Legacy {
					m.flags.Legacy = !m.flags.Legacy
				}
				if parsedFlags.CopyCmd {
					m.flags.CopyCmd = !m.flags.CopyCmd
				}
				if parsedFlags.UseTmux {
					m.flags.UseTmux = !m.flags.UseTmux
				}
				m.input.SetValue("")
				m.updatePromptStyle()
				m.updateSuggestions()
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.updateSuggestions()
	return m, cmd
}

func (m tuiModel) View() string {
	if m.quitting {
		return ""
	}

	// Settings mode view
	if m.mode == modeSettings {
		return m.settingsView()
	}

	// Tmux overview mode
	if m.mode == modeTmuxOverview {
		return m.tmuxOverviewView()
	}

	// Plugin manager mode
	if m.mode == modePluginManager {
		return m.pluginView()
	}

	// Plugin settings mode
	if m.mode == modePluginSettings {
		return m.pluginSettingsView()
	}

	var sb strings.Builder

	// Banner
	sb.WriteString(renderBanner())

	// Header line
	headerLine := styleHeader.Render("woossh") + "  " + m.version + "  ·  "
	if m.cfg.GlobalJumphost && m.cfg.JumpServer != "" {
		juser := m.cfg.JumpUser
		if juser != "" {
			headerLine += "Jump: " + juser + "@" + m.cfg.JumpServer + "  ·  "
		} else {
			headerLine += "Jump: " + m.cfg.JumpServer + "  ·  "
		}
	}
	if m.cfg.SSHUser != "" {
		headerLine += "User: " + m.cfg.SSHUser
	}
	sb.WriteString(headerLine + "\n")
	sb.WriteString(styleRule.Render(strings.Repeat("─", m.width)) + "\n")

	// Table
	w := computeColWidths(m.hosts, m.cfg)
	sb.WriteString(renderTableHeader(w) + "\n")

	display := m.filteredHosts
	visible := m.visibleRows()
	total := len(display)
	end := m.scrollOffset + visible
	if end > total {
		end = total
	}

	if total > 0 {
		slice := display[m.scrollOffset:end]
		for _, h := range slice {
			// Star prefix for favorites
			hostname := h.Hostname
			if m.favorites[hostname] {
				hostname = "★ " + hostname
			}
			// Render row with starred hostname
			sb.WriteString(renderTableRowWithName(h, m.cfg, w, hostname) + "\n")
		}
	}

	// Pad with empty lines to keep layout stable (fixed height for table area)
	rowsRendered := end - m.scrollOffset
	if rowsRendered < 0 {
		rowsRendered = 0
	}
	for i := rowsRendered; i < visible; i++ {
		sb.WriteString("\n")
	}

	sb.WriteString("\n")

	// Hint lines
	sb.WriteString(styleDim.Render("  Tab / type to autocomplete  ·  Enter to connect  ·  Ctrl+C to quit") + "\n")
	sb.WriteString(styleDim.Render("  /o direct  ·  /v verbose  ·  /d dry-run  ·  /l legacy  ·  /c copy  ·  /t tmux (stackable, e.g. /o/v)") + "\n")
	sb.WriteString(styleDim.Render("  Ctrl+S settings  ·  Ctrl+F favorite  ·  Ctrl+Y toggle /c  ·  Ctrl+T tmux  ·  Ctrl+O overview  ·  Ctrl+P plugins") + "\n")

	// Scroll indicator
	if m.needsScroll() {
		indicator := fmt.Sprintf("  ↑ ↓ to scroll  [%d–%d of %d]", m.scrollOffset+1, end, total)
		sb.WriteString(styleDim.Render(indicator) + "\n")
	}

	// Prompt
	sb.WriteString(m.input.View())

	return sb.String()
}

// ─── Settings mode ─────────────────────────────────────────────────────────────

func (m *tuiModel) initSettings() {
	m.settingsCats = []string{"Allgemein", "SSH", "Tmux", "TUI", "Favoriten"}
	m.settingsItems = []settingsNavItem{
		{category: "Allgemein", key: "hosts_path",   label: "Hosts CSV Pfad",   kind: "string"},
		{category: "Allgemein", key: "config_path",  label: "Config INI Pfad",  kind: "string"},
		{category: "Allgemein", key: "default_user", label: "Default User",     kind: "string"},
		{category: "Allgemein", key: "ssh_port",     label: "SSH Port",         kind: "int"},
		{category: "SSH",       key: "identity_file",label: "Identity File",    kind: "string"},
		{category: "SSH",       key: "jump_host",    label: "Jump Host",        kind: "string"},
		{category: "SSH",       key: "forward_agent",label: "Agent Forwarding", kind: "bool"},
		{category: "SSH",       key: "timeout",      label: "Timeout (s)",      kind: "int"},
		{category: "Tmux",      key: "use_tmux",     label: "Tmux nutzen",     kind: "bool"},
		{category: "Tmux",      key: "socket_path",  label: "Socket Pfad",      kind: "string"},
		{category: "Tmux",      key: "session_prefix",label: "Session Prefix",  kind: "string"},
		{category: "TUI",       key: "color_scheme", label: "Farbschema",       kind: "enum", enumOpts: []string{"default", "dark", "light", "monokai", "nord", "gruvbox", "dracula", "solarized"}},
		{category: "TUI",       key: "show_ip",      label: "IP-Spalte",        kind: "bool"},
		{category: "TUI",       key: "show_port",    label: "Port-Spalte",      kind: "bool"},
		{category: "TUI", key: "show_favorites", label: "Favoriten anzeigen", kind: "bool"},
		{category: "TUI",       key: "pager_lines",  label: "Zeilen pro Seite", kind: "int"},
		{category: "Favoriten", key: "favorites_file",label: "Favoriten Datei",  kind: "string"},
		{category: "Favoriten", key: "favorite_sort",label: "Sortierung",       kind: "enum", enumOpts: []string{"name", "manual"}},
	}
}

func (m *tuiModel) getCfgVal(key string) string {
	switch key {
	case "hosts_path":     return m.cfg.HostsPath
	case "config_path":    return m.cfg.ConfigPath
	case "default_user":   return m.cfg.DefaultUser
	case "ssh_port":       return intDisplay(m.cfg.SSHPort)
	case "identity_file":  return m.cfg.IdentityFile
	case "jump_host":      return m.cfg.JumpHost
	case "forward_agent":  return boolDisplay(m.cfg.ForwardAgent)
	case "timeout":        return intDisplay(m.cfg.Timeout)
	case "use_tmux":       return boolDisplay(m.cfg.UseTmux)
	case "socket_path":    return m.cfg.SocketPath
	case "session_prefix": return m.cfg.SessionPrefix
	case "color_scheme":   return m.cfg.ColorScheme
	case "show_ip":        return boolDisplay(m.cfg.ShowIP)
	case "show_port":      return boolDisplay(m.cfg.ShowPort)
	case "show_favorites": return boolDisplay(m.cfg.ShowFavorites)
	case "pager_lines":    return intDisplay(m.cfg.PagerLines)
	case "favorites_file": return m.cfg.FavoritesFile
	case "favorite_sort":  return m.cfg.FavoriteSort
	}
	return ""
}

func (m *tuiModel) setCfgVal(key, val string) {
	switch key {
	case "hosts_path":     m.cfg.HostsPath = val
	case "config_path":    m.cfg.ConfigPath = val
	case "default_user":   m.cfg.DefaultUser = val
	case "identity_file":  m.cfg.IdentityFile = val
	case "jump_host":      m.cfg.JumpHost = val
	case "socket_path":    m.cfg.SocketPath = val
	case "session_prefix": m.cfg.SessionPrefix = val
	case "favorites_file": m.cfg.FavoritesFile = val
	case "favorite_sort":  m.cfg.FavoriteSort = val
	case "color_scheme":
		m.cfg.ColorScheme = val
		m.applyScheme()
	}
}

func (m *tuiModel) toggleCfgBool(key string) {
	switch key {
	case "forward_agent":  m.cfg.ForwardAgent = !m.cfg.ForwardAgent
	case "use_tmux":       m.cfg.UseTmux = !m.cfg.UseTmux
	case "show_ip":        m.cfg.ShowIP = !m.cfg.ShowIP
	case "show_port":      m.cfg.ShowPort = !m.cfg.ShowPort
	case "show_favorites": m.cfg.ShowFavorites = !m.cfg.ShowFavorites
	}
}

func (m *tuiModel) setCfgInt(key string, val int) {
	switch key {
	case "ssh_port":    m.cfg.SSHPort = val
	case "timeout":     m.cfg.Timeout = val
	case "pager_lines": m.cfg.PagerLines = val
	}
}

func boolDisplay(v bool) string {
	if v {
		return "[ja]"
	}
	return "[nein]"
}

func intDisplay(v int) string {
	if v == 0 {
		return ""
	}
	return fmt.Sprintf("%d", v)
}

// settingsView renders the settings TUI.
func (m tuiModel) settingsView() string {
	if m.quitting {
		return ""
	}
	var sb strings.Builder

	sb.WriteString(styleHeader.Render("  ⚙  Settings — woossh "+m.version) + "\n")
	sb.WriteString(styleRule.Render(strings.Repeat("─", m.width)) + "\n\n")

	// Build category index -> item ranges
	catStart := make(map[string]int)
	catEnd := make(map[string]int)
	for i, item := range m.settingsItems {
		if _, ok := catStart[item.category]; !ok {
			catStart[item.category] = i
		}
		catEnd[item.category] = i + 1
	}

	// Render each category
	for ci, cat := range m.settingsCats {
		start := catStart[cat]
		end := catEnd[cat]
		activeCat := ci == m.settingsCatIdx

		// Category header
		var catLine string
		if activeCat {
			catLine = styleHeader.Render("  ── " + cat + " ──")
		} else {
			catLine = styleDim.Render("  ── " + cat + " ──")
		}
		sb.WriteString(catLine + "\n")

		// Items in category
		for ii := start; ii < end; ii++ {
			item := m.settingsItems[ii]
			activeItem := activeCat && ii-start == m.settingsItemIdx
			val := m.getCfgVal(item.key)

			var prefix, suffix string
			if activeItem && m.settingsEditState == settingsEditing {
				// Show edit buffer
				prefix = "  > "
				suffix = stylePromptFlag.Render(" [" + m.settingsEditBuf + "]")
			} else if activeItem {
				prefix = styleCyan.Render("  ▸ ")
				suffix = "  " + styleDim.Render("[Enter] edit")
			} else {
				prefix = "    "
				suffix = ""
			}

			var valStyle string
			if activeItem {
				valStyle = styleHeader.Render(val)
			} else {
				valStyle = val
			}

			line := prefix + pad(item.label, 22) + "  " + valStyle + suffix
			sb.WriteString(line + "\n")
		}
		sb.WriteString("\n")
	}

	// Footer
	sb.WriteString(styleRule.Render(strings.Repeat("─", m.width)) + "\n")
	sb.WriteString(styleDim.Render("  [↑↓] Navigate  [Tab] Category  [Enter] Edit  [Esc] or [Ctrl+S] Save & Close") + "\n")

	return sb.String()
}

// ─── Tmux Overview ──────────────────────────────────────────────────────────

// refreshTmuxSessions reloads the tmux session list.
func (m *tuiModel) refreshTmuxSessions() {
	sessions, err := tmux.List(m.cfg.SessionPrefix)
	if err != nil {
		m.tmuxError = err.Error()
		m.tmuxSessions = nil
		return
	}
	m.tmuxError = ""
	m.tmuxSessions = sessions
	if m.tmuxScrollOffset >= len(m.tmuxSessions) && len(m.tmuxSessions) > 0 {
		m.tmuxScrollOffset = len(m.tmuxSessions) - 1
	}
}

// tmuxOverviewView renders the tmux session overview.
func (m tuiModel) tmuxOverviewView() string {
	if m.quitting {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(styleHeader.Render("  📡  Tmux Session Overview — woossh "+m.version) + "\n")
	sb.WriteString(styleRule.Render(strings.Repeat("─", m.width)) + "\n\n")

	if m.tmuxError != "" {
		sb.WriteString(styleYellow.Render("  ⚠ " + m.tmuxError) + "\n\n")
		sb.WriteString(styleDim.Render("  [Esc] or [q] back") + "\n")
		return sb.String()
	}

	if len(m.tmuxSessions) == 0 {
		sb.WriteString(styleDim.Render("  No active tmux sessions\n") + "\n")
		sb.WriteString(styleDim.Render("  [Esc] or [q] back") + "\n")
		return sb.String()
	}

	// Header row
	hostW := 20
	pidW := 7
	timeW := 20
	statusW := 10

	header := "  " +
		pad("Host", hostW) + "  " +
		pad("PID", pidW) + "  " +
		pad("Created", timeW) + "  " +
		pad("Status", statusW) + "  Windows"
	sb.WriteString(styleColHead.Render(header) + "\n")
	sb.WriteString(styleRule.Render(strings.Repeat("─", m.width)) + "\n")

	// Data rows
	visible := m.visibleRows()
	if visible < 3 {
		visible = 10
	}
	end := m.tmuxScrollOffset + visible
	if end > len(m.tmuxSessions) {
		end = len(m.tmuxSessions)
	}
	slice := m.tmuxSessions[m.tmuxScrollOffset:end]

	for _, s := range slice {
		// Status emoji
		status := "🟡 detached"
		if s.Attached {
			status = "🟢 attached"
		}
		if s.PID <= 0 {
			status = "🔴 zombie"
		}

		created := s.Created.Format("2006-01-02 15:04")
		windows := fmt.Sprintf("%d", s.Windows)

		row := "  " +
			pad(s.Host, hostW) + "  " +
			pad(fmt.Sprintf("%d", s.PID), pidW) + "  " +
			pad(created, timeW) + "  " +
			pad(status, statusW) + "  " + windows
		sb.WriteString(row + "\n")
	}

	// Padding
	for i := len(slice); i < visible; i++ {
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(styleDim.Render("  [↑↓] Scroll  [Enter] Attach  [k] Kill  [d] Detach All  [Esc] or [q] Back") + "\n")

	// Scroll indicator
	if len(m.tmuxSessions) > visible {
		indicator := fmt.Sprintf("  ↑ ↓ to scroll  [%d–%d of %d]", m.tmuxScrollOffset+1, end, len(m.tmuxSessions))
		sb.WriteString(styleDim.Render(indicator) + "\n")
	}

	return sb.String()
}

// saveSettings persists the current config to disk.
func (m *tuiModel) saveSettings() {
	_ = config.Save(m.cfg)
}

// applySettingsEdit applies the edit buffer to the config.
func (m *tuiModel) applySettingsEdit() {
	item := m.currentSettingsItem()
	if item == nil {
		return
	}
	switch item.kind {
	case "string":
		m.setCfgVal(item.key, m.settingsEditBuf)
	case "int":
		val := 0
		for _, c := range m.settingsEditBuf {
			if c >= '0' && c <= '9' {
				val = val*10 + int(c-'0')
			}
		}
		m.setCfgInt(item.key, val)
	}
}

// currentSettingsItem returns the currently selected settings item, or nil.
func (m *tuiModel) currentSettingsItem() *settingsNavItem {
	if m.settingsCatIdx < 0 || m.settingsCatIdx >= len(m.settingsCats) {
		return nil
	}
	cat := m.settingsCats[m.settingsCatIdx]
	idx := 0
	for i, item := range m.settingsItems {
		if item.category == cat {
			if idx == m.settingsItemIdx {
				return &m.settingsItems[i]
			}
			idx++
		}
	}
	return nil
}

// cycleSettingsEnum cycles to the next enum value for the given item.
func (m *tuiModel) cycleSettingsEnum(item *settingsNavItem) {
	current := m.getCfgVal(item.key)
	next := false
	for _, opt := range item.enumOpts {
		if next {
			m.setCfgVal(item.key, opt)
			return
		}
		if opt == current {
			next = true
		}
	}
	// Wrap around to first
	if len(item.enumOpts) > 0 {
		m.setCfgVal(item.key, item.enumOpts[0])
	}
}

// Run launches the interactive TUI and returns the user's selection.
func Run(cfg config.Config, hosts []model.HostEntry, version string, pluginMgr *plugin.Manager) (Result, error) {
	m := initialModel(cfg, hosts, version, pluginMgr)
	p := tea.NewProgram(m, tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return Result{}, err
	}
	fm := final.(tuiModel)
	if fm.err != nil {
		return Result{}, fm.err
	}
	return fm.result, nil
}
