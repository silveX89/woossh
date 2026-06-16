package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// SessionInfo holds info about a single tmux session.
type SessionInfo struct {
	Name     string
	Host     string
	PID      int
	Created  time.Time
	Attached bool
	Windows  int
}

// prefix is the default session name prefix for woossh sessions.
const DefaultPrefix = "woossh_"

// Available checks if tmux is installed.
func Available() bool {
	_, err := exec.LookPath("tmux")
	return err == nil
}

// sessionName builds a unique session name for a host.
// It appends a numeric suffix if a session for that host already exists.
func sessionName(host string, prefix string) string {
	sessions, _ := listRaw()
	// Sanitize: replace dots with underscores (tmux interprets . as pane separator)
	safeHost := strings.ReplaceAll(host, ".", "_")
	base := prefix + safeHost
	name := base
	// Build a set of existing session names for fast lookup
	existing := make(map[string]bool, len(sessions))
	for _, s := range sessions {
		existing[s] = true
	}
	// Keep incrementing suffix until we find a free name
	for suffix := 1; existing[name]; suffix++ {
		name = fmt.Sprintf("%s_%d", base, suffix)
	}
	return name
}

// listRaw returns session names from tmux list-sessions.
func listRaw() ([]string, error) {
	cmd := exec.Command("tmux", "list-sessions", "-F", "#{session_name}")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var names []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			names = append(names, l)
		}
	}
	return names, nil
}

// Create starts a new tmux session running ssh to the given user@host.
// Returns the session name.
func Create(user, host, prefix string, port int, sshArgs []string) (string, error) {
	name := sessionName(host, prefix)

	// Build the ssh command
	sshCmd := "ssh"
	if port > 0 && port != 22 {
		sshCmd += fmt.Sprintf(" -p %d", port)
	}
	for _, arg := range sshArgs {
		sshCmd += " " + arg
	}
	if user != "" {
		sshCmd += " " + user + "@" + host
	} else {
		sshCmd += " " + host
	}

	args := []string{"new-session", "-d", "-s", name, sshCmd}
	cmd := exec.Command("tmux", args...)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("tmux create: %w", err)
	}
	return name, nil
}

// List returns all tmux sessions with detailed info.
func List(prefix string) ([]SessionInfo, error) {
	cmd := exec.Command("tmux", "list-sessions", "-F",
		"#{session_name}\t#{session_created}\t#{session_attached}\t#{pane_pid}\t#{window_count}")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var sessions []SessionInfo
	for _, line := range lines {
		parts := strings.Split(line, "\t")
		if len(parts) < 5 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		// Extract host from session name
		host := strings.TrimPrefix(name, prefix)
		// Strip numeric suffix like _1, _2 if present
		if idx := strings.LastIndex(host, "_"); idx > 0 {
			// Only strip if the suffix is purely numeric
			suffix := host[idx+1:]
			if _, err := strconv.Atoi(suffix); err == nil {
				host = host[:idx]
			}
		}
		// Convert sanitized host back for display (underscores → dots)
		host = strings.ReplaceAll(host, "_", ".")

		createdUnix, _ := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
		created := time.Unix(createdUnix, 0)
		attached := strings.TrimSpace(parts[2]) == "1"
		pid, _ := strconv.Atoi(strings.TrimSpace(parts[3]))
		windows, _ := strconv.Atoi(strings.TrimSpace(parts[4]))

		sessions = append(sessions, SessionInfo{
			Name:     name,
			Host:     host,
			PID:      pid,
			Created:  created,
			Attached: attached,
			Windows:  windows,
		})
	}
	return sessions, nil
}

// Attach attaches to a tmux session. This replaces the current process.
func Attach(name string) error {
	cmd := exec.Command("tmux", "attach-session", "-t", name)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Kill kills a named tmux session.
func Kill(name string) error {
	return exec.Command("tmux", "kill-session", "-t", name).Run()
}

// DetachAll sends detach to all woossh sessions.
func DetachAll(prefix string) {
	sessions, _ := List(prefix)
	for _, s := range sessions {
		if s.Attached {
			// Can only detach from sessions we're attached to
			_ = exec.Command("tmux", "detach-client", "-s", s.Name).Run()
		}
	}
}

// Cleanup kills all woossh tmux sessions older than the given age.
func Cleanup(prefix string, maxAge time.Duration) (int, error) {
	sessions, err := List(prefix)
	if err != nil {
		return 0, err
	}
	now := time.Now()
	count := 0
	for _, s := range sessions {
		if !s.Attached && now.Sub(s.Created) > maxAge {
			if err := Kill(s.Name); err == nil {
				count++
			}
		}
	}
	return count, nil
}