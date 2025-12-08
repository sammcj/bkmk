package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"
	"github.com/sammcj/bkmk/internal/config"
	"github.com/sammcj/bkmk/internal/history"
	"github.com/sammcj/bkmk/internal/runner"
)

type viewMode int

const (
	viewGroups viewMode = iota
	viewCommands
	viewSearch
	viewAddGroup
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
	actionCmd       *config.FlatCommand
	actionCursor    int
	actionResult    string
	actionError     string
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
	}
}

func NewWithHistory(cfg *config.Config) Model {
	m := New(cfg)
	m.startInHistory = true
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

func (m *Model) createFormInputs(fields []string, placeholders []string, values []string) {
	m.formInputs = make([]textinput.Model, len(fields))
	inputWidth := m.width - 4
	if inputWidth < 20 {
		inputWidth = 20
	}
	for i := range fields {
		ti := textinput.New()
		ti.Placeholder = placeholders[i]
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
		m.mode = viewHistory
		if err := m.loadHistory(); err != nil {
			m.historyError = err.Error()
		}
		m.historySearch.Focus()
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
	case viewAddGroup, viewAddCommand, viewEditCommand, viewHistoryAddDetails:
		if m.formFocus < len(m.formInputs) {
			var cmd tea.Cmd
			m.formInputs[m.formFocus], cmd = m.formInputs[m.formFocus].Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle form modes first
	switch m.mode {
	case viewAddGroup:
		return m.handleAddGroupKey(msg)
	case viewAddCommand:
		return m.handleAddCommandKey(msg)
	case viewEditCommand:
		return m.handleEditCommandKey(msg)
	case viewDeleteConfirm:
		return m.handleDeleteConfirmKey(msg)
	case viewHistory:
		return m.handleHistoryKey(msg)
	case viewHistorySelectGroup:
		return m.handleHistorySelectGroupKey(msg)
	case viewHistoryAddDetails:
		return m.handleHistoryAddDetailsKey(msg)
	case viewActionSelect:
		return m.handleActionSelectKey(msg)
	}

	switch msg.String() {
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit

	case "q":
		if m.mode == viewSearch {
			var cmd tea.Cmd
			m.searchInput, cmd = m.searchInput.Update(msg)
			m.updateFilter()
			return m, cmd
		}
		m.quitting = true
		return m, tea.Quit

	case "esc":
		switch m.mode {
		case viewSearch:
			m.mode = viewGroups
			m.searchInput.Blur()
			m.searchInput.SetValue("")
			m.cursor = 0
		case viewCommands:
			m.mode = viewGroups
			m.cursor = m.selectedGroup
		}
		return m, nil

	case "/":
		if m.mode != viewSearch {
			m.mode = viewSearch
			m.searchInput.Focus()
			m.cursor = 0
			m.updateFilter()
			return m, textinput.Blink
		}

	case "h":
		if m.mode == viewGroups || m.mode == viewCommands {
			m.previousMode = m.mode
			m.mode = viewHistory
			m.cursor = 0
			m.historyError = ""
			if err := m.loadHistory(); err != nil {
				m.historyError = err.Error()
			}
			m.historySearch.SetValue("")
			m.historySearch.Focus()
			return m, textinput.Blink
		}

	case "a":
		if m.mode == viewGroups {
			m.previousMode = viewGroups
			m.mode = viewAddGroup
			m.createFormInputs(
				[]string{"name"},
				[]string{"Group name"},
				[]string{},
			)
			return m, textinput.Blink
		}
		if m.mode == viewCommands {
			m.previousMode = viewCommands
			m.mode = viewAddCommand
			m.createFormInputs(
				[]string{"name", "command", "description"},
				[]string{"Command name", "Command to run", "Description (optional)"},
				[]string{},
			)
			return m, textinput.Blink
		}

	case "e":
		if m.mode == viewCommands && len(m.commands) > 0 && m.cursor < len(m.commands) {
			cmd := m.commands[m.cursor]
			m.editingCmd = &cmd
			m.editingCmdIdx = m.cursor
			m.previousMode = viewCommands
			m.mode = viewEditCommand
			m.createFormInputs(
				[]string{"name", "command", "description"},
				[]string{"Command name", "Command to run", "Description (optional)"},
				[]string{cmd.Name, cmd.Command, cmd.Description},
			)
			return m, textinput.Blink
		}

	case "d", "backspace", "delete":
		if m.mode == viewGroups && len(m.groups) > 0 && m.cursor < len(m.groups) {
			m.deleteTarget = deleteGroup
			m.deleteGroupName = m.groups[m.cursor].Name
			m.previousMode = viewGroups
			m.mode = viewDeleteConfirm
			return m, nil
		}
		if m.mode == viewCommands && len(m.commands) > 0 && m.cursor < len(m.commands) {
			m.deleteTarget = deleteCommand
			m.deleteGroupName = m.groups[m.selectedGroup].Name
			m.deleteCmdName = m.commands[m.cursor].Name
			m.previousMode = viewCommands
			m.mode = viewDeleteConfirm
			return m, nil
		}

	case "up", "k":
		if m.mode == viewSearch {
			var cmd tea.Cmd
			m.searchInput, cmd = m.searchInput.Update(msg)
			m.updateFilter()
			return m, cmd
		}
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil

	case "down", "j":
		if m.mode == viewSearch {
			var cmd tea.Cmd
			m.searchInput, cmd = m.searchInput.Update(msg)
			m.updateFilter()
			return m, cmd
		}
		maxCursor := m.maxCursor()
		if m.cursor < maxCursor {
			m.cursor++
		}
		return m, nil

	case "ctrl+n":
		if m.mode == viewSearch {
			maxCursor := len(m.filtered) - 1
			if m.cursor < maxCursor {
				m.cursor++
			}
		}
		return m, nil

	case "ctrl+p":
		if m.mode == viewSearch {
			if m.cursor > 0 {
				m.cursor--
			}
		}
		return m, nil

	case "enter":
		return m.handleSelect()

	case "tab":
		if m.mode == viewGroups && len(m.groups) > 0 {
			m.mode = viewCommands
			m.selectedGroup = m.cursor
			m.commands = m.groups[m.cursor].Commands
			m.cursor = 0
		}
		return m, nil
	}

	if m.mode == viewSearch {
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		m.updateFilter()
		return m, cmd
	}

	return m, nil
}

func (m Model) handleHistoryKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "esc":
		m.mode = m.previousMode
		m.historySearch.Blur()
		m.historySearch.SetValue("")
		m.cursor = 0
		return m, nil
	case "ctrl+n", "down":
		maxCursor := len(m.filteredHistory) - 1
		if m.cursor < maxCursor {
			m.cursor++
		}
		return m, nil
	case "ctrl+p", "up":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	case "enter":
		if len(m.filteredHistory) > 0 && m.cursor < len(m.filteredHistory) {
			m.selectedHistCmd = m.filteredHistory[m.cursor].Command
			m.historySearch.Blur()

			// Move to group selection
			m.mode = viewHistorySelectGroup
			m.cursor = 0
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.historySearch, cmd = m.historySearch.Update(msg)
	m.updateHistoryFilter()
	return m, cmd
}

func (m Model) handleHistorySelectGroupKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "esc":
		m.mode = viewHistory
		m.historySearch.Focus()
		m.cursor = 0
		return m, textinput.Blink
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	case "down", "j":
		// +1 for "Create new group" option
		maxCursor := len(m.groups)
		if m.cursor < maxCursor {
			m.cursor++
		}
		return m, nil
	case "enter":
		if m.cursor == len(m.groups) {
			// Create new group option selected
			m.mode = viewAddGroup
			m.previousMode = viewHistorySelectGroup
			m.createFormInputs(
				[]string{"name"},
				[]string{"Group name"},
				[]string{},
			)
			return m, textinput.Blink
		}
		// Existing group selected
		m.selectedGroup = m.cursor
		m.mode = viewHistoryAddDetails
		m.createFormInputs(
			[]string{"name", "description"},
			[]string{"Command name", "Description (optional)"},
			[]string{},
		)
		return m, textinput.Blink
	}
	return m, nil
}

func (m Model) handleHistoryAddDetailsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "esc":
		m.mode = viewHistorySelectGroup
		m.formError = ""
		m.cursor = m.selectedGroup
		return m, nil
	case "enter":
		if m.formFocus < len(m.formInputs)-1 {
			m.formInputs[m.formFocus].Blur()
			m.formFocus++
			m.formInputs[m.formFocus].Focus()
			return m, textinput.Blink
		}
		// Submit
		name := m.formInputs[0].Value()
		description := m.formInputs[1].Value()

		if name == "" {
			m.formError = "Command name cannot be empty"
			return m, nil
		}

		groupName := m.groups[m.selectedGroup].Name
		if err := m.config.AddCommand(groupName, name, m.selectedHistCmd, description); err != nil {
			m.formError = err.Error()
			return m, nil
		}
		if err := m.config.Save(); err != nil {
			m.formError = "Failed to save: " + err.Error()
			return m, nil
		}
		m.refreshData()

		// Go back to groups view with the group selected
		m.mode = viewCommands
		m.commands = m.groups[m.selectedGroup].Commands
		m.cursor = len(m.commands) - 1
		m.selectedHistCmd = ""
		return m, nil
	case "tab":
		m.formInputs[m.formFocus].Blur()
		m.formFocus = (m.formFocus + 1) % len(m.formInputs)
		m.formInputs[m.formFocus].Focus()
		return m, textinput.Blink
	case "shift+tab":
		m.formInputs[m.formFocus].Blur()
		m.formFocus = (m.formFocus - 1 + len(m.formInputs)) % len(m.formInputs)
		m.formInputs[m.formFocus].Focus()
		return m, textinput.Blink
	}

	var cmd tea.Cmd
	m.formInputs[m.formFocus], cmd = m.formInputs[m.formFocus].Update(msg)
	return m, cmd
}

