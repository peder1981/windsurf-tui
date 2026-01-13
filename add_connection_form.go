package main

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type formField int

const (
	fieldDriver formField = iota
	fieldName
	fieldHost
	fieldPort
	fieldUser
	fieldPassword
	fieldDatabase
	fieldSSLMode
	fieldPath
)

var driverLabels = map[ConnectionType]string{
	ConnectionPostgres: "PostgreSQL",
	ConnectionSQLite:   "SQLite",
}

type AddConnectionForm struct {
	connectionInfo  *ConnectionInfo
	cursor          int
	field           int
	isConfirmed     bool
	cancelled       bool
	validationError string
	mode            string
	fieldLabelWidth int
}

func NewAddConnectionForm() *AddConnectionForm {
	return &AddConnectionForm{
		connectionInfo: &ConnectionInfo{
			Type:     ConnectionPostgres,
			Name:     "",
			Host:     "localhost",
			Port:     5432,
			User:     "postgres",
			Password: "",
			Database: "",
			SSLMode:  "disable",
			Path:     "",
		},
		cursor:          0,
		field:           0,
		fieldLabelWidth: 22,
		mode:            "add",
	}
}

func (acf *AddConnectionForm) Init() tea.Cmd { return nil }

func (acf *AddConnectionForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyUp:
			acf.moveField(-1)
		case tea.KeyDown:
			acf.moveField(1)
		case tea.KeyTab:
			acf.moveField(1)
		case tea.KeyEnter:
			if acf.validate() {
				acf.isConfirmed = true
			}
		case tea.KeyEscape:
			acf.isConfirmed = true
			acf.cancelled = true
		case tea.KeyBackspace:
			acf.deleteChar()
		case tea.KeyLeft:
			if acf.currentField() == fieldDriver {
				acf.toggleDriver(-1)
			} else {
				acf.moveCursor(-1)
			}
		case tea.KeyRight:
			if acf.currentField() == fieldDriver {
				acf.toggleDriver(1)
			} else {
				acf.moveCursor(1)
			}
		case tea.KeyCtrlT:
			acf.toggleDriver(1)
		case tea.KeyRunes:
			if len(msg.Runes) > 0 && !msg.Alt {
				acf.addRunes(msg.Runes)
			}
		case tea.KeySpace:
			if !msg.Alt {
				acf.addRunes([]rune{' '})
			}
		}
	}
	return acf, nil
}

func (acf *AddConnectionForm) visibleFields() []formField {
	fields := []formField{fieldDriver, fieldName}
	if acf.connectionInfo.Type == ConnectionSQLite {
		fields = append(fields, fieldPath)
	} else {
		fields = append(fields, fieldHost, fieldPort, fieldUser, fieldPassword, fieldDatabase, fieldSSLMode)
	}
	return fields
}

func (acf *AddConnectionForm) currentField() formField {
	fields := acf.visibleFields()
	if len(fields) == 0 {
		return fieldDriver
	}
	if acf.field < 0 {
		acf.field = 0
	}
	if acf.field >= len(fields) {
		acf.field = len(fields) - 1
	}
	return fields[acf.field]
}

func (acf *AddConnectionForm) moveField(delta int) {
	fields := acf.visibleFields()
	if len(fields) == 0 {
		return
	}
	acf.field += delta
	if acf.field < 0 {
		acf.field = 0
	} else if acf.field >= len(fields) {
		acf.field = len(fields) - 1
	}
	acf.syncCursorToField()
}

func (acf *AddConnectionForm) syncCursorToField() {
	if acf.currentField() == fieldDriver {
		acf.cursor = 0
		return
	}
	val := acf.currentFieldValue()
	if acf.cursor > len(val) {
		acf.cursor = len(val)
	}
}

func (acf *AddConnectionForm) currentFieldValue() string {
	info := acf.connectionInfo
	switch acf.currentField() {
	case fieldName:
		return info.Name
	case fieldHost:
		return info.Host
	case fieldPort:
		if info.Port <= 0 {
			return ""
		}
		return fmt.Sprintf("%d", info.Port)
	case fieldUser:
		return info.User
	case fieldPassword:
		return info.Password
	case fieldDatabase:
		return info.Database
	case fieldSSLMode:
		return info.SSLMode
	case fieldPath:
		return info.Path
	default:
		return ""
	}
}

func (acf *AddConnectionForm) moveCursor(delta int) {
	if acf.currentField() == fieldDriver {
		return
	}
	acf.cursor += delta
	if acf.cursor < 0 {
		acf.cursor = 0
	}
	val := acf.currentFieldValue()
	if acf.cursor > len(val) {
		acf.cursor = len(val)
	}
}

