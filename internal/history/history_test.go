package history

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseHistoryLine(t *testing.T) {
	tests := []struct {
		input       string
		expected    string
		hasTimestamp bool
	}{
		// Regular commands (no timestamp)
		{"docker ps -a", "docker ps -a", false},
		{"git status", "git status", false},

		// Zsh extended format (with timestamp)
		{": 1699000000:0;docker build -t test .", "docker build -t test .", true},
		{": 1699000000:0;git commit -m \"test\"", "git commit -m \"test\"", true},

		// Should skip short/common commands
		{"ls", "", false},
		{"cd", "", false},
		{"pwd", "", false},
		{"", "", false},
		{"a", "", false},

		// Whitespace handling
		{"  docker ps  ", "docker ps", false},
	}

	for _, tt := range tests {
		result, ts := parseHistoryLine(tt.input)
		if result != tt.expected {
			t.Errorf("parseHistoryLine(%q) = %q, want %q", tt.input, result, tt.expected)
		}
		if tt.hasTimestamp && ts.IsZero() {
			t.Errorf("parseHistoryLine(%q) should have timestamp", tt.input)
		}
		if !tt.hasTimestamp && !ts.IsZero() {
			t.Errorf("parseHistoryLine(%q) should not have timestamp", tt.input)
		}
	}
}

func TestReadHistoryFrom(t *testing.T) {
	tmpDir := t.TempDir()
	histFile := filepath.Join(tmpDir, ".test_history")

	// Write test history (oldest first, newest last - typical shell history order)
	content := `docker build -t myimage .
: 1699000000:0;kubectl get pods
git status
docker ps -a
ls
cd
`
	if err := os.WriteFile(histFile, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write test history: %v", err)
	}

	entries, err := ReadHistoryFrom(histFile, 10)
	if err != nil {
		t.Fatalf("ReadHistoryFrom failed: %v", err)
	}

	// Should have unique entries (no duplicates), excluding ls/cd
	// docker ps -a, git status, docker build, kubectl get pods = 4 unique
	if len(entries) != 4 {
		t.Errorf("Expected 4 unique entries, got %d", len(entries))
		for i, e := range entries {
			t.Logf("Entry %d: %s", i, e.Command)
		}
	}

	// Most recent should be first (reading from end of file)
	if len(entries) > 0 && entries[0].Command != "docker ps -a" {
		t.Errorf("Expected most recent command 'docker ps -a', got %q", entries[0].Command)
	}
}

func TestReadHistoryFromLimit(t *testing.T) {
	tmpDir := t.TempDir()
	histFile := filepath.Join(tmpDir, ".test_history")

	// Write many commands
	content := ""
	for i := range 100 {
		content += "command" + string(rune('0'+i%10)) + "\n"
	}
	if err := os.WriteFile(histFile, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write test history: %v", err)
	}

	entries, err := ReadHistoryFrom(histFile, 5)
	if err != nil {
		t.Fatalf("ReadHistoryFrom failed: %v", err)
	}

	if len(entries) != 5 {
		t.Errorf("Expected 5 entries with limit, got %d", len(entries))
	}
}