func (m Model) handleAddGroupKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "esc":
		m.mode = m.previousMode
		m.formError = ""
		return m, nil
	case "enter":
		name := m.formInputs[0].Value()
		if name == "" {
			m.formError = "Group name cannot be empty"
			return m, nil
		}
		if err := m.config.AddGroup(name); err != nil {
			m.formError = err.Error()
			return m, nil
		}
		if err := m.config.Save(); err != nil {
			m.formError = "Failed to save: " + err.Error()
			return m, nil
		}
		m.refreshData()

		// If we came from history group selection, go to add details
		if m.previousMode == viewHistorySelectGroup {
			m.selectedGroup = len(m.groups) - 1
			m.mode = viewHistoryAddDetails
			m.createFormInputs(
				[]string{"name", "description"},
				[]string{"Command name", "Description (optional)"},
				[]string{},
			)
			return m, textinput.Blink
		}

		m.mode = viewGroups
		m.cursor = len(m.groups) - 1
		return m, nil
	case "tab", "shift+tab":
		return m, nil
	}

	var cmd tea.Cmd
	m.formInputs[m.formFocus], cmd = m.formInputs[m.formFocus].Update(msg)
	return m, cmd
}

func (m Model) handleAddCommandKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "esc":
		m.mode = m.previousMode
		m.formError = ""
		return m, nil
	case "enter":
		if m.formFocus < len(m.formInputs)-1 {
			m.formInputs[m.formFocus].Blur()
			m.formFocus++
			m.formInputs[m.formFocus].Focus()
			return m, textinput.Blink
		}
		// Submit
		name := m.formInputs[0].Value()
		command := m.formInputs[1].Value()
		description := m.formInputs[2].Value()

		if name == "" {
			m.formError = "Command name cannot be empty"
			return m, nil
		}
		if command == "" {
			m.formError = "Command cannot be empty"
			return m, nil
		}

		groupName := m.groups[m.selectedGroup].Name
		if err := m.config.AddCommand(groupName, name, command, description); err != nil {
			m.formError = err.Error()
			return m, nil
		}
		if err := m.config.Save(); err != nil {
			m.formError = "Failed to save: " + err.Error()
			return m, nil
		}
		m.refreshData()
		m.mode = viewCommands
		m.cursor = len(m.commands) - 1
		return m, nil
	case "tab":
		m.formInputs[m.formFocus].Blur()
		m.formFocus = (m.formFocus + 1) % len(m.formInputs)
		m.formInputs[m.formFocus].Focus()
		return m, textinput.Blink
	case "shift+tab":
		m.formInputs[m.formFocus].Blur()
		m.formFocus = (m.formFocus - 1 + len(m.formInputs)) % len(m.formInputs)
		m.formInputs[m.formFocus].Focus()
		return m, textinput.Blink
	}

	var cmd tea.Cmd
	m.formInputs[m.formFocus], cmd = m.formInputs[m.formFocus].Update(msg)
	return m, cmd
}

