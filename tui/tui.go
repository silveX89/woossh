package tui

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"

	"github.com/silveX89/woossh/config"
	"github.com/silveX89/woossh/model"
	sshpkg "github.com/silveX89/woossh/ssh"
)

// Result is returned from Run after the user makes a selection.
type Result struct {
	Target string
	Flags  sshpkg.Flags
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

// ─── Bubbletea model ─────────────────────────────────────────────────────────

type tuiModel struct {
	hosts   []model.HostEntry
	cfg     config.Config
	version string
	history []string

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
}

func initialModel(cfg config.Config, hosts []model.HostEntry, version string) tuiModel {
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

	m := tuiModel{
		hosts:        hosts,
		cfg:          cfg,
		version:      version,
		history:      history,
		input:        ti,
		allHostnames: hostnames,
		width:        80,
		height:       24,
	}
	m.updatePromptStyle()
	m.updateSuggestions()
	return m
}

func (m *tuiModel) visibleRows() int {
	rows := m.height - 23
	if rows < 3 {
		rows = 3
	}
	return rows
}

func (m *tuiModel) needsScroll() bool {
	return len(m.hosts) > m.visibleRows()
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
		// Show history when nothing typed
		m.input.SetSuggestions(m.history)
		return
	}

	// Fuzzy match against hostnames
	matches := fuzzy.Find(searchTerm, m.allHostnames)
	suggestions := make([]string, 0, len(matches))
	for _, match := range matches {
		suggestions = append(suggestions, match.Str)
	}
	m.input.SetSuggestions(suggestions)
}

// tryConsumeSlashCommand checks if the current input value is exactly a slash
// command token. Called when the user presses Space.
func (m *tuiModel) tryConsumeSlashCommand(val string) bool {
	trimmed := strings.TrimSpace(val)
	changed := false
	switch trimmed {
	case "/o":
		m.flags.BypassJumphost = true
		changed = true
	case "/v":
		m.flags.Verbose = true
		changed = true
	case "/d":
		m.flags.DryRun = true
		changed = true
	case "/l":
		m.flags.Legacy = true
		changed = true
	}
	return changed
}

func (m tuiModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch {
		case msg.Type == tea.KeyCtrlC:
			m.err = errors.New("interrupted")
			m.quitting = true
			return m, tea.Quit

		case msg.Type == tea.KeyEnter:
			raw := m.input.Value()
			// Parse any typed slash prefixes
			target, extraFlags := sshpkg.ParseSlashPrefixes(raw)
			// Merge interactively-set flags with typed-prefix flags
			merged := sshpkg.Flags{
				BypassJumphost: m.flags.BypassJumphost || extraFlags.BypassJumphost,
				Verbose:        m.flags.Verbose || extraFlags.Verbose,
				DryRun:         m.flags.DryRun || extraFlags.DryRun,
				Legacy:         m.flags.Legacy || extraFlags.Legacy,
			}
			target = strings.TrimSpace(target)
			if target == "" {
				return m, nil
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

		case msg.Type == tea.KeyRunes && string(msg.Runes) == " ":
			val := m.input.Value()
			if m.tryConsumeSlashCommand(val) {
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

	visible := m.visibleRows()
	total := len(m.hosts)
	end := m.scrollOffset + visible
	if end > total {
		end = total
	}
	slice := m.hosts[m.scrollOffset:end]

	for _, h := range slice {
		sb.WriteString(renderTableRow(h, m.cfg, w) + "\n")
	}

	sb.WriteString("\n")

	// Hint lines
	sb.WriteString(styleDim.Render("  Tab / type to autocomplete  ·  Enter to connect  ·  Ctrl+C to quit") + "\n")
	sb.WriteString(styleDim.Render("  /o direct  ·  /v verbose  ·  /d dry-run  ·  /l legacy  (stackable, e.g. /o/v)") + "\n")

	// Scroll indicator
	if m.needsScroll() {
		indicator := fmt.Sprintf("  ↑ ↓ to scroll  [%d–%d of %d]", m.scrollOffset+1, end, total)
		sb.WriteString(styleDim.Render(indicator) + "\n")
	}

	// Prompt
	sb.WriteString(m.input.View())

	return sb.String()
}

// ─── Public entry point ───────────────────────────────────────────────────────

// Run launches the interactive TUI and returns the user's selection.
func Run(cfg config.Config, hosts []model.HostEntry, version string) (Result, error) {
	m := initialModel(cfg, hosts, version)
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
