package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func newTestManager(t *testing.T) *DefaultServiceManager {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "services")
	mgr, err := NewServiceManager(dir, 1)
	if err != nil {
		t.Fatalf("NewServiceManager: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })
	return mgr
}

func TestCreate(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	svc, err := mgr.Create(ctx, "test-svc", "Test Service")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if svc.ID != "test-svc" {
		t.Errorf("ID = %q, want test-svc", svc.ID)
	}
	if svc.Name != "Test Service" {
		t.Errorf("Name = %q, want Test Service", svc.Name)
	}

	// DB file should exist
	dbPath := filepath.Join(mgr.servicesDir, "test-svc.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("expected .db file to exist")
	}
}

func TestCreate_Duplicate(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	if _, err := mgr.Create(ctx, "dup", "First"); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	if _, err := mgr.Create(ctx, "dup", "Second"); err != ErrServiceExists {
		t.Fatalf("second Create error = %v, want ErrServiceExists", err)
	}
}

func TestCreate_InvalidID(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	cases := []string{"", "UPPER", "has space", "bad_underscore", "-leading", "trailing-"}
	for _, id := range cases {
		if _, err := mgr.Create(ctx, id, "name"); err == nil {
			t.Errorf("Create(%q) should have failed", id)
		}
	}
}

func TestGet(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	_, _ = mgr.Create(ctx, "exists", "Exists")
	svc, err := mgr.Get(ctx, "exists")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if svc.ID != "exists" {
		t.Errorf("ID = %q, want exists", svc.ID)
	}
}

func TestGet_NotFound(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	_, err := mgr.Get(ctx, "nope")
	if err != ErrServiceNotFound {
		t.Fatalf("Get error = %v, want ErrServiceNotFound", err)
	}
}

func TestDelete_SoftDelete(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	_, _ = mgr.Create(ctx, "to-delete", "Delete Me")
	dbPath := filepath.Join(mgr.servicesDir, "to-delete.db")

	if err := mgr.Delete(ctx, "to-delete"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Should not be found in registry
	if _, err := mgr.Get(ctx, "to-delete"); err != ErrServiceNotFound {
		t.Error("expected ErrServiceNotFound after delete")
	}

	// .db file should still exist on disk (soft delete)
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("expected .db file to remain after soft delete")
	}
}

func TestGetDBConnection(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	_, _ = mgr.Create(ctx, "db-test", "DB Test")
	db, err := mgr.GetDBConnection(ctx, "db-test")
	if err != nil {
		t.Fatalf("GetDBConnection: %v", err)
	}

	var result int
	if err := db.QueryRow("SELECT 1").Scan(&result); err != nil {
		t.Fatalf("query on connection: %v", err)
	}
}

func TestLoadExistingOnRestart(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "services")
	ctx := context.Background()

	// First manager: create a service
	mgr1, err := NewServiceManager(dir, 1)
	if err != nil {
		t.Fatalf("NewServiceManager 1: %v", err)
	}
	_, _ = mgr1.Create(ctx, "persist", "Persistent")
	_ = mgr1.Close()

	// Second manager: should discover the service from disk
	mgr2, err := NewServiceManager(dir, 1)
	if err != nil {
		t.Fatalf("NewServiceManager 2: %v", err)
	}
	defer func() { _ = mgr2.Close() }()

	svc, err := mgr2.Get(ctx, "persist")
	if err != nil {
		t.Fatalf("Get after restart: %v", err)
	}
	if svc.Name != "Persistent" {
		t.Errorf("Name = %q, want Persistent", svc.Name)
	}
}

func TestList(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	_, _ = mgr.Create(ctx, "alpha", "Alpha")
	_, _ = mgr.Create(ctx, "beta", "Beta")

	services, err := mgr.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(services) != 2 {
		t.Errorf("len = %d, want 2", len(services))
	}
}
