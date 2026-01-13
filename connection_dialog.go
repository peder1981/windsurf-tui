package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ConnectionDialog struct {
	choices       []string
	connections   []*ConnectionInfo
	cursor        int
	selectedIndex int
	isConfirmed   bool
	connectionMgr *ConnectionManager
}

func NewConnectionDialog(connectionMgr *ConnectionManager) *ConnectionDialog {
	cd := &ConnectionDialog{
		cursor:        0,
		selectedIndex: -1,
		isConfirmed:   false,
		connectionMgr: connectionMgr,
	}
	cd.refreshChoices()
	return cd
}

func (cd *ConnectionDialog) refreshChoices() {
	cd.connections = cd.connectionMgr.GetSavedConnections()

	cd.choices = make([]string, 0, len(cd.connections)+1)
	cd.choices = append(cd.choices, "🔧 Nova conexão (Ctrl+N)")

	for _, conn := range cd.connections {
		cd.choices = append(cd.choices, fmt.Sprintf("📁 %s (%s:%d)", conn.Name, conn.Host, conn.Port))
	}

	if len(cd.choices) == 0 {
		cd.cursor = 0
		return
	}

	if cd.cursor >= len(cd.choices) {
		cd.cursor = len(cd.choices) - 1
	}
	if cd.cursor < 0 {
		cd.cursor = 0
	}
}

func (cd *ConnectionDialog) ReloadChoices() {
	cd.refreshChoices()
	cd.selectedIndex = -1
	cd.isConfirmed = false
}

func (cd *ConnectionDialog) Init() tea.Cmd {
	return nil
}

func (cd *ConnectionDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyUp:
			if cd.cursor > 0 {
				cd.cursor--
			}
		case tea.KeyDown:
			if cd.cursor < len(cd.choices)-1 {
				cd.cursor++
			}
		case tea.KeyEnter, tea.KeySpace:
			if cd.cursor == 0 {
				cd.isConfirmed = true
				cd.selectedIndex = -1
			} else {
				cd.isConfirmed = true
				cd.selectedIndex = cd.cursor - 1
			}
		case tea.KeyEscape:
			cd.isConfirmed = true
			cd.selectedIndex = -1
		}
	}

	return cd, nil
}

func (cd *ConnectionDialog) View() string {
	title := "🌐 XTreeGold - PostgreSQL Navigator"
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFD700")).
		Bold(true).
		Align(lipgloss.Center).
		Width(60)

	content := titleStyle.Render(title) + "\n\n"

	content += lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Render("Select a connection or create a new one:\n\n")

	for i, choice := range cd.choices {
		cursor := " "
		if cd.cursor == i {
			cursor = ">"
		}

		style := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF"))

		if cd.cursor == i {
			style = style.Background(lipgloss.Color("#4169E1"))
		}

		content += fmt.Sprintf("%s %s\n", cursor, style.Render(choice))
	}

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#808080")).
		Italic(true)

	content += "\n" + helpStyle.Render("↑/↓ Navigate | Enter Select | Escape Exit")

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#4169E1")).
		Padding(1, 2)

	return border.Render(content)
}

func (cd *ConnectionDialog) GetSelectedConnection() *ConnectionInfo {
	if cd.selectedIndex < 0 {
		return nil
	}

	savedConns := cd.connectionMgr.GetSavedConnections()
	if cd.selectedIndex < len(savedConns) {
		return savedConns[cd.selectedIndex]
	}

	return nil
}

func (cd *ConnectionDialog) ShouldAddNewConnection() bool {
	return cd.cursor == 0 && cd.isConfirmed
}

func (cd *ConnectionDialog) IsConfirmed() bool {
	return cd.isConfirmed
}