func (m Model) handleEditCommandKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "esc":
		m.mode = m.previousMode
		m.formError = ""
		m.editingCmd = nil
		return m, nil
	case "enter":
		if m.formFocus < len(m.formInputs)-1 {
			m.formInputs[m.formFocus].Blur()
			m.formFocus++
			m.formInputs[m.formFocus].Focus()
			return m, textinput.Blink
		}
		// Submit
		newName := m.formInputs[0].Value()
		newCommand := m.formInputs[1].Value()
		newDescription := m.formInputs[2].Value()

		if newName == "" {
			m.formError = "Command name cannot be empty"
			return m, nil
		}
		if newCommand == "" {
			m.formError = "Command cannot be empty"
			return m, nil
		}

		groupName := m.groups[m.selectedGroup].Name

		// Remove old and add new (to handle name changes)
		if err := m.config.RemoveCommand(groupName, m.editingCmd.Name); err != nil {
			m.formError = err.Error()
			return m, nil
		}
		if err := m.config.AddCommand(groupName, newName, newCommand, newDescription); err != nil {
			// Try to restore old command on failure
			_ = m.config.AddCommand(groupName, m.editingCmd.Name, m.editingCmd.Command, m.editingCmd.Description)
			m.formError = err.Error()
			return m, nil
		}
		if err := m.config.Save(); err != nil {
			m.formError = "Failed to save: " + err.Error()
			return m, nil
		}
		m.refreshData()
		m.mode = viewCommands
		m.editingCmd = nil
		return m, nil
	case "tab":
		m.formInputs[m.formFocus].Blur()
		m.formFocus = (m.formFocus + 1) % len(m.formInputs)
		m.formInputs[m.formFocus].Focus()
		return m, textinput.Blink
	case "shift+tab":
		m.formInputs[m.formFocus].Blur()
		m.formFocus = (m.formFocus - 1 + len(m.formInputs)) % len(m.formInputs)
		m.formInputs[m.formFocus].Focus()
		return m, textinput.Blink
	}

	var cmd tea.Cmd
	m.formInputs[m.formFocus], cmd = m.formInputs[m.formFocus].Update(msg)
	return m, cmd
}

