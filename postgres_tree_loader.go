package main

import (
	"database/sql"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type PostgresTreeLoader struct {
	db          *sql.DB
	connInfo    *ConnectionInfo
	connections map[string]*sql.DB
}

func (ptl *PostgresTreeLoader) getDatabaseConnection(databaseName string) (*sql.DB, error) {
	if databaseName == "" || ptl.connInfo == nil || databaseName == ptl.connInfo.Database {
		return ptl.db, nil
	}

	if db, ok := ptl.connections[databaseName]; ok {
		return db, nil
	}

	connInfo := *ptl.connInfo
	connInfo.Database = databaseName

	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		connInfo.Host,
		connInfo.Port,
		connInfo.User,
		connInfo.Password,
		connInfo.Database,
		connInfo.SSLMode,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection for database %s: %w", databaseName, err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database %s: %w", databaseName, err)
	}

	ptl.connections[databaseName] = db
	return db, nil
}

func quoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func NewPostgresTreeLoader(db *sql.DB, connInfo *ConnectionInfo) *PostgresTreeLoader {
	return &PostgresTreeLoader{
		db:          db,
		connInfo:    connInfo,
		connections: make(map[string]*sql.DB),
	}
}

func (ptl *PostgresTreeLoader) LoadTree(serverName string) (*TreeNode, error) {
	root := &TreeNode{
		ID:       "root",
		Name:     "PostgreSQL Servers",
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
	}

	databases, err := ptl.loadDatabases()
	if err != nil {
		return nil, fmt.Errorf("failed to load databases: %w", err)
	}

	for _, db := range databases {
		db.Parent = serverNode
		serverNode.Children = append(serverNode.Children, db)
	}

	root.Children = append(root.Children, serverNode)
	serverNode.Parent = root

	return root, nil
}

func (ptl *PostgresTreeLoader) loadDatabases() ([]*TreeNode, error) {
	query := `
		SELECT 
			datname as database_name,
			pg_size_pretty(pg_database_size(oid)) as database_size,
			pg_stat_get_db_tuples_returned(oid) as estimated_rows
		FROM pg_database
		WHERE datistemplate = false
		ORDER BY datname
	`

	rows, err := ptl.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var databases []*TreeNode
	for rows.Next() {
		var dbName, dbSize string
		var estimatedRows int64

		if err := rows.Scan(&dbName, &dbSize, &estimatedRows); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		database := &TreeNode{
			ID:    fmt.Sprintf("db_%s", dbName),
			Name:  dbName,
			Type:  NodeDatabase,
			Path:  dbName,
			Level: 1,
			Metadata: NodeMetadata{
				Size:     dbSize,
				RowCount: estimatedRows,
			},
			Children: make([]*TreeNode, 0),
		}

		// Don't load schemas initially - will be loaded on demand
		// schemas, err := ptl.loadSchemas(dbName)
		// if err != nil {
		// 	return nil, fmt.Errorf("failed to load schemas for %s: %w", dbName, err)
		// }

		// for _, schema := range schemas {
		// 	schema.Parent = database
		// 	database.Children = append(database.Children, schema)
		// }

		databases = append(databases, database)
	}

	return databases, nil
}

func (ptl *PostgresTreeLoader) loadSchemas(databaseName string) ([]*TreeNode, error) {
	dbConn, err := ptl.getDatabaseConnection(databaseName)
	if err != nil {
		return nil, err
	}
	query := `
		SELECT 
			s.schema_name,
			COALESCE(t.table_count, 0) as table_count
		FROM information_schema.schemata s
		LEFT JOIN (
			SELECT table_schema, COUNT(*) as table_count
			FROM information_schema.tables
			WHERE table_type = 'BASE TABLE'
			GROUP BY table_schema
		) t ON s.schema_name = t.table_schema
		WHERE s.schema_name NOT IN ('pg_catalog', 'information_schema')
		ORDER BY s.schema_name
	`

	rows, err := dbConn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var schemas []*TreeNode
	for rows.Next() {
		var schemaName string
		var tableCount int

		if err := rows.Scan(&schemaName, &tableCount); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		schema := &TreeNode{
			ID:    fmt.Sprintf("schema_%s_%s", databaseName, schemaName),
			Name:  schemaName,
			Type:  NodeSchema,
			Path:  fmt.Sprintf("%s.%s", databaseName, schemaName),
			Level: 2,
			Metadata: NodeMetadata{
				Count: tableCount,
			},
			Children: make([]*TreeNode, 0),
		}

		// Don't load tables initially - will be loaded on demand
		// tables, err := ptl.loadTables(databaseName, schemaName)
		// if err != nil {
		// 	return nil, fmt.Errorf("failed to load tables for %s: %w", schemaName, err)
		// }

		// for _, table := range tables {
		// 	table.Parent = schema
		// 	schema.Children = append(schema.Children, table)
		// }

		schemas = append(schemas, schema)
	}

	return schemas, nil
}

