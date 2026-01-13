package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type PaneRenderer struct {
	styles PaneStyles
}

type PaneStyles struct {
	Header    lipgloss.Style
	Body      lipgloss.Style
	Selected  lipgloss.Style
	Normal    lipgloss.Style
	Focused   lipgloss.Style
	Unfocused lipgloss.Style
	Border    lipgloss.Style
	Status    lipgloss.Style
}

func NewPaneRenderer() *PaneRenderer {
	return &PaneRenderer{
		styles: PaneStyles{
			Header:    lipgloss.NewStyle().Background(lipgloss.Color("#1a1a1a")).Foreground(lipgloss.Color("#FFD700")).Bold(true).Padding(0, 1),
			Body:      lipgloss.NewStyle().Background(lipgloss.Color("#000000")).Foreground(lipgloss.Color("#FFFFFF")),
			Selected:  lipgloss.NewStyle().Foreground(lipgloss.Color("#000000")).Background(lipgloss.Color("#FFD700")),
			Normal:    lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")),
			Focused:   lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#00FF00")),
			Unfocused: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#666666")),
			Status:    lipgloss.NewStyle().Foreground(lipgloss.Color("#808080")),
		},
	}
}

func (pr *PaneRenderer) RenderPanes(paneModel *PaneModel, width, height int) string {
	topHeight := height / 2
	bottomHeight := height - topHeight

	paneWidth := width / 3
	if paneWidth < 20 {
		paneWidth = 20
	}

	var topPanes []string
	paneNames := []string{"Databases", "Schemas", "Tables"}

	for i := 0; i < 3; i++ {
		paneType := PaneType(i)
		pane := paneModel.GetPane(paneType)
		isFocused := paneModel.GetFocus() == paneType
		paneViewport := topHeight - 4
		if paneViewport < 1 {
			paneViewport = 1
		}
		paneModel.SetPaneViewport(paneType, paneViewport)
		paneContent := pr.renderPane(pane, paneNames[i], paneWidth, topHeight-2, isFocused)
		topPanes = append(topPanes, paneContent)
	}

	topRow := lipgloss.JoinHorizontal(lipgloss.Top, topPanes...)

	isDataFocused := paneModel.GetFocus() == PaneData
	dataPane := pr.renderDataPane(paneModel, "Data", width, bottomHeight-2, isDataFocused)

	return lipgloss.JoinVertical(lipgloss.Left, topRow, dataPane)
}

func (pr *PaneRenderer) renderPane(pane *PaneState, title string, width, height int, isFocused bool) string {
	header := pr.renderPaneHeader(title, isFocused)
	body := pr.renderPaneBody(pane, width, height-2, isFocused)
	footer := pr.renderPaneFooter(pane)

	content := header + "\n" + body + "\n" + footer

	borderStyle := pr.styles.Unfocused
	if isFocused {
		borderStyle = pr.styles.Focused
	}

	return borderStyle.Width(width).Height(height).Render(content)
}

func (pr *PaneRenderer) renderPaneHeader(title string, isFocused bool) string {
	if isFocused {
		return pr.styles.Header.Render("► " + title)
	}
	return pr.styles.Header.Render("  " + title)
}

func (pr *PaneRenderer) renderPaneBody(pane *PaneState, width, height int, isFocused bool) string {
	if len(pane.Nodes) == 0 {
		return pr.styles.Body.Render("  (empty)")
	}

	start := pane.Offset
	if start >= len(pane.Nodes) {
		start = len(pane.Nodes) - 1
	}

	end := start + height
	if end > len(pane.Nodes) {
		end = len(pane.Nodes)
	}

	var content string
	for i := start; i < end; i++ {
		if i >= len(pane.Nodes) {
			break
		}
		node := pane.Nodes[i]
		isSelected := i == pane.SelectedIdx
		line := pr.renderPaneItem(node, isSelected, isFocused)
		content += line + "\n"
	}

	return content
}