func (m Model) handleDeleteConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "esc", "n", "N":
		m.mode = m.previousMode
		return m, nil
	case "y", "Y", "enter":
		var err error
		if m.deleteTarget == deleteGroup {
			err = m.config.RemoveGroup(m.deleteGroupName)
		} else {
			err = m.config.RemoveCommand(m.deleteGroupName, m.deleteCmdName)
		}
		if err != nil {
			m.formError = err.Error()
			return m, nil
		}
		if err := m.config.Save(); err != nil {
			m.formError = "Failed to save: " + err.Error()
			return m, nil
		}
		m.refreshData()
		if m.deleteTarget == deleteGroup {
			m.mode = viewGroups
			if m.cursor >= len(m.groups) {
				m.cursor = max(0, len(m.groups)-1)
			}
		} else {
			m.mode = viewCommands
			if m.cursor >= len(m.commands) {
				m.cursor = max(0, len(m.commands)-1)
			}
		}
		return m, nil
	}
	return m, nil
}

func (m *Model) handleSelect() (tea.Model, tea.Cmd) {
	switch m.mode {
	case viewGroups:
		if len(m.groups) > 0 && m.cursor < len(m.groups) {
			m.mode = viewCommands
			m.selectedGroup = m.cursor
			m.commands = m.groups[m.cursor].Commands
			m.cursor = 0
		}
	case viewCommands:
		if len(m.commands) > 0 && m.cursor < len(m.commands) {
			cmd := m.commands[m.cursor]
			m.actionCmd = &config.FlatCommand{
				ID:            cmd.ID,
				GroupName:     m.groups[m.selectedGroup].Name,
				Name:          cmd.Name,
				Command:       cmd.Command,
				Description:   cmd.Description,
				DefaultAction: cmd.DefaultAction,
			}
			// If default action is set, execute it directly
			if cmd.DefaultAction == config.ActionCopy {
				return m.executeAction(config.ActionCopy)
			}
			if cmd.DefaultAction == config.ActionRun {
				return m.executeAction(config.ActionRun)
			}
			// Show action selection
			m.previousMode = viewCommands
			m.mode = viewActionSelect
			m.actionCursor = 0
			m.actionResult = ""
			m.actionError = ""
		}
	case viewSearch:
		if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
			selected := m.filtered[m.cursor]
			m.actionCmd = &selected
			// If default action is set, execute it directly
			if selected.DefaultAction == config.ActionCopy {
				return m.executeAction(config.ActionCopy)
			}
			if selected.DefaultAction == config.ActionRun {
				return m.executeAction(config.ActionRun)
			}
			// Show action selection
			m.previousMode = viewSearch
			m.mode = viewActionSelect
			m.actionCursor = 0
			m.actionResult = ""
			m.actionError = ""
		}
	}
	return m, nil
}