func (ptl *PostgresTreeLoader) loadTables(databaseName, schemaName string) ([]*TreeNode, error) {
	dbConn, err := ptl.getDatabaseConnection(databaseName)
	if err != nil {
		return nil, err
	}
	query := `
		SELECT
			tablename,
			pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as table_size,
			pg_stat_get_tuples_returned(c.oid) as row_count
		FROM pg_tables t
		JOIN pg_class c ON c.relname = t.tablename
		WHERE schemaname = $1
		ORDER BY tablename
	`

	rows, err := dbConn.Query(query, schemaName)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var tables []*TreeNode
	for rows.Next() {
		var tableName, tableSize string
		var rowCount int64

		if err := rows.Scan(&tableName, &tableSize, &rowCount); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		table := &TreeNode{
			ID:    fmt.Sprintf("table_%s_%s_%s", databaseName, schemaName, tableName),
			Name:  tableName,
			Type:  NodeTable,
			Path:  fmt.Sprintf("%s.%s.%s", databaseName, schemaName, tableName),
			Level: 3,
			Metadata: NodeMetadata{
				Size:     tableSize,
				RowCount: rowCount,
			},
			Children: make([]*TreeNode, 0),
		}

		// Don't load columns initially - will be loaded on demand
		// columns, err := ptl.loadColumns(databaseName, schemaName, tableName)
		// if err != nil {
		// 	return nil, fmt.Errorf("failed to load columns for %s: %w", tableName, err)
		// }

		// for _, column := range columns {
		// 	column.Parent = table
		// 	table.Children = append(table.Children, column)
		// }

		tables = append(tables, table)
	}

	return tables, nil
}

func (ptl *PostgresTreeLoader) loadColumns(databaseName, schemaName, tableName string) ([]*TreeNode, error) {
	dbConn, err := ptl.getDatabaseConnection(databaseName)
	if err != nil {
		return nil, err
	}
	query := `
		SELECT 
			column_name,
			data_type,
			is_nullable,
			column_default,
			(
				SELECT COUNT(*) 
				FROM information_schema.key_column_usage kcu
				WHERE kcu.table_schema = $1
				AND kcu.table_name = $2
				AND kcu.column_name = $3
				AND EXISTS (
					SELECT 1 
					FROM information_schema.table_constraints tc
					WHERE tc.constraint_name = kcu.constraint_name
					AND tc.constraint_type = 'PRIMARY KEY'
				)
			) as is_primary_key
		FROM information_schema.columns
		WHERE table_schema = $1 AND table_name = $2
		ORDER BY ordinal_position
	`

	rows, err := dbConn.Query(query, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var columns []*TreeNode
	for rows.Next() {
		var columnName, dataType, defaultValue string
		var isNullable bool
		var isPrimaryKey int

		if err := rows.Scan(&columnName, &dataType, &isNullable, &defaultValue, &isPrimaryKey); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		column := &TreeNode{
			ID:    fmt.Sprintf("col_%s_%s_%s_%s", databaseName, schemaName, tableName, columnName),
			Name:  columnName,
			Type:  NodeColumn,
			Path:  fmt.Sprintf("%s.%s.%s.%s", databaseName, schemaName, tableName, columnName),
			Level: 4,
			Metadata: NodeMetadata{
				DataType:     dataType,
				IsNullable:   isNullable,
				DefaultValue: defaultValue,
				PrimaryKey:   isPrimaryKey > 0,
			},
			Children: make([]*TreeNode, 0),
		}

		columns = append(columns, column)
	}

	return columns, nil
}

func (ptl *PostgresTreeLoader) LoadTreeAsync(serverName string) tea.Cmd {
	return func() tea.Msg {
		tree, err := ptl.LoadTree(serverName)
		if err != nil {
			return ErrMsg{err}
		}
		return TreeLoadedMsg{tree: tree}
	}
}

func (ptl *PostgresTreeLoader) GetTableData(databaseName, schemaName, tableName string, limit, offset int) ([]map[string]interface{}, error) {
	dbConn, err := ptl.getDatabaseConnection(databaseName)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`
		SELECT * FROM %s.%s
		ORDER BY ctid
		LIMIT %d OFFSET %d
	`, quoteIdentifier(schemaName), quoteIdentifier(tableName), limit, offset)

	rows, err := dbConn.Query(query)
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

func (ptl *PostgresTreeLoader) ExecuteQuery(queryStr string) ([]map[string]interface{}, error) {
	queryStr = strings.TrimSpace(queryStr)
	if queryStr == "" {
		return nil, nil
	}

	rows, err := ptl.db.Query(queryStr)
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

func (ptl *PostgresTreeLoader) GetTableRowCount(databaseName, schemaName, tableName string) (int64, error) {
	dbConn, err := ptl.getDatabaseConnection(databaseName)
	if err != nil {
		return 0, err
	}

	query := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM %s.%s
	`, quoteIdentifier(schemaName), quoteIdentifier(tableName))

	var count int64
	if err := dbConn.QueryRow(query).Scan(&count); err != nil {
		return 0, fmt.Errorf("query failed: %w", err)
	}

	return count, nil
}

func (ptl *PostgresTreeLoader) buildPathFromParent(node *TreeNode) string {
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

func (ptl *PostgresTreeLoader) LoadChildren(node *TreeNode) error {
	if node == nil || len(node.Children) > 0 {
		return nil
	}

	parts := strings.Split(node.Path, ".")

	switch node.Type {
	case NodeDatabase:
		if len(parts) >= 1 {
			schemas, err := ptl.loadSchemas(parts[0])
			if err != nil {
				return err
			}
			for _, schema := range schemas {
				schema.Parent = node
				node.Children = append(node.Children, schema)
			}
		}
	case NodeSchema:
		if len(parts) >= 2 {
			tables, err := ptl.loadTables(parts[0], parts[1])
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
			columns, err := ptl.loadColumns(parts[0], parts[1], parts[2])
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
