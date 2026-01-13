package main

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type SQLiteTreeLoader struct {
	db       *sql.DB
	connInfo *ConnectionInfo
}

func NewSQLiteTreeLoader(db *sql.DB, connInfo *ConnectionInfo) *SQLiteTreeLoader {
	return &SQLiteTreeLoader{
		db:       db,
		connInfo: connInfo,
	}
}

func (stl *SQLiteTreeLoader) LoadTree(serverName string) (*TreeNode, error) {
	root := &TreeNode{
		ID:       "root",
		Name:     "SQLite Connections",
		Type:     NodeServer,
		Level:    -1,
		Children: make([]*TreeNode, 0),
	}

	serverNode := &TreeNode{
		ID:       serverName,
		Name:     serverName,
		Type:     NodeServer,
		Path:     serverName,
		Level:    0,
		Children: make([]*TreeNode, 0),
		Metadata: NodeMetadata{},
	}

	dbNode := &TreeNode{
		ID:       fmt.Sprintf("sqlite_db_%s", stl.databaseLabel()),
		Name:     stl.databaseLabel(),
		Type:     NodeDatabase,
		Path:     stl.databaseLabel(),
		Level:    1,
		Children: make([]*TreeNode, 0),
		Metadata: NodeMetadata{
			ContextType: "sqlite",
		},
	}

	dbNode.Parent = serverNode
	serverNode.Children = append(serverNode.Children, dbNode)
	serverNode.Parent = root
	root.Children = append(root.Children, serverNode)

	return root, nil
}

func (stl *SQLiteTreeLoader) LoadTreeAsync(serverName string) tea.Cmd {
	return func() tea.Msg {
		tree, err := stl.LoadTree(serverName)
		if err != nil {
			return ErrMsg{err}
		}
		return TreeLoadedMsg{tree: tree}
	}
}

func (stl *SQLiteTreeLoader) LoadChildren(node *TreeNode) error {
	if node == nil || len(node.Children) > 0 {
		return nil
	}

	parts := strings.Split(node.Path, ".")

	switch node.Type {
	case NodeDatabase:
		schemas := stl.loadSchemas(parts[0])
		for _, schema := range schemas {
			schema.Parent = node
			node.Children = append(node.Children, schema)
		}
	case NodeSchema:
		if len(parts) >= 2 {
			tables, err := stl.loadTables(parts[0], parts[1])
			if err != nil {
				return err
			}
			for _, table := range tables {
				table.Parent = node
				node.Children = append(node.Children, table)
			}
		}
	case NodeTable:
		if len(parts) >= 3 {
			columns, err := stl.loadColumns(parts[0], parts[2])
			if err != nil {
				return err
			}
			for _, column := range columns {
				column.Parent = node
				node.Children = append(node.Children, column)
			}
		}
	}

	return nil
}

func (stl *SQLiteTreeLoader) loadSchemas(databaseName string) []*TreeNode {
	schema := &TreeNode{
		ID:    fmt.Sprintf("sqlite_schema_%s_main", databaseName),
		Name:  "main",
		Type:  NodeSchema,
		Path:  fmt.Sprintf("%s.%s", databaseName, "main"),
		Level: 2,
		Metadata: NodeMetadata{
			ContextType: "sqlite",
		},
		Children: make([]*TreeNode, 0),
	}
	return []*TreeNode{schema}
}

func (stl *SQLiteTreeLoader) loadTables(databaseName, schemaName string) ([]*TreeNode, error) {
	const query = `
		SELECT name
		FROM sqlite_master
		WHERE type = 'table'
			AND name NOT LIKE 'sqlite_%'
		ORDER BY name;
	`

	rows, err := stl.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list tables: %w", err)
	}
	defer rows.Close()

	var tables []*TreeNode
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, fmt.Errorf("failed to scan table: %w", err)
		}

		tableNode := &TreeNode{
			ID:    fmt.Sprintf("sqlite_table_%s_%s", databaseName, tableName),
			Name:  tableName,
			Type:  NodeTable,
			Path:  fmt.Sprintf("%s.%s.%s", databaseName, schemaName, tableName),
			Level: 3,
			Metadata: NodeMetadata{
				ContextType: "sqlite",
			},
			Children: make([]*TreeNode, 0),
		}

		tables = append(tables, tableNode)
	}

	return tables, nil
}

