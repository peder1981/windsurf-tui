package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type XTreeGoldApp struct {
	tree              *TreeModel
	navigator         *TreeNavigator
	renderer          *TreeRenderer
	paneModel         *PaneModel
	paneNavigator     *PaneNavigator
	paneRenderer      *PaneRenderer
	connectionMgr     *ConnectionManager
	postgresLoader    *PostgresTreeLoader
	styles            AppStyles
	initialized       bool
	currentServer     string
	currentConnection *ConnectionInfo
	width             int
	height            int
	queryEditor       *QueryEditor
	dataViewer        *DataViewer
	dataEditor        *TextInput
	dataEditMode      DataEditMode
	dataEditRow       int
	dataEditColumn    string
	focusMode         FocusMode
	connectionDialog  *ConnectionDialog
	addConnectionForm *AddConnectionForm
	connectionStep    ConnectionStep
}

type AppStyles struct {
	Header  lipgloss.Style
	Body    lipgloss.Style
	Footer  lipgloss.Style
	Error   lipgloss.Style
	Success lipgloss.Style
}

type FocusMode int

const (
	FocusTree FocusMode = iota
	FocusQuery
	FocusData
	FocusConnectionDialog
	FocusAddConnectionForm
)

type DataEditMode int

const (
	DataEditNone DataEditMode = iota
	DataEditUpdateCell
	DataEditInsertRow
)

type ConnectionStep int

const (
	StepSelectConnection ConnectionStep = iota
	StepAddConnection
	StepConnected
)

type ErrMsg struct {
	err error
}

func (e ErrMsg) Error() string {
	return e.err.Error()
}

type TreeLoadedMsg struct {
	tree *TreeNode
}

type ExecuteQueryMsg struct {
	query string
}

type LoadTableDataMsg struct {
	database string
	schema   string
	table    string
	rowIndex int
	colIndex int
}

type UpdateCellMsg struct {
	database string
	schema   string
	table    string
	ctid     string
	column   string
	value    interface{}
	rowIndex int
	colIndex int
}

type InsertRowMsg struct {
	database string
	schema   string
	table    string
	values   map[string]interface{}
	rowIndex int
	colIndex int
}

type DeleteRowMsg struct {
	database string
	schema   string
	table    string
	ctid     string
	rowIndex int
	colIndex int
}

type SearchResultMsg struct {
	node *TreeNode
}

type FocusModeMsg struct {
	focusMode FocusMode
}

func NewXTreeGoldApp() (*XTreeGoldApp, error) {
	connMgr, err := NewConnectionManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create connection manager: %w", err)
	}

	tree := NewTreeModel(nil)
	paneModel := NewPaneModel()
	app := &XTreeGoldApp{
		tree:           &tree,
		navigator:      NewTreeNavigator(&tree),
		renderer:       NewTreeRenderer(),
		paneModel:      paneModel,
		paneNavigator:  NewPaneNavigator(paneModel),
		paneRenderer:   NewPaneRenderer(),
		connectionMgr:  connMgr,
		connectionStep: StepSelectConnection,
		width:          80,
		height:         24,
		queryEditor:    NewQueryEditor(),
		dataViewer:     NewDataViewer(),
		dataEditor:     NewTextInput(),
		styles: AppStyles{
			Header:  lipgloss.NewStyle().Background(lipgloss.Color("#1a1a1a")).Foreground(lipgloss.Color("#FFD700")).Bold(true).Padding(0, 1),
			Body:    lipgloss.NewStyle().Background(lipgloss.Color("#000000")).Foreground(lipgloss.Color("#FFFFFF")),
			Footer:  lipgloss.NewStyle().Background(lipgloss.Color("#1a1a1a")).Foreground(lipgloss.Color("#808080")).Padding(0, 1),
			Error:   lipgloss.NewStyle().Background(lipgloss.Color("#FF0000")).Foreground(lipgloss.Color("#FFFFFF")).Padding(0, 1),
			Success: lipgloss.NewStyle().Background(lipgloss.Color("#00FF00")).Foreground(lipgloss.Color("#000000")).Padding(0, 1),
		},
		focusMode: FocusConnectionDialog,
	}

	return app, nil
}

