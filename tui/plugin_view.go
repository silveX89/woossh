package tui

import (
	"fmt"
	"strings"

	"github.com/silveX89/woossh/plugin"
)

// pluginView renders the plugin manager UI.
func (m tuiModel) pluginView() string {
	if m.quitting {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(styleHeader.Render("  Plugin Manager — woossh "+m.version) + "\n")
	sb.WriteString(styleRule.Render(strings.Repeat("─", m.width)) + "\n\n")

	entries := m.pluginEntries
	if len(entries) == 0 {
		sb.WriteString(styleDim.Render("  No plugins found") + "\n\n")
		sb.WriteString(styleDim.Render("  [s] Repo settings  [Esc/q] Back") + "\n")
		return sb.String()
	}

	nameW := 22
	verW := 14
	statusW := 12

	header := "  " +
		pad("Plugin", nameW) + "  " +
		pad("Version", verW) + "  " +
		pad("Status", statusW) + "  " +
		"Source"
	sb.WriteString(styleColHead.Render(header) + "\n")
	sb.WriteString(styleRule.Render(strings.Repeat("─", m.width)) + "\n")

	visible := m.visibleRows()
	if visible < 3 {
		visible = 10
	}
	end := m.pluginScrollOffset + visible
	if end > len(entries) {
		end = len(entries)
	}
	slice := entries[m.pluginScrollOffset:end]

	for i, e := range slice {
		globalIdx := m.pluginScrollOffset + i
		arrow := "  "
		if globalIdx == m.pluginScrollOffset {
			arrow = styleCyan.Render("▸ ")
		}

		var statusStr string
		switch e.Status {
		case plugin.PluginStatusEnabled:
			statusStr = styleYellow.Render("enabled")
		case plugin.PluginDownloaded:
			statusStr = styleCyan.Render("downloaded")
		default:
			statusStr = styleDim.Render("available")
		}

		name := e.Name
		if name == "" {
			name = e.ID
		}
		ver := e.Version
		if ver == "" {
			ver = "—"
		}
		source := e.Source
		if source == "" {
			source = "builtin"
		}
		// Truncate long source URLs to fit
		maxSrc := m.width - nameW - verW - statusW - 10
		if maxSrc < 10 {
			maxSrc = 10
		}
		if len(source) > maxSrc {
			source = "…" + source[len(source)-maxSrc+1:]
		}

		row := arrow +
			pad(name, nameW) + "  " +
			pad(ver, verW) + "  " +
			pad(statusStr, statusW) + "  " +
			source
		sb.WriteString(row + "\n")
	}

	for i := len(slice); i < visible; i++ {
		sb.WriteString("\n")
	}

	if m.pluginOpStatus != "" {
		sb.WriteString("\n" + styleDim.Render("  "+m.pluginOpStatus) + "\n")
	} else {
		sb.WriteString("\n")
	}

	sb.WriteString(styleDim.Render("  [↑↓] Scroll  [Enter] Toggle  [d] Download  [r] Remove  [s] Repos  [Esc/q] Back") + "\n")

	if len(entries) > visible {
		indicator := fmt.Sprintf("  ↑ ↓ to scroll  [%d–%d of %d]", m.pluginScrollOffset+1, end, len(entries))
		sb.WriteString(styleDim.Render(indicator) + "\n")
	}

	return sb.String()
}

// pluginRepoView renders the plugin repository management UI.
func (m tuiModel) pluginRepoView() string {
	if m.quitting {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(styleHeader.Render("  Plugin Repos — woossh "+m.version) + "\n")
	sb.WriteString(styleRule.Render(strings.Repeat("─", m.width)) + "\n\n")

	var repos []plugin.PluginRepo
	if m.pluginMgr != nil {
		repos = m.pluginMgr.GetRepos()
	}

	if len(repos) == 0 {
		sb.WriteString(styleDim.Render("  No repositories configured") + "\n\n")
	} else {
		nameW := 24
		urlW := m.width - nameW - 16
		if urlW < 20 {
			urlW = 20
		}

		header := "  " + pad("Name", nameW) + "  " + pad("URL", urlW) + "  " + "Enabled"
		sb.WriteString(styleColHead.Render(header) + "\n")
		sb.WriteString(styleRule.Render(strings.Repeat("─", m.width)) + "\n")

		for i, r := range repos {
			arrow := "  "
			if i == m.pluginRepoOffset {
				arrow = styleCyan.Render("▸ ")
			}

			enabledStr := styleDim.Render("no")
			if r.Enabled {
				enabledStr = styleYellow.Render("yes")
			}

			url := r.URL
			if len(url) > urlW {
				url = "…" + url[len(url)-urlW+1:]
			}

			row := arrow + pad(r.Name, nameW) + "  " + pad(url, urlW) + "  " + enabledStr
			sb.WriteString(row + "\n")
		}
	}

	sb.WriteString("\n")

	if m.pluginRepoAddMode {
		sb.WriteString(styleHeader.Render("  Add repo URL: ") + m.pluginRepoAddBuf + styleDim.Render("█") + "\n")
		sb.WriteString(styleDim.Render("  [Enter] confirm  [Esc] cancel") + "\n")
	} else {
		sb.WriteString(styleDim.Render("  [↑↓] Scroll  [Enter] Toggle enabled  [a] Add  [d] Delete  [Esc/q] Back") + "\n")
	}

	return sb.String()
}

// pluginDisplayVal returns a human-friendly display value for a plugin setting.
func pluginDisplayVal(v string, def plugin.SettingDef) string {
	if v == "" {
		v = def.Default
	}
	switch def.Kind {
	case "bool":
		if v == "true" || v == "yes" || v == "1" {
			return "[yes]"
		}
		return "[no]"
	case "int":
		return v
	default:
		return v
	}
}

// pluginSettingsView renders the settings for a specific plugin.
func (m tuiModel) pluginSettingsView() string {
	if m.quitting {
		return ""
	}

	settings := m.pluginMgr.PluginManifest(m.pluginSettingsPluginID)
	var sb strings.Builder

	sb.WriteString(styleHeader.Render("  Plugin Settings: "+settings.Name+" — woossh "+m.version) + "\n")
	sb.WriteString(styleRule.Render(strings.Repeat("─", m.width)) + "\n\n")

	if len(settings.Settings) == 0 {
		sb.WriteString(styleDim.Render("  No settings defined for this plugin") + "\n\n")
		sb.WriteString(styleDim.Render("  [Esc/q] Back") + "\n")
		return sb.String()
	}

	visible := m.visibleRows()
	if visible < 3 {
		visible = 10
	}
	end := m.pluginSettingsOffset + visible
	if end > len(settings.Settings) {
		end = len(settings.Settings)
	}
	slice := settings.Settings[m.pluginSettingsOffset:end]

	labelW := 24
	for _, s := range settings.Settings {
		if len(s.Label) > labelW {
			labelW = len(s.Label)
		}
	}

	for i, def := range slice {
		globalIdx := m.pluginSettingsOffset + i
		val := m.pluginMgr.GetPluginSetting(m.pluginSettingsPluginID, def.Key)
		valStr := pluginDisplayVal(val, def)

		var prefix, suffix string
		if globalIdx == m.pluginSettingsOffset {
			if m.pluginSettingsEditState == pluginSettingsEditing {
				prefix = "  > "
				suffix = stylePromptFlag.Render(" [" + m.pluginSettingsEditBuf + "]")
			} else {
				prefix = styleCyan.Render("  ▸ ")
				suffix = "  " + styleDim.Render("[Enter] edit")
			}
		} else {
			prefix = "    "
			suffix = ""
		}

		var valStyle string
		if globalIdx == m.pluginSettingsOffset {
			valStyle = styleHeader.Render(valStr)
		} else {
			valStyle = valStr
		}

		line := prefix + pad(def.Label, labelW) + "  " + valStyle + suffix
		sb.WriteString(line + "\n")

		if globalIdx == m.pluginSettingsOffset && len(def.EnumOpts) > 0 && m.pluginSettingsEditState != pluginSettingsEditing {
			opts := "      " + styleDim.Render("Options: "+strings.Join(def.EnumOpts, ", "))
			sb.WriteString(opts + "\n")
		}
	}

	for i := len(slice); i < visible; i++ {
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(styleRule.Render(strings.Repeat("─", m.width)) + "\n")
	sb.WriteString(styleDim.Render("  [↑↓] Scroll  [Enter] Edit/Toggle  [Esc/q] Back to Plugin Manager") + "\n")

	if len(settings.Settings) > visible {
		indicator := fmt.Sprintf("  ↑ ↓ to scroll  [%d–%d of %d]", m.pluginSettingsOffset+1, end, len(settings.Settings))
		sb.WriteString(styleDim.Render(indicator) + "\n")
	}

	return sb.String()
}

// applyPluginSettingEdit applies the edit buffer to the current plugin setting.
func (m *tuiModel) applyPluginSettingEdit() {
	settings := m.pluginMgr.PluginManifest(m.pluginSettingsPluginID)
	if settings == nil || m.pluginSettingsOffset >= len(settings.Settings) {
		return
	}
	def := settings.Settings[m.pluginSettingsOffset]
	m.pluginMgr.SetPluginSetting(m.pluginSettingsPluginID, def.Key, m.pluginSettingsEditBuf)
}
