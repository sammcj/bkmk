package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigLoadSave(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.yaml")

	cfg := &Config{
		Groups: []Group{
			{
				Name: "docker",
				Commands: []Command{
					{Name: "ps", Command: "docker ps -a", Description: "List containers"},
				},
			},
		},
	}

	if err := cfg.SaveTo(path); err != nil {
		t.Fatalf("SaveTo failed: %v", err)
	}

	loaded, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom failed: %v", err)
	}

	if len(loaded.Groups) != 1 {
		t.Errorf("expected 1 group, got %d", len(loaded.Groups))
	}

	if loaded.Groups[0].Name != "docker" {
		t.Errorf("expected group name 'docker', got %q", loaded.Groups[0].Name)
	}

	if len(loaded.Groups[0].Commands) != 1 {
		t.Errorf("expected 1 command, got %d", len(loaded.Groups[0].Commands))
	}

	cmd := loaded.Groups[0].Commands[0]
	if cmd.Name != "ps" || cmd.Command != "docker ps -a" {
		t.Errorf("command mismatch: got %+v", cmd)
	}
}

func TestConfigLoadNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "nonexistent.yaml")

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom should not fail for non-existent file: %v", err)
	}

	if cfg == nil {
		t.Fatal("expected empty config, got nil")
	}

	if len(cfg.Groups) != 0 {
		t.Errorf("expected 0 groups, got %d", len(cfg.Groups))
	}
}

func TestAddGroup(t *testing.T) {
	cfg := &Config{}

	if err := cfg.AddGroup("test"); err != nil {
		t.Fatalf("AddGroup failed: %v", err)
	}

	if len(cfg.Groups) != 1 {
		t.Errorf("expected 1 group, got %d", len(cfg.Groups))
	}

	if cfg.Groups[0].Name != "test" {
		t.Errorf("expected group name 'test', got %q", cfg.Groups[0].Name)
	}

	// Adding duplicate should fail
	if err := cfg.AddGroup("test"); err == nil {
		t.Error("expected error when adding duplicate group")
	}
}

func TestAddCommand(t *testing.T) {
	cfg := &Config{}
	if err := cfg.AddGroup("docker"); err != nil {
		t.Fatalf("AddGroup failed: %v", err)
	}

	if err := cfg.AddCommand("docker", "ps", "docker ps", "List containers"); err != nil {
		t.Fatalf("AddCommand failed: %v", err)
	}

	group := cfg.GetGroup("docker")
	if group == nil {
		t.Fatal("group not found")
	}

	if len(group.Commands) != 1 {
		t.Errorf("expected 1 command, got %d", len(group.Commands))
	}

	cmd := group.Commands[0]
	if cmd.Name != "ps" {
		t.Errorf("expected command name 'ps', got %q", cmd.Name)
	}

	// Adding to non-existent group should fail
	if err := cfg.AddCommand("nonexistent", "test", "test", ""); err == nil {
		t.Error("expected error when adding to non-existent group")
	}

	// Adding duplicate command should fail
	if err := cfg.AddCommand("docker", "ps", "docker ps -a", ""); err == nil {
		t.Error("expected error when adding duplicate command")
	}
}

func TestRemoveGroup(t *testing.T) {
	cfg := &Config{}
	if err := cfg.AddGroup("test1"); err != nil {
		t.Fatalf("AddGroup test1 failed: %v", err)
	}
	if err := cfg.AddGroup("test2"); err != nil {
		t.Fatalf("AddGroup test2 failed: %v", err)
	}

	if err := cfg.RemoveGroup("test1"); err != nil {
		t.Fatalf("RemoveGroup failed: %v", err)
	}

	if len(cfg.Groups) != 1 {
		t.Errorf("expected 1 group, got %d", len(cfg.Groups))
	}

	if cfg.Groups[0].Name != "test2" {
		t.Errorf("expected remaining group 'test2', got %q", cfg.Groups[0].Name)
	}

	// Removing non-existent should fail
	if err := cfg.RemoveGroup("nonexistent"); err == nil {
		t.Error("expected error when removing non-existent group")
	}
}

func TestRemoveCommand(t *testing.T) {
	cfg := &Config{}
	if err := cfg.AddGroup("docker"); err != nil {
		t.Fatalf("AddGroup failed: %v", err)
	}
	if err := cfg.AddCommand("docker", "ps", "docker ps", ""); err != nil {
		t.Fatalf("AddCommand ps failed: %v", err)
	}
	if err := cfg.AddCommand("docker", "logs", "docker logs", ""); err != nil {
		t.Fatalf("AddCommand logs failed: %v", err)
	}

	if err := cfg.RemoveCommand("docker", "ps"); err != nil {
		t.Fatalf("RemoveCommand failed: %v", err)
	}

	group := cfg.GetGroup("docker")
	if len(group.Commands) != 1 {
		t.Errorf("expected 1 command, got %d", len(group.Commands))
	}

	if group.Commands[0].Name != "logs" {
		t.Errorf("expected remaining command 'logs', got %q", group.Commands[0].Name)
	}

	// Removing from non-existent group should fail
	if err := cfg.RemoveCommand("nonexistent", "test"); err == nil {
		t.Error("expected error when removing from non-existent group")
	}

	// Removing non-existent command should fail
	if err := cfg.RemoveCommand("docker", "nonexistent"); err == nil {
		t.Error("expected error when removing non-existent command")
	}
}