func (app *XTreeGoldApp) handleDataEditInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if app.dataEditor == nil {
		app.cancelDataEdit()
		return app, nil
	}

	switch msg.Type {
	case tea.KeyEscape:
		app.cancelDataEdit()
		return app, nil
	case tea.KeyEnter:
		return app.commitDataEdit()
	}

	if app.dataEditor.HandleKey(msg) {
		return app, nil
	}

	return app, nil
}

func (app *XTreeGoldApp) beginCellEdit() {
	if app.dataEditor == nil || !app.paneModel.HasDataContext() {
		return
	}

	rowIdx := app.paneModel.GetSelectedDataRowIndex()
	colName := app.paneModel.GetSelectedDataColumnName()
	if rowIdx < 0 || colName == "" {
		return
	}

	_, _, value := app.paneModel.GetSelectedDataCell()
	app.dataEditMode = DataEditUpdateCell
	app.dataEditRow = rowIdx
	app.dataEditColumn = colName
	app.dataEditor.SetWidth(max(app.width-4, 20))
	if value == nil {
		app.dataEditor.SetValue("")
	} else {
		app.dataEditor.SetValue(fmt.Sprintf("%v", value))
	}
	app.dataEditor.SetPlaceholder("")
}

func (app *XTreeGoldApp) beginInsertRow() {
	if app.dataEditor == nil || !app.paneModel.HasDataContext() {
		return
	}

	app.dataEditMode = DataEditInsertRow
	app.dataEditRow = -1
	app.dataEditColumn = ""
	app.dataEditor.SetWidth(max(app.width-4, 30))
	app.dataEditor.SetValue("")
	app.dataEditor.SetPlaceholder("coluna=valor, outra=valor2")
}

func (app *XTreeGoldApp) requestDeleteRow() tea.Cmd {
	if !app.paneModel.HasDataContext() {
		return nil
	}
	rowIdx := app.paneModel.GetSelectedDataRowIndex()
	ctid := app.paneModel.GetSelectedRowCTID()
	if ctid == "" {
		return nil
	}
	db, schema, table := app.paneModel.GetDataContext()
	colIdx := app.paneModel.GetSelectedDataColIndex()
	targetRow := rowIdx - 1
	return func() tea.Msg {
		return DeleteRowMsg{
			database: db,
			schema:   schema,
			table:    table,
			ctid:     ctid,
			rowIndex: targetRow,
			colIndex: colIdx,
		}
	}
}

func (app *XTreeGoldApp) commitDataEdit() (tea.Model, tea.Cmd) {
	switch app.dataEditMode {
	case DataEditUpdateCell:
		return app.commitUpdateCell()
	case DataEditInsertRow:
		return app.commitInsertRow()
	default:
		app.cancelDataEdit()
		return app, nil
	}
}

func (app *XTreeGoldApp) commitUpdateCell() (tea.Model, tea.Cmd) {
	if !app.paneModel.HasDataContext() {
		app.cancelDataEdit()
		return app, nil
	}

	rowIdx := app.dataEditRow
	if rowIdx < 0 {
		rowIdx = app.paneModel.GetSelectedDataRowIndex()
	}
	colName := app.dataEditColumn
	if colName == "" {
		colName = app.paneModel.GetSelectedDataColumnName()
	}
	if colName == "" {
		app.cancelDataEdit()
		return app, nil
	}

	ctid := app.paneModel.GetRowCTID(rowIdx)
	if ctid == "" {
		app.cancelDataEdit()
		return app, nil
	}

	currentValue := app.getCurrentCellValue(rowIdx, colName)
	converted := app.convertInputValue(app.dataEditor.Value(), currentValue)
	db, schema, table := app.paneModel.GetDataContext()
	colIdx := app.paneModel.GetColumnIndexByName(colName)
	app.cancelDataEdit()

	return app, func() tea.Msg {
		return UpdateCellMsg{
			database: db,
			schema:   schema,
			table:    table,
			ctid:     ctid,
			column:   colName,
			value:    converted,
			rowIndex: rowIdx,
			colIndex: colIdx,
		}
	}
}

