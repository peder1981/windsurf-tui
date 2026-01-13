package main

import (
	"database/sql"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type TreeNode struct {
	ID       string
	Name     string
	Type     NodeType
	Path     string
	Children []*TreeNode
	Expanded bool
	Selected bool
	Metadata NodeMetadata
	Level    int
	Parent   *TreeNode
}

type NodeType int

const (
	NodeServer NodeType = iota
	NodeDatabase
	NodeSchema
	NodeTable
	NodeColumn
)

type NodeMetadata struct {
	Size         string
	Modified     string
	Count        int
	ContextType  string
	URI          string
	TableSize    string
	RowCount     int64
	DataType     string
	IsNullable   bool
	DefaultValue string
	PrimaryKey   bool
}

type TreeModel struct {
	db           *sql.DB
	root         *TreeNode
	selectedNode *TreeNode
	error        error
	searchMode   bool
	searchQuery  string
	viewport     Viewport
	focusPane    FocusPane
}

type Viewport struct {
	Offset   int
	Height   int
	Width    int
	Position int
}

type FocusPane int

const (
	PaneTree FocusPane = iota
	PaneDetails
	PanePreview
)

func NewTreeModel(db *sql.DB) TreeModel {
	return TreeModel{
		db:        db,
		root:      &TreeNode{Name: "Root", Type: NodeServer, Level: -1},
		viewport:  Viewport{Height: 20, Width: 80},
		focusPane: PaneTree,
	}
}

func (nt NodeType) String() string {
	switch nt {
	case NodeServer:
		return "Server"
	case NodeDatabase:
		return "Database"
	case NodeSchema:
		return "Schema"
	case NodeTable:
		return "Table"
	case NodeColumn:
		return "Column"
	default:
		return "Unknown"
	}
}

func (tn *TreeNode) GetIcon() string {
	switch tn.Type {
	case NodeServer:
		if tn.Expanded {
			return "🌐"
		}
		return "🖧"
	case NodeDatabase:
		if tn.Expanded {
			return "📂"
		}
		return "📁"
	case NodeSchema:
		if tn.Expanded {
			return "📂"
		}
		return "📁"
	case NodeTable:
		return "📊"
	case NodeColumn:
		return "🔹"
	default:
		return "❓"
	}
}

func (tn *TreeNode) GetDisplayString() string {
	icon := tn.GetIcon()

	var metadata string
	switch tn.Type {
	case NodeServer:
		metadata = fmt.Sprintf(" (%d databases)", tn.Metadata.Count)
	case NodeDatabase:
		metadata = fmt.Sprintf(" [%s]", tn.Metadata.Size)
	case NodeSchema:
		metadata = fmt.Sprintf(" (%d tables)", tn.Metadata.Count)
	case NodeTable:
		metadata = fmt.Sprintf(" %s", tn.Metadata.Size)
	case NodeColumn:
		metadata = fmt.Sprintf(" %s", tn.Metadata.DataType)
	}

	indent := strings.Repeat("  ", tn.Level+1)
	return fmt.Sprintf("%s%s %s%s", indent, icon, tn.Name, metadata)
}

func (tn *TreeNode) HasChildren() bool {
	return len(tn.Children) > 0
}

func (tn *TreeNode) IsLeaf() bool {
	return len(tn.Children) == 0
}

func (tn *TreeNode) ToggleExpanded() {
	if tn.HasChildren() {
		tn.Expanded = !tn.Expanded
	}
}

func (tn *TreeNode) Expand() {
	if tn.HasChildren() {
		tn.Expanded = true
	}
}

func (tn *TreeNode) Collapse() {
	tn.Expanded = false
}

func (tm *TreeModel) GetAllVisibleNodes() []*TreeNode {
	return tm.getVisibleNodes(tm.root, 0)
}

func (tm *TreeModel) getVisibleNodes(node *TreeNode, index int) []*TreeNode {
	var nodes []*TreeNode

	if node.Level >= 0 || node.Level == -1 {
		if node.Level >= 0 {
			nodes = append(nodes, node)
		}

		if node.Level == -1 || node.Expanded {
			for _, child := range node.Children {
				childNodes := tm.getVisibleNodes(child, index+len(nodes))
				nodes = append(nodes, childNodes...)
			}
		}
	}

	return nodes
}

func (tm *TreeModel) GetSelectedNode() *TreeNode {
	return tm.selectedNode
}

func (tm *TreeModel) SetSelectedNode(node *TreeNode) {
	if tm.selectedNode != nil {
		tm.selectedNode.Selected = false
	}
	tm.selectedNode = node
	if node != nil {
		node.Selected = true
		tm.ensureParentsExpanded(node)
	}
}

func (tm *TreeModel) ensureParentsExpanded(node *TreeNode) {
	if node == nil || node.Parent == nil {
		return
	}
	parent := node.Parent
	parent.Expanded = true
	tm.ensureParentsExpanded(parent)
}

func (tm *TreeModel) FindNodeByID(id string) *TreeNode {
	return tm.findNodeByID(tm.root, id)
}

func (tm *TreeModel) findNodeByID(node *TreeNode, id string) *TreeNode {
	if node.ID == id {
		return node
	}

	for _, child := range node.Children {
		if found := tm.findNodeByID(child, id); found != nil {
			return found
		}
	}

	return nil
}

func (tm *TreeModel) GetNodeCount() int {
	return tm.countNodes(tm.root)
}

func (tm *TreeModel) countNodes(node *TreeNode) int {
	count := 1
	for _, child := range node.Children {
		count += tm.countNodes(child)
	}
	return count
}

func (tm *TreeModel) Init() tea.Cmd {
	return nil
}

func (tm *TreeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return tm, nil
}

func (tm *TreeModel) View() string {
	return "TreeModel View"
}
