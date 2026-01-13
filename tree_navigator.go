package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type TreeNavigator struct {
	model          *TreeModel
	app            interface{}
	postgresLoader *PostgresTreeLoader
}

func NewTreeNavigator(model *TreeModel) *TreeNavigator {
	return &TreeNavigator{model: model}
}

func (tn *TreeNavigator) SetPostgresLoader(loader *PostgresTreeLoader) {
	tn.postgresLoader = loader
}

func (tn *TreeNavigator) SetApp(app interface{}) {
	tn.app = app
}

func (tn *TreeNavigator) HandleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyUp:
		return tn.navigateUp()
	case tea.KeyDown:
		return tn.navigateDown()
	case tea.KeyLeft:
		return tn.navigateLeft()
	case tea.KeyRight:
		return tn.navigateRight()
	case tea.KeyEnter:
		return tn.navigateEnter()
	case tea.KeyBackspace:
		return tn.navigateBack()
	case tea.KeyHome:
		return tn.navigateHome()
	case tea.KeyEnd:
		return tn.navigateEnd()
	case tea.KeyPgUp:
		return tn.navigatePageUp()
	case tea.KeyPgDown:
		return tn.navigatePageDown()
	case tea.KeyCtrlD:
		return tn.viewTableData()
	case tea.KeyF5:
		return tn.refreshTree()
	case tea.KeyEscape:
		return tn.exitSearch()
	case tea.KeyCtrlQ:
		return tn.openQueryEditor()
	default:
		if tn.model.searchMode {
			return tn.handleSearchInput(msg)
		}
		return tn.model, nil
	}
}

func (tn *TreeNavigator) navigateUp() (tea.Model, tea.Cmd) {
	visibleNodes := tn.model.GetAllVisibleNodes()
	if len(visibleNodes) == 0 {
		return tn.model, nil
	}

	currentPos := tn.model.viewport.Position
	if currentPos > 0 {
		tn.model.viewport.Position--
		tn.model.SetSelectedNode(visibleNodes[currentPos-1])
	}

	return tn.model, nil
}

func (tn *TreeNavigator) navigateDown() (tea.Model, tea.Cmd) {
	visibleNodes := tn.model.GetAllVisibleNodes()
	if len(visibleNodes) == 0 {
		return tn.model, nil
	}

	currentPos := tn.model.viewport.Position
	if currentPos < len(visibleNodes)-1 {
		tn.model.viewport.Position++
		tn.model.SetSelectedNode(visibleNodes[currentPos+1])
	}

	return tn.model, nil
}

func (tn *TreeNavigator) navigateLeft() (tea.Model, tea.Cmd) {
	selected := tn.model.GetSelectedNode()
	if selected == nil {
		return tn.model, nil
	}

	if selected.HasChildren() && selected.Expanded {
		selected.Collapse()
	} else if selected.Parent != nil && selected.Parent.Level >= 0 {
		tn.selectNode(selected.Parent)
	}

	return tn.model, nil
}

func (tn *TreeNavigator) navigateRight() (tea.Model, tea.Cmd) {
	selected := tn.model.GetSelectedNode()
	if selected == nil {
		return tn.model, nil
	}

	if selected.HasChildren() && !selected.Expanded {
		selected.Expand()
	} else if !selected.HasChildren() && tn.postgresLoader != nil {
		// Try to load children on demand
		if err := tn.postgresLoader.LoadChildren(selected); err == nil && selected.HasChildren() {
			selected.Expand()
		}
	}

	return tn.model, nil
}

func (tn *TreeNavigator) navigateEnter() (tea.Model, tea.Cmd) {
	selected := tn.model.GetSelectedNode()
	if selected == nil {
		return tn.model, nil
	}

	if selected.HasChildren() && !selected.Expanded {
		selected.Expand()
		if len(selected.Children) > 0 {
			tn.selectNode(selected.Children[0])
		}
	}

	return tn.model, nil
}

func (tn *TreeNavigator) navigateBack() (tea.Model, tea.Cmd) {
	selected := tn.model.GetSelectedNode()
	if selected != nil && selected.Parent != nil && selected.Parent.Level >= 0 {
		tn.selectNode(selected.Parent)
	}

	return tn.model, nil
}

func (tn *TreeNavigator) navigateHome() (tea.Model, tea.Cmd) {
	if len(tn.model.root.Children) > 0 {
		tn.selectNode(tn.model.root.Children[0])
	}

	return tn.model, nil
}

func (tn *TreeNavigator) navigateEnd() (tea.Model, tea.Cmd) {
	visibleNodes := tn.model.GetAllVisibleNodes()
	if len(visibleNodes) > 0 {
		lastNode := visibleNodes[len(visibleNodes)-1]
		tn.selectNode(lastNode)
	}

	return tn.model, nil
}

