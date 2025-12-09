package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sahilm/fuzzy"
	"github.com/sammcj/bkmk/internal/config"
	"github.com/sammcj/bkmk/internal/history"
)

type viewMode int

const (
	viewGroups viewMode = iota
	viewCommands
	viewSearch
	viewAddGroup
	viewEditGroup
	viewAddCommand
	viewEditCommand
	viewDeleteConfirm
	viewHistory
	viewHistorySelectGroup
	viewHistoryAddDetails
	viewActionSelect
)

type deleteTarget int

const (
	deleteCommand deleteTarget = iota
	deleteGroup
)

type Model struct {
	config       *config.Config
	groups       []config.Group
	commands     []config.Command
	flatCommands []config.FlatCommand
	filtered     []config.FlatCommand

	cursor        int
	selectedGroup int
	mode          viewMode
	previousMode  viewMode
	searchInput   textinput.Model
	width         int
	height        int
	selected      *config.FlatCommand
	quitting      bool

	// Form inputs for add/edit
	formInputs    []textinput.Model
	formFocus     int
	formError     string
	editingCmd    *config.Command
	editingCmdIdx int
	editingGroup  string

	// Delete confirmation
	deleteTarget    deleteTarget
	deleteGroupName string
	deleteCmdName   string

	// History browser
	historyEntries  []history.Entry
	filteredHistory []history.Entry
	historySearch   textinput.Model
	selectedHistCmd string
	historyError    string

	// Start in history mode flag
	startInHistory bool

	// Action selection
	actionCmd    *config.FlatCommand
	actionCursor int
	actionResult string
	actionError  string

	// Config path for display
	configPath string
}

func New(cfg *config.Config) Model {
	ti := textinput.New()
	ti.Placeholder = "Search commands..."
	ti.CharLimit = 256
	ti.Width = 50

	histSearch := textinput.New()
	histSearch.Placeholder = "Search history..."
	histSearch.CharLimit = 256
	histSearch.Width = 50

	cfgPath, _ := config.DefaultPath()

	return Model{
		config:        cfg,
		groups:        cfg.Groups,
		flatCommands:  cfg.FlatCommands(),
		filtered:      cfg.FlatCommands(),
		searchInput:   ti,
		historySearch: histSearch,
		mode:          viewGroups,
		width:         80,
		height:        24,
		configPath:    cfgPath,
	}
}

func NewWithHistory(cfg *config.Config) Model {
	m := New(cfg)
	m.startInHistory = true
	m.mode = viewHistory
	if err := m.loadHistory(); err != nil {
		m.historyError = err.Error()
	}
	m.historySearch.Focus()
	return m
}

// NewWithLastCommand creates a TUI that starts directly in group selection
// with the provided command pre-filled, for quickly bookmarking a command.
func NewWithLastCommand(cfg *config.Config, command string) Model {
	m := New(cfg)
	m.selectedHistCmd = command
	m.mode = viewHistorySelectGroup
	m.cursor = 0
	return m
}

func (m *Model) refreshData() {
	m.groups = m.config.Groups
	m.flatCommands = m.config.FlatCommands()
	m.filtered = m.flatCommands
	if m.selectedGroup < len(m.groups) && m.selectedGroup >= 0 {
		m.commands = m.groups[m.selectedGroup].Commands
	}
}

func (m *Model) createFormInputs(placeholders []string, values []string) {
	m.formInputs = make([]textinput.Model, len(placeholders))
	inputWidth := max(m.width-4, 20)
	for i, placeholder := range placeholders {
		ti := textinput.New()
		ti.Placeholder = placeholder
		ti.CharLimit = 512
		ti.Width = inputWidth
		if i < len(values) {
			ti.SetValue(values[i])
		}
		m.formInputs[i] = ti
	}
	m.formFocus = 0
	m.formError = ""
	if len(m.formInputs) > 0 {
		m.formInputs[0].Focus()
	}
}

func (m *Model) loadHistory() error {
	entries, err := history.ReadHistory(500)
	if err != nil {
		return err
	}
	m.historyEntries = entries
	m.filteredHistory = entries
	return nil
}

func (m *Model) updateHistoryFilter() {
	query := m.historySearch.Value()
	if query == "" {
		m.filteredHistory = m.historyEntries
		return
	}

	searchItems := make([]string, len(m.historyEntries))
	for i, entry := range m.historyEntries {
		searchItems[i] = entry.Command
	}

	matches := fuzzy.Find(query, searchItems)
	m.filteredHistory = make([]history.Entry, len(matches))
	for i, match := range matches {
		m.filteredHistory[i] = m.historyEntries[match.Index]
	}

	if m.cursor >= len(m.filteredHistory) {
		m.cursor = max(0, len(m.filteredHistory)-1)
	}
}

func (m Model) Init() tea.Cmd {
	if m.startInHistory {
		return textinput.Blink
	}
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.searchInput.Width = msg.Width - 4
		m.historySearch.Width = msg.Width - 4
		for i := range m.formInputs {
			m.formInputs[i].Width = msg.Width - 4
		}
		return m, nil
	}

	// Update active input
	switch m.mode {
	case viewSearch:
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		m.updateFilter()
		return m, cmd
	case viewHistory:
		var cmd tea.Cmd
		m.historySearch, cmd = m.historySearch.Update(msg)
		m.updateHistoryFilter()
		return m, cmd
	case viewAddGroup, viewEditGroup, viewAddCommand, viewEditCommand, viewHistoryAddDetails:
		if m.formFocus < len(m.formInputs) {
			var cmd tea.Cmd
			m.formInputs[m.formFocus], cmd = m.formInputs[m.formFocus].Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m *Model) updateFilter() {
	query := m.searchInput.Value()
	if query == "" {
		m.filtered = m.flatCommands
		return
	}

	searchItems := make([]string, len(m.flatCommands))
	for i, cmd := range m.flatCommands {
		searchItems[i] = cmd.GroupName + " " + cmd.Name + " " + cmd.Command + " " + cmd.Description
	}

	matches := fuzzy.Find(query, searchItems)
	m.filtered = make([]config.FlatCommand, len(matches))
	for i, match := range matches {
		m.filtered[i] = m.flatCommands[match.Index]
	}

	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
}

func (m Model) selectedGroupValid() bool {
	return m.selectedGroup >= 0 && m.selectedGroup < len(m.groups)
}

func (m Model) maxCursor() int {
	switch m.mode {
	case viewGroups:
		return max(0, len(m.groups)-1)
	case viewCommands:
		return max(0, len(m.commands)-1)
	case viewSearch:
		return max(0, len(m.filtered)-1)
	case viewHistory:
		return max(0, len(m.filteredHistory)-1)
	case viewHistorySelectGroup:
		return len(m.groups) // +1 for "create new" option
	}
	return 0
}

// historyPageSize returns the number of visible history items for page up/down
func (m Model) historyPageSize() int {
	reservedLines := 8
	pageSize := m.height - reservedLines
	if pageSize < 5 {
		pageSize = 5
	}
	return pageSize
}

func (m Model) Selected() *config.FlatCommand {
	return m.selected
}

// ActionResult returns the action result message
func (m Model) ActionResult() string {
	return m.actionResult
}
