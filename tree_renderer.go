package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type TreeRenderer struct {
	styles TreeStyles
}

type TreeStyles struct {
	Normal    lipgloss.Style
	Selected  lipgloss.Style
	Expanded  lipgloss.Style
	Collapsed lipgloss.Style
	Project   lipgloss.Style
	File      lipgloss.Style
	Context   lipgloss.Style
	Search    lipgloss.Style
	Status    lipgloss.Style
	Border    lipgloss.Style
	Header    lipgloss.Style
}

func NewTreeRenderer() *TreeRenderer {
	return &TreeRenderer{
		styles: TreeStyles{
			Normal:    lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")),
			Selected:  lipgloss.NewStyle().Foreground(lipgloss.Color("#000000")).Background(lipgloss.Color("#FFD700")),
			Expanded:  lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")),
			Collapsed: lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFF00")),
			Project:   lipgloss.NewStyle().Foreground(lipgloss.Color("#00BFFF")).Bold(true),
			File:      lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")),
			Context:   lipgloss.NewStyle().Foreground(lipgloss.Color("#FF8C00")),
			Search:    lipgloss.NewStyle().Foreground(lipgloss.Color("#00FFFF")).Background(lipgloss.Color("#000080")),
			Status:    lipgloss.NewStyle().Foreground(lipgloss.Color("#808080")),
			Border:    lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1),
			Header:    lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700")).Bold(true).Align(lipgloss.Center),
		},
	}
}

func (tr *TreeRenderer) RenderTree(model *TreeModel) string {
	visibleNodes := model.GetAllVisibleNodes()
	if len(visibleNodes) == 0 {
		return tr.renderEmptyState("No data available")
	}

	content := ""
	viewport := model.viewport

	start := viewport.Offset
	if start >= len(visibleNodes) {
		start = len(visibleNodes) - 1
		if start < 0 {
			start = 0
		}
	}

	end := start + viewport.Height
	if end > len(visibleNodes) {
		end = len(visibleNodes)
	}

	tr.ensureSelectionVisible(model, visibleNodes, &start, &end)

	for i := start; i < end; i++ {
		if i >= len(visibleNodes) {
			break
		}
		node := visibleNodes[i]
		line := tr.renderNode(node, i == model.viewport.Position)
		content += line + "\n"
	}

	scrollInfo := tr.renderScrollInfo(start, end, len(visibleNodes))
	content += scrollInfo

	return tr.styles.Border.Render(content)
}

func (tr *TreeRenderer) renderNode(node *TreeNode, isSelected bool) string {
	indent := strings.Repeat("  ", node.Level+1)

	var connector string
	if node.Level >= 0 {
		if node.HasChildren() {
			if node.Expanded {
				connector = "├─ "
			} else {
				connector = "├─ "
			}
		} else {
			connector = "└─ "
		}
	} else {
		connector = ""
	}

	icon := node.GetIcon()
	var style lipgloss.Style

	if isSelected {
		style = tr.styles.Selected
	} else {
		switch node.Type {
		case NodeServer:
			style = tr.styles.Project
		case NodeDatabase:
			style = tr.styles.Project
		case NodeSchema:
			style = tr.styles.File
		case NodeTable:
			style = tr.styles.Context
		case NodeColumn:
			style = tr.styles.Normal
		default:
			style = tr.styles.Normal
		}
	}

	var expandIndicator string
	if node.HasChildren() && node.Level >= 0 {
		if node.Expanded {
			expandIndicator = "[-]"
		} else {
			expandIndicator = "[+]"
		}
	}

	nameText := node.Name
	metadata := tr.formatMetadata(node)
	if metadata != "" {
		nameText += " " + metadata
	}

	line := style.Render(fmt.Sprintf("%s%s%s %s", indent, connector, icon, nameText))

	if expandIndicator != "" {
		line += " " + tr.styles.Normal.Render(expandIndicator)
	}

	return line
}

func (tr *TreeRenderer) formatMetadata(node *TreeNode) string {
	switch node.Type {
	case NodeServer:
		if node.Metadata.Count > 0 {
			return fmt.Sprintf("(%d databases)", node.Metadata.Count)
		}
	case NodeDatabase:
		if node.Metadata.Size != "" {
			return fmt.Sprintf("[%s]", node.Metadata.Size)
		}
		if node.Metadata.RowCount > 0 {
			return fmt.Sprintf("~%d rows", node.Metadata.RowCount)
		}
	case NodeSchema:
		if node.Metadata.Count > 0 {
			return fmt.Sprintf("(%d tables)", node.Metadata.Count)
		}
	case NodeTable:
		if node.Metadata.Size != "" {
			return fmt.Sprintf("%s", node.Metadata.Size)
		}
		if node.Metadata.RowCount > 0 {
			return fmt.Sprintf("%d rows", node.Metadata.RowCount)
		}
	case NodeColumn:
		pkIndicator := ""
		if node.Metadata.PrimaryKey {
			pkIndicator = " 🔑"
		}
		nullableIndicator := ""
		if node.Metadata.IsNullable {
			nullableIndicator = " NULL"
		}
		return fmt.Sprintf("%s%s%s", node.Metadata.DataType, nullableIndicator, pkIndicator)
	}
	return ""
}

