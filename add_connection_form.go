package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type AddConnectionForm struct {
	connectionInfo  *ConnectionInfo
	cursor          int
	field           int
	isConfirmed     bool
	validationError string
	mode            string
	fieldLabelWidth int
}

func NewAddConnectionForm() *AddConnectionForm {
	return &AddConnectionForm{
		connectionInfo: &ConnectionInfo{
			Name:     "",
			Host:     "localhost",
			Port:     5432,
			User:     "postgres",
			Password: "",
			Database: "",
			SSLMode:  "disable",
		},
		cursor:          0,
		field:           0,
		isConfirmed:     false,
		mode:            "add",
		fieldLabelWidth: 18,
	}
}

func (acf *AddConnectionForm) Init() tea.Cmd {
	return nil
}

func (acf *AddConnectionForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyUp:
			if acf.field > 0 {
				acf.field--
				acf.syncCursorToField()
			}
		case tea.KeyDown:
			if acf.field < 6 {
				acf.field++
				acf.syncCursorToField()
			}
		case tea.KeyEnter:
			if acf.validate() {
				acf.isConfirmed = true
			} else {
				acf.validationError = "Please fill in required fields"
			}
		case tea.KeyEscape:
			acf.isConfirmed = true
		case tea.KeyBackspace:
			acf.deleteChar()
		case tea.KeyLeft:
			if acf.cursor > 0 {
				acf.cursor--
			}
		case tea.KeyRight:
			acf.moveCursorRight()
		default:
			if len(msg.Runes) > 0 {
				acf.addChar(string(msg.Runes))
			}
		}
	}

	return acf, nil
}

func (acf *AddConnectionForm) syncCursorToField() {
	switch acf.field {
	case 0:
		if acf.cursor > len(acf.connectionInfo.Name) {
			acf.cursor = len(acf.connectionInfo.Name)
		}
	case 1:
		if acf.cursor > len(acf.connectionInfo.Host) {
			acf.cursor = len(acf.connectionInfo.Host)
		}
	case 2:
		if acf.cursor > len(acf.connectionInfo.User) {
			acf.cursor = len(acf.connectionInfo.User)
		}
	case 3:
		if acf.cursor > len(acf.connectionInfo.Password) {
			acf.cursor = len(acf.connectionInfo.Password)
		}
	case 4:
		if acf.cursor > len(acf.connectionInfo.Database) {
			acf.cursor = len(acf.connectionInfo.Database)
		}
	case 5:
		if acf.cursor > len(acf.connectionInfo.SSLMode) {
			acf.cursor = len(acf.connectionInfo.SSLMode)
		}
	case 6:
		portStr := fmt.Sprintf("%d", acf.connectionInfo.Port)
		if acf.cursor > len(portStr) {
			acf.cursor = len(portStr)
		}
	}
}

func (acf *AddConnectionForm) moveCursorRight() {
	switch acf.field {
	case 0:
		if acf.cursor < len(acf.connectionInfo.Name) {
			acf.cursor++
		}
	case 1:
		if acf.cursor < len(acf.connectionInfo.Host) {
			acf.cursor++
		}
	case 2:
		if acf.cursor < len(acf.connectionInfo.User) {
			acf.cursor++
		}
	case 3:
		if acf.cursor < len(acf.connectionInfo.Password) {
			acf.cursor++
		}
	case 4:
		if acf.cursor < len(acf.connectionInfo.Database) {
			acf.cursor++
		}
	case 5:
		if acf.cursor < len(acf.connectionInfo.SSLMode) {
			acf.cursor++
		}
	case 6:
		portStr := fmt.Sprintf("%d", acf.connectionInfo.Port)
		if acf.cursor < len(portStr) {
			acf.cursor++
		}
	}
}

func (acf *AddConnectionForm) addChar(char string) {
	switch acf.field {
	case 0:
		acf.connectionInfo.Name = acf.insertAtCursor(acf.connectionInfo.Name, char)
		acf.cursor++
	case 1:
		acf.connectionInfo.Host = acf.insertAtCursor(acf.connectionInfo.Host, char)
		acf.cursor++
	case 2:
		acf.connectionInfo.User = acf.insertAtCursor(acf.connectionInfo.User, char)
		acf.cursor++
	case 3:
		acf.connectionInfo.Password = acf.insertAtCursor(acf.connectionInfo.Password, char)
		acf.cursor++
	case 4:
		acf.connectionInfo.Database = acf.insertAtCursor(acf.connectionInfo.Database, char)
		acf.cursor++
	case 5:
		acf.connectionInfo.SSLMode = acf.insertAtCursor(acf.connectionInfo.SSLMode, char)
		acf.cursor++
	case 6:
		portStr := fmt.Sprintf("%d", acf.connectionInfo.Port)
		portStr = acf.insertAtCursor(portStr, char)
		acf.cursor++
		var port int
		fmt.Sscanf(portStr, "%d", &port)
		acf.connectionInfo.Port = port
	}
}

