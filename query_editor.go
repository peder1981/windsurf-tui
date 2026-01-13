package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type QueryEditor struct {
	value  string
	cursor int
	width  int
	height int
}

func NewQueryEditor() *QueryEditor {
	return &QueryEditor{
		value:  "",
		cursor: 0,
		width:  80,
		height: 20,
	}
}

func (qe *QueryEditor) SetValue(value string) {
	qe.value = value
	if qe.cursor > len(value) {
		qe.cursor = len(value)
	}
}

func (qe *QueryEditor) GetValue() string {
	return qe.value
}

func (qe *QueryEditor) Init() tea.Cmd {
	return nil
}

func (qe *QueryEditor) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyLeft:
			if qe.cursor > 0 {
				qe.cursor--
			}
		case tea.KeyRight:
			if qe.cursor < len(qe.value) {
				qe.cursor++
			}
		case tea.KeyUp:
			qe.moveCursorUp()
		case tea.KeyDown:
			qe.moveCursorDown()
		case tea.KeyBackspace:
			if qe.cursor > 0 {
				qe.value = qe.value[:qe.cursor-1] + qe.value[qe.cursor:]
				qe.cursor--
			}
		case tea.KeyDelete:
			if qe.cursor < len(qe.value) {
				qe.value = qe.value[:qe.cursor] + qe.value[qe.cursor+1:]
			}
		case tea.KeyEnter:
			// Enter executes the query
			return qe, func() tea.Msg {
				return ExecuteQueryMsg{query: qe.value}
			}
		case tea.KeyCtrlJ:
			// Ctrl+J adds a newline
			qe.value += "\n"
			qe.cursor++
		case tea.KeyHome:
			qe.moveToLineStart()
		case tea.KeyEnd:
			qe.moveToLineEnd()
		case tea.KeyCtrlV:
			return qe, qe.paste()
		default:
			if len(msg.Runes) > 0 {
				qe.value = qe.value[:qe.cursor] + string(msg.Runes) + qe.value[qe.cursor:]
				qe.cursor += len(msg.Runes)
			}
		}
	}

	return qe, nil
}

func (qe *QueryEditor) paste() tea.Cmd {
	return func() tea.Msg {
		return nil
	}
}

func (qe *QueryEditor) View() string {
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#00FFFF")).
		Padding(0, 1)

	helpText := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#808080")).
		Render("Type your SQL query here. Press Enter to execute, Ctrl+J for newline, Esc to cancel.")

	editor := border.Render(qe.value)

	if qe.value == "" {
		editor = border.Render("")
	}

	return helpText + "\n\n" + editor
}

func (qe *QueryEditor) moveCursorUp() {
	lines := strings.Split(qe.value, "\n")
	if len(lines) > 1 {
		currentLine := qe.getCurrentLine()
		posInLine := qe.getPosInCurrentLine()

		if currentLine > 0 {
			previousLineLen := len(lines[currentLine-1])
			if posInLine > previousLineLen {
				qe.cursor -= posInLine - previousLineLen
			}
			qe.cursor -= len(lines[currentLine]) + 1
		}
	}
}

func (qe *QueryEditor) moveCursorDown() {
	lines := strings.Split(qe.value, "\n")
	if len(lines) > 1 {
		currentLine := qe.getCurrentLine()
		posInLine := qe.getPosInCurrentLine()

		if currentLine < len(lines)-1 {
			qe.cursor += len(lines[currentLine]) + 1
			nextLineLen := len(lines[currentLine+1])
			if posInLine > nextLineLen {
				qe.cursor -= posInLine - nextLineLen
			}
		}
	}
}

func (qe *QueryEditor) moveToLineStart() {
	lines := strings.Split(qe.value, "\n")
	currentLine := qe.getCurrentLine()

	if currentLine == 0 {
		qe.cursor = 0
	} else {
		for i := 0; i < currentLine; i++ {
			qe.cursor -= len(lines[i]) + 1
		}
	}
}

func (qe *QueryEditor) moveToLineEnd() {
	lines := strings.Split(qe.value, "\n")
	currentLine := qe.getCurrentLine()

	if currentLine < len(lines)-1 {
		for i := currentLine; i < len(lines)-1; i++ {
			qe.cursor += len(lines[i]) + 1
		}
	} else {
		qe.cursor = len(qe.value)
	}
}

func (qe *QueryEditor) getCurrentLine() int {
	lines := strings.Split(qe.value[:qe.cursor], "\n")
	return len(lines) - 1
}

func (qe *QueryEditor) getPosInCurrentLine() int {
	if qe.cursor == 0 {
		return 0
	}

	lines := strings.Split(qe.value[:qe.cursor], "\n")
	return len(lines[len(lines)-1])
}
