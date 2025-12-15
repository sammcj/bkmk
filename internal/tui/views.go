package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderHeader() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205"))

	configStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	left := titleStyle.Render("bkmk: Command Bookmarks")
	right := configStyle.Render("Config: " + m.configPath)

	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	gap := m.width - leftWidth - rightWidth

	if gap < 2 {
		return left + "\n"
	}

	return left + strings.Repeat(" ", gap) + right + "\n"
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
	case viewAllCommands:
		content = m.viewAllCommands()
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
	itemStyle := lipgloss.NewStyle().PaddingLeft(2)
	selectedStyle := lipgloss.NewStyle().
		PaddingLeft(2).
		Foreground(lipgloss.Color("170")).
		Bold(true)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginTop(1)

	s := m.renderHeader() + "\n"

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

	s += "\n" + helpStyle.Render("j/k navigate | enter select | a add | e edit | d delete | s show all | h history | o open config | / search | q quit")

	return s
}

func (m Model) viewCommands() string {
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

	s := m.renderHeader()
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

	s += helpStyle.Render("j/k navigate | enter select | a add | e edit | d delete | s show all | h history | o open config | esc back | q quit")

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

	s := titleStyle.Render("bkmk: Search Commands") + "\n\n"
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

func (m Model) viewAllCommands() string {
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

	s := titleStyle.Render("bkmk: All Bookmarks") + "\n\n"

	if len(m.flatCommands) == 0 {
		s += itemStyle.Render("No bookmarks yet.") + "\n"
	} else {
		idStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

		// Calculate available lines for items
		reservedLines := 6
		availableLines := m.height - reservedLines
		if availableLines < 5 {
			availableLines = 5
		}

		displayCount := min(availableLines, len(m.flatCommands))
		totalItems := len(m.flatCommands)

		// Calculate scroll offset to keep cursor visible
		offset := 0
		if m.cursor >= displayCount {
			offset = m.cursor - displayCount + 1
		}
		if offset+displayCount > totalItems {
			offset = max(0, totalItems-displayCount)
		}

		endIdx := min(offset+displayCount, totalItems)
		for i := offset; i < endIdx; i++ {
			cmd := m.flatCommands[i]
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

		// Show scroll indicators
		if offset > 0 || endIdx < totalItems {
			scrollInfo := ""
			if offset > 0 {
				scrollInfo = fmt.Sprintf("↑ %d more above", offset)
			}
			if endIdx < totalItems {
				if scrollInfo != "" {
					scrollInfo += " | "
				}
				scrollInfo += fmt.Sprintf("↓ %d more below", totalItems-endIdx)
			}
			s += itemStyle.Render(scrollInfo) + "\n"
		}
	}

	s += helpStyle.Render("j/k navigate | enter select | esc back | q quit")

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

	selectedCmdStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("170")).
		Bold(true)

	cmdStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196"))

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginTop(1)

	s := titleStyle.Render("bkmk: Shell History") + "\n\n"
	s += m.historySearch.View() + "\n\n"

	if m.historyError != "" {
		s += errorStyle.Render("Error: "+m.historyError) + "\n"
	} else if len(m.filteredHistory) == 0 {
		s += itemStyle.Render("No matching commands in history.") + "\n"
	} else {
		// Calculate available lines for history items
		// Header: title (1) + blank (1) + search (1) + blank (1) = 4
		// Footer: blank (1) + help (1) + optional "more" line (1) = 3
		reservedLines := 8
		availableLines := m.height - reservedLines
		if availableLines < 5 {
			availableLines = 5 // minimum visible items
		}

		displayCount := min(availableLines, len(m.filteredHistory))
		totalItems := len(m.filteredHistory)

		// Calculate scroll offset to keep cursor visible
		offset := 0
		if m.cursor >= displayCount {
			offset = m.cursor - displayCount + 1
		}
		if offset+displayCount > totalItems {
			offset = max(0, totalItems-displayCount)
		}

		timeStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("246"))
		selectedTimeStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("170"))

		endIdx := min(offset+displayCount, totalItems)
		for i := offset; i < endIdx; i++ {
			entry := m.filteredHistory[i]
			cursor := "  "
			style := itemStyle
			cmdSt := cmdStyle
			timeSt := timeStyle
			if m.cursor == i {
				cursor = "> "
				style = selectedStyle
				cmdSt = selectedCmdStyle
				timeSt = selectedTimeStyle
			}

			// Format timestamp if available
			timeStr := ""
			if !entry.Timestamp.IsZero() {
				timeStr = entry.Timestamp.Local().Format("02 Jan 15:04") + " "
			}
			timeWidth := len(timeStr)

			// Truncate long commands for display (account for timestamp width)
			cmd := entry.Command
			maxCmdWidth := m.width - 10 - timeWidth
			if len(cmd) > maxCmdWidth && maxCmdWidth > 10 {
				cmd = cmd[:maxCmdWidth-3] + "..."
			}

			if timeStr != "" {
				s += style.Render(cursor) + timeSt.Render(timeStr) + cmdSt.Render(cmd) + "\n"
			} else {
				s += style.Render(cursor) + cmdSt.Render(cmd) + "\n"
			}
		}

		// Show scroll indicators
		if offset > 0 || endIdx < totalItems {
			scrollInfo := ""
			if offset > 0 {
				scrollInfo = fmt.Sprintf("↑ %d more above", offset)
			}
			if endIdx < totalItems {
				if scrollInfo != "" {
					scrollInfo += " | "
				}
				scrollInfo += fmt.Sprintf("↓ %d more below", totalItems-endIdx)
			}
			s += itemStyle.Render(scrollInfo) + "\n"
		}
	}

	s += "\n" + helpStyle.Render("↑/↓ navigate | pgup/pgdn page | enter select | esc back")

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

	s := titleStyle.Render("bkmk: Select Group") + "\n\n"

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

	s := titleStyle.Render("bkmk: Add from History") + "\n"
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

	s := titleStyle.Render("bkmk: Add Group") + "\n\n"
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

	s := titleStyle.Render("bkmk: Rename Group") + "\n\n"
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

	s := titleStyle.Render("bkmk: Add Command") + "\n"
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

	s := titleStyle.Render("bkmk: Edit Command") + "\n"
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

	s := titleStyle.Render("bkmk: Confirm Delete") + "\n\n"

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

	s := titleStyle.Render("bkmk: Select Action") + "\n\n"

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
