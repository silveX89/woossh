package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/silveX89/woossh/config"
	"github.com/silveX89/woossh/model"
	sshpkg "github.com/silveX89/woossh/ssh"
	"github.com/silveX89/woossh/tui"
)

const Version = "v0.1.0"

func main() {
	args := os.Args[1:]

	cfg, _ := config.Load()
	hosts, _ := model.LoadHosts()

	// --list-hosts: print all hostnames for shell tab completion
	if len(args) == 1 && args[0] == "--list-hosts" {
		for _, h := range hosts {
			fmt.Println(h.Hostname)
		}
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

	if chosenFlags.DryRun {
		fmt.Println(sshpkg.CommandLine(entry, cfg, chosenFlags))
		os.Exit(0)
	}

	// Append to history before connecting
	tui.AppendHistory(entry.Hostname)

	exitCode := sshpkg.Connect(entry, cfg, chosenFlags)
	os.Exit(exitCode)
}