func (stl *SQLiteTreeLoader) loadColumns(databaseName, tableName string) ([]*TreeNode, error) {
	query := fmt.Sprintf(`PRAGMA table_info(%s);`, quoteIdentifier(tableName))

	rows, err := stl.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list columns: %w", err)
	}
	defer rows.Close()

	var columns []*TreeNode
	for rows.Next() {
		var (
			cid        int
			name       string
			dataType   string
			notNull    int
			defaultVal sql.NullString
			pk         int
		)

		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultVal, &pk); err != nil {
			return nil, fmt.Errorf("failed to scan column: %w", err)
		}

		colNode := &TreeNode{
			ID:    fmt.Sprintf("sqlite_col_%s_%s", tableName, name),
			Name:  name,
			Type:  NodeColumn,
			Path:  fmt.Sprintf("%s.%s.%s.%s", databaseName, "main", tableName, name),
			Level: 4,
			Metadata: NodeMetadata{
				DataType:     dataType,
				IsNullable:   notNull == 0,
				DefaultValue: defaultVal.String,
				PrimaryKey:   pk > 0,
			},
		}
		columns = append(columns, colNode)
	}

	return columns, nil
}

func (stl *SQLiteTreeLoader) GetTableData(database, schema, table string, limit, offset int) ([]map[string]interface{}, error) {
	query := fmt.Sprintf(`
		SELECT rowid AS "__rowid", *
		FROM %s
		LIMIT %d OFFSET %d
	`, quoteIdentifier(table), limit, offset)

	rows, err := stl.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			row[col] = values[i]
		}
		results = append(results, row)
	}

	return results, nil
}

func (stl *SQLiteTreeLoader) UpdateCell(database, schema, table, column, rowID string, value interface{}) error {
	query := fmt.Sprintf(`
		UPDATE %s
		SET %s = ?
		WHERE rowid = ?
	`, quoteIdentifier(table), quoteIdentifier(column))

	idValue, err := parseRowID(rowID)
	if err != nil {
		return err
	}

	if _, err := stl.db.Exec(query, value, idValue); err != nil {
		return fmt.Errorf("failed to update %s: %w", table, err)
	}
	return nil
}

func (stl *SQLiteTreeLoader) InsertRow(database, schema, table string, values map[string]interface{}) error {
	if len(values) == 0 {
		return fmt.Errorf("no values provided for insert")
	}

	columns := make([]string, 0, len(values))
	for col := range values {
		columns = append(columns, col)
	}
	sort.Strings(columns)

	var placeholders []string
	var args []interface{}
	for range columns {
		placeholders = append(placeholders, "?")
	}
	for _, col := range columns {
		args = append(args, values[col])
	}

	var quotedCols []string
	for _, col := range columns {
		quotedCols = append(quotedCols, quoteIdentifier(col))
	}

	query := fmt.Sprintf(`
		INSERT INTO %s (%s)
		VALUES (%s)
	`, quoteIdentifier(table), strings.Join(quotedCols, ", "), strings.Join(placeholders, ", "))

	if _, err := stl.db.Exec(query, args...); err != nil {
		return fmt.Errorf("failed to insert row: %w", err)
	}

	return nil
}

func (stl *SQLiteTreeLoader) DeleteRow(database, schema, table, rowID string) error {
	query := fmt.Sprintf(`
		DELETE FROM %s
		WHERE rowid = ?
	`, quoteIdentifier(table))

	idValue, err := parseRowID(rowID)
	if err != nil {
		return err
	}

	if _, err := stl.db.Exec(query, idValue); err != nil {
		return fmt.Errorf("failed to delete row: %w", err)
	}
	return nil
}

func (stl *SQLiteTreeLoader) ExecuteQuery(query string) ([]map[string]interface{}, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}

	rows, err := stl.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			row[col] = values[i]
		}
		results = append(results, row)
	}

	return results, nil
}

func (stl *SQLiteTreeLoader) GetTableRowCount(database, schema, table string) (int64, error) {
	query := fmt.Sprintf(`SELECT COUNT(*) FROM %s`, quoteIdentifier(table))

	var count int64
	if err := stl.db.QueryRow(query).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count rows: %w", err)
	}

	return count, nil
}

func (stl *SQLiteTreeLoader) databaseLabel() string {
	if stl.connInfo == nil {
		return "sqlite"
	}
	if stl.connInfo.Database != "" {
		return stl.connInfo.Database
	}
	if stl.connInfo.Path != "" {
		return filepath.Base(stl.connInfo.Path)
	}
	return stl.connInfo.Name
}

func parseRowID(rowID string) (interface{}, error) {
	if rowID == "" {
		return nil, fmt.Errorf("rowid is empty")
	}
	if i, err := strconv.ParseInt(rowID, 10, 64); err == nil {
		return i, nil
	}
	return rowID, nil
}
