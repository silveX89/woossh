package model

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type HostEntry struct {
	Hostname string
	Host     string
	Port     int
	User     string
	JumpHost string
	JumpUser string
	Notes    string
	Legacy   bool
}

func configDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "woossh")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "woossh")
}

func ConfigDir() string {
	return configDir()
}

func findFile(name string) string {
	if _, err := os.Stat(name); err == nil {
		return name
	}
	p := filepath.Join(configDir(), name)
	if _, err := os.Stat(p); err == nil {
		return p
	}
	return ""
}

func LoadHosts() ([]HostEntry, error) {
	path := findFile("hosts.csv")
	if path == "" {
		return nil, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.TrimLeadingSpace = true
	r.FieldsPerRecord = -1

	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	// Filter blank lines and comments
	var filtered [][]string
	for _, rec := range records {
		if len(rec) == 0 {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(rec[0]), "#") {
			continue
		}
		filtered = append(filtered, rec)
	}

	if len(filtered) == 0 {
		return nil, nil
	}

	entries := parseRecords(filtered)
	sort.Slice(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].Hostname) < strings.ToLower(entries[j].Hostname)
	})
	return entries, nil
}

func parseRecords(records [][]string) []HostEntry {
	if len(records) == 0 {
		return nil
	}

	header := records[0]
	headerLower := make([]string, len(header))
	for i, h := range header {
		headerLower[i] = strings.ToLower(strings.TrimSpace(h))
	}

	hasCol := func(name string) bool {
		for _, h := range headerLower {
			if h == name {
				return true
			}
		}
		return false
	}

	colIdx := func(name string) int {
		for i, h := range headerLower {
			if h == name {
				return i
			}
		}
		return -1
	}

	// Determine if first row is a header
	isHeader := false
	for _, h := range headerLower {
		if h == "hostname" || h == "name" || h == "host" || h == "ip address" || h == "addr" {
			isHeader = true
			break
		}
	}

	if !isHeader {
		// Raw list: col[0] = hostname
		var entries []HostEntry
		for _, rec := range records {
			if len(rec) > 0 && strings.TrimSpace(rec[0]) != "" {
				entries = append(entries, HostEntry{Hostname: strings.TrimSpace(rec[0])})
			}
		}
		return entries
	}

	dataRows := records[1:]

	if hasCol("hostname") {
		return parseFullFormat(dataRows, colIdx)
	} else if hasCol("name") && hasCol("ip address") {
		return parseXIQFormat(dataRows, colIdx)
	} else if hasCol("host") && hasCol("addr") {
		return parseTwoColFormat(dataRows, colIdx)
	} else {
		// Generic: col[0]=hostname, col[1]=IP
		var entries []HostEntry
		for _, rec := range dataRows {
			if len(rec) == 0 {
				continue
			}
			e := HostEntry{Hostname: strings.TrimSpace(rec[0])}
			if len(rec) > 1 {
				e.Host = strings.TrimSpace(rec[1])
			}
			entries = append(entries, e)
		}
		return entries
	}
}

func getCol(rec []string, i int) string {
	if i < 0 || i >= len(rec) {
		return ""
	}
	return strings.TrimSpace(rec[i])
}

func parseBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "yes" || s == "true" || s == "1"
}

func parsePort(s string) int {
	if s == "" {
		return 0
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

func parseFullFormat(rows [][]string, colIdx func(string) int) []HostEntry {
	iHostname := colIdx("hostname")
	iHost := colIdx("host")
	iPort := colIdx("port")
	iUser := colIdx("user")
	iJumphost := colIdx("jumphost")
	iJumpuser := colIdx("jumpuser")
	iNotes := colIdx("notes")
	iLegacy := colIdx("legacy")

	var entries []HostEntry
	for _, rec := range rows {
		hostname := getCol(rec, iHostname)
		if hostname == "" {
			continue
		}
		e := HostEntry{
			Hostname: hostname,
			Host:     getCol(rec, iHost),
			Port:     parsePort(getCol(rec, iPort)),
			User:     getCol(rec, iUser),
			JumpHost: getCol(rec, iJumphost),
			JumpUser: getCol(rec, iJumpuser),
			Notes:    getCol(rec, iNotes),
			Legacy:   parseBool(getCol(rec, iLegacy)),
		}
		entries = append(entries, e)
	}
	return entries
}

func parseXIQFormat(rows [][]string, colIdx func(string) int) []HostEntry {
	iName := colIdx("name")
	iIP := colIdx("ip address")
	iPort := colIdx("port")
	iUser := colIdx("user")
	iJumphost := colIdx("jumphost")
	iJumpuser := colIdx("jumpuser")
	iNotes := colIdx("notes")
	iLegacy := colIdx("legacy")

	var entries []HostEntry
	for _, rec := range rows {
		hostname := getCol(rec, iName)
		if hostname == "" {
			continue
		}
		e := HostEntry{
			Hostname: hostname,
			Host:     getCol(rec, iIP),
			Port:     parsePort(getCol(rec, iPort)),
			User:     getCol(rec, iUser),
			JumpHost: getCol(rec, iJumphost),
			JumpUser: getCol(rec, iJumpuser),
			Notes:    getCol(rec, iNotes),
			Legacy:   parseBool(getCol(rec, iLegacy)),
		}
		entries = append(entries, e)
	}
	return entries
}

func parseTwoColFormat(rows [][]string, colIdx func(string) int) []HostEntry {
	iHost := colIdx("host")
	iAddr := colIdx("addr")

	var entries []HostEntry
	for _, rec := range rows {
		hostname := getCol(rec, iHost)
		if hostname == "" {
			continue
		}
		entries = append(entries, HostEntry{
			Hostname: hostname,
			Host:     getCol(rec, iAddr),
		})
	}
	return entries
}

// FindEntry resolves typed text to a HostEntry using the 4-step priority from the spec.
func FindEntry(hosts []HostEntry, typed string) HostEntry {
	if typed == "" {
		return HostEntry{Hostname: typed, Host: typed}
	}
	lower := strings.ToLower(typed)

	// 1. Exact hostname match
	for _, h := range hosts {
		if strings.ToLower(h.Hostname) == lower {
			return h
		}
	}

	// 2. Unique prefix match
	var prefixMatches []HostEntry
	for _, h := range hosts {
		if strings.HasPrefix(strings.ToLower(h.Hostname), lower) {
			prefixMatches = append(prefixMatches, h)
		}
	}
	if len(prefixMatches) == 1 {
		return prefixMatches[0]
	}

	// 3. Exact IP/host field match
	for _, h := range hosts {
		if strings.ToLower(h.Host) == lower {
			return h
		}
	}

	// 4. Literal fallback
	return HostEntry{Hostname: typed, Host: typed}
}