func (app *XTreeGoldApp) commitInsertRow() (tea.Model, tea.Cmd) {
	if !app.paneModel.HasDataContext() {
		app.cancelDataEdit()
		return app, nil
	}

	raw := strings.TrimSpace(app.dataEditor.Value())
	if raw == "" {
		return app, nil
	}

	pairs := ParseKeyValueInput(raw)
	if len(pairs) == 0 {
		return app, nil
	}

	values := make(map[string]interface{}, len(pairs))
	for col, val := range pairs {
		values[col] = app.convertInputValue(val, nil)
	}

	db, schema, table := app.paneModel.GetDataContext()
	rowIdx := app.paneModel.GetSelectedDataRowIndex()
	app.cancelDataEdit()

	return app, func() tea.Msg {
		return InsertRowMsg{
			database: db,
			schema:   schema,
			table:    table,
			values:   values,
			rowIndex: rowIdx,
			colIndex: 0,
		}
	}
}

func (app *XTreeGoldApp) cancelDataEdit() {
	app.dataEditMode = DataEditNone
	app.dataEditRow = -1
	app.dataEditColumn = ""
	if app.dataEditor != nil {
		app.dataEditor.Reset()
	}
}

func (app *XTreeGoldApp) dataEditPrompt() string {
	switch app.dataEditMode {
	case DataEditUpdateCell:
		return fmt.Sprintf("Editar %s (linha %d)", app.dataEditColumn, app.dataEditRow+1)
	case DataEditInsertRow:
		return "Inserir linha (use formato coluna=valor, ...)"
	default:
		return ""
	}
}

func (app *XTreeGoldApp) convertInputValue(input string, current interface{}) interface{} {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" || strings.EqualFold(trimmed, "null") {
		return nil
	}

	if strings.EqualFold(trimmed, "true") {
		return true
	}
	if strings.EqualFold(trimmed, "false") {
		return false
	}

	if i, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(trimmed, 64); err == nil {
		return f
	}

	timeFormats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range timeFormats {
		if t, err := time.Parse(layout, trimmed); err == nil {
			return t
		}
	}

	return trimmed
}

