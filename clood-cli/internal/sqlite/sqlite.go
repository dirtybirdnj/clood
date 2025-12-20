// Package sqlite provides SQLite query capabilities for clood.
// Uses pure-Go SQLite (no CGO) for portability.
package sqlite

import (
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

// QueryResult represents the result of a query
type QueryResult struct {
	Columns []string        `json:"columns"`
	Rows    [][]interface{} `json:"rows"`
	Count   int             `json:"count"`
}

// Query executes a SELECT query and returns results as JSON-friendly structure
func Query(dbPath, query string, args ...interface{}) (*QueryResult, error) {
	// Safety check - only allow SELECT statements
	queryLower := strings.TrimSpace(strings.ToLower(query))
	if !strings.HasPrefix(queryLower, "select") &&
		!strings.HasPrefix(queryLower, "pragma") &&
		!strings.HasPrefix(queryLower, "explain") {
		return nil, fmt.Errorf("only SELECT, PRAGMA, and EXPLAIN queries are allowed")
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	result := &QueryResult{
		Columns: columns,
		Rows:    [][]interface{}{},
	}

	// Scan rows
	for rows.Next() {
		// Create a slice of interface{} to hold the values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Convert []byte to string for better JSON output
		for i, v := range values {
			if b, ok := v.([]byte); ok {
				values[i] = string(b)
			}
		}

		result.Rows = append(result.Rows, values)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	result.Count = len(result.Rows)
	return result, nil
}

// TableInfo represents information about a table
type TableInfo struct {
	Name    string       `json:"name"`
	Columns []ColumnInfo `json:"columns"`
}

// ColumnInfo represents a column in a table
type ColumnInfo struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	NotNull    bool   `json:"not_null"`
	PrimaryKey bool   `json:"primary_key"`
	Default    string `json:"default,omitempty"`
}

// Schema returns the schema for a specific table
func Schema(dbPath, tableName string) (*TableInfo, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Use PRAGMA table_info to get column information
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return nil, fmt.Errorf("failed to get table info: %w", err)
	}
	defer rows.Close()

	info := &TableInfo{
		Name:    tableName,
		Columns: []ColumnInfo{},
	}

	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dfltValue sql.NullString

		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			return nil, fmt.Errorf("failed to scan column info: %w", err)
		}

		col := ColumnInfo{
			Name:       name,
			Type:       colType,
			NotNull:    notNull == 1,
			PrimaryKey: pk == 1,
		}
		if dfltValue.Valid {
			col.Default = dfltValue.String
		}

		info.Columns = append(info.Columns, col)
	}

	if len(info.Columns) == 0 {
		return nil, fmt.Errorf("table not found: %s", tableName)
	}

	return info, nil
}

// Tables returns a list of all tables in the database
func Tables(dbPath string) ([]string, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT name FROM sqlite_master
		WHERE type='table' AND name NOT LIKE 'sqlite_%'
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list tables: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("failed to scan table name: %w", err)
		}
		tables = append(tables, name)
	}

	return tables, nil
}

// DatabaseInfo returns full schema information for all tables
func DatabaseInfo(dbPath string) ([]TableInfo, error) {
	tables, err := Tables(dbPath)
	if err != nil {
		return nil, err
	}

	var infos []TableInfo
	for _, table := range tables {
		info, err := Schema(dbPath, table)
		if err != nil {
			continue // Skip tables we can't read
		}
		infos = append(infos, *info)
	}

	return infos, nil
}
