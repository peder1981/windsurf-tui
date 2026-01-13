package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type TextInput struct {
	value       string
	cursor      int
	width       int
	placeholder string
}

func NewTextInput() *TextInput {
	return &TextInput{
		value:       "",
		cursor:      0,
		width:       60,
		placeholder: "",
	}
}

func (ti *TextInput) SetPlaceholder(placeholder string) {
	ti.placeholder = placeholder
}

func (ti *TextInput) SetWidth(width int) {
	if width < 10 {
		width = 10
	}
	ti.width = width
}

func (ti *TextInput) SetValue(value string) {
	ti.value = value
	if ti.cursor > len(value) {
		ti.cursor = len(value)
	}
}

func (ti *TextInput) Value() string {
	return ti.value
}

func (ti *TextInput) Reset() {
	ti.value = ""
	ti.cursor = 0
}

func (ti *TextInput) HandleKey(msg tea.KeyMsg) bool {
	switch msg.Type {
	case tea.KeyLeft:
		if ti.cursor > 0 {
			ti.cursor--
		}
		return true
	case tea.KeyRight:
		if ti.cursor < len(ti.value) {
			ti.cursor++
		}
		return true
	case tea.KeyHome:
		ti.cursor = 0
		return true
	case tea.KeyEnd:
		ti.cursor = len(ti.value)
		return true
	case tea.KeyBackspace:
		if ti.cursor > 0 {
			ti.value = ti.value[:ti.cursor-1] + ti.value[ti.cursor:]
			ti.cursor--
		}
		return true
	case tea.KeyDelete:
		if ti.cursor < len(ti.value) {
			ti.value = ti.value[:ti.cursor] + ti.value[ti.cursor+1:]
		}
		return true
	}

	if len(msg.Runes) > 0 {
		ti.value = ti.value[:ti.cursor] + string(msg.Runes) + ti.value[ti.cursor:]
		ti.cursor += len(msg.Runes)
		return true
	}

	return false
}

func (ti *TextInput) View(prompt string) string {
	value := ti.value
	if value == "" && ti.placeholder != "" {
		value = ti.placeholder
	}

	display := value
	if len(display) > ti.width {
		start := len(display) - ti.width
		display = display[start:]
	}

	style := lipgloss.NewStyle().
		Width(ti.width+2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#00FF00")).
		Padding(0, 1)

	return lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700")).Render(prompt),
		style.Render(display))
}

func ParseKeyValueInput(input string) map[string]string {
	values := make(map[string]string)
	pairs := strings.Split(input, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if key == "" {
			continue
		}
		values[key] = val
	}
	return values
}
