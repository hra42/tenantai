package database

import (
	"testing"
)

func TestOpenDB_InMemory(t *testing.T) {
	db, err := OpenDB("", 1)
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	defer func() { _ = db.Close() }()

	var result int
	if err := db.QueryRow("SELECT 42").Scan(&result); err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if result != 42 {
		t.Errorf("got %d, want 42", result)
	}
}

func TestInitializeSchema(t *testing.T) {
	db, err := OpenDB("", 1)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer func() { _ = db.Close() }()

	if err := InitializeSchema(db); err != nil {
		t.Fatalf("first init failed: %v", err)
	}

	// Idempotent: run again without error
	if err := InitializeSchema(db); err != nil {
		t.Fatalf("second init failed: %v", err)
	}

	// Verify tables exist
	var name string
	err = db.QueryRow("SELECT table_name FROM information_schema.tables WHERE table_name = 'conversations'").Scan(&name)
	if err != nil {
		t.Fatalf("conversations table not found: %v", err)
	}

	err = db.QueryRow("SELECT table_name FROM information_schema.tables WHERE table_name = 'service_metadata'").Scan(&name)
	if err != nil {
		t.Fatalf("service_metadata table not found: %v", err)
	}
}
