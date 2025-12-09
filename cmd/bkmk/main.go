package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sammcj/bkmk/internal/config"
	"github.com/sammcj/bkmk/internal/history"
	"github.com/sammcj/bkmk/internal/runner"
	"github.com/sammcj/bkmk/internal/tui"
)

// Build-time variables set via ldflags
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

func main() {
	if len(os.Args) < 2 {
		runTUI()
		return
	}

	switch os.Args[1] {
	case "add-group", "ag", "--add-group":
		addGroup()
	case "add", "a", "--add":
		addCommand()
	case "remove-group", "rg", "--remove-group":
		removeGroup()
	case "remove", "rm", "--remove":
		removeCommand()
	case "list", "ls", "--list":
		listAll()
	case "history", "hist", "--history":
		runHistoryTUI()
	case "last", "-l", "--last":
		addLastCommand()
	case "suggest", "freq", "--suggest":
		suggestCommands()
	case "version", "-v", "--version":
		printVersion()
	case "help", "-h", "--help":
		printHelp()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		printHelp()
		os.Exit(1)
	}
}

func runTUI() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	m := tui.New(cfg)
	p := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}

	// Handle both value and pointer types from TUI
	var selected *config.FlatCommand
	var result string

	switch m := finalModel.(type) {
	case tui.Model:
		selected = m.Selected()
		result = m.ActionResult()
	case *tui.Model:
		selected = m.Selected()
		result = m.ActionResult()
	}

	if selected != nil {
		switch result {
		case "run":
			fmt.Printf("Running: %s\n", selected.Command)
			if err := runner.RunCommand(selected.Command); err != nil {
				fmt.Fprintf(os.Stderr, "Error running command: %v\n", err)
				os.Exit(1)
			}
		case "Copied to clipboard":
			fmt.Println("Copied to clipboard:", selected.Command)
		}
	}
}

func runHistoryTUI() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	m := tui.NewWithHistory(cfg)
	p := tea.NewProgram(m, tea.WithAltScreen())

	_, err = p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}

func addLastCommand() {
	// Get recent commands from shell history, filtering out bkmk commands
	entries, err := history.ReadHistory(20)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading shell history: %v\n", err)
		os.Exit(1)
	}

	// Find the first command that isn't a bkmk command
	var lastCmd string
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Command, "bkmk") {
			lastCmd = entry.Command
			break
		}
	}

	if lastCmd == "" {
		fmt.Fprintln(os.Stderr, "No commands found in shell history (excluding bkmk commands)")
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	m := tui.NewWithLastCommand(cfg, lastCmd)
	p := tea.NewProgram(m, tea.WithAltScreen())

	_, err = p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}

func addGroup() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: bkmk add-group <name>")
		os.Exit(1)
	}

	name := os.Args[2]
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if err := cfg.AddGroup(name); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := cfg.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Group %q created\n", name)
}

func addCommand() {
	if len(os.Args) < 5 {
		fmt.Fprintln(os.Stderr, "Usage: bkmk add <group> <name> <command> [description]")
		os.Exit(1)
	}

	groupName := os.Args[2]
	cmdName := os.Args[3]
	command := os.Args[4]
	var description string
	if len(os.Args) > 5 {
		description = strings.Join(os.Args[5:], " ")
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if err := cfg.AddCommand(groupName, cmdName, command, description); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := cfg.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Command %q added to group %q\n", cmdName, groupName)
}

func removeGroup() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: bkmk remove-group <name>")
		os.Exit(1)
	}

	name := os.Args[2]
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if err := cfg.RemoveGroup(name); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := cfg.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Group %q removed\n", name)
}

func removeCommand() {
	if len(os.Args) < 4 {
		fmt.Fprintln(os.Stderr, "Usage: bkmk remove <group> <name>")
		os.Exit(1)
	}

	groupName := os.Args[2]
	cmdName := os.Args[3]

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if err := cfg.RemoveCommand(groupName, cmdName); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := cfg.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Command %q removed from group %q\n", cmdName, groupName)
}

func listAll() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if len(cfg.Groups) == 0 {
		fmt.Println("No groups configured.")
		return
	}

	for _, g := range cfg.Groups {
		fmt.Printf("\n[%s]\n", g.Name)
		if len(g.Commands) == 0 {
			fmt.Println("  (no commands)")
			continue
		}
		for _, cmd := range g.Commands {
			fmt.Printf("  [%d] %s: %s\n", cmd.ID, cmd.Name, cmd.Command)
			if cmd.Description != "" {
				fmt.Printf("      # %s\n", cmd.Description)
			}
		}
	}
	fmt.Println()
}

func suggestCommands() {
	const (
		defaultDays    = 60
		defaultMinArgs = 2
		defaultLimit   = 20
	)

	commands, err := history.GetFrequentCommands(defaultDays, defaultMinArgs, defaultLimit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading shell history: %v\n", err)
		os.Exit(1)
	}

	if len(commands) == 0 {
		fmt.Println("No frequently used commands found matching criteria.")
		fmt.Println("(Looking for commands with 2+ arguments from the last 60 days)")
		return
	}

	fmt.Println("Frequently used commands (good candidates for bookmarking):")
	fmt.Println()

	for i, cmd := range commands {
		fmt.Printf("%2d. [%dx] %s\n", i+1, cmd.Count, cmd.Command)
	}

	fmt.Println()
	fmt.Println("Add one with: bkmk add <group> \"<name>\" \"<command>\"")
}

func printVersion() {
	fmt.Printf("bkmk %s (commit: %s, built: %s)\n", Version, Commit, BuildDate)
}

func printHelp() {
	help := `bkmk - Command Bookmark Manager

Usage:
  bkmk                              Launch interactive TUI
  bkmk add-group <name>             Create a new group (alias: ag)
  bkmk add <group> <name> <cmd>     Add a command to a group (alias: a)
       [description]
  bkmk remove-group <name>          Remove a group (alias: rg)
  bkmk remove <group> <name>        Remove a command (alias: rm)
  bkmk list                         List all groups and commands (alias: ls)
  bkmk history                      Browse shell history to add commands (alias: hist)
  bkmk last                         Bookmark the last command from shell history (alias: -l)
  bkmk suggest                      Show frequently used commands to bookmark (alias: freq)
  bkmk version                      Show version information
  bkmk help                         Show this help message

TUI Controls:
  j/k, ↑/↓     Navigate
  Enter, Tab   Select group / execute command
  /            Search all commands (fuzzy)
  h            Browse shell history
  a            Add group/command
  e            Edit command
  d            Delete (with confirmation)
  Ctrl+N/P     Navigate in search/history
  Esc          Go back
  q, Ctrl+C    Quit

Examples:
  bkmk add-group docker
  bkmk add docker ps "docker ps -a" "List all containers"
  bkmk add docker logs "docker logs -f" "Follow container logs"
  bkmk history
  bkmk last                         # Bookmark the command you just ran

Config: ~/.config/bkmk/config.yaml
`
	fmt.Print(help)
}
