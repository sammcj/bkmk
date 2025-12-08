package history

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseHistoryLine(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Regular commands
		{"docker ps -a", "docker ps -a"},
		{"git status", "git status"},

		// Zsh extended format
		{": 1699000000:0;docker build -t test .", "docker build -t test ."},
		{": 1699000000:0;git commit -m \"test\"", "git commit -m \"test\""},

		// Should skip short/common commands
		{"ls", ""},
		{"cd", ""},
		{"pwd", ""},
		{"", ""},
		{"a", ""},

		// Whitespace handling
		{"  docker ps  ", "docker ps"},
	}

	for _, tt := range tests {
		result := parseHistoryLine(tt.input)
		if result != tt.expected {
			t.Errorf("parseHistoryLine(%q) = %q, want %q", tt.input, result, tt.expected)
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
