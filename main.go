package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/silveX89/woossh/config"
	"github.com/silveX89/woossh/model"
	sshpkg "github.com/silveX89/woossh/ssh"
	"github.com/silveX89/woossh/tmux"
	"github.com/silveX89/woossh/tui"
)

const Version = "v0.2.0"

func main() {
	args := os.Args[1:]

	cfg, _ := config.Load()
	hosts, _ := model.LoadHosts(cfg.HostsPath)

	// --list-hosts: print all hostnames for shell tab completion
	if len(args) == 1 && args[0] == "--list-hosts" {
		for _, h := range hosts {
			fmt.Println(h.Hostname)
		}
		os.Exit(0)
	}

	// --version / -v: print version and exit
	if len(args) == 1 && (args[0] == "--version" || args[0] == "-v") {
		fmt.Println("woossh", Version)
		os.Exit(0)
	}

	// --cleanup: kill stale tmux sessions
	if len(args) >= 1 && args[0] == "--cleanup" {
		cleaned, _ := tmux.Cleanup(tmux.DefaultPrefix, 24*time.Hour)
		fmt.Printf("Cleaned up %d stale tmux sessions\n", cleaned)
		os.Exit(0)
	}

	// --import-ssh-config: parse ~/.ssh/config and merge into hosts.csv
	if len(args) >= 1 && args[0] == "--import-ssh-config" {
		sshPath := filepath.Join(os.Getenv("HOME"), ".ssh", "config")
		if len(args) >= 2 {
			sshPath = args[1]
		}
		imported, err := model.ImportSSHConfig(sshPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, "woossh: error importing", sshPath+":", err)
			os.Exit(1)
		}
		merged := model.MergeHosts(hosts, imported)
		skipped := 0
		for _, h := range imported {
			for _, ex := range hosts {
				if h.Hostname == ex.Hostname {
					skipped++
					break
				}
			}
		}
		fmt.Printf("Imported %d hosts from %s (%d skipped, already in hosts.csv)\n", len(imported), sshPath, skipped)
		fmt.Println("Total hosts after merge:", len(merged))
		os.Exit(0)
	}

	var chosenTarget string
	var chosenFlags sshpkg.Flags

	if len(args) >= 1 && !strings.HasPrefix(args[0], "--") {
		// Direct connect — join all args and parse slash prefixes
		// e.g. woossh server  or  woossh /d server  or  woossh /o/v server
		joined := strings.Join(args, " ")
		chosenTarget, chosenFlags = sshpkg.ParseSlashPrefixes(joined)
	} else {
		// Launch interactive TUI
		result, err := tui.Run(cfg, hosts, Version)
		if err != nil {
			if err.Error() == "interrupted" {
				os.Exit(130)
			}
			fmt.Fprintln(os.Stderr, "woossh:", err)
			os.Exit(1)
		}
		// Tmux overview attach
		if result.TmuxAttach != "" {
			if err := tmux.Attach(result.TmuxAttach); err != nil {
				os.Exit(1)
			}
			os.Exit(0)
		}
		if result.Target == "" {
			fmt.Fprintln(os.Stderr, "woossh: no host provided")
			os.Exit(1)
		}
		chosenTarget = result.Target
		chosenFlags = result.Flags
	}

	entry := model.FindEntry(hosts, chosenTarget)

	// Per-host legacy flag overrides the interactive flag
	if entry.Legacy {
		chosenFlags.Legacy = true
	}

	if chosenFlags.DryRun || chosenFlags.CopyCmd {
		line := sshpkg.CommandLine(entry, cfg, chosenFlags)
		if chosenFlags.CopyCmd {
			fmt.Println("📋", line)
		} else {
			fmt.Println(line)
		}
		os.Exit(0)
	}

	// Append to history before connecting
	tui.AppendHistory(entry.Hostname)

	// Tmux mode: create session, then optionally attach
	if chosenFlags.UseTmux {
		if !tmux.Available() {
			fmt.Fprintln(os.Stderr, "woossh: tmux not found in PATH, ignoring /t flag")
		} else {
			prefix := cfg.SessionPrefix
			if prefix == "" {
				prefix = "woossh_"
			}
			sshArgs := sshpkg.BuildArgs(entry, cfg, chosenFlags)
			// Remove the user@host target from args (it's passed separately)
			target := sshArgs[len(sshArgs)-1]
			sshArgs = sshArgs[:len(sshArgs)-1]
			// Split target into user and host
			user := ""
			host := target
			if at := strings.LastIndex(target, "@"); at >= 0 {
				user = target[:at]
				host = target[at+1:]
			}
			sessionName, err := tmux.Create(user, host, prefix, entry.Port, sshArgs)
			if err != nil {
				fmt.Fprintln(os.Stderr, "woossh: tmux error:", err)
				os.Exit(1)
			}
			fmt.Printf("📡 Created tmux session: %s\n", sessionName)
			// Attach interactively
			err = tmux.Attach(sessionName)
			if err != nil {
				os.Exit(1)
			}
			os.Exit(0)
		}
	}

	exitCode := sshpkg.Connect(entry, cfg, chosenFlags)
	os.Exit(exitCode)
}
