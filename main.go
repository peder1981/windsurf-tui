package main

import (
	"fmt"

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
}

type UpdateCellMsg struct {
	database string
	schema   string
	table    string
	ctid     string
	column   string
	value    string
	rowIndex int
	colIndex int
}

type InsertRowMsg struct {
	database string
	schema   string
	table    string
	values   map[string]string
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
			app.paneModel.SetFocus(PaneData)
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
	switch msg.Type {
	case tea.KeyEscape:
		app.focusMode = FocusTree
		return app, nil
	case tea.KeyCtrlQ:
		app.focusMode = FocusQuery
		return app, nil
	default:
		return app, nil
	}
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
	footer := "Data View | ESC: Return to Tree | Ctrl+Q: Query"
	content := app.styles.Header.Render(header) + "\n"
	dataView := app.paneRenderer.renderDataPane(app.paneModel, "Data", width, bodyHeight, app.paneModel.GetFocus() == PaneData)
	content += dataView + "\n"
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