func (tn *TreeNavigator) navigatePageUp() (tea.Model, tea.Cmd) {
	visibleNodes := tn.model.GetAllVisibleNodes()
	pageSize := tn.model.viewport.Height

	currentPos := tn.model.viewport.Position
	newPos := currentPos - pageSize
	if newPos < 0 {
		newPos = 0
	}

	tn.model.viewport.Position = newPos
	if newPos < len(visibleNodes) {
		tn.model.SetSelectedNode(visibleNodes[newPos])
	}

	return tn.model, nil
}

func (tn *TreeNavigator) navigatePageDown() (tea.Model, tea.Cmd) {
	visibleNodes := tn.model.GetAllVisibleNodes()
	pageSize := tn.model.viewport.Height

	currentPos := tn.model.viewport.Position
	newPos := currentPos + pageSize
	if newPos >= len(visibleNodes) {
		newPos = len(visibleNodes) - 1
	}

	tn.model.viewport.Position = newPos
	if newPos < len(visibleNodes) {
		tn.model.SetSelectedNode(visibleNodes[newPos])
	}

	return tn.model, nil
}

func (tn *TreeNavigator) startSearch() (tea.Model, tea.Cmd) {
	tn.model.searchMode = true
	tn.model.searchQuery = ""
	return tn.model, nil
}

func (tn *TreeNavigator) exitSearch() (tea.Model, tea.Cmd) {
	tn.model.searchMode = false
	tn.model.searchQuery = ""
	return tn.model, nil
}

func (tn *TreeNavigator) handleSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape:
		return tn.exitSearch()
	case tea.KeyEnter:
		return tn.model, tn.findNextMatch()
	case tea.KeyBackspace:
		if len(tn.model.searchQuery) > 0 {
			tn.model.searchQuery = tn.model.searchQuery[:len(tn.model.searchQuery)-1]
		}
	default:
		if len(msg.Runes) > 0 {
			tn.model.searchQuery += string(msg.Runes)
		}
	}

	return tn.model, tn.findNextMatch()
}

func (tn *TreeNavigator) findNextMatch() tea.Cmd {
	return func() tea.Msg {
		visibleNodes := tn.model.GetAllVisibleNodes()
		for _, node := range visibleNodes {
			if tn.nodeMatches(node) {
				return SearchResultMsg{node: node}
			}
		}
		return searchResultMsg{node: nil}
	}
}

func (tn *TreeNavigator) nodeMatches(node *TreeNode) bool {
	if tn.model.searchQuery == "" {
		return true
	}
	query := fmt.Sprintf("%s", tn.model.searchQuery)
	return fmt.Sprintf("%v", node.Name) == query ||
		fmt.Sprintf("%v", node.Path) == query
}

func (tn *TreeNavigator) selectNode(node *TreeNode) {
	tn.model.SetSelectedNode(node)

	visibleNodes := tn.model.GetAllVisibleNodes()
	for i, visibleNode := range visibleNodes {
		if visibleNode.ID == node.ID {
			if i < tn.model.viewport.Offset {
				tn.model.viewport.Offset = i
			} else if i >= tn.model.viewport.Offset+tn.model.viewport.Height {
				tn.model.viewport.Offset = i - tn.model.viewport.Height + 1
			}
			tn.model.viewport.Position = i - tn.model.viewport.Offset
			break
		}
	}
}

func (tn *TreeNavigator) insertItem() (tea.Model, tea.Cmd) {
	return tn.model, nil
}

func (tn *TreeNavigator) deleteItem() (tea.Model, tea.Cmd) {
	return tn.model, nil
}

func (tn *TreeNavigator) renameItem() (tea.Model, tea.Cmd) {
	return tn.model, nil
}

func (tn *TreeNavigator) refreshTree() (tea.Model, tea.Cmd) {
	return tn.model, nil
}

func (tn *TreeNavigator) copyItem() (tea.Model, tea.Cmd) {
	return tn.model, nil
}

func (tn *TreeNavigator) pasteItem() (tea.Model, tea.Cmd) {
	return tn.model, nil
}

func (tn *TreeNavigator) quit() (tea.Model, tea.Cmd) {
	return tn.model, tea.Quit
}

func (tn *TreeNavigator) openQueryEditor() (tea.Model, tea.Cmd) {
	return tn.model, func() tea.Msg {
		return FocusModeMsg{FocusQuery}
	}
}

func (tn *TreeNavigator) viewTableData() (tea.Model, tea.Cmd) {
	selected := tn.model.GetSelectedNode()
	if selected == nil || selected.Type != NodeTable {
		return tn.model, nil
	}

	parts := strings.Split(selected.Path, ".")
	if len(parts) != 3 {
		return tn.model, nil
	}

	database := parts[0]
	schema := parts[1]
	table := parts[2]

	return tn.model, func() tea.Msg {
		return LoadTableDataMsg{
			database: database,
			schema:   schema,
			table:    table,
		}
	}
}

type searchResultMsg struct {
	node *TreeNode
}