func (app *XTreeGoldApp) getCurrentCellValue(rowIdx int, column string) interface{} {
	data := app.paneModel.GetData()
	if rowIdx < 0 || rowIdx >= len(data) {
		return nil
	}
	if column == "" {
		column = app.paneModel.GetSelectedDataColumnName()
	}
	if column == "" {
		return nil
	}
	row := data[rowIdx]
	return row[column]
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (app *XTreeGoldApp) Init() tea.Cmd {
	app.connectionDialog = NewConnectionDialog(app.connectionMgr)
	app.addConnectionForm = NewAddConnectionForm()
	return nil
}

func (app *XTreeGoldApp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		app.width = msg.Width
		app.height = msg.Height
		if app.focusMode == FocusTree {
			return app, nil
		}
		return app, nil
	case tea.KeyMsg:
		if app.focusMode == FocusConnectionDialog {
			return app.handleConnectionDialog(msg)
		} else if app.focusMode == FocusAddConnectionForm {
			return app.handleAddConnectionForm(msg)
		} else if app.focusMode == FocusTree && app.paneNavigator != nil {
			_, cmd := app.paneNavigator.HandleKeyMsg(msg)
			return app, cmd
		} else if app.focusMode == FocusQuery {
			return app.handleQueryInput(msg)
		} else if app.focusMode == FocusData {
			return app.handleDataView(msg)
		}

		return app, nil
	case ErrMsg:
		app.tree.error = msg.err
		app.connectionStep = StepSelectConnection
		return app, nil
	case TreeLoadedMsg:
		app.tree.root = msg.tree
		app.tree.SetSelectedNode(nil)

		// Populate pane model with databases
		if len(msg.tree.Children) > 0 && len(msg.tree.Children[0].Children) > 0 {
			databases := msg.tree.Children[0].Children
			app.paneModel.SetPaneNodes(PaneDatabases, databases, msg.tree.Children[0])
			app.tree.SetSelectedNode(msg.tree.Children[0])
		}

		app.initialized = true
		app.connectionStep = StepConnected
		return app, nil
	case SearchResultMsg:
		if msg.node != nil {
			app.navigator.selectNode(msg.node)
		}
		return app, nil
	case ExecuteQueryMsg:
		if app.postgresLoader != nil {
			results, err := app.postgresLoader.ExecuteQuery(msg.query)
			if err != nil {
				app.tree.error = err
				return app, nil
			}
			app.dataViewer.SetResults(results)
			app.focusMode = FocusData
		}
		return app, nil
	case LoadTableDataMsg:
		if app.postgresLoader != nil {
			results, err := app.postgresLoader.GetTableData(msg.database, msg.schema, msg.table, 100, 0)
			if err != nil {
				app.tree.error = err
				return app, nil
			}
			app.paneModel.SetData(results)
			app.paneModel.SetDataContext(msg.database, msg.schema, msg.table)
			app.paneModel.SetFocus(PaneData)
			app.paneModel.SetDataSelection(msg.rowIndex, msg.colIndex)
			app.focusMode = FocusData
		}
		return app, nil
	case UpdateCellMsg:
		if app.postgresLoader != nil {
			if err := app.postgresLoader.UpdateCell(msg.database, msg.schema, msg.table, msg.column, msg.ctid, msg.value); err != nil {
				app.tree.error = err
				return app, nil
			}
			return app, func() tea.Msg {
				return LoadTableDataMsg{
					database: msg.database,
					schema:   msg.schema,
					table:    msg.table,
					rowIndex: msg.rowIndex,
					colIndex: msg.colIndex,
				}
			}
		}
		return app, nil
	case InsertRowMsg:
		if app.postgresLoader != nil {
			if err := app.postgresLoader.InsertRow(msg.database, msg.schema, msg.table, msg.values); err != nil {
				app.tree.error = err
				return app, nil
			}
			return app, func() tea.Msg {
				return LoadTableDataMsg{
					database: msg.database,
					schema:   msg.schema,
					table:    msg.table,
					rowIndex: msg.rowIndex,
					colIndex: msg.colIndex,
				}
			}
		}
		return app, nil
	case DeleteRowMsg:
		if app.postgresLoader != nil {
			if err := app.postgresLoader.DeleteRow(msg.database, msg.schema, msg.table, msg.ctid); err != nil {
				app.tree.error = err
				return app, nil
			}
			targetRow := msg.rowIndex
			if targetRow < 0 {
				targetRow = 0
			}
			return app, func() tea.Msg {
				return LoadTableDataMsg{
					database: msg.database,
					schema:   msg.schema,
					table:    msg.table,
					rowIndex: targetRow,
					colIndex: msg.colIndex,
				}
			}
		}
		return app, nil
	case FocusModeMsg:
		app.focusMode = msg.focusMode
		return app, nil
	default:
		return app, nil
	}
}

func (app *XTreeGoldApp) handleConnectionDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	dialog := app.connectionDialog
	model, cmd := dialog.Update(msg)
	app.connectionDialog = model.(*ConnectionDialog)

	if dialog.IsConfirmed() {
		if dialog.ShouldAddNewConnection() {
			app.focusMode = FocusAddConnectionForm
			app.connectionStep = StepAddConnection
		} else if conn := dialog.GetSelectedConnection(); conn != nil {
			db, err := app.connectionMgr.Connect(conn)
			if err != nil {
				return app, func() tea.Msg {
					return ErrMsg{fmt.Errorf("connection failed: %w", err)}
				}
			}
			app.currentServer = conn.Name
			app.currentConnection = conn
			tree := NewTreeModel(db)
			app.tree = &tree
			app.navigator = NewTreeNavigator(app.tree)
			app.postgresLoader = NewPostgresTreeLoader(db, conn)
			app.navigator.SetPostgresLoader(app.postgresLoader)
			app.paneNavigator.SetPostgresLoader(app.postgresLoader)
			app.focusMode = FocusTree
			app.connectionStep = StepConnected
			app.initialized = false
			return app, app.postgresLoader.LoadTreeAsync(app.currentServer)
		} else {
			return app, tea.Quit
		}
	}

	return app, cmd
}

