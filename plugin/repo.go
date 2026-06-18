package plugin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type githubDirEntry struct {
	Name string `json:"name"`
	Type string `json:"type"` // "dir" | "file"
}

// DiscoverRepoPlugins fetches the top-level directory listing of a GitHub
// repository and returns one PluginEntry per subdirectory found.
// API failures are treated as an empty list rather than a hard error so the
// TUI degrades gracefully when offline.
func (m *Manager) DiscoverRepoPlugins(repo PluginRepo) ([]PluginEntry, error) {
	url := repo.URL
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "github.com/")

	parts := strings.SplitN(strings.Trim(url, "/"), "/", 3)
	if len(parts) < 2 {
		return nil, fmt.Errorf("unsupported repo URL: %s", repo.URL)
	}

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents", parts[0], parts[1])

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, nil // offline — return empty list
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil // non-200 — return empty list
	}

	var items []githubDirEntry
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("decode GitHub API response: %w", err)
	}

	pDir := pluginsDir()
	var entries []PluginEntry
	for _, item := range items {
		if item.Type != "dir" {
			continue
		}
		pluginURL := strings.TrimSuffix(repo.URL, "/") + "/" + item.Name
		localDir := pluginLocalDir(pluginURL, pDir)

		status := PluginAvailable
		if _, err := os.Stat(localDir); err == nil {
			if m.IsEnabled(item.Name) {
				status = PluginStatusEnabled
			} else {
				status = PluginDownloaded
			}
		}

		entries = append(entries, PluginEntry{
			ID:     item.Name,
			Name:   item.Name,
			Source: pluginURL,
			Status: status,
		})
	}
	return entries, nil
}