func (acf *AddConnectionForm) toggleDriver(delta int) {
	drivers := []ConnectionType{ConnectionPostgres, ConnectionSQLite}
	current := acf.connectionInfo.Type
	idx := 0
	for i, d := range drivers {
		if d == current {
			idx = i
			break
		}
	}
	idx = (idx + delta + len(drivers)) % len(drivers)
	acf.connectionInfo.Type = drivers[idx]
	acf.field = 0 // keep cursor on driver when toggling
	acf.cursor = 0
	acf.validationError = ""
}

func (acf *AddConnectionForm) addChar(char string) {
	if acf.currentField() == fieldDriver {
		return
	}

	switch acf.currentField() {
	case fieldName:
		acf.connectionInfo.Name = acf.insertAtCursor(acf.connectionInfo.Name, char)
	case fieldHost:
		acf.connectionInfo.Host = acf.insertAtCursor(acf.connectionInfo.Host, char)
	case fieldPort:
		portStr := acf.insertAtCursor(acf.currentFieldValue(), char)
		acf.setPortFromString(portStr)
	case fieldUser:
		acf.connectionInfo.User = acf.insertAtCursor(acf.connectionInfo.User, char)
	case fieldPassword:
		acf.connectionInfo.Password = acf.insertAtCursor(acf.connectionInfo.Password, char)
	case fieldDatabase:
		acf.connectionInfo.Database = acf.insertAtCursor(acf.connectionInfo.Database, char)
	case fieldSSLMode:
		acf.connectionInfo.SSLMode = acf.insertAtCursor(acf.connectionInfo.SSLMode, char)
	case fieldPath:
		acf.connectionInfo.Path = acf.insertAtCursor(acf.connectionInfo.Path, char)
	}
	acf.cursor++
}

func (acf *AddConnectionForm) addRunes(runes []rune) {
	for _, r := range runes {
		acf.addChar(string(r))
	}
}

func (acf *AddConnectionForm) deleteChar() {
	if acf.currentField() == fieldDriver || acf.cursor == 0 {
		return
	}

	switch acf.currentField() {
	case fieldName:
		acf.connectionInfo.Name = acf.deleteFromCursor(acf.connectionInfo.Name)
	case fieldHost:
		acf.connectionInfo.Host = acf.deleteFromCursor(acf.connectionInfo.Host)
	case fieldPort:
		portStr := acf.deleteFromCursor(acf.currentFieldValue())
		acf.cursor--
		acf.setPortFromString(portStr)
		return
	case fieldUser:
		acf.connectionInfo.User = acf.deleteFromCursor(acf.connectionInfo.User)
	case fieldPassword:
		acf.connectionInfo.Password = acf.deleteFromCursor(acf.connectionInfo.Password)
	case fieldDatabase:
		acf.connectionInfo.Database = acf.deleteFromCursor(acf.connectionInfo.Database)
	case fieldSSLMode:
		acf.connectionInfo.SSLMode = acf.deleteFromCursor(acf.connectionInfo.SSLMode)
	case fieldPath:
		acf.connectionInfo.Path = acf.deleteFromCursor(acf.connectionInfo.Path)
	}
	acf.cursor--
}

func (acf *AddConnectionForm) setPortFromString(value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		acf.connectionInfo.Port = 0
		return
	}
	port, err := strconv.Atoi(value)
	if err != nil {
		return
	}
	acf.connectionInfo.Port = port
}

func (acf *AddConnectionForm) insertAtCursor(current, char string) string {
	if acf.cursor > len(current) {
		return current + char
	}
	return current[:acf.cursor] + char + current[acf.cursor:]
}

func (acf *AddConnectionForm) deleteFromCursor(current string) string {
	if acf.cursor == 0 || len(current) == 0 {
		return current
	}
	if acf.cursor >= len(current) {
		return current[:len(current)-1]
	}
	return current[:acf.cursor-1] + current[acf.cursor:]
}

func (acf *AddConnectionForm) validate() bool {
	info := acf.connectionInfo
	if strings.TrimSpace(info.Name) == "" {
		acf.validationError = "Nome da conexão é obrigatório"
		return false
	}

	switch info.Type {
	case ConnectionSQLite:
		if strings.TrimSpace(info.Path) == "" {
			acf.validationError = "Informe o caminho do arquivo SQLite"
			return false
		}
	default:
		switch {
		case strings.TrimSpace(info.Host) == "":
			acf.validationError = "Host é obrigatório para PostgreSQL"
			return false
		case info.Port <= 0:
			acf.validationError = "Porta inválida"
			return false
		case strings.TrimSpace(info.Database) == "":
			acf.validationError = "Database é obrigatório"
			return false
		}
	}

	acf.validationError = ""
	return true
}