func (app *XTreeGoldApp) handleAddConnectionForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	form := app.addConnectionForm
	model, cmd := form.Update(msg)
	app.addConnectionForm = model.(*AddConnectionForm)

	if form.IsConfirmed() {
		if form.IsCancelled() {
			app.focusMode = FocusConnectionDialog
			app.connectionStep = StepSelectConnection
		} else if conn := form.GetConnectionInfo(); conn != nil {
			if err := app.connectionMgr.SaveConnection(conn); err != nil {
				return app, func() tea.Msg {
					return ErrMsg{fmt.Errorf("failed to save connection: %w", err)}
				}
			}
			db, err := app.connectionMgr.Connect(conn)
			if err != nil {
				return app, func() tea.Msg {
					return ErrMsg{fmt.Errorf("connection failed: %w", err)}
				}
			}
			app.currentServer = conn.Name
			app.currentConnection = conn
			tree := NewTreeModel(db)
			app.tree = &tree
			app.navigator = NewTreeNavigator(app.tree)
			app.postgresLoader = NewPostgresTreeLoader(db, conn)
			app.navigator.SetPostgresLoader(app.postgresLoader)
			app.paneNavigator.SetPostgresLoader(app.postgresLoader)
			app.focusMode = FocusTree
			app.connectionStep = StepConnected
			app.initialized = false
			return app, app.postgresLoader.LoadTreeAsync(app.currentServer)
		}
	}

	return app, cmd
}

func (app *XTreeGoldApp) handleQueryInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape:
		app.focusMode = FocusTree
		return app, nil
	case tea.KeyEnter:
		query := app.queryEditor.GetValue()
		if query != "" {
			return app, func() tea.Msg {
				return ExecuteQueryMsg{query: query}
			}
		}
		return app, nil
	default:
		model, cmd := app.queryEditor.Update(msg)
		app.queryEditor = model.(*QueryEditor)
		return app, cmd
	}
}

func (app *XTreeGoldApp) OpenQueryEditor() tea.Cmd {
	app.focusMode = FocusQuery
	return nil
}

func (app *XTreeGoldApp) handleDataView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if app.dataEditMode != DataEditNone {
		return app.handleDataEditInput(msg)
	}

	switch msg.Type {
	case tea.KeyEscape:
		app.focusMode = FocusTree
		return app, nil
	case tea.KeyCtrlQ:
		app.focusMode = FocusQuery
		return app, nil
	case tea.KeyUp:
		app.paneModel.MoveDataSelection(-1, 0)
		return app, nil
	case tea.KeyDown:
		app.paneModel.MoveDataSelection(1, 0)
		return app, nil
	case tea.KeyLeft:
		app.paneModel.MoveDataSelection(0, -1)
		return app, nil
	case tea.KeyRight:
		app.paneModel.MoveDataSelection(0, 1)
		return app, nil
	case tea.KeyPgUp:
		app.paneModel.MoveDataSelection(-app.paneModel.GetDataViewportRows(), 0)
		return app, nil
	case tea.KeyPgDown:
		app.paneModel.MoveDataSelection(app.paneModel.GetDataViewportRows(), 0)
		return app, nil
	case tea.KeyHome:
		app.paneModel.SetDataSelection(app.paneModel.GetSelectedDataRowIndex(), 0)
		return app, nil
	case tea.KeyEnd:
		app.paneModel.SetDataSelection(
			app.paneModel.GetSelectedDataRowIndex(),
			app.paneModel.GetDataColCount()-1,
		)
		return app, nil
	case tea.KeyEnter:
		app.beginCellEdit()
		return app, nil
	}

	switch strings.ToLower(msg.String()) {
	case "ctrl+q":
		app.focusMode = FocusQuery
		return app, nil
	case "ctrl+n":
		app.beginInsertRow()
		return app, nil
	case "ctrl+d":
		return app, app.requestDeleteRow()
	}

	return app, nil
}