func (m Model) handleActionSelectKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "esc":
		m.mode = m.previousMode
		m.actionCmd = nil
		return m, nil
	case "up", "k":
		if m.actionCursor > 0 {
			m.actionCursor--
		}
		return m, nil
	case "down", "j":
		if m.actionCursor < 2 { // run, copy, cancel
			m.actionCursor++
		}
		return m, nil
	case "r":
		return m.executeAction(config.ActionRun)
	case "c":
		return m.executeAction(config.ActionCopy)
	case "enter":
		switch m.actionCursor {
		case 0: // Run
			return m.executeAction(config.ActionRun)
		case 1: // Copy
			return m.executeAction(config.ActionCopy)
		case 2: // Cancel
			m.mode = m.previousMode
			m.actionCmd = nil
			return m, nil
		}
	}
	return m, nil
}

func (m *Model) executeAction(action config.ActionType) (tea.Model, tea.Cmd) {
	if m.actionCmd == nil {
		return m, nil
	}

	switch action {
	case config.ActionCopy:
		if err := runner.CopyToClipboard(m.actionCmd.Command); err != nil {
			m.actionError = err.Error()
			return m, nil
		}
		m.selected = m.actionCmd
		m.actionResult = "Copied to clipboard"
		m.quitting = true
		return m, tea.Quit
	case config.ActionRun:
		m.selected = m.actionCmd
		m.actionResult = "run"
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
}

// ActionResult returns the action result message
func (m Model) ActionResult() string {
	return m.actionResult
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

func (m Model) Selected() *config.FlatCommand {
	return m.selected
}

func (m Model) View() string {
	if m.quitting && m.selected != nil {
		return ""
	}

	var content string
	switch m.mode {
	case viewGroups:
		content = m.viewGroups()
	case viewCommands:
		content = m.viewCommands()
	case viewSearch:
		content = m.viewSearch()
	case viewAddGroup:
		content = m.viewAddGroup()
	case viewAddCommand:
		content = m.viewAddCommand()
	case viewEditCommand:
		content = m.viewEditCommand()
	case viewDeleteConfirm:
		content = m.viewDeleteConfirm()
	case viewHistory:
		content = m.viewHistory()
	case viewHistorySelectGroup:
		content = m.viewHistorySelectGroup()
	case viewHistoryAddDetails:
		content = m.viewHistoryAddDetails()
	case viewActionSelect:
		content = m.viewActionSelect()
	}

	return content
}

func (m Model) viewGroups() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		MarginBottom(1)

	itemStyle := lipgloss.NewStyle().PaddingLeft(2)
	selectedStyle := lipgloss.NewStyle().
		PaddingLeft(2).
		Foreground(lipgloss.Color("170")).
		Bold(true)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginTop(1)

	s := titleStyle.Render("Command Bookmarks") + "\n\n"

	if len(m.groups) == 0 {
		s += itemStyle.Render("No groups yet. Press 'a' to add one.") + "\n"
	} else {
		for i, g := range m.groups {
			cursor := "  "
			style := itemStyle
			if m.cursor == i {
				cursor = "> "
				style = selectedStyle
			}
			cmdCount := len(g.Commands)
			plural := "s"
			if cmdCount == 1 {
				plural = ""
			}
			countStr := fmt.Sprintf("%d", cmdCount)
			s += style.Render(cursor+g.Name) + lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(" ("+countStr+" cmd"+plural+")") + "\n"
		}
	}

	s += "\n" + helpStyle.Render("j/k navigate | enter select | a add | d delete | h history | / search | q quit")

	return s
}

func (m Model) viewCommands() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		MarginBottom(1)

	groupStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("141")).
		MarginBottom(1)

	itemStyle := lipgloss.NewStyle().PaddingLeft(2)
	selectedStyle := lipgloss.NewStyle().
		PaddingLeft(2).
		Foreground(lipgloss.Color("170")).
		Bold(true)

	cmdStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Italic(true)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginTop(1)

	s := titleStyle.Render("Command Bookmarks") + "\n"
	s += groupStyle.Render("Group: "+m.groups[m.selectedGroup].Name) + "\n\n"

	if len(m.commands) == 0 {
		s += itemStyle.Render("No commands in this group. Press 'a' to add one.") + "\n"
	} else {
		idStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
		for i, cmd := range m.commands {
			cursor := "  "
			style := itemStyle
			if m.cursor == i {
				cursor = "> "
				style = selectedStyle
			}
			idTag := idStyle.Render(fmt.Sprintf("[%d] ", cmd.ID))
			line := style.Render(cursor) + idTag + style.Render(cmd.Name)
			line += "\n" + itemStyle.Render("    ") + cmdStyle.Render(cmd.Command)
			if cmd.Description != "" {
				line += "\n" + itemStyle.Render("    ") + descStyle.Render(cmd.Description)
			}
			s += line + "\n\n"
		}
	}

	s += helpStyle.Render("j/k navigate | enter select | a add | e edit | d delete | h history | esc back | q quit")

	return s
}

