package service

import (
	"errors"
	"fmt"
	"regexp"
	"time"
)

var serviceIDRegex = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

var (
	ErrServiceNotFound = errors.New("service not found")
	ErrServiceExists   = errors.New("service already exists")
)

type Service struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	CreatedAt time.Time              `json:"created_at"`
	Config    map[string]interface{} `json:"config,omitempty"`
}

func ValidateServiceID(id string) error {
	if len(id) == 0 {
		return fmt.Errorf("service ID cannot be empty")
	}
	if len(id) > 63 {
		return fmt.Errorf("service ID must be at most 63 characters")
	}
	if !serviceIDRegex.MatchString(id) {
		return fmt.Errorf("service ID must be lowercase alphanumeric with hyphens (e.g. my-service)")
	}
	return nil
}
