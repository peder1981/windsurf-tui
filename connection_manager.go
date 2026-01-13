package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

type ConnectionType string

const (
	ConnectionPostgres ConnectionType = "postgres"
	ConnectionSQLite   ConnectionType = "sqlite"
)

type ConnectionInfo struct {
	Name     string         `json:"name"`
	Type     ConnectionType `json:"type"`
	Host     string         `json:"host,omitempty"`
	Port     int            `json:"port,omitempty"`
	User     string         `json:"user,omitempty"`
	Password string         `json:"password,omitempty"`
	Database string         `json:"database"`
	SSLMode  string         `json:"sslmode,omitempty"`
	Path     string         `json:"path,omitempty"`
}

type ConnectionManager struct {
	connections      map[string]*sql.DB
	savedConnections map[string]*ConnectionInfo
	configPath       string
}

func NewConnectionManager() (*ConnectionManager, error) {
	homeConfig := filepath.Join(os.Getenv("HOME"), ".windsurf-tui", "connections.json")

	var configPath string
	cwd, err := os.Getwd()
	if err == nil {
		localConfig := filepath.Join(cwd, "connections.json")
		if _, err := os.Stat(localConfig); err == nil {
			configPath = localConfig
		}
	}

	if configPath == "" {
		configPath = homeConfig
	}

	cm := &ConnectionManager{
		connections:      make(map[string]*sql.DB),
		savedConnections: make(map[string]*ConnectionInfo),
		configPath:       configPath,
	}

	if err := os.MkdirAll(filepath.Dir(cm.configPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := cm.LoadSavedConnections(); err != nil {
		return nil, fmt.Errorf("failed to load saved connections: %w", err)
	}

	return cm, nil
}

func (cm *ConnectionManager) Connect(connInfo *ConnectionInfo) (*sql.DB, error) {
	var (
		driver  string
		connStr string
	)

	switch connInfo.Type {
	case ConnectionSQLite:
		driver = "sqlite3"
		if strings.TrimSpace(connInfo.Path) == "" {
			return nil, fmt.Errorf("sqlite connection requires a file path")
		}
		connStr = connInfo.Path
	default:
		driver = "postgres"
		connStr = fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			connInfo.Host,
			connInfo.Port,
			connInfo.User,
			connInfo.Password,
			connInfo.Database,
			connInfo.SSLMode,
		)
	}

	db, err := sql.Open(driver, connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	cm.connections[connInfo.Name] = db
	return db, nil
}

func (cm *ConnectionManager) GetConnection(name string) (*sql.DB, bool) {
	db, ok := cm.connections[name]
	return db, ok
}

func (cm *ConnectionManager) Disconnect(name string) error {
	if db, ok := cm.connections[name]; ok {
		if err := db.Close(); err != nil {
			return err
		}
		delete(cm.connections, name)
	}
	return nil
}

func (cm *ConnectionManager) DisconnectAll() error {
	var lastErr error
	for name := range cm.connections {
		if err := cm.Disconnect(name); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

func (cm *ConnectionManager) SaveConnection(connInfo *ConnectionInfo) error {
	cm.savedConnections[connInfo.Name] = connInfo
	return cm.SaveConnections()
}

func (cm *ConnectionManager) SaveConnections() error {
	connections := make([]*ConnectionInfo, 0, len(cm.savedConnections))
	for _, conn := range cm.savedConnections {
		connections = append(connections, conn)
	}

	data, err := json.MarshalIndent(connections, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal connections: %w", err)
	}

	if err := os.WriteFile(cm.configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write connections file: %w", err)
	}

	return nil
}

func (cm *ConnectionManager) LoadSavedConnections() error {
	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read connections file: %w", err)
	}

	var connections []*ConnectionInfo
	if err := json.Unmarshal(data, &connections); err != nil {
		return fmt.Errorf("failed to unmarshal connections: %w", err)
	}

	for _, conn := range connections {
		cm.savedConnections[conn.Name] = conn
	}

	return nil
}

func (cm *ConnectionManager) GetSavedConnections() []*ConnectionInfo {
	connections := make([]*ConnectionInfo, 0, len(cm.savedConnections))
	for _, conn := range cm.savedConnections {
		copyConn := *conn
		connections = append(connections, &copyConn)
	}
	sort.Slice(connections, func(i, j int) bool {
		return strings.ToLower(connections[i].Name) < strings.ToLower(connections[j].Name)
	})
	return connections
}

func (cm *ConnectionManager) DeleteConnection(name string) error {
	delete(cm.savedConnections, name)
	return cm.SaveConnections()
}

func (cm *ConnectionManager) GetConnectionNames() []string {
	names := make([]string, 0, len(cm.connections))
	for name := range cm.connections {
		names = append(names, name)
	}
	return names
}

func (cm *ConnectionManager) Close() error {
	return cm.DisconnectAll()
}

func (cm *ConnectionManager) CreateDefaultConnectionIfNone() (*ConnectionInfo, error) {
	if len(cm.savedConnections) > 0 {
		return cm.GetSavedConnections()[0], nil
	}

	defaultConn := &ConnectionInfo{
		Name:     "localhost",
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "",
		Database: "postgres",
		SSLMode:  "disable",
	}

	if err := cm.SaveConnection(defaultConn); err != nil {
		return nil, fmt.Errorf("failed to save default connection: %w", err)
	}

	return defaultConn, nil
}