func (acf *AddConnectionForm) View() string {
	title := "🔧 Nova Conexão"
	if acf.mode == "edit" {
		title = "✏️ Editar Conexão"
	}

	var lines []string
	header := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFD700")).
		Bold(true).
		Width(74).
		Align(lipgloss.Center).
		Render(title)
	lines = append(lines, header, "")

	for idx, field := range acf.visibleFields() {
		line := acf.renderField(field, acf.field == idx)
		lines = append(lines, line)
	}

	if acf.validationError != "" {
		errLine := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF6B6B")).
			Bold(true).
			Render("⚠ " + acf.validationError)
		lines = append(lines, "", errLine)
	}

	helpText := "↑/↓ Navega | ←/→ Move cursor (Driver alterna) | Tab Avança | Ctrl+T Troca Driver | Enter Salva | Esc Cancela"
	helpLine := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Italic(true).
		Render(helpText)
	lines = append(lines, "", helpLine)

	container := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#4169E1")).
		Padding(1, 2).
		Render(strings.Join(lines, "\n"))

	return container
}

func (acf *AddConnectionForm) renderField(field formField, focused bool) string {
	label := acf.fieldLabel(field)
	value := acf.displayValue(field)

	lineStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
	if focused {
		lineStyle = lineStyle.Background(lipgloss.Color("#1F3B73"))
	}

	labelStyled := lipgloss.NewStyle().
		Width(acf.fieldLabelWidth).
		Align(lipgloss.Right).
		Bold(true).
		Render(label)

	valueStyled := lipgloss.NewStyle().
		Width(38).
		Render(value)

	cursorMarker := " "
	if focused {
		cursorMarker = "▶"
	}

	return lineStyle.Render(fmt.Sprintf("%s %s %s", cursorMarker, labelStyled, valueStyled))
}

func (acf *AddConnectionForm) displayValue(field formField) string {
	info := acf.connectionInfo
	switch field {
	case fieldDriver:
		label := driverLabels[info.Type]
		if label == "" {
			label = string(info.Type)
		}
		return fmt.Sprintf("%s (Ctrl+T)", label)
	case fieldName:
		if info.Name == "" {
			return "(nome amigável)"
		}
		return info.Name
	case fieldHost:
		if info.Host == "" {
			return "(ex: localhost)"
		}
		return info.Host
	case fieldPort:
		if info.Port <= 0 {
			return "(ex: 5432)"
		}
		return fmt.Sprintf("%d", info.Port)
	case fieldUser:
		if info.User == "" {
			return "(opcional)"
		}
		return info.User
	case fieldPassword:
		if info.Password == "" {
			return "(opcional)"
		}
		return strings.Repeat("•", len(info.Password))
	case fieldDatabase:
		if info.Database == "" {
			return "(database)"
		}
		return info.Database
	case fieldSSLMode:
		if info.SSLMode == "" {
			return "(disable/require/verify-full)"
		}
		return info.SSLMode
	case fieldPath:
		if info.Path == "" {
			return "(ex: /dados/app.db)"
		}
		return fmt.Sprintf("%s (%s)", filepath.Base(info.Path), info.Path)
	default:
		return ""
	}
}

func (acf *AddConnectionForm) fieldLabel(field formField) string {
	switch field {
	case fieldDriver:
		return "Driver:"
	case fieldName:
		return "Nome:"
	case fieldHost:
		return "Host:"
	case fieldPort:
		return "Porta:"
	case fieldUser:
		return "Usuário:"
	case fieldPassword:
		return "Senha:"
	case fieldDatabase:
		return "Database:"
	case fieldSSLMode:
		return "SSL Mode:"
	case fieldPath:
		return "Arquivo SQLite:"
	default:
		return ""
	}
}

func (acf *AddConnectionForm) GetConnectionInfo() *ConnectionInfo {
	return acf.connectionInfo
}

func (acf *AddConnectionForm) SetConnectionInfo(info *ConnectionInfo) {
	if info == nil {
		return
	}
	acf.connectionInfo = info
	if acf.connectionInfo.Type == "" {
		acf.connectionInfo.Type = ConnectionPostgres
	}
	acf.mode = "edit"
	acf.field = 0
	acf.cursor = 0
	acf.validationError = ""
}

func (acf *AddConnectionForm) IsConfirmed() bool {
	return acf.isConfirmed
}

func (acf *AddConnectionForm) IsCancelled() bool {
	return acf.cancelled
}
