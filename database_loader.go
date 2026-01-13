package main

import tea "github.com/charmbracelet/bubbletea"

type DatabaseLoader interface {
	LoadTreeAsync(serverName string) tea.Cmd
	LoadChildren(node *TreeNode) error
	GetTableData(database, schema, table string, limit, offset int) ([]map[string]interface{}, error)
	UpdateCell(database, schema, table, column, rowID string, value interface{}) error
	InsertRow(database, schema, table string, values map[string]interface{}) error
	DeleteRow(database, schema, table, rowID string) error
	ExecuteQuery(query string) ([]map[string]interface{}, error)
}
