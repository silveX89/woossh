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
	sb.WriteString(styleHeader.Render("  🔌  Plugin Manager — woossh "+m.version) + "\n")
	sb.WriteString(styleRule.Render(strings.Repeat("─", m.width)) + "\n\n")

	plugins := plugin.All()
	if len(plugins) == 0 {
		sb.WriteString(styleDim.Render("  No plugins registered") + "\n\n")
		sb.WriteString(styleDim.Render("  [Esc] or [q] back") + "\n")
		return sb.String()
	}

	// Header row
	nameW := 24
	verW := 10
	statusW := 10

	header := "  " +
		pad("Plugin", nameW) + "  " +
		pad("Version", verW) + "  " +
		pad("Status", statusW) + "  " +
		"Source"
	sb.WriteString(styleColHead.Render(header) + "\n")
	sb.WriteString(styleRule.Render(strings.Repeat("─", m.width)) + "\n")

	// Data rows
	visible := m.visibleRows()
	if visible < 3 {
		visible = 10
	}
	end := m.pluginScrollOffset + visible
	if end > len(plugins) {
		end = len(plugins)
	}
	slice := plugins[m.pluginScrollOffset:end]

	for _, p := range slice {
		mf := p.Manifest()

		// Build prefix arrow
		globalIdx := indexOfPlugin(plugins, p.ID())
		arrow := "  "
		if globalIdx == m.pluginScrollOffset {
			arrow = styleCyan.Render("▸ ")
		}

		enabled := m.pluginMgr != nil && m.pluginMgr.IsEnabled(p.ID())
		status := styleDim.Render("disabled")
		if enabled {
			status = styleYellow.Render("enabled")
		}

		source := mf.RepoURL
		if source == "" {
			source = "builtin"
		}

		row := arrow +
			pad(mf.Name, nameW) + "  " +
			pad(mf.Version, verW) + "  " +
			pad(status, statusW) + "  " +
			source
		sb.WriteString(row + "\n")
	}

	// Padding
	for i := len(slice); i < visible; i++ {
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(styleDim.Render("  [↑↓] Scroll  [Enter] Toggle enable/disable  [s] Settings  [Esc] or [q] Back") + "\n")

	// Scroll indicator
	if len(plugins) > visible {
		indicator := fmt.Sprintf("  ↑ ↓ to scroll  [%d–%d of %d]", m.pluginScrollOffset+1, end, len(plugins))
		sb.WriteString(styleDim.Render(indicator) + "\n")
	}

	return sb.String()
}

// indexOfPlugin returns the index of a plugin with the given ID in the slice.
func indexOfPlugin(plugins []plugin.Plugin, id string) int {
	for i, p := range plugins {
		if p.ID() == id {
			return i
		}
	}
	return -1
}

// pluginDisplayVal returns a human-friendly display value for a plugin setting.
func pluginDisplayVal(v string, def plugin.SettingDef) string {
	if v == "" {
		v = def.Default
	}
	switch def.Kind {
	case "bool":
		if v == "true" || v == "yes" || v == "1" {
			return "[ja]"
		}
		return "[nein]"
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

	sb.WriteString(styleHeader.Render("  ⚙  Plugin Settings: "+settings.Name+" — woossh "+m.version)+"\n")
	sb.WriteString(styleRule.Render(strings.Repeat("─", m.width))+"\n\n")

	if len(settings.Settings) == 0 {
		sb.WriteString(styleDim.Render("  No settings defined for this plugin")+"\n\n")
		sb.WriteString(styleDim.Render("  [Esc] or [q] back")+"\n")
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

		// Show enum options if applicable
		if globalIdx == m.pluginSettingsOffset && len(def.EnumOpts) > 0 && m.pluginSettingsEditState != pluginSettingsEditing {
			opts := "      " + styleDim.Render("Options: "+strings.Join(def.EnumOpts, ", "))
			sb.WriteString(opts + "\n")
		}
	}

	// Padding
	for i := len(slice); i < visible; i++ {
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(styleRule.Render(strings.Repeat("─", m.width))+"\n")
	sb.WriteString(styleDim.Render("  [↑↓] Scroll  [Enter] Edit/Toggle  [Esc] or [q] Back to Plugin Manager")+"\n")

	// Scroll indicator
	if len(settings.Settings) > visible {
		indicator := fmt.Sprintf("  ↑ ↓ to scroll  [%d–%d of %d]", m.pluginSettingsOffset+1, end, len(settings.Settings))
		sb.WriteString(styleDim.Render(indicator) + "\n")
	}

	return sb.String()
}

// applyPluginSettingEdit wendet den Edit-Buffer auf die aktuelle Plugin-Einstellung an.
func (m *tuiModel) applyPluginSettingEdit() {
	settings := m.pluginMgr.PluginManifest(m.pluginSettingsPluginID)
	if settings == nil || m.pluginSettingsOffset >= len(settings.Settings) {
		return
	}
	def := settings.Settings[m.pluginSettingsOffset]
	m.pluginMgr.SetPluginSetting(m.pluginSettingsPluginID, def.Key, m.pluginSettingsEditBuf)
}