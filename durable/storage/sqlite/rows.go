package sqlite

import (
	"fmt"

	chordsqlite "github.com/tmih06/ompig/chord/sqlite"
)

// rowInt reads an INTEGER column.
func rowInt(row SqliteRow, column string) (int64, error) { return chordsqlite.RowInt(row, column) }

// rowText reads a TEXT column.
func rowText(row SqliteRow, column string) (string, error) {
	switch value := row[column].(type) {
	case string:
		return value, nil
	case []byte:
		return string(value), nil
	}
	return "", fmt.Errorf("SQLite column %s is not text: %v", column, row[column])
}
