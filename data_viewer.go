package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type DataViewer struct {
	results     []map[string]interface{}
	columns     []string
	verticalPos int
	width       int
	height      int
}

func NewDataViewer() *DataViewer {
	return &DataViewer{
		results:     make([]map[string]interface{}, 0),
		columns:     make([]string, 0),
		verticalPos: 0,
		width:       80,
		height:      20,
	}
}

func (dv *DataViewer) SetResults(results []map[string]interface{}) {
	dv.results = results
	if len(results) > 0 {
		columns := make([]string, 0, len(results[0]))
		for col := range results[0] {
			columns = append(columns, col)
		}
		dv.columns = columns
	}
	dv.verticalPos = 0
}

func (dv *DataViewer) GetResults() []map[string]interface{} {
	return dv.results
}

func (dv *DataViewer) View() string {
	if len(dv.results) == 0 {
		return dv.renderEmptyState()
	}

	return dv.renderTable()
}

func (dv *DataViewer) renderEmptyState() string {
	emptyMsg := "No data to display"

	return lipgloss.NewStyle().
		Width(60).
		Align(lipgloss.Center).
		Height(10).
		Background(lipgloss.Color("#1a1a1a")).
		Border(lipgloss.RoundedBorder()).
		Padding(2, 1).
		Render(emptyMsg)
}

func (dv *DataViewer) renderTable() string {
	columnWidths := dv.calculateColumnWidths()
	header := dv.renderHeader(columnWidths)
	separator := dv.renderSeparator(columnWidths)
	body := dv.renderBody(columnWidths)

	return header + "\n" + separator + "\n" + body
}

func (dv *DataViewer) calculateColumnWidths() map[string]int {
	widths := make(map[string]int)

	for _, col := range dv.columns {
		widths[col] = len(col)
	}

	for _, row := range dv.results {
		for col, val := range row {
			valStr := fmt.Sprintf("%v", val)
			if len(valStr) > widths[col] {
				widths[col] = len(valStr)
			}
		}
	}

	for col := range widths {
		if widths[col] > 30 {
			widths[col] = 30
		}
		if widths[col] < 10 {
			widths[col] = 10
		}
	}

	return widths
}

func (dv *DataViewer) renderHeader(columnWidths map[string]int) string {
	parts := make([]string, 0, len(dv.columns))

	for _, col := range dv.columns {
		style := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFD700")).
			Bold(true).
			Width(columnWidths[col])

		parts = append(parts, style.Render(col))
	}

	return strings.Join(parts, " ")
}

func (dv *DataViewer) renderSeparator(columnWidths map[string]int) string {
	parts := make([]string, 0, len(dv.columns))

	for _, col := range dv.columns {
		style := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#808080"))

		parts = append(parts, style.Render(strings.Repeat("-", columnWidths[col])))
	}

	return strings.Join(parts, " ")
}

func (dv *DataViewer) renderBody(columnWidths map[string]int) string {
	maxRows := dv.height - 4
	startRow := dv.verticalPos

	if startRow+maxRows > len(dv.results) {
		startRow = len(dv.results) - maxRows
	}
	if startRow < 0 {
		startRow = 0
	}

	endRow := startRow + maxRows
	if endRow > len(dv.results) {
		endRow = len(dv.results)
	}

	rows := make([]string, 0, endRow-startRow)

	for i := startRow; i < endRow; i++ {
		row := dv.results[i]
		rowStrs := dv.renderRow(row, columnWidths)
		rows = append(rows, rowStrs...)
	}

	return strings.Join(rows, "\n")
}

func (dv *DataViewer) renderRow(row map[string]interface{}, columnWidths map[string]int) []string {
	parts := make([]string, 0, len(dv.columns))

	for _, col := range dv.columns {
		val := row[col]
		valStr := fmt.Sprintf("%v", val)

		if len(valStr) > 30 {
			valStr = valStr[:27] + "..."
		}

		style := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Width(columnWidths[col])

		parts = append(parts, style.Render(valStr))
	}

	return []string{strings.Join(parts, " ")}
}

func (dv *DataViewer) ScrollUp() {
	if dv.verticalPos > 0 {
		dv.verticalPos--
	}
}

func (dv *DataViewer) ScrollDown() {
	visibleRows := dv.height - 4
	if dv.verticalPos < len(dv.results)-visibleRows {
		dv.verticalPos++
	}
}

func (dv *DataViewer) ScrollToTop() {
	dv.verticalPos = 0
}

func (dv *DataViewer) ScrollToBottom() {
	visibleRows := dv.height - 4
	dv.verticalPos = len(dv.results) - visibleRows
	if dv.verticalPos < 0 {
		dv.verticalPos = 0
	}
}
