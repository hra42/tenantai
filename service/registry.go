package service

import (
	"database/sql"
	"sync"
)

type registry struct {
	mu       sync.RWMutex
	services map[string]*Service
	dbs      map[string]*sql.DB
}

func newRegistry() *registry {
	return &registry{
		services: make(map[string]*Service),
		dbs:      make(map[string]*sql.DB),
	}
}

func (r *registry) get(id string) (*Service, *sql.DB, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	svc, ok := r.services[id]
	if !ok {
		return nil, nil, false
	}
	return svc, r.dbs[id], true
}

func (r *registry) set(svc *Service, db *sql.DB) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.services[svc.ID] = svc
	r.dbs[svc.ID] = db
}

func (r *registry) delete(id string) (*sql.DB, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	db, ok := r.dbs[id]
	if !ok {
		return nil, false
	}
	delete(r.services, id)
	delete(r.dbs, id)
	return db, true
}

func (r *registry) list() []*Service {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*Service, 0, len(r.services))
	for _, svc := range r.services {
		result = append(result, svc)
	}
	return result
}
