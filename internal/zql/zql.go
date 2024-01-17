package zql

import (
	"database/sql"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func Sqlite3(dbName string) (*sql.DB, error) {
	return sql.Open("sqlite3", dbName)
}