func (pr *PaneRenderer) renderPaneItem(node *TreeNode, isSelected, isFocused bool) string {
	icon := node.GetIcon()
	name := node.Name
	metadata := pr.formatPaneMetadata(node)

	if metadata != "" {
		name += " " + metadata
	}

	line := fmt.Sprintf("  %s %s", icon, name)

	if isSelected {
		return pr.styles.Selected.Render(line)
	}

	return pr.styles.Normal.Render(line)
}

func (pr *PaneRenderer) formatPaneMetadata(node *TreeNode) string {
	switch node.Type {
	case NodeDatabase:
		if node.Metadata.Size != "" {
			return fmt.Sprintf("[%s]", node.Metadata.Size)
		}
	case NodeSchema:
		if node.Metadata.Count > 0 {
			return fmt.Sprintf("(%d)", node.Metadata.Count)
		}
	case NodeTable:
		if node.Metadata.Size != "" {
			return node.Metadata.Size
		}
	case NodeColumn:
		pk := ""
		if node.Metadata.PrimaryKey {
			pk = " 🔑"
		}
		return fmt.Sprintf("%s%s", node.Metadata.DataType, pk)
	}
	return ""
}

func (pr *PaneRenderer) renderPaneFooter(pane *PaneState) string {
	if len(pane.Nodes) == 0 {
		return ""
	}

	selectedNode := pane.GetSelectedNode()
	if selectedNode == nil {
		return ""
	}

	info := fmt.Sprintf("%d/%d", pane.SelectedIdx+1, len(pane.Nodes))
	if selectedNode.Path != "" {
		info += " | " + selectedNode.Name
	}

	return pr.styles.Status.Render(info)
}

func (pr *PaneRenderer) renderDataPane(paneModel *PaneModel, title string, width, height int, isFocused bool) string {
	bodyHeight := height - 2
	if bodyHeight < 1 {
		bodyHeight = 1
	}
	paneModel.SetDataViewport(bodyHeight)
	paneModel.SetDataViewportWidth(width - 4)

	header := pr.renderPaneHeader(title, isFocused)
	body := pr.renderDataBody(paneModel, width, height-2, isFocused)
	footer := pr.renderDataFooter(paneModel)

	content := header + "\n" + body + "\n" + footer

	borderStyle := pr.styles.Unfocused
	if isFocused {
		borderStyle = pr.styles.Focused
	}

	return borderStyle.Width(width).Height(height).Render(content)
}

func (pr *PaneRenderer) renderDataBody(paneModel *PaneModel, width, height int, isFocused bool) string {
	data := paneModel.GetData()
	columns := paneModel.GetDataColumns()
	if len(data) == 0 || len(columns) == 0 {
		return pr.styles.Body.Render("  (no data)")
	}

	rowOffset := paneModel.GetDataRowOffset()
	visibleRows := paneModel.GetDataViewportRows()
	endRow := rowOffset + visibleRows
	if endRow > len(data) {
		endRow = len(data)
	}

	colOffset := paneModel.GetDataColOffset()
	visibleColumns := pr.visibleColumns(columns[colOffset:], data, width-4)

	columnWidths := pr.calculateColumnWidths(visibleColumns, data[rowOffset:endRow])

	var lines []string
	var headerParts []string
	for _, col := range visibleColumns {
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700")).Bold(true).Width(columnWidths[col])
		headerParts = append(headerParts, style.Render(col))
	}
	lines = append(lines, strings.Join(headerParts, " "))
	lines = append(lines, strings.Repeat("-", width-4))

	maxRows := height - 3
	if maxRows < 1 {
		maxRows = 1
	}
	for rowIdx := rowOffset; rowIdx < len(data) && rowIdx < rowOffset+maxRows; rowIdx++ {
		row := data[rowIdx]
		var rowParts []string
		for _, col := range visibleColumns {
			val := row[col]
			valStr := fmt.Sprintf("%v", val)
			if len(valStr) > columnWidths[col] {
				truncateWidth := columnWidths[col]
				if truncateWidth > 3 {
					valStr = valStr[:truncateWidth-3] + "..."
				} else {
					valStr = valStr[:truncateWidth]
				}
			}
			style := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Width(columnWidths[col])
			rowParts = append(rowParts, style.Render(valStr))
		}
		lines = append(lines, strings.Join(rowParts, " "))
	}

	return strings.Join(lines, "\n")
}

