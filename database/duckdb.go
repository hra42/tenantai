package database

import (
	"database/sql"
	"fmt"

	_ "github.com/duckdb/duckdb-go/v2"
)

func OpenDB(path string, maxConns int) (*sql.DB, error) {
	db, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, fmt.Errorf("opening duckdb at %s: %w", path, err)
	}

	db.SetMaxOpenConns(maxConns)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("pinging duckdb at %s: %w", path, err)
	}

	return db, nil
}