func (m Model) viewSearch() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		MarginBottom(1)

	itemStyle := lipgloss.NewStyle().PaddingLeft(2)
	selectedStyle := lipgloss.NewStyle().
		PaddingLeft(2).
		Foreground(lipgloss.Color("170")).
		Bold(true)

	groupTagStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("141")).
		Background(lipgloss.Color("236")).
		Padding(0, 1)

	cmdStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Italic(true)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginTop(1)

	s := titleStyle.Render("Search Commands") + "\n\n"
	s += m.searchInput.View() + "\n\n"

	if len(m.filtered) == 0 {
		s += itemStyle.Render("No matching commands.") + "\n"
	} else {
		idStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
		displayCount := min(10, len(m.filtered))
		for i := 0; i < displayCount; i++ {
			cmd := m.filtered[i]
			cursor := "  "
			style := itemStyle
			if m.cursor == i {
				cursor = "> "
				style = selectedStyle
			}
			idTag := idStyle.Render(fmt.Sprintf("[%d] ", cmd.ID))
			line := style.Render(cursor) + idTag + style.Render(cmd.Name) + " " + groupTagStyle.Render(cmd.GroupName)
			line += "\n" + itemStyle.Render("    ") + cmdStyle.Render(cmd.Command)
			if cmd.Description != "" {
				line += "\n" + itemStyle.Render("    ") + descStyle.Render(cmd.Description)
			}
			s += line + "\n\n"
		}
		if len(m.filtered) > displayCount {
			s += itemStyle.Render("... and more results") + "\n"
		}
	}

	s += helpStyle.Render("ctrl+n/p navigate | enter select | esc back | q quit")

	return s
}

