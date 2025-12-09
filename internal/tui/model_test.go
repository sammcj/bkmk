package tui

import (
	"testing"

	"github.com/sammcj/bkmk/internal/config"
)

func TestNew(t *testing.T) {
	cfg := &config.Config{
		Groups: []config.Group{
			{Name: "test", Commands: []config.Command{}},
		},
	}

	m := New(cfg)

	if m.mode != viewGroups {
		t.Errorf("New() should start in viewGroups mode, got %v", m.mode)
	}

	if len(m.groups) != 1 {
		t.Errorf("Expected 1 group, got %d", len(m.groups))
	}
}

func TestNewWithHistory(t *testing.T) {
	cfg := &config.Config{
		Groups: []config.Group{
			{Name: "test", Commands: []config.Command{}},
		},
	}

	m := NewWithHistory(cfg)

	if m.mode != viewHistory {
		t.Errorf("NewWithHistory() should start in viewHistory mode, got %v", m.mode)
	}

	if !m.startInHistory {
		t.Error("NewWithHistory() should set startInHistory to true")
	}
}

func TestNewWithLastCommand(t *testing.T) {
	cfg := &config.Config{
		Groups: []config.Group{
			{Name: "docker", Commands: []config.Command{}},
			{Name: "git", Commands: []config.Command{}},
		},
	}

	testCmd := "docker build -t myimage ."
	m := NewWithLastCommand(cfg, testCmd)

	if m.mode != viewHistorySelectGroup {
		t.Errorf("NewWithLastCommand() should start in viewHistorySelectGroup mode, got %v", m.mode)
	}

	if m.selectedHistCmd != testCmd {
		t.Errorf("Expected selectedHistCmd to be %q, got %q", testCmd, m.selectedHistCmd)
	}

	if m.cursor != 0 {
		t.Errorf("Expected cursor to be 0, got %d", m.cursor)
	}

	if len(m.groups) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(m.groups))
	}
}

func TestNewWithLastCommand_EmptyGroups(t *testing.T) {
	cfg := &config.Config{
		Groups: []config.Group{},
	}

	testCmd := "some command"
	m := NewWithLastCommand(cfg, testCmd)

	if m.mode != viewHistorySelectGroup {
		t.Errorf("NewWithLastCommand() should start in viewHistorySelectGroup mode, got %v", m.mode)
	}

	if m.selectedHistCmd != testCmd {
		t.Errorf("Expected selectedHistCmd to be %q, got %q", testCmd, m.selectedHistCmd)
	}

	if len(m.groups) != 0 {
		t.Errorf("Expected 0 groups, got %d", len(m.groups))
	}
}
