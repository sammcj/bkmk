package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const maxBackups = 20

type ActionType string

const (
	ActionNone ActionType = "none"
	ActionCopy ActionType = "copy"
	ActionRun  ActionType = "run"
)

type Command struct {
	ID            int        `yaml:"id"`
	Name          string     `yaml:"name"`
	Command       string     `yaml:"command"`
	Description   string     `yaml:"description,omitempty"`
	DefaultAction ActionType `yaml:"default_action,omitempty"`
}

type Group struct {
	Name     string    `yaml:"name"`
	Commands []Command `yaml:"commands"`
}

type Config struct {
	Groups    []Group `yaml:"groups"`
	NextID    int     `yaml:"next_id,omitempty"`
}

func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".config", "bkmk", "config.yaml"), nil
}

func Load() (*Config, error) {
	path, err := DefaultPath()
	if err != nil {
		return nil, err
	}
	return LoadFrom(path)
}

func LoadFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{Groups: []Group{}, NextID: 1}, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Migration: assign IDs to commands that don't have them
	cfg.migrateIDs()

	return &cfg, nil
}

func (c *Config) migrateIDs() {
	// Find max existing ID
	maxID := 0
	for _, g := range c.Groups {
		for _, cmd := range g.Commands {
			if cmd.ID > maxID {
				maxID = cmd.ID
			}
		}
	}

	// Set NextID if not set or too low
	if c.NextID <= maxID {
		c.NextID = maxID + 1
	}

	// Assign IDs to commands without them
	for i := range c.Groups {
		for j := range c.Groups[i].Commands {
			if c.Groups[i].Commands[j].ID == 0 {
				c.Groups[i].Commands[j].ID = c.NextID
				c.NextID++
			}
		}
	}
}

func (c *Config) Save() error {
	path, err := DefaultPath()
	if err != nil {
		return err
	}
	return c.SaveTo(path)
}

func (c *Config) SaveTo(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create backup of existing config if it exists
	if _, err := os.Stat(path); err == nil {
		if err := createBackup(path); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// createBackup creates a timestamped backup and prunes old backups beyond maxBackups.
func createBackup(path string) error {
	dir := filepath.Dir(path)
	base := filepath.Base(path)

	// Read existing file
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Create backup with timestamp (including microseconds for uniqueness)
	timestamp := time.Now().Format("20060102-150405.000000")
	backupName := fmt.Sprintf("%s.bak.%s", base, timestamp)
	backupPath := filepath.Join(dir, backupName)

	if err := os.WriteFile(backupPath, data, 0o644); err != nil {
		return err
	}

	// Prune old backups
	return pruneBackups(dir, base)
}

// pruneBackups removes oldest backups if count exceeds maxBackups.
func pruneBackups(dir, base string) error {
	pattern := filepath.Join(dir, base+".bak.*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}

	if len(matches) <= maxBackups {
		return nil
	}

	// Sort by filename (timestamp in name ensures chronological order)
	sort.Strings(matches)

	// Remove oldest backups
	toDelete := matches[:len(matches)-maxBackups]
	for _, path := range toDelete {
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("failed to remove old backup %s: %w", path, err)
		}
	}

	return nil
}

// ListBackups returns available backup files sorted oldest to newest.
func ListBackups() ([]string, error) {
	path, err := DefaultPath()
	if err != nil {
		return nil, err
	}

	dir := filepath.Dir(path)
	base := filepath.Base(path)
	pattern := filepath.Join(dir, base+".bak.*")

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	sort.Strings(matches)
	return matches, nil
}

// RestoreBackup restores config from a backup file.
func RestoreBackup(backupPath string) error {
	path, err := DefaultPath()
	if err != nil {
		return err
	}

	// Verify backup exists and has correct prefix
	base := filepath.Base(path)
	if !strings.HasPrefix(filepath.Base(backupPath), base+".bak.") {
		return fmt.Errorf("invalid backup file: %s", backupPath)
	}

	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup: %w", err)
	}

	// Validate it's valid YAML config
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("backup contains invalid config: %w", err)
	}

	// Create backup of current before restoring
	if _, err := os.Stat(path); err == nil {
		if err := createBackup(path); err != nil {
			return fmt.Errorf("failed to backup current config: %w", err)
		}
	}

	return os.WriteFile(path, data, 0o644)
}

func (c *Config) AddGroup(name string) error {
	for _, g := range c.Groups {
		if g.Name == name {
			return fmt.Errorf("group %q already exists", name)
		}
	}
	c.Groups = append(c.Groups, Group{Name: name, Commands: []Command{}})
	return nil
}

func (c *Config) AddCommand(groupName, cmdName, command, description string) error {
	return c.AddCommandWithAction(groupName, cmdName, command, description, ActionNone)
}

func (c *Config) AddCommandWithAction(groupName, cmdName, command, description string, action ActionType) error {
	for i, g := range c.Groups {
		if g.Name == groupName {
			for _, cmd := range g.Commands {
				if cmd.Name == cmdName {
					return fmt.Errorf("command %q already exists in group %q", cmdName, groupName)
				}
			}
			c.Groups[i].Commands = append(c.Groups[i].Commands, Command{
				ID:            c.NextID,
				Name:          cmdName,
				Command:       command,
				Description:   description,
				DefaultAction: action,
			})
			c.NextID++
			return nil
		}
	}
	return fmt.Errorf("group %q not found", groupName)
}

