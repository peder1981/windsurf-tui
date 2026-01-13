package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type PaneNavigator struct {
	paneModel      *PaneModel
	postgresLoader *PostgresTreeLoader
}

func NewPaneNavigator(paneModel *PaneModel) *PaneNavigator {
	return &PaneNavigator{
		paneModel: paneModel,
	}
}

func (pn *PaneNavigator) SetPostgresLoader(loader *PostgresTreeLoader) {
	pn.postgresLoader = loader
}

func (pn *PaneNavigator) HandleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyUp:
		pn.paneModel.MoveSelection(-1)
		return pn.paneModel, nil
	case tea.KeyDown:
		pn.paneModel.MoveSelection(1)
		return pn.paneModel, nil
	case tea.KeyLeft:
		pn.navigateLeft()
		return pn.paneModel, nil
	case tea.KeyRight:
		return pn.navigateRight()
	case tea.KeyEnter:
		return pn.navigateRight()
	case tea.KeyTab:
		pn.cycleFocus()
		return pn.paneModel, nil
	case tea.KeyCtrlQ:
		return pn.openQueryEditor()
	case tea.KeyCtrlX:
		return pn.paneModel, tea.Quit
	case tea.KeyEscape:
		return pn.navigateUp()
	default:
		return pn.paneModel, nil
	}
}

func (pn *PaneNavigator) navigateLeft() {
	currentFocus := pn.paneModel.GetFocus()
	if currentFocus > PaneDatabases {
		pn.paneModel.SetFocus(currentFocus - 1)
	}
}

func (pn *PaneNavigator) navigateUp() (tea.Model, tea.Cmd) {
	currentFocus := pn.paneModel.GetFocus()

	// If at databases pane, quit
	if currentFocus == PaneDatabases {
		return pn.paneModel, tea.Quit
	}

	// Otherwise, move up to previous pane
	pn.navigateLeft()
	return pn.paneModel, nil
}

func (pn *PaneNavigator) navigateRight() (tea.Model, tea.Cmd) {
	currentFocus := pn.paneModel.GetFocus()
	selectedNode := pn.paneModel.GetSelectedNode(currentFocus)

	if selectedNode == nil {
		return pn.paneModel, nil
	}

	// If at Tables pane and selected node is a table, load data
	if currentFocus == PaneTables && selectedNode.Type == NodeTable {
		return pn.loadTableData(selectedNode)
	}

	// Try to load children if not already loaded
	if len(selectedNode.Children) == 0 && pn.postgresLoader != nil {
		if err := pn.postgresLoader.LoadChildren(selectedNode); err != nil {
			return pn.paneModel, nil
		}
	}

	// If node has children, move to next pane
	if len(selectedNode.Children) > 0 && currentFocus < PaneData {
		nextPane := currentFocus + 1
		pn.paneModel.SetPaneNodes(nextPane, selectedNode.Children, selectedNode)
		pn.paneModel.SetFocus(nextPane)
	}

	return pn.paneModel, nil
}

func (pn *PaneNavigator) loadTableData(tableNode *TreeNode) (tea.Model, tea.Cmd) {
	path := pn.buildPath(tableNode)
	parts := splitPath(path)
	if len(parts) < 3 {
		return pn.paneModel, nil
	}

	parts = parts[len(parts)-3:]
	database := parts[0]
	schema := parts[1]
	table := parts[2]

	return pn.paneModel, func() tea.Msg {
		return LoadTableDataMsg{
			database: database,
			schema:   schema,
			table:    table,
		}
	}
}

func (pn *PaneNavigator) cycleFocus() {
	currentFocus := pn.paneModel.GetFocus()
	nextFocus := currentFocus + 1
	if nextFocus > PaneData {
		nextFocus = PaneDatabases
	}
	pn.paneModel.SetFocus(nextFocus)
}

func (pn *PaneNavigator) viewTableData() (tea.Model, tea.Cmd) {
	selectedNode := pn.paneModel.GetSelectedNode(PaneTables)
	if selectedNode == nil || selectedNode.Type != NodeTable {
		return pn.paneModel, nil
	}

	// Build path from parent chain
	path := pn.buildPath(selectedNode)
	parts := splitPath(path)
	if len(parts) != 3 {
		return pn.paneModel, nil
	}

	database := parts[0]
	schema := parts[1]
	table := parts[2]

	return pn.paneModel, func() tea.Msg {
		return LoadTableDataMsg{
			database: database,
			schema:   schema,
			table:    table,
		}
	}
}

func (pn *PaneNavigator) openQueryEditor() (tea.Model, tea.Cmd) {
	return pn.paneModel, func() tea.Msg {
		return FocusModeMsg{FocusQuery}
	}
}

func (pn *PaneNavigator) buildPath(node *TreeNode) string {
	if node == nil {
		return ""
	}

	var parts []string
	current := node

	for current != nil {
		parts = append([]string{current.Name}, parts...)
		current = current.Parent
	}

	return strings.Join(parts, ".")
}

func splitPath(path string) []string {
	parts := make([]string, 0)
	current := ""
	for _, c := range path {
		if c == '.' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}
