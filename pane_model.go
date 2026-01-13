package main

import (
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

var hiddenDataColumns = map[string]bool{
	"__ctid": true,
}

type PaneType int

const (
	PaneDatabases PaneType = iota
	PaneSchemas
	PaneTables
	PaneData
)

type PaneState struct {
	PaneType       PaneType
	Nodes          []*TreeNode
	SelectedIdx    int
	Offset         int
	ParentNode     *TreeNode
	Expanded       bool
	ViewportHeight int
}

type PaneModel struct {
	panes             [4]*PaneState
	focus             PaneType
	data              []map[string]interface{}
	dataColumns       []string
	dataRowOffset     int
	dataColOffset     int
	dataViewportRows  int
	dataViewportWidth int
	dataSelectedRow   int
	dataSelectedCol   int
	dataDatabase      string
	dataSchema        string
	dataTable         string
}

func NewPaneModel() *PaneModel {
	return &PaneModel{
		panes: [4]*PaneState{
			{PaneType: PaneDatabases, SelectedIdx: 0, Offset: 0, Expanded: true},
			{PaneType: PaneSchemas, SelectedIdx: 0, Offset: 0, Expanded: false},
			{PaneType: PaneTables, SelectedIdx: 0, Offset: 0, Expanded: false},
			{PaneType: PaneData, SelectedIdx: 0, Offset: 0, Expanded: false},
		},
		focus:             PaneDatabases,
		data:              make([]map[string]interface{}, 0),
		dataColumns:       make([]string, 0),
		dataViewportRows:  10,
		dataViewportWidth: 80,
	}
}

func (pm *PaneModel) Init() tea.Cmd {
	return nil
}

func (pm *PaneModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return pm, nil
}

func (pm *PaneModel) View() string {
	return "PaneModel View"
}

func (pm *PaneModel) SetPaneNodes(paneType PaneType, nodes []*TreeNode, parentNode *TreeNode) {
	pane := pm.panes[paneType]
	pane.Nodes = nodes
	pane.ParentNode = parentNode
	pane.SelectedIdx = 0
	pane.Offset = 0
	pane.Expanded = len(nodes) > 0

	for _, node := range nodes {
		node.Parent = parentNode
	}
}

func (pm *PaneModel) GetSelectedNode(paneType PaneType) *TreeNode {
	pane := pm.panes[paneType]
	if pane.SelectedIdx >= 0 && pane.SelectedIdx < len(pane.Nodes) {
		return pane.Nodes[pane.SelectedIdx]
	}
	return nil
}

func (pm *PaneModel) SetFocus(paneType PaneType) {
	pm.focus = paneType
}

func (pm *PaneModel) GetFocus() PaneType {
	return pm.focus
}

func (pm *PaneModel) MoveSelection(direction int) {
	pane := pm.panes[pm.focus]
	newIdx := pane.SelectedIdx + direction

	if newIdx >= 0 && newIdx < len(pane.Nodes) {
		pane.SelectedIdx = newIdx

		// Adjust offset if needed
		visibleRows := pane.ViewportHeight
		if visibleRows <= 0 {
			visibleRows = 10
		}
		if pane.SelectedIdx < pane.Offset {
			pane.Offset = pane.SelectedIdx
		} else if pane.SelectedIdx >= pane.Offset+visibleRows {
			pane.Offset = pane.SelectedIdx - visibleRows + 1
		}
	}
}

func (pm *PaneModel) GetPane(paneType PaneType) *PaneState {
	return pm.panes[paneType]
}

func (pm *PaneModel) SetData(data []map[string]interface{}) {
	pm.data = data
	pm.dataRowOffset = 0
	pm.dataColOffset = 0
	pm.dataColumns = pm.buildDataColumns()
	pm.dataSelectedRow = 0
	pm.dataSelectedCol = 0
}

func (pm *PaneModel) GetData() []map[string]interface{} {
	return pm.data
}

func (pm *PaneModel) buildDataColumns() []string {
	if len(pm.data) == 0 {
		return nil
	}

	columns := make([]string, 0, len(pm.data[0]))
	for col := range pm.data[0] {
		if hiddenDataColumns[strings.ToLower(col)] || hiddenDataColumns[col] {
			continue
		}
		columns = append(columns, col)
	}
	sort.Strings(columns)
	return columns
}

func (pm *PaneModel) GetDataColumns() []string {
	return pm.dataColumns
}

func (pm *PaneModel) SetPaneViewport(paneType PaneType, height int) {
	if height < 1 {
		height = 1
	}
	pm.panes[paneType].ViewportHeight = height
}

func (pm *PaneModel) SetDataViewport(bodyHeight int) {
	rows := bodyHeight - 3
	if rows < 1 {
		rows = 1
	}
	pm.dataViewportRows = rows
}

func (pm *PaneModel) SetDataViewportWidth(width int) {
	if width < 20 {
		width = 20
	}
	pm.dataViewportWidth = width
}

func (pm *PaneModel) GetDataViewportRows() int {
	if pm.dataViewportRows < 1 {
		return 1
	}
	return pm.dataViewportRows
}

func (pm *PaneModel) GetDataViewportWidth() int {
	if pm.dataViewportWidth < 20 {
		return 20
	}
	return pm.dataViewportWidth
}

func (pm *PaneModel) GetDataRowOffset() int {
	if pm.dataRowOffset < 0 {
		return 0
	}
	return pm.dataRowOffset
}

func (pm *PaneModel) GetDataColOffset() int {
	if pm.dataColOffset < 0 {
		return 0
	}
	return pm.dataColOffset
}

func (pm *PaneModel) ScrollDataRows(delta int) {
	if len(pm.data) == 0 {
		return
	}
	pm.dataRowOffset += delta
	maxOffset := len(pm.data) - pm.GetDataViewportRows()
	if maxOffset < 0 {
		maxOffset = 0
	}
	if pm.dataRowOffset < 0 {
		pm.dataRowOffset = 0
	} else if pm.dataRowOffset > maxOffset {
		pm.dataRowOffset = maxOffset
	}
}

func (pm *PaneModel) ScrollDataCols(delta int) {
	if len(pm.dataColumns) == 0 {
		return
	}
	pm.dataColOffset += delta
	maxOffset := len(pm.dataColumns) - pm.visibleColumnCount()
	if maxOffset < 0 {
		maxOffset = 0
	}
	if pm.dataColOffset < 0 {
		pm.dataColOffset = 0
	} else if pm.dataColOffset > maxOffset {
		pm.dataColOffset = maxOffset
	}
}

func (pm *PaneModel) visibleColumnCount() int {
	maxWidth := pm.GetDataViewportWidth()
	if maxWidth < 20 {
		maxWidth = 20
	}
	widthRemaining := maxWidth
	count := 0
	for _, col := range pm.dataColumns {
		colWidth := len(col)
		if colWidth < 8 {
			colWidth = 8
		}
		if colWidth > 30 {
			colWidth = 30
		}
		space := 1
		if count == 0 {
			space = 0
		}
		if widthRemaining-colWidth-space < 0 {
			break
		}
		widthRemaining -= colWidth + space
		count++
	}
	if count == 0 && len(pm.dataColumns) > 0 {
		return 1
	}
	return count
}

func (pm *PaneModel) MoveDataSelection(rowDelta, colDelta int) {
	if len(pm.data) == 0 || len(pm.dataColumns) == 0 {
		return
	}

	pm.dataSelectedRow += rowDelta
	if pm.dataSelectedRow < 0 {
		pm.dataSelectedRow = 0
	} else if pm.dataSelectedRow >= len(pm.data) {
		pm.dataSelectedRow = len(pm.data) - 1
	}

	pm.dataSelectedCol += colDelta
	if pm.dataSelectedCol < 0 {
		pm.dataSelectedCol = 0
	} else if pm.dataSelectedCol >= len(pm.dataColumns) {
		pm.dataSelectedCol = len(pm.dataColumns) - 1
	}

	pm.ensureDataSelectionVisible()
}

func (pm *PaneModel) ensureDataSelectionVisible() {
	visibleRows := pm.GetDataViewportRows()
	if pm.dataSelectedRow < pm.dataRowOffset {
		pm.dataRowOffset = pm.dataSelectedRow
	} else if pm.dataSelectedRow >= pm.dataRowOffset+visibleRows {
		pm.dataRowOffset = pm.dataSelectedRow - visibleRows + 1
	}

	visibleCols := pm.visibleColumnCount()
	if visibleCols < 1 {
		visibleCols = 1
	}
	if pm.dataSelectedCol < pm.dataColOffset {
		pm.dataColOffset = pm.dataSelectedCol
	} else if pm.dataSelectedCol >= pm.dataColOffset+visibleCols {
		pm.dataColOffset = pm.dataSelectedCol - visibleCols + 1
	}

	if pm.dataRowOffset < 0 {
		pm.dataRowOffset = 0
	}
	if pm.dataColOffset < 0 {
		pm.dataColOffset = 0
	}
}

func (pm *PaneModel) SetDataSelection(row, col int) {
	if row < 0 {
		row = 0
	}
	if row >= len(pm.data) {
		row = len(pm.data) - 1
		if row < 0 {
			row = 0
		}
	}
	if col < 0 {
		col = 0
	}
	if col >= len(pm.dataColumns) {
		col = len(pm.dataColumns) - 1
		if col < 0 {
			col = 0
		}
	}
	pm.dataSelectedRow = row
	pm.dataSelectedCol = col
	pm.ensureDataSelectionVisible()
}

func (pm *PaneModel) GetSelectedDataCell() (row map[string]interface{}, column string, value interface{}) {
	if len(pm.data) == 0 || len(pm.dataColumns) == 0 {
		return nil, "", nil
	}

	if pm.dataSelectedRow < 0 || pm.dataSelectedRow >= len(pm.data) {
		return nil, "", nil
	}
	if pm.dataSelectedCol < 0 || pm.dataSelectedCol >= len(pm.dataColumns) {
		return nil, "", nil
	}

	row = pm.data[pm.dataSelectedRow]
	column = pm.dataColumns[pm.dataSelectedCol]
	value = row[column]
	return
}

func (pm *PaneModel) SetDataContext(database, schema, table string) {
	pm.dataDatabase = database
	pm.dataSchema = schema
	pm.dataTable = table
}

func (pm *PaneModel) GetDataContext() (database, schema, table string) {
	return pm.dataDatabase, pm.dataSchema, pm.dataTable
}

func (pm *PaneModel) HasDataContext() bool {
	return pm.dataDatabase != "" && pm.dataSchema != "" && pm.dataTable != ""
}

func (pm *PaneModel) GetSelectedDataRowIndex() int {
	return pm.dataSelectedRow
}

func (pm *PaneModel) GetSelectedDataColIndex() int {
	return pm.dataSelectedCol
}

func (pm *PaneModel) GetSelectedDataColumnName() string {
	if pm.dataSelectedCol >= 0 && pm.dataSelectedCol < len(pm.dataColumns) {
		return pm.dataColumns[pm.dataSelectedCol]
	}
	return ""
}

func (pm *PaneModel) GetDataRowCount() int {
	return len(pm.data)
}

func (pm *PaneModel) GetDataColCount() int {
	return len(pm.dataColumns)
}

func (pm *PaneModel) GetRowCTID(rowIdx int) string {
	if rowIdx < 0 || rowIdx >= len(pm.data) {
		return ""
	}
	row := pm.data[rowIdx]
	if val, ok := row["__ctid"]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}

func (pm *PaneModel) GetSelectedRowCTID() string {
	return pm.GetRowCTID(pm.dataSelectedRow)
}
