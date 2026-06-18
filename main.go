package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/silveX89/woossh/config"
	"github.com/silveX89/woossh/model"
	"github.com/silveX89/woossh/plugin"
	sshpkg "github.com/silveX89/woossh/ssh"
	"github.com/silveX89/woossh/tmux"
	"github.com/silveX89/woossh/tui"
)

const Version = "v0.3.2"

func main() {
	args := os.Args[1:]

	cfg, _ := config.Load()
	hosts, _ := model.LoadHosts(cfg.HostsPath)

	// Plugin-Manager initialisieren
	pluginMgr := plugin.NewManager(&cfg, &hosts)
	_ = pluginMgr.LoadState()
	pluginMgr.InitAll()
	defer pluginMgr.Shutdown()

	// Hooks: Hosts anreichern
	hosts = pluginMgr.Hooks().FireHostLoaded(hosts)

	// Plugin subcommand: woossh plugin install/list/remove/help
	if len(args) >= 1 && args[0] == "plugin" {
		runPluginCmd(args[1:], pluginMgr)
		os.Exit(0)
	}

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

	// Plugin CLI commands (--cleanup, etc.) — intercept unknown flags
	if len(args) >= 1 && strings.HasPrefix(args[0], "--") {
		if handled, code := pluginMgr.Hooks().FireCLICommand(args); handled {
			os.Exit(code)
		}
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
	var err error
	var result tui.Result

	if len(args) >= 1 && !strings.HasPrefix(args[0], "--") {
		// Direct connect — join all args and parse slash prefixes
		// e.g. woossh server  or  woossh /d server  or  woossh /o/v server
		joined := strings.Join(args, " ")
		chosenTarget, chosenFlags = sshpkg.ParseSlashPrefixes(joined)
	} else {
		// Launch interactive TUI
		result, err = tui.Run(cfg, hosts, Version, pluginMgr)
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

	// Plugin Hooks: BeforeConnect (Plugins können die Verbindung modifizieren oder abbrechen)
	entry, chosenFlags, err = pluginMgr.Hooks().FireBeforeConnect(entry, chosenFlags)
	if err != nil {
		fmt.Fprintln(os.Stderr, "woossh: plugin intercept:", err)
		os.Exit(1)
	}

	exitCode := sshpkg.Connect(entry, cfg, chosenFlags)
	pluginMgr.Hooks().FireAfterConnect(entry, exitCode)
	os.Exit(exitCode)
}

// runPluginCmd handles woossh plugin install/list/remove/help.
func runPluginCmd(args []string, mgr *plugin.Manager) {
	if len(args) == 0 {
		printPluginHelp()
		return
	}
	switch args[0] {
	case "install":
		runInstall(args[1:], mgr)
	case "remove", "rm", "uninstall":
		runRemove(args[1:], mgr)
	case "list", "ls":
		runList(args[1:], mgr)
	case "help", "--help", "-h":
		printPluginHelp()
	default:
		fmt.Fprintf(os.Stderr, "woossh: unknown plugin command %q\n", args[0])
		printPluginHelp()
		os.Exit(1)
	}
}

func printPluginHelp() {
	fmt.Println(`Usage: woossh plugin <command> [options]

Plugin management commands:
  install <url> [--rebuild]  Install a plugin from a Git URL
  remove <id>                Remove an installed plugin
  list                       List all plugins (builtin + external)
  help                       Show this help`)
}

func runInstall(args []string, mgr *plugin.Manager) {
	rebuild := false
	var url string
	for _, a := range args {
		if a == "--rebuild" {
			rebuild = true
		} else if !strings.HasPrefix(a, "-") {
			url = a
		}
	}
	if url == "" {
		fmt.Fprintln(os.Stderr, "usage: woossh plugin install <url> [--rebuild]")
		os.Exit(1)
	}
	if err := mgr.InstallPlugin(url, rebuild); err != nil {
		fmt.Fprintf(os.Stderr, "woossh: plugin install: %v\n", err)
		os.Exit(1)
	}
}

func runRemove(args []string, mgr *plugin.Manager) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: woossh plugin remove <id>")
		os.Exit(1)
	}
	id := args[0]
	if err := mgr.RemovePlugin(id); err != nil {
		fmt.Fprintf(os.Stderr, "woossh: plugin remove: %v\n", err)
		os.Exit(1)
	}
}

func runList(args []string, mgr *plugin.Manager) {
	entries, err := mgr.ListPlugins()
	if err != nil {
		fmt.Fprintf(os.Stderr, "woossh: plugin list: %v\n", err)
		os.Exit(1)
	}
	if len(entries) == 0 {
		fmt.Println("No plugins installed or registered.")
		return
	}

	// Header
	fmt.Printf("%-24s %-20s %-10s %-10s %-10s  %s\n",
		"ID", "Name", "Version", "Status", "Trust", "Source")
	fmt.Println(strings.Repeat("─", 100))

	for _, e := range entries {
		status := "available"
		switch e.Status {
		case plugin.PluginStatusEnabled:
			status = "enabled"
		case plugin.PluginDownloaded:
			status = "downloaded"
		case plugin.PluginAvailable:
			status = "available"
		default:
			status = "disabled"
			if e.Status == plugin.PluginAvailable && !e.Enabled {
				status = "available"
			}
		}

		trustStr := "official"
		switch e.Trust {
		case plugin.TrustCommunity:
			trustStr = "community"
		case plugin.TrustUnknown:
			trustStr = "unknown"
		}

		source := e.Source
		if source == "" {
			source = "builtin"
		}

		fmt.Printf("%-24s %-20s %-10s %-10s %-10s  %s\n",
			e.ID, e.Name, e.Version, status, trustStr, source)
	}
}
