package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sammcj/bkmk/internal/config"
	"github.com/sammcj/bkmk/internal/runner"
)

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle form modes first
	switch m.mode {
	case viewAddGroup:
		return m.handleAddGroupKey(msg)
	case viewEditGroup:
		return m.handleEditGroupKey(msg)
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

	case "o":
		if m.mode == viewGroups || m.mode == viewCommands {
			configPath, err := config.DefaultPath()
			if err != nil {
				return m, nil
			}
			editorCmd, err := runner.OpenInEditor(configPath)
			if err != nil {
				return m, nil
			}
			return m, tea.ExecProcess(editorCmd, func(err error) tea.Msg {
				return nil
			})
		}

	case "a":
		if m.mode == viewGroups {
			m.previousMode = viewGroups
			m.mode = viewAddGroup
			m.createFormInputs(
				[]string{"Group name"},
				[]string{},
			)
			return m, textinput.Blink
		}
		if m.mode == viewCommands {
			m.previousMode = viewCommands
			m.mode = viewAddCommand
			m.createFormInputs(
				[]string{"Command name", "Command to run", "Description (optional)"},
				[]string{},
			)
			return m, textinput.Blink
		}

	case "e":
		if m.mode == viewGroups && len(m.groups) > 0 && m.cursor < len(m.groups) {
			m.editingGroup = m.groups[m.cursor].Name
			m.previousMode = viewGroups
			m.mode = viewEditGroup
			m.createFormInputs(
				[]string{"Group name"},
				[]string{m.editingGroup},
			)
			return m, textinput.Blink
		}
		if m.mode == viewCommands && len(m.commands) > 0 && m.cursor < len(m.commands) {
			cmd := m.commands[m.cursor]
			m.editingCmd = &cmd
			m.editingCmdIdx = m.cursor
			m.previousMode = viewCommands
			m.mode = viewEditCommand
			m.createFormInputs(
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
		if m.mode == viewCommands && len(m.commands) > 0 && m.cursor < len(m.commands) && m.selectedGroupValid() {
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
	// Calculate page size for page up/down
	pageSize := m.historyPageSize()

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
	case "pgdown", "ctrl+down":
		maxCursor := len(m.filteredHistory) - 1
		m.cursor = min(m.cursor+pageSize, maxCursor)
		return m, nil
	case "pgup", "ctrl+up":
		m.cursor = max(m.cursor-pageSize, 0)
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
				[]string{"Group name"},
				[]string{},
			)
			return m, textinput.Blink
		}
		// Existing group selected
		m.selectedGroup = m.cursor
		m.mode = viewHistoryAddDetails
		m.createFormInputs(
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

		if !m.selectedGroupValid() {
			m.formError = "No group selected"
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

func (m Model) handleEditGroupKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "esc":
		m.mode = m.previousMode
		m.formError = ""
		m.editingGroup = ""
		return m, nil
	case "enter":
		newName := m.formInputs[0].Value()
		if newName == "" {
			m.formError = "Group name cannot be empty"
			return m, nil
		}
		if err := m.config.RenameGroup(m.editingGroup, newName); err != nil {
			m.formError = err.Error()
			return m, nil
		}
		if err := m.config.Save(); err != nil {
			m.formError = "Failed to save: " + err.Error()
			return m, nil
		}
		m.refreshData()
		m.mode = viewGroups
		m.editingGroup = ""
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

		if !m.selectedGroupValid() {
			m.formError = "No group selected"
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

		if !m.selectedGroupValid() {
			m.formError = "No group selected"
			return m, nil
		}
		groupName := m.groups[m.selectedGroup].Name

		// Update existing command (preserves ID)
		if err := m.config.UpdateCommand(groupName, m.editingCmd.Name, newName, newCommand, newDescription); err != nil {
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