func TestFlatCommands(t *testing.T) {
	cfg := &Config{
		Groups: []Group{
			{
				Name: "docker",
				Commands: []Command{
					{Name: "ps", Command: "docker ps"},
					{Name: "logs", Command: "docker logs"},
				},
			},
			{
				Name: "git",
				Commands: []Command{
					{Name: "status", Command: "git status"},
				},
			},
		},
	}

	flat := cfg.FlatCommands()
	if len(flat) != 3 {
		t.Errorf("expected 3 flat commands, got %d", len(flat))
	}

	// Verify group names are preserved
	dockerCount := 0
	gitCount := 0
	for _, cmd := range flat {
		switch cmd.GroupName {
		case "docker":
			dockerCount++
		case "git":
			gitCount++
		}
	}

	if dockerCount != 2 || gitCount != 1 {
		t.Errorf("unexpected group distribution: docker=%d, git=%d", dockerCount, gitCount)
	}
}

func TestSaveCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "nested", "deep", "config.yaml")

	cfg := &Config{Groups: []Group{{Name: "test"}}}

	if err := cfg.SaveTo(path); err != nil {
		t.Fatalf("SaveTo failed to create directories: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("config file was not created")
	}
}

func TestBackupCreatedOnSave(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.yaml")

	// Create initial config
	cfg := &Config{Groups: []Group{{Name: "initial"}}}
	if err := cfg.SaveTo(path); err != nil {
		t.Fatalf("initial SaveTo failed: %v", err)
	}

	// Save again - this should create a backup
	cfg.Groups[0].Name = "updated"
	if err := cfg.SaveTo(path); err != nil {
		t.Fatalf("second SaveTo failed: %v", err)
	}

	// Check backup was created
	matches, err := filepath.Glob(filepath.Join(tmpDir, "config.yaml.bak.*"))
	if err != nil {
		t.Fatalf("Glob failed: %v", err)
	}

	if len(matches) != 1 {
		t.Errorf("expected 1 backup file, got %d", len(matches))
	}

	// Verify backup contains original content
	if len(matches) > 0 {
		data, err := os.ReadFile(matches[0])
		if err != nil {
			t.Fatalf("failed to read backup: %v", err)
		}
		if !strings.Contains(string(data), "initial") {
			t.Error("backup should contain original 'initial' group name")
		}
	}
}

func TestBackupPruning(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.yaml")

	cfg := &Config{Groups: []Group{{Name: "test"}}}

	// Create initial file
	if err := cfg.SaveTo(path); err != nil {
		t.Fatalf("initial SaveTo failed: %v", err)
	}

	// Save 25 times to exceed maxBackups (20)
	for i := 0; i < 25; i++ {
		cfg.Groups[0].Name = "test" + string(rune('A'+i))
		if err := cfg.SaveTo(path); err != nil {
			t.Fatalf("SaveTo iteration %d failed: %v", i, err)
		}
	}

	// Check we have exactly maxBackups
	matches, err := filepath.Glob(filepath.Join(tmpDir, "config.yaml.bak.*"))
	if err != nil {
		t.Fatalf("Glob failed: %v", err)
	}

	if len(matches) != maxBackups {
		t.Errorf("expected %d backups, got %d", maxBackups, len(matches))
	}
}

func TestRestoreBackup(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.yaml")

	// Create and save original config
	cfg := &Config{Groups: []Group{{Name: "original"}}}
	if err := cfg.SaveTo(path); err != nil {
		t.Fatalf("initial SaveTo failed: %v", err)
	}

	// Save updated config (creates backup of original)
	cfg.Groups[0].Name = "updated"
	if err := cfg.SaveTo(path); err != nil {
		t.Fatalf("second SaveTo failed: %v", err)
	}

	// Find the backup
	matches, err := filepath.Glob(filepath.Join(tmpDir, "config.yaml.bak.*"))
	if err != nil || len(matches) == 0 {
		t.Fatal("no backup found")
	}

	// Restore using the internal function (RestoreBackup uses DefaultPath)
	backupData, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("failed to read backup: %v", err)
	}
	if err := os.WriteFile(path, backupData, 0o644); err != nil {
		t.Fatalf("failed to restore: %v", err)
	}

	// Verify restoration
	loaded, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom failed: %v", err)
	}

	if loaded.Groups[0].Name != "original" {
		t.Errorf("expected restored group name 'original', got %q", loaded.Groups[0].Name)
	}
}