func (m Model) viewHistory() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		MarginBottom(1)

	itemStyle := lipgloss.NewStyle().PaddingLeft(2)
	selectedStyle := lipgloss.NewStyle().
		PaddingLeft(2).
		Foreground(lipgloss.Color("170")).
		Bold(true)

	cmdStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196"))

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginTop(1)

	s := titleStyle.Render("Shell History") + "\n\n"
	s += m.historySearch.View() + "\n\n"

	if m.historyError != "" {
		s += errorStyle.Render("Error: "+m.historyError) + "\n"
	} else if len(m.filteredHistory) == 0 {
		s += itemStyle.Render("No matching commands in history.") + "\n"
	} else {
		displayCount := min(15, len(m.filteredHistory))
		for i := 0; i < displayCount; i++ {
			entry := m.filteredHistory[i]
			cursor := "  "
			style := itemStyle
			if m.cursor == i {
				cursor = "> "
				style = selectedStyle
			}
			// Truncate long commands for display
			cmd := entry.Command
			if len(cmd) > m.width-10 && m.width > 20 {
				cmd = cmd[:m.width-13] + "..."
			}
			s += style.Render(cursor) + cmdStyle.Render(cmd) + "\n"
		}
		if len(m.filteredHistory) > displayCount {
			s += itemStyle.Render(fmt.Sprintf("... and %d more", len(m.filteredHistory)-displayCount)) + "\n"
		}
	}

	s += "\n" + helpStyle.Render("ctrl+n/p navigate | enter select | esc back")

	return s
}

func (m Model) viewHistorySelectGroup() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		MarginBottom(1)

	cmdPreviewStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		MarginBottom(1)

	itemStyle := lipgloss.NewStyle().PaddingLeft(2)
	selectedStyle := lipgloss.NewStyle().
		PaddingLeft(2).
		Foreground(lipgloss.Color("170")).
		Bold(true)

	newGroupStyle := lipgloss.NewStyle().
		PaddingLeft(2).
		Foreground(lipgloss.Color("141")).
		Italic(true)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginTop(1)

	s := titleStyle.Render("Select Group") + "\n\n"

	// Show selected command preview
	cmdPreview := m.selectedHistCmd
	if len(cmdPreview) > m.width-4 && m.width > 20 {
		cmdPreview = cmdPreview[:m.width-7] + "..."
	}
	s += cmdPreviewStyle.Render("Command: "+cmdPreview) + "\n\n"

	// List groups
	for i, g := range m.groups {
		cursor := "  "
		style := itemStyle
		if m.cursor == i {
			cursor = "> "
			style = selectedStyle
		}
		s += style.Render(cursor+g.Name) + "\n"
	}

	// "Create new group" option
	cursor := "  "
	style := newGroupStyle
	if m.cursor == len(m.groups) {
		cursor = "> "
		style = lipgloss.NewStyle().
			PaddingLeft(2).
			Foreground(lipgloss.Color("170")).
			Bold(true).
			Italic(true)
	}
	s += style.Render(cursor+"+ Create new group") + "\n"

	s += "\n" + helpStyle.Render("j/k navigate | enter select | esc back")

	return s
}

func (m Model) viewHistoryAddDetails() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		MarginBottom(1)

	cmdPreviewStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		MarginBottom(1)

	groupStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("141")).
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("141"))

	focusedLabelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("170")).
		Bold(true)

	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		MarginTop(1)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginTop(1)

	s := titleStyle.Render("Add from History") + "\n"
	s += groupStyle.Render("Group: "+m.groups[m.selectedGroup].Name) + "\n"

	// Show selected command preview
	cmdPreview := m.selectedHistCmd
	if len(cmdPreview) > m.width-4 && m.width > 20 {
		cmdPreview = cmdPreview[:m.width-7] + "..."
	}
	s += cmdPreviewStyle.Render("Command: "+cmdPreview) + "\n\n"

	labels := []string{"Name:", "Description:"}
	for i, label := range labels {
		style := labelStyle
		if i == m.formFocus {
			style = focusedLabelStyle
		}
		s += style.Render(label) + "\n"
		s += m.formInputs[i].View() + "\n\n"
	}

	if m.formError != "" {
		s += errorStyle.Render("Error: "+m.formError) + "\n"
	}

	s += helpStyle.Render("tab next field | enter submit | esc back")

	return s
}

