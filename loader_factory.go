package main

import (
	"database/sql"
	"fmt"
)

func NewDatabaseLoader(db *sql.DB, connInfo *ConnectionInfo) (DatabaseLoader, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	if connInfo == nil {
		return nil, fmt.Errorf("connection info is required")
	}

	connType := connInfo.Type
	if connType == "" {
		connType = ConnectionPostgres
	}

	switch connType {
	case ConnectionSQLite:
		return NewSQLiteTreeLoader(db, connInfo), nil
	case ConnectionPostgres:
		fallthrough
	default:
		return NewPostgresTreeLoader(db, connInfo), nil
	}
}