func TestReadHistoryNonExistent(t *testing.T) {
	_, err := ReadHistoryFrom("/nonexistent/path/.history", 10)
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestGetHistoryPath_HISTFILEPriority(t *testing.T) {
	// Create a temp directory with test history files
	tmpDir := t.TempDir()

	// Create a custom HISTFILE location
	customHistFile := filepath.Join(tmpDir, ".custom_history")
	if err := os.WriteFile(customHistFile, []byte("custom command\n"), 0o644); err != nil {
		t.Fatalf("Failed to create custom history file: %v", err)
	}

	// Create fake zsh and bash history files
	zshHistory := filepath.Join(tmpDir, ".zsh_history")
	if err := os.WriteFile(zshHistory, []byte("zsh command\n"), 0o644); err != nil {
		t.Fatalf("Failed to create zsh history file: %v", err)
	}

	bashHistory := filepath.Join(tmpDir, ".bash_history")
	if err := os.WriteFile(bashHistory, []byte("bash command\n"), 0o644); err != nil {
		t.Fatalf("Failed to create bash history file: %v", err)
	}

	// Set HISTFILE to custom location
	oldHistFile := os.Getenv("HISTFILE")
	os.Setenv("HISTFILE", customHistFile)
	defer os.Setenv("HISTFILE", oldHistFile)

	// GetHistoryPath should prioritise HISTFILE over default locations
	path, err := GetHistoryPath()
	if err != nil {
		t.Fatalf("GetHistoryPath failed: %v", err)
	}

	if path != customHistFile {
		t.Errorf("GetHistoryPath should prioritise HISTFILE.\nGot: %s\nWant: %s", path, customHistFile)
	}
}

func TestGetHistoryPath_DotHistoryFallback(t *testing.T) {
	// This tests that ~/.history is checked as a fallback
	// when standard locations don't exist and HISTFILE isn't set
	tmpDir := t.TempDir()

	// Clear HISTFILE
	oldHistFile := os.Getenv("HISTFILE")
	os.Unsetenv("HISTFILE")
	defer os.Setenv("HISTFILE", oldHistFile)

	// Create only ~/.history (not zsh or bash)
	dotHistory := filepath.Join(tmpDir, ".history")
	if err := os.WriteFile(dotHistory, []byte(": 1699000000:0;test command\n"), 0o644); err != nil {
		t.Fatalf("Failed to create .history file: %v", err)
	}

	// Test with a mock home directory - we need GetHistoryPathWithHome for this
	path, err := GetHistoryPathWithHome(tmpDir)
	if err != nil {
		t.Fatalf("GetHistoryPathWithHome failed: %v", err)
	}

	if path != dotHistory {
		t.Errorf("GetHistoryPathWithHome should find ~/.history.\nGot: %s\nWant: %s", path, dotHistory)
	}
}

func TestGetHistoryPath_FallbackOrder(t *testing.T) {
	// Save and clear HISTFILE
	oldHistFile := os.Getenv("HISTFILE")
	os.Unsetenv("HISTFILE")
	defer os.Setenv("HISTFILE", oldHistFile)

	// This test verifies the fallback order when HISTFILE is not set
	// We can't easily test this without mocking the home directory,
	// but we can at least verify the function doesn't error when
	// HISTFILE is unset and default files may or may not exist
	_, err := GetHistoryPath()
	// The function may or may not find a history file depending on the system
	// We're mainly testing it doesn't panic
	_ = err
}

func TestGetHistoryPath_HISTFILENonExistent(t *testing.T) {
	// Set HISTFILE to a non-existent path
	oldHistFile := os.Getenv("HISTFILE")
	os.Setenv("HISTFILE", "/nonexistent/path/.history")
	defer os.Setenv("HISTFILE", oldHistFile)

	// When HISTFILE points to non-existent file, should fall back to defaults
	// This tests the current behaviour - HISTFILE should still be checked first
	// but if it doesn't exist, fallback is appropriate
	_, _ = GetHistoryPath()
	// Just verifying no panic - actual result depends on system state
}

func TestCountArgs(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"ls", 1},
		{"docker ps", 2},
		{"docker ps -a", 3},
		{"git commit -m \"hello world\"", 4},
		{"echo 'single quoted'", 2},
		{"kubectl get pods -n default", 5},
		{"", 0},
		{"   ", 0},
		{"cmd   with   spaces", 3},
		{`echo "nested 'quotes' here"`, 2},
	}

	for _, tt := range tests {
		result := countArgs(tt.input)
		if result != tt.expected {
			t.Errorf("countArgs(%q) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}

func TestGetFrequentCommandsFrom(t *testing.T) {
	tmpDir := t.TempDir()
	histFile := filepath.Join(tmpDir, ".test_history")

	// Write test history with repeated commands (zsh format with recent timestamps)
	// Commands must be 13+ chars to pass filter
	content := `: 1765200000:0;docker ps --all
: 1765200001:0;git status -s
: 1765200002:0;docker ps --all
: 1765200003:0;kubectl get pods -n default
: 1765200004:0;docker ps --all
: 1765200005:0;kubectl get pods -n default
: 1765200006:0;ls
: 1765200007:0;git commit -m "test message"
`
	if err := os.WriteFile(histFile, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write test history: %v", err)
	}

	// Get frequent commands with 2+ args
	commands, err := GetFrequentCommandsFrom(histFile, 365, 2, 10)
	if err != nil {
		t.Fatalf("GetFrequentCommandsFrom failed: %v", err)
	}

	// Should have: docker ps --all (3x), kubectl get pods -n default (2x), git commit -m "test message" (1x)
	// git status -s is only 13 chars so included
	if len(commands) < 3 {
		t.Errorf("Expected at least 3 commands, got %d", len(commands))
		for _, c := range commands {
			t.Logf("  %dx: %s", c.Count, c.Command)
		}
	}

	// Most frequent should be first
	if len(commands) > 0 && commands[0].Command != "docker ps --all" {
		t.Errorf("Expected 'docker ps --all' as most frequent, got %q", commands[0].Command)
	}

	if len(commands) > 0 && commands[0].Count != 3 {
		t.Errorf("Expected count of 3 for 'docker ps --all', got %d", commands[0].Count)
	}
}

func TestGetFrequentCommandsFrom_MinArgs(t *testing.T) {
	tmpDir := t.TempDir()
	histFile := filepath.Join(tmpDir, ".test_history")

	// Commands must be 13+ chars
	content := `docker container list --all
git status --short
kubectl get pods -n default
`
	if err := os.WriteFile(histFile, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write test history: %v", err)
	}

	// Require 4+ args
	commands, err := GetFrequentCommandsFrom(histFile, 365, 4, 10)
	if err != nil {
		t.Fatalf("GetFrequentCommandsFrom failed: %v", err)
	}

	// Should only have: docker container list --all (4 args), kubectl get pods -n default (5 args)
	// git status --short has 3 args so filtered out
	if len(commands) != 2 {
		t.Errorf("Expected 2 commands with 4+ args, got %d", len(commands))
		for _, c := range commands {
			t.Logf("  %dx: %s (args: %d)", c.Count, c.Command, countArgs(c.Command))
		}
	}
}

func TestGetFrequentCommandsFrom_Limit(t *testing.T) {
	tmpDir := t.TempDir()
	histFile := filepath.Join(tmpDir, ".test_history")

	// Commands must be 13+ chars
	content := `command1 argument1
command2 argument1
command3 argument1
command4 argument1
command5 argument1
`
	if err := os.WriteFile(histFile, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write test history: %v", err)
	}

	commands, err := GetFrequentCommandsFrom(histFile, 365, 2, 3)
	if err != nil {
		t.Fatalf("GetFrequentCommandsFrom failed: %v", err)
	}

	if len(commands) != 3 {
		t.Errorf("Expected 3 commands (limit), got %d", len(commands))
	}
}

func TestGetFrequentCommandsFrom_SkipsBkmk(t *testing.T) {
	tmpDir := t.TempDir()
	histFile := filepath.Join(tmpDir, ".test_history")

	// Commands must be 13+ chars
	content := `bkmk add docker ps
bkmk suggest --all
docker ps --all --format json
`
	if err := os.WriteFile(histFile, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write test history: %v", err)
	}

	commands, err := GetFrequentCommandsFrom(histFile, 365, 2, 10)
	if err != nil {
		t.Fatalf("GetFrequentCommandsFrom failed: %v", err)
	}

	// Should only have docker command, bkmk commands filtered out
	if len(commands) != 1 {
		t.Errorf("Expected 1 command (bkmk filtered), got %d", len(commands))
		for _, c := range commands {
			t.Logf("  %s", c.Command)
		}
	}

	if len(commands) > 0 && commands[0].Command != "docker ps --all --format json" {
		t.Errorf("Expected 'docker ps --all --format json', got %q", commands[0].Command)
	}
}

func TestIsMultilineFragment(t *testing.T) {
	tests := []struct {
		input      string
		isFragment bool
	}{
		// Line continuations
		{"git push \\", true},
		{"docker run \\", true},

		// Too long
		{strings.Repeat("x", 301), true},
		{strings.Repeat("x", 300), false},

		// Invalid starts (fragments)
		{`"num_ctx": 512}}'`, true},
		{`("=" * 50)`, true},
		{`) >> "$GITHUB_STEP_SUMMARY"`, true},
		{`-H "X-GitHub-Api-Version"`, true},
		{"", true},

		// Valid shell commands
		{"docker ps -a", false},
		{"git commit -m 'message'", false},
		{"kubectl get pods", false},
		{"./script.sh", false},
		{"/usr/bin/env bash", false},
		{"~/bin/tool", false},
		{"$HOME/bin/tool arg", false},
	}

	for _, tt := range tests {
		result := isMultilineFragment(tt.input)
		if result != tt.isFragment {
			t.Errorf("isMultilineFragment(%q) = %v, want %v", tt.input, result, tt.isFragment)
		}
	}
}
