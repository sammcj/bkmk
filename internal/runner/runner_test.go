package runner

import (
	"os"
	"testing"
)

func TestRunCommand(t *testing.T) {
	// Test running a simple command
	err := RunCommand("true")
	if err != nil {
		t.Errorf("RunCommand('true') failed: %v", err)
	}
}

func TestRunCommandWithOutput(t *testing.T) {
	// Test that command actually executes (writes to a temp file)
	tmpFile, err := os.CreateTemp("", "bkmk-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	// Run a command that writes to the file
	err = RunCommand("echo 'test content' > " + tmpPath)
	if err != nil {
		t.Errorf("RunCommand failed: %v", err)
	}

	// Verify the file was written
	content, err := os.ReadFile(tmpPath)
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}

	if string(content) != "test content\n" {
		t.Errorf("Expected 'test content\\n', got %q", string(content))
	}
}

func TestRunCommandFailure(t *testing.T) {
	// Test that failing commands return an error
	err := RunCommand("false")
	if err == nil {
		t.Error("Expected error from RunCommand('false'), got nil")
	}
}

func TestRunCommandUsesShell(t *testing.T) {
	// Test that the command uses shell features (pipes, etc.)
	tmpFile, err := os.CreateTemp("", "bkmk-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	// Run a command with shell features (pipe)
	err = RunCommand("echo 'hello' | cat > " + tmpPath)
	if err != nil {
		t.Errorf("RunCommand with pipe failed: %v", err)
	}

	content, err := os.ReadFile(tmpPath)
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}

	if string(content) != "hello\n" {
		t.Errorf("Expected 'hello\\n', got %q", string(content))
	}
}

func TestOpenInEditor(t *testing.T) {
	tests := []struct {
		name       string
		editorEnv  string
		wantEditor string
	}{
		{
			name:       "uses EDITOR env var",
			editorEnv:  "nano",
			wantEditor: "nano",
		},
		{
			name:       "falls back to vi when EDITOR not set",
			editorEnv:  "",
			wantEditor: "vi",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalEditor := os.Getenv("EDITOR")
			defer os.Setenv("EDITOR", originalEditor)

			if tt.editorEnv == "" {
				os.Unsetenv("EDITOR")
			} else {
				os.Setenv("EDITOR", tt.editorEnv)
			}

			cmd, err := OpenInEditor("/tmp/test")
			if err != nil {
				t.Fatalf("OpenInEditor failed: %v", err)
			}

			if cmd.Path == "" {
				t.Fatal("Expected command path to be set")
			}

			if len(cmd.Args) < 2 {
				t.Fatalf("Expected at least 2 args, got %d", len(cmd.Args))
			}

			if cmd.Args[1] != "/tmp/test" {
				t.Errorf("Expected path arg '/tmp/test', got %q", cmd.Args[1])
			}
		})
	}
}