func (c *Config) RemoveGroup(name string) error {
	for i, g := range c.Groups {
		if g.Name == name {
			c.Groups = append(c.Groups[:i], c.Groups[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("group %q not found", name)
}

func (c *Config) RemoveCommand(groupName, cmdNameOrID string) error {
	for i, g := range c.Groups {
		if g.Name == groupName {
			for j, cmd := range g.Commands {
				if cmd.Name == cmdNameOrID || strconv.Itoa(cmd.ID) == cmdNameOrID {
					c.Groups[i].Commands = append(g.Commands[:j], g.Commands[j+1:]...)
					return nil
				}
			}
			return fmt.Errorf("command %q not found in group %q", cmdNameOrID, groupName)
		}
	}
	return fmt.Errorf("group %q not found", groupName)
}

// RemoveCommandByID removes a command by its global ID (no group needed)
func (c *Config) RemoveCommandByID(id int) error {
	for i, g := range c.Groups {
		for j, cmd := range g.Commands {
			if cmd.ID == id {
				c.Groups[i].Commands = append(g.Commands[:j], g.Commands[j+1:]...)
				return nil
			}
		}
	}
	return fmt.Errorf("command with ID %d not found", id)
}

func (c *Config) GetGroup(name string) *Group {
	for i := range c.Groups {
		if c.Groups[i].Name == name {
			return &c.Groups[i]
		}
	}
	return nil
}

// GetCommandByID finds a command by its global ID
func (c *Config) GetCommandByID(id int) (*Command, string) {
	for _, g := range c.Groups {
		for i := range g.Commands {
			if g.Commands[i].ID == id {
				return &g.Commands[i], g.Name
			}
		}
	}
	return nil, ""
}

// GetCommand finds a command by name or ID within a group
func (c *Config) GetCommand(groupName, cmdNameOrID string) (*Command, error) {
	group := c.GetGroup(groupName)
	if group == nil {
		return nil, fmt.Errorf("group %q not found", groupName)
	}

	for i := range group.Commands {
		cmd := &group.Commands[i]
		if cmd.Name == cmdNameOrID || strconv.Itoa(cmd.ID) == cmdNameOrID {
			return cmd, nil
		}
	}
	return nil, fmt.Errorf("command %q not found in group %q", cmdNameOrID, groupName)
}

// UpdateCommand updates a command found by name or ID
func (c *Config) UpdateCommand(groupName, cmdNameOrID, newName, newCommand, newDescription string) error {
	return c.UpdateCommandWithAction(groupName, cmdNameOrID, newName, newCommand, newDescription, "")
}

// UpdateCommandWithAction updates a command with optional action change
func (c *Config) UpdateCommandWithAction(groupName, cmdNameOrID, newName, newCommand, newDescription string, newAction ActionType) error {
	for i, g := range c.Groups {
		if g.Name == groupName {
			for j, cmd := range g.Commands {
				if cmd.Name == cmdNameOrID || strconv.Itoa(cmd.ID) == cmdNameOrID {
					// Check if new name conflicts with another command
					if newName != cmd.Name {
						for k, other := range g.Commands {
							if k != j && other.Name == newName {
								return fmt.Errorf("command %q already exists in group %q", newName, groupName)
							}
						}
					}
					c.Groups[i].Commands[j].Name = newName
					c.Groups[i].Commands[j].Command = newCommand
					c.Groups[i].Commands[j].Description = newDescription
					if newAction != "" {
						c.Groups[i].Commands[j].DefaultAction = newAction
					}
					return nil
				}
			}
			return fmt.Errorf("command %q not found in group %q", cmdNameOrID, groupName)
		}
	}
	return fmt.Errorf("group %q not found", groupName)
}

// SetCommandAction sets the default action for a command
func (c *Config) SetCommandAction(id int, action ActionType) error {
	for i, g := range c.Groups {
		for j := range g.Commands {
			if c.Groups[i].Commands[j].ID == id {
				c.Groups[i].Commands[j].DefaultAction = action
				return nil
			}
		}
	}
	return fmt.Errorf("command with ID %d not found", id)
}

func (c *Config) AllCommands() []Command {
	var all []Command
	for _, g := range c.Groups {
		all = append(all, g.Commands...)
	}
	return all
}

type FlatCommand struct {
	ID            int
	GroupName     string
	Name          string
	Command       string
	Description   string
	DefaultAction ActionType
}

func (c *Config) FlatCommands() []FlatCommand {
	var all []FlatCommand
	for _, g := range c.Groups {
		for _, cmd := range g.Commands {
			all = append(all, FlatCommand{
				ID:            cmd.ID,
				GroupName:     g.Name,
				Name:          cmd.Name,
				Command:       cmd.Command,
				Description:   cmd.Description,
				DefaultAction: cmd.DefaultAction,
			})
		}
	}
	return all
}