func (app *XTreeGoldApp) View() string {
	if app.tree.error != nil {
		return app.renderError(app.tree.error)
	}

	switch app.connectionStep {
	case StepSelectConnection:
		return app.connectionDialog.View()
	case StepAddConnection:
		return app.addConnectionForm.View()
	case StepConnected:
		return app.renderMainView()
	default:
		return "Loading..."
	}
}

func (app *XTreeGoldApp) renderMainView() string {
	if !app.initialized {
		return app.renderLoading()
	}

	width, height := app.width, app.height
	if width == 0 {
		width = 80
	}
	if height == 0 {
		height = 24
	}
	headerHeight := 1
	footerHeight := 2
	bodyHeight := height - headerHeight - footerHeight

	// Update viewport height to match terminal
	app.tree.viewport.Height = bodyHeight

	header := app.renderer.renderHeader()

	switch app.focusMode {
	case FocusTree:
		return app.renderTreeView(width, height, bodyHeight, header)
	case FocusQuery:
		return app.renderQueryView(width, height, bodyHeight, header)
	case FocusData:
		return app.renderDataView(width, height, bodyHeight, header)
	default:
		return ""
	}
}

func (app *XTreeGoldApp) renderTreeView(width, height, bodyHeight int, header string) string {
	header = app.paneRenderer.renderHeader()
	footer := app.paneRenderer.renderStatus(app.paneModel)

	content := app.styles.Header.Render(header) + "\n"

	panesView := app.paneRenderer.RenderPanes(app.paneModel, width, bodyHeight)
	content += panesView + "\n"

	content += app.styles.Footer.Render(footer)
	return content
}

func (app *XTreeGoldApp) renderQueryView(width, height, bodyHeight int, header string) string {
	footer := "SQL Editor | ESC: Return to Tree | Enter: Execute Query | Ctrl+J: Newline"
	content := app.styles.Header.Render(header) + "\n"
	queryView := app.queryEditor.View()
	content += queryView + "\n"
	content += app.styles.Footer.Render(footer)
	return content
}

func (app *XTreeGoldApp) renderDataView(width, height, bodyHeight int, header string) string {
	footer := "Data View | ESC: Return to Tree | Ctrl+Q: Query | Enter: Edit | Ctrl+N: Insert | Ctrl+D: Delete"
	content := app.styles.Header.Render(header) + "\n"
	dataView := app.paneRenderer.renderDataPane(app.paneModel, "Data", width, bodyHeight, app.paneModel.GetFocus() == PaneData)
	content += dataView + "\n"
	if app.dataEditMode != DataEditNone && app.dataEditor != nil {
		content += app.dataEditor.View(app.dataEditPrompt()) + "\n"
	}
	content += app.styles.Footer.Render(footer)
	return content
}

func (app *XTreeGoldApp) renderError(err error) string {
	errorMsg := fmt.Sprintf("❌ Error: %v", err)
	instructions := "Press Escape to continue"

	content := app.styles.Error.Render(errorMsg) + "\n\n"
	content += app.styles.Body.Render(instructions)

	return lipgloss.NewStyle().
		Width(60).
		Align(lipgloss.Center).
		Height(10).
		Border(lipgloss.RoundedBorder()).
		Padding(2, 1).
		Render(content)
}

func (app *XTreeGoldApp) renderLoading() string {
	loadingText := "🔄 Loading tree structure from database..."

	return lipgloss.NewStyle().
		Width(50).
		Align(lipgloss.Center).
		Height(8).
		Border(lipgloss.RoundedBorder()).
		Padding(2, 1).
		Render(loadingText)
}

func (app *XTreeGoldApp) truncateToHeight(content string, maxHeight int) string {
	lines := lipgloss.NewStyle().MaxHeight(maxHeight).Render(content)
	return lines
}

func main() {
	app, err := NewXTreeGoldApp()
	if err != nil {
		fmt.Printf("Failed to initialize application: %v\n", err)
		return
	}

	p := tea.NewProgram(
		app,
		tea.WithAltScreen(),
		tea.WithMouseAllMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Application error: %v\n", err)
	}
}