func (tr *TreeRenderer) renderScrollInfo(start, end, total int) string {
	if total <= 1 {
		return ""
	}

	scrollInfo := fmt.Sprintf("Lines %d-%d of %d", start+1, end, total)
	return tr.styles.Status.Render(scrollInfo)
}

func (tr *TreeRenderer) renderEmptyState(message string) string {
	emptyContent := tr.styles.Normal.Render(fmt.Sprintf("📁 %s", message))
	container := lipgloss.NewStyle().
		Width(60).
		Align(lipgloss.Center).
		Height(10).
		Background(lipgloss.Color("#1a1a1a")).
		Border(lipgloss.RoundedBorder()).
		Padding(2, 1).
		Render(emptyContent)

	return container
}

func (tr *TreeRenderer) ensureSelectionVisible(tm *TreeModel, visibleNodes []*TreeNode, start, end *int) {
	selectedNode := tm.GetSelectedNode()
	if selectedNode == nil {
		return
	}

	selectedIndex := -1
	for i, node := range visibleNodes {
		if node.ID == selectedNode.ID {
			selectedIndex = i
			break
		}
	}

	if selectedIndex == -1 {
		return
	}

	if selectedIndex < *start {
		*start = selectedIndex
		*end = *start + tm.viewport.Height
		if *end > len(visibleNodes) {
			*end = len(visibleNodes)
		}
		tm.viewport.Offset = *start
	} else if selectedIndex >= *end {
		*end = selectedIndex + 1
		*start = *end - tm.viewport.Height
		if *start < 0 {
			*start = 0
		}
		tm.viewport.Offset = *start
	}

	tm.viewport.Position = selectedIndex - *start
}

func (tr *TreeRenderer) renderHeader() string {
	title := "🗂 XTreeGold - PostgreSQL Navigator"
	return tr.styles.Header.Render(title)
}

func (tr *TreeRenderer) renderStatus(tm *TreeModel) string {
	selectedNode := tm.GetSelectedNode()
	var status string

	if selectedNode != nil {
		path := tr.buildPath(selectedNode)
		status = fmt.Sprintf("Selected: %s | Path: %s | Total: %d items",
			selectedNode.Name, path, tm.GetNodeCount())
	} else {
		status = fmt.Sprintf("Total: %d items", tm.GetNodeCount())
	}

	if tm.searchMode {
		status = fmt.Sprintf("🔍 Search: %s | %s", tm.searchQuery, status)
	}

	return tr.styles.Status.Render(status)
}

func (tr *TreeRenderer) buildPath(node *TreeNode) string {
	if node == nil || node.Level < 0 {
		return "/"
	}

	path := node.Name
	current := node.Parent

	for current != nil && current.Level >= 0 {
		path = current.Name + " > " + path
		current = current.Parent
	}

	return path
}

func (tr *TreeRenderer) RenderDetails(node *TreeNode) string {
	if node == nil {
		return tr.renderEmptyState("No item selected")
	}

	content := fmt.Sprintf("Type: %s\n", node.Type.String())
	content += fmt.Sprintf("Name: %s\n", node.Name)
	content += fmt.Sprintf("ID: %s\n", node.ID)

	if node.Path != "" {
		content += fmt.Sprintf("Path: %s\n", node.Path)
	}

	content += "\nMetadata:\n"

	if node.Metadata.Size != "" {
		content += fmt.Sprintf("  Size: %s\n", node.Metadata.Size)
	}

	if node.Metadata.Modified != "" {
		content += fmt.Sprintf("  Modified: %s\n", node.Metadata.Modified)
	}

	if node.Metadata.Count > 0 {
		content += fmt.Sprintf("  Count: %d\n", node.Metadata.Count)
	}

	if node.Metadata.ContextType != "" {
		content += fmt.Sprintf("  Type: %s\n", node.Metadata.ContextType)
	}

	if node.Metadata.URI != "" {
		content += fmt.Sprintf("  URI: %s\n", node.Metadata.URI)
	}

	if node.Metadata.DataType != "" {
		content += fmt.Sprintf("  Data Type: %s\n", node.Metadata.DataType)
	}

	if node.Metadata.IsNullable {
		content += fmt.Sprintf("  Nullable: Yes\n")
	}

	if node.Metadata.PrimaryKey {
		content += fmt.Sprintf("  Primary Key: Yes\n")
	}

	if node.HasChildren() {
		content += fmt.Sprintf("\nChildren: %d\n", len(node.Children))
		content += fmt.Sprintf("Expanded: %t\n", node.Expanded)
	}

	return tr.styles.Border.Render(content)
}
