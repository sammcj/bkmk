package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

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
	case viewEditGroup:
		content = m.viewEditGroup()
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

	s += "\n" + helpStyle.Render("j/k navigate | enter select | a add | e edit | d delete | h history | o open config | / search | q quit")

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
	if !m.selectedGroupValid() {
		s += groupStyle.Render("No group selected") + "\n\n"
		return s
	}
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

	s += helpStyle.Render("j/k navigate | enter select | a add | e edit | d delete | h history | o open config | esc back | q quit")

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
		for i := range displayCount {
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
		for i := range displayCount {
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
	groupName := "Unknown"
	if m.selectedGroupValid() {
		groupName = m.groups[m.selectedGroup].Name
	}
	s += groupStyle.Render("Group: "+groupName) + "\n"

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

func (m Model) viewEditGroup() string {
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

	s := titleStyle.Render("Rename Group") + "\n\n"
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
	groupName := "Unknown"
	if m.selectedGroupValid() {
		groupName = m.groups[m.selectedGroup].Name
	}
	s += lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("Group: "+groupName) + "\n\n"

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
	groupName := "Unknown"
	if m.selectedGroupValid() {
		groupName = m.groups[m.selectedGroup].Name
	}
	s += lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("Group: "+groupName) + "\n\n"

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