func (acf *AddConnectionForm) deleteChar() {
	switch acf.field {
	case 0:
		if acf.cursor > 0 {
			acf.connectionInfo.Name = acf.deleteFromCursor(acf.connectionInfo.Name)
			acf.cursor--
		}
	case 1:
		if acf.cursor > 0 {
			acf.connectionInfo.Host = acf.deleteFromCursor(acf.connectionInfo.Host)
			acf.cursor--
		}
	case 2:
		if acf.cursor > 0 {
			acf.connectionInfo.User = acf.deleteFromCursor(acf.connectionInfo.User)
			acf.cursor--
		}
	case 3:
		if acf.cursor > 0 {
			acf.connectionInfo.Password = acf.deleteFromCursor(acf.connectionInfo.Password)
			acf.cursor--
		}
	case 4:
		if acf.cursor > 0 {
			acf.connectionInfo.Database = acf.deleteFromCursor(acf.connectionInfo.Database)
			acf.cursor--
		}
	case 5:
		if acf.cursor > 0 {
			acf.connectionInfo.SSLMode = acf.deleteFromCursor(acf.connectionInfo.SSLMode)
			acf.cursor--
		}
	case 6:
		portStr := fmt.Sprintf("%d", acf.connectionInfo.Port)
		if acf.cursor > 0 {
			portStr = acf.deleteFromCursor(portStr)
			acf.cursor--
			var port int
			fmt.Sscanf(portStr, "%d", &port)
			acf.connectionInfo.Port = port
		}
	}
}

func (acf *AddConnectionForm) insertAtCursor(current, char string) string {
	if acf.cursor > len(current) {
		return current + char
	}
	return current[:acf.cursor] + char + current[acf.cursor:]
}

func (acf *AddConnectionForm) deleteFromCursor(current string) string {
	if acf.cursor <= 1 {
		if len(current) <= 1 {
			return ""
		}
		return current[1:]
	}
	if acf.cursor > len(current) {
		return current[:len(current)-1]
	}
	return current[:acf.cursor-1] + current[acf.cursor:]
}

func (acf *AddConnectionForm) validate() bool {
	return strings.TrimSpace(acf.connectionInfo.Name) != "" &&
		strings.TrimSpace(acf.connectionInfo.Host) != "" &&
		strings.TrimSpace(acf.connectionInfo.Database) != "" &&
		acf.connectionInfo.Port > 0
}

func (acf *AddConnectionForm) View() string {
	title := "🔧 Configure Connection"
	if acf.mode == "edit" {
		title = "✏️ Edit Connection"
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFD700")).
		Bold(true).
		Align(lipgloss.Center).
		Width(70)

	content := titleStyle.Render(title) + "\n\n"

	fields := []struct {
		label string
		value string
		field int
	}{
		{"Connection Name:", acf.connectionInfo.Name, 0},
		{"Host:", acf.connectionInfo.Host, 1},
		{"Port:", fmt.Sprintf("%d", acf.connectionInfo.Port), 6},
		{"User:", acf.connectionInfo.User, 2},
		{"Password:", acf.connectionInfo.Password, 3},
		{"Database:", acf.connectionInfo.Database, 4},
		{"SSL Mode:", acf.connectionInfo.SSLMode, 5},
	}

	labelWidth := acf.fieldLabelWidth
	valueWidth := 35

	for i, field := range fields {
		lineStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
		if acf.field == i {
			lineStyle = lineStyle.Background(lipgloss.Color("#2B6CB0"))
		}

		value := field.value
		if field.field == 3 && value != "" {
			value = strings.Repeat("*", len(value))
		}

		labelStyled := lipgloss.NewStyle().
			Width(labelWidth).
			Align(lipgloss.Right).
			Bold(true).
			Render(field.label)

		valueStyled := lipgloss.NewStyle().
			Width(valueWidth).
			Render(value)

		cursorMarker := " "
		if acf.field == i {
			cursorMarker = ">"
		}

		content += lineStyle.Render(fmt.Sprintf("%s %s %s", cursorMarker, labelStyled, valueStyled)) + "\n"
	}

	if acf.validationError != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF6B6B")).
			Bold(true)
		content += "\n" + errorStyle.Render(acf.validationError) + "\n"
	}

	helpText := "↑/↓ Navega | ←/→ Move Cursor | Enter Salva | Esc Cancela | Tab Alterna Campo"
	if acf.mode == "edit" {
		helpText = "↑/↓ Navega | ←/→ Move Cursor | Enter Atualiza | Esc Cancela Edição"
	}

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#808080")).
		Italic(true)

	content += "\n" + helpStyle.Render(helpText)

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#4169E1")).
		Padding(1, 2)

	return border.Render(content)
}

func (acf *AddConnectionForm) GetConnectionInfo() *ConnectionInfo {
	return acf.connectionInfo
}

func (acf *AddConnectionForm) SetConnectionInfo(info *ConnectionInfo) {
	if info == nil {
		return
	}
	acf.connectionInfo = info
	acf.mode = "edit"
	acf.field = 0
	acf.cursor = len(info.Name)
}

func (acf *AddConnectionForm) IsConfirmed() bool {
	return acf.isConfirmed
}

func (acf *AddConnectionForm) IsCancelled() bool {
	return acf.isConfirmed && strings.TrimSpace(acf.connectionInfo.Name) == ""
}