func (pr *PaneRenderer) renderDataFooter(paneModel *PaneModel) string {
	data := paneModel.GetData()
	cols := paneModel.GetDataColumns()
	if len(data) == 0 || len(cols) == 0 {
		return ""
	}
	rowOffset := paneModel.GetDataRowOffset()
	visibleRows := paneModel.GetDataViewportRows()
	rowEnd := rowOffset + visibleRows
	if rowEnd > len(data) {
		rowEnd = len(data)
	}

	colOffset := paneModel.GetDataColOffset()
	colEnd := colOffset + len(pr.visibleColumns(cols[colOffset:], data, paneModel.GetDataViewportWidth()))
	if colEnd > len(cols) {
		colEnd = len(cols)
	}

	info := fmt.Sprintf("rows %d-%d of %d | cols %d-%d of %d",
		rowOffset+1, rowEnd, len(data), colOffset+1, colEnd, len(cols))
	return pr.styles.Status.Render(info)
}

func (pr *PaneRenderer) visibleColumns(columns []string, data []map[string]interface{}, maxWidth int) []string {
	if maxWidth < 20 {
		maxWidth = 20
	}

	var selected []string
	currentWidth := 0

	for _, col := range columns {
		width := pr.computeColumnWidth(col, data)
		if width < 8 {
			width = 8
		}
		if width > 30 {
			width = 30
		}

		space := 1
		if len(selected) == 0 {
			space = 0
		}

		if currentWidth+width+space > maxWidth {
			break
		}

		selected = append(selected, col)
		currentWidth += width + space
	}

	if len(selected) == 0 && len(columns) > 0 {
		selected = append(selected, columns[0])
	}

	return selected
}

func (pr *PaneRenderer) computeColumnWidth(column string, data []map[string]interface{}) int {
	maxWidth := len(column)
	for _, row := range data {
		val := row[column]
		valStr := fmt.Sprintf("%v", val)
		if len(valStr) > maxWidth {
			maxWidth = len(valStr)
		}
	}
	return maxWidth
}

func (pr *PaneRenderer) calculateColumnWidths(columns []string, data []map[string]interface{}) map[string]int {
	widths := make(map[string]int, len(columns))
	for _, col := range columns {
		width := pr.computeColumnWidth(col, data)
		if width > 30 {
			width = 30
		}
		if width < 8 {
			width = 8
		}
		widths[col] = width
	}
	return widths
}

func (pr *PaneRenderer) renderHeader() string {
	title := "🗂 XTreeGold - PostgreSQL Navigator (dBeaver Style)"
	return pr.styles.Header.Render(title)
}

func (pr *PaneRenderer) renderStatus(paneModel *PaneModel) string {
	focusNames := []string{"Databases", "Schemas", "Tables", "Data"}
	currentFocus := paneModel.GetFocus()

	var parts []string
	for i, name := range focusNames {
		if PaneType(i) == currentFocus {
			parts = append(parts, fmt.Sprintf("[%s]", name))
		} else {
			parts = append(parts, name)
		}
	}

	status := strings.Join(parts, " | ")
	status += " | Tab: Switch Pane | ←/→: Navigate | Enter: Drill Down | Ctrl+Q: Query | ESC: Back/Quit | Ctrl+X: Quit"

	return pr.styles.Status.Render(status)
}

func (ps *PaneState) GetSelectedNode() *TreeNode {
	if ps.SelectedIdx >= 0 && ps.SelectedIdx < len(ps.Nodes) {
		return ps.Nodes[ps.SelectedIdx]
	}
	return nil
}