func (m Model) viewAddGroup() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("141")).
		MarginBottom(1)

	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		MarginTop(1)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginTop(1)

	s := titleStyle.Render("Add Group") + "\n\n"
	s += labelStyle.Render("Name:") + "\n"
	s += m.formInputs[0].View() + "\n"

	if m.formError != "" {
		s += errorStyle.Render("Error: "+m.formError) + "\n"
	}

	s += "\n" + helpStyle.Render("enter submit | esc cancel")

	return s
}

func (m Model) viewAddCommand() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("141"))

	focusedLabelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("170")).
		Bold(true)

	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		MarginTop(1)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginTop(1)

	labels := []string{"Name:", "Command:", "Description:"}

	s := titleStyle.Render("Add Command") + "\n"
	s += lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("Group: "+m.groups[m.selectedGroup].Name) + "\n\n"

	for i, label := range labels {
		style := labelStyle
		if i == m.formFocus {
			style = focusedLabelStyle
		}
		s += style.Render(label) + "\n"
		s += m.formInputs[i].View() + "\n\n"
	}

	if m.formError != "" {
		s += errorStyle.Render("Error: "+m.formError) + "\n"
	}

	s += helpStyle.Render("tab next field | enter submit | esc cancel")

	return s
}

func (m Model) viewEditCommand() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("141"))

	focusedLabelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("170")).
		Bold(true)

	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		MarginTop(1)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginTop(1)

	labels := []string{"Name:", "Command:", "Description:"}

	s := titleStyle.Render("Edit Command") + "\n"
	s += lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("Group: "+m.groups[m.selectedGroup].Name) + "\n\n"

	for i, label := range labels {
		style := labelStyle
		if i == m.formFocus {
			style = focusedLabelStyle
		}
		s += style.Render(label) + "\n"
		s += m.formInputs[i].View() + "\n\n"
	}

	if m.formError != "" {
		s += errorStyle.Render("Error: "+m.formError) + "\n"
	}

	s += helpStyle.Render("tab next field | enter submit | esc cancel")

	return s
}

func (m Model) viewDeleteConfirm() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("196")).
		MarginBottom(1)

	messageStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		MarginBottom(1)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginTop(1)

	s := titleStyle.Render("Confirm Delete") + "\n\n"

	if m.deleteTarget == deleteGroup {
		s += messageStyle.Render(fmt.Sprintf("Delete group '%s' and all its commands?", m.deleteGroupName)) + "\n"
	} else {
		s += messageStyle.Render(fmt.Sprintf("Delete command '%s' from group '%s'?", m.deleteCmdName, m.deleteGroupName)) + "\n"
	}

	s += "\n" + helpStyle.Render("y confirm | n/esc cancel")

	return s
}

func (m Model) viewActionSelect() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		MarginBottom(1)

	cmdPreviewStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		MarginBottom(1)

	itemStyle := lipgloss.NewStyle().PaddingLeft(2)
	selectedStyle := lipgloss.NewStyle().
		PaddingLeft(2).
		Foreground(lipgloss.Color("170")).
		Bold(true)

	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		MarginTop(1)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginTop(1)

	s := titleStyle.Render("Select Action") + "\n\n"

	if m.actionCmd != nil {
		// Show command preview
		cmdPreview := m.actionCmd.Command
		if len(cmdPreview) > m.width-4 && m.width > 20 {
			cmdPreview = cmdPreview[:m.width-7] + "..."
		}
		s += cmdPreviewStyle.Render("Command: "+cmdPreview) + "\n\n"
	}

	actions := []struct {
		key  string
		name string
	}{
		{"r", "Run command"},
		{"c", "Copy to clipboard"},
		{"", "Cancel"},
	}

	for i, action := range actions {
		cursor := "  "
		style := itemStyle
		if m.actionCursor == i {
			cursor = "> "
			style = selectedStyle
		}
		if action.key != "" {
			s += style.Render(fmt.Sprintf("%s[%s] %s", cursor, action.key, action.name)) + "\n"
		} else {
			s += style.Render(cursor+action.name) + "\n"
		}
	}

	if m.actionError != "" {
		s += "\n" + errorStyle.Render("Error: "+m.actionError)
	}

	s += "\n" + helpStyle.Render("j/k navigate | enter select | r run | c copy | esc cancel")

	return s
}
