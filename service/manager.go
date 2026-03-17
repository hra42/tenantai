package service

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hra42/tenantai/database"
)

type ServiceManager interface {
	Create(ctx context.Context, id, name string) (*Service, error)
	Get(ctx context.Context, id string) (*Service, error)
	List(ctx context.Context) ([]*Service, error)
	Delete(ctx context.Context, id string) error
	GetDBConnection(ctx context.Context, id string) (*sql.DB, error)
	Close() error
}

type DefaultServiceManager struct {
	reg         *registry
	servicesDir string
	maxConns    int
}

func NewServiceManager(servicesDir string, maxConns int) (*DefaultServiceManager, error) {
	if err := os.MkdirAll(servicesDir, 0755); err != nil {
		return nil, fmt.Errorf("creating services directory: %w", err)
	}

	mgr := &DefaultServiceManager{
		reg:         newRegistry(),
		servicesDir: servicesDir,
		maxConns:    maxConns,
	}

	if err := mgr.loadExisting(); err != nil {
		return nil, fmt.Errorf("loading existing services: %w", err)
	}

	return mgr, nil
}

func (m *DefaultServiceManager) loadExisting() error {
	entries, err := os.ReadDir(m.servicesDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".db") {
			continue
		}

		id := strings.TrimSuffix(entry.Name(), ".db")
		dbPath := filepath.Join(m.servicesDir, entry.Name())

		db, err := database.OpenDB(dbPath, m.maxConns)
		if err != nil {
			return fmt.Errorf("opening db for service %s: %w", id, err)
		}

		name := id // fallback
		var storedName string
		err = db.QueryRow("SELECT value FROM service_metadata WHERE key = 'name'").Scan(&storedName)
		if err == nil {
			name = storedName
		}

		info, _ := entry.Info()
		createdAt := time.Now()
		if info != nil {
			createdAt = info.ModTime()
		}

		svc := &Service{
			ID:        id,
			Name:      name,
			CreatedAt: createdAt,
		}
		m.reg.set(svc, db)
	}

	return nil
}

func (m *DefaultServiceManager) Create(_ context.Context, id, name string) (*Service, error) {
	if err := ValidateServiceID(id); err != nil {
		return nil, err
	}

	if _, _, exists := m.reg.get(id); exists {
		return nil, ErrServiceExists
	}

	dbPath := filepath.Join(m.servicesDir, id+".db")
	db, err := database.OpenDB(dbPath, m.maxConns)
	if err != nil {
		return nil, fmt.Errorf("creating db for service %s: %w", id, err)
	}

	if err := database.InitializeSchema(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("initializing schema for service %s: %w", id, err)
	}

	_, err = db.Exec("INSERT INTO service_metadata (key, value) VALUES ('name', ?)", name)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("storing service metadata for %s: %w", id, err)
	}

	svc := &Service{
		ID:        id,
		Name:      name,
		CreatedAt: time.Now(),
	}
	m.reg.set(svc, db)

	return svc, nil
}

func (m *DefaultServiceManager) Get(_ context.Context, id string) (*Service, error) {
	svc, _, ok := m.reg.get(id)
	if !ok {
		return nil, ErrServiceNotFound
	}
	return svc, nil
}

func (m *DefaultServiceManager) List(_ context.Context) ([]*Service, error) {
	return m.reg.list(), nil
}

func (m *DefaultServiceManager) Delete(_ context.Context, id string) error {
	db, ok := m.reg.delete(id)
	if !ok {
		return ErrServiceNotFound
	}
	return db.Close()
}

func (m *DefaultServiceManager) GetDBConnection(_ context.Context, id string) (*sql.DB, error) {
	_, db, ok := m.reg.get(id)
	if !ok {
		return nil, ErrServiceNotFound
	}
	return db, nil
}

func (m *DefaultServiceManager) Close() error {
	for _, svc := range m.reg.list() {
		if db, ok := m.reg.delete(svc.ID); ok {
			_ = db.Close()
		}
	}
	return nil
}
