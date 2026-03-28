package ssh

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/silveX89/woossh/config"
	"github.com/silveX89/woossh/model"
)

// Flags holds the active slash-command flags for a connection.
type Flags struct {
	BypassJumphost bool // /o
	Verbose        bool // /v
	DryRun         bool // /d
	Legacy         bool // /l
}

// ParseSlashPrefixes strips leading /o /v /d /l prefixes from a typed string
// and returns the remaining hostname and the parsed flags.
func ParseSlashPrefixes(s string) (string, Flags) {
	var flags Flags
	for {
		s = strings.TrimSpace(s)
		if strings.HasPrefix(s, "/o") && (len(s) == 2 || s[2] == '/' || s[2] == ' ') {
			flags.BypassJumphost = true
			s = s[2:]
		} else if strings.HasPrefix(s, "/v") && (len(s) == 2 || s[2] == '/' || s[2] == ' ') {
			flags.Verbose = true
			s = s[2:]
		} else if strings.HasPrefix(s, "/d") && (len(s) == 2 || s[2] == '/' || s[2] == ' ') {
			flags.DryRun = true
			s = s[2:]
		} else if strings.HasPrefix(s, "/l") && (len(s) == 2 || s[2] == '/' || s[2] == ' ') {
			flags.Legacy = true
			s = s[2:]
		} else {
			break
		}
	}
	return strings.TrimSpace(s), flags
}

// FlagString returns a compact display string for active flags, e.g. "[o][v]".
func (f Flags) FlagString() string {
	var parts []string
	if f.BypassJumphost {
		parts = append(parts, "[o]")
	}
	if f.Verbose {
		parts = append(parts, "[v]")
	}
	if f.DryRun {
		parts = append(parts, "[d]")
	}
	if f.Legacy {
		parts = append(parts, "[l]")
	}
	return strings.Join(parts, "")
}

// Any returns true if any flag is set.
func (f Flags) Any() bool {
	return f.BypassJumphost || f.Verbose || f.DryRun || f.Legacy
}

// BuildArgs constructs the ssh argument list for the given entry, config and flags.
func BuildArgs(entry model.HostEntry, cfg config.Config, flags Flags) []string {
	var args []string

	if flags.Verbose {
		args = append(args, "-v")
	}

	if flags.Legacy || entry.Legacy {
		args = append(args, "-o", "HostKeyAlgorithms=+ssh-rsa", "-o", "PubkeyAcceptedAlgorithms=+ssh-rsa")
	}

	// Jump host
	if !flags.BypassJumphost {
		jumphost := entry.JumpHost
		if jumphost == "" && cfg.GlobalJumphost {
			jumphost = cfg.JumpServer
		}
		if jumphost != "" {
			jumpuser := entry.JumpUser
			if jumpuser == "" {
				jumpuser = cfg.JumpUser
			}
			var jumpArg string
			if jumpuser != "" {
				jumpArg = jumpuser + "@" + jumphost
			} else {
				jumpArg = jumphost
			}
			args = append(args, "-J", jumpArg)
		}
	}

	// Port
	if entry.Port != 0 && entry.Port != 22 {
		args = append(args, "-p", strconv.Itoa(entry.Port))
	}

	// Target
	target := entry.Host
	if target == "" {
		target = entry.Hostname
	}

	user := entry.User
	if user == "" {
		user = cfg.SSHUser
	}

	if user != "" {
		args = append(args, user+"@"+target)
	} else {
		args = append(args, target)
	}

	return args
}

// CommandLine returns the full ssh command as a printable string.
func CommandLine(entry model.HostEntry, cfg config.Config, flags Flags) string {
	args := BuildArgs(entry, cfg, flags)
	return "ssh " + strings.Join(args, " ")
}

// Connect runs ssh as a subprocess and returns its exit code.
func Connect(entry model.HostEntry, cfg config.Config, flags Flags) int {
	sshPath, err := exec.LookPath("ssh")
	if err != nil {
		fmt.Fprintln(os.Stderr, "woossh: ssh not found in PATH")
		return 127
	}

	args := BuildArgs(entry, cfg, flags)
	cmd := exec.Command(sshPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		return 1
	}
	return 0
}
