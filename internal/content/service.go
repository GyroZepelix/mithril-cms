package content

import (
	"context"
	"fmt"
	"sync"

	"github.com/GyroZepelix/mithril-cms/internal/audit"
	"github.com/GyroZepelix/mithril-cms/internal/schema"
	"github.com/GyroZepelix/mithril-cms/internal/server"
)

// Service implements the business logic for content CRUD operations.
type Service struct {
	repo         *Repository
	mu           sync.RWMutex
	schemas      map[string]schema.ContentType
	auditService *audit.Service
}

// NewService creates a new content Service. The audit service is optional;
// if nil, audit events are silently skipped.
func NewService(repo *Repository, schemas map[string]schema.ContentType, auditService *audit.Service) *Service {
	return &Service{
		repo:         repo,
		schemas:      schemas,
		auditService: auditService,
	}
}

// UpdateSchemas replaces the in-memory schema map. This is called after a
// schema refresh to ensure the service uses the latest content type definitions.
func (s *Service) UpdateSchemas(schemas map[string]schema.ContentType) {
	s.mu.Lock()
	s.schemas = schemas
	s.mu.Unlock()
}

// getSchema safely retrieves a schema by name with read locking.
func (s *Service) getSchema(name string) (schema.ContentType, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ct, ok := s.schemas[name]
	return ct, ok
}

// logAudit sends an audit event if the audit service is configured.
func (s *Service) logAudit(ctx context.Context, event audit.Event) {
	if s.auditService != nil {
		s.auditService.Log(ctx, event)
	}
}

// tableName returns the PostgreSQL table name for a content type.
func tableName(ctName string) string {
	return "ct_" + ctName
}

// ValidationError is returned when content data fails validation.
type ValidationError struct {
	Fields []server.FieldError
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed: %d field errors", len(e.Fields))
}

// List retrieves a paginated list of content entries.
func (s *Service) List(ctx context.Context, contentType string, q QueryParams, publishedOnly bool) ([]map[string]any, int, error) {
	ct, ok := s.getSchema(contentType)
	if !ok {
		return nil, 0, ErrNotFound
	}

	entries, total, err := s.repo.List(ctx, tableName(ct.Name), ct.Fields, q, publishedOnly)
	if err != nil {
		return nil, 0, fmt.Errorf("listing %s entries: %w", contentType, err)
	}

	return entries, total, nil
}

// GetByID retrieves a single content entry by ID.
func (s *Service) GetByID(ctx context.Context, contentType, id string, publishedOnly bool) (map[string]any, error) {
	ct, ok := s.getSchema(contentType)
	if !ok {
		return nil, ErrNotFound
	}

	entry, err := s.repo.GetByID(ctx, tableName(ct.Name), ct.Fields, id, publishedOnly)
	if err != nil {
		return nil, fmt.Errorf("getting %s entry: %w", contentType, err)
	}

	return entry, nil
}

// Create validates and inserts a new content entry as a draft.
func (s *Service) Create(ctx context.Context, contentType string, data map[string]any, adminID string) (map[string]any, error) {
	ct, ok := s.getSchema(contentType)
	if !ok {
		return nil, ErrNotFound
	}

	if errs := ValidateEntry(ct, data, false); len(errs) > 0 {
		return nil, &ValidationError{Fields: errs}
	}

	entry, err := s.repo.Insert(ctx, tableName(ct.Name), ct.Fields, data, adminID)
	if err != nil {
		return nil, fmt.Errorf("creating %s entry: %w", contentType, err)
	}

	if id, ok := entry["id"].(string); ok {
		s.logAudit(ctx, audit.Event{
			Action:     "entry.create",
			ActorID:    adminID,
			Resource:   contentType,
			ResourceID: id,
		})
	}

	return entry, nil
}

// Update validates and updates an existing content entry.
func (s *Service) Update(ctx context.Context, contentType, id string, data map[string]any, adminID string) (map[string]any, error) {
	ct, ok := s.getSchema(contentType)
	if !ok {
		return nil, ErrNotFound
	}

	if errs := ValidateEntry(ct, data, true); len(errs) > 0 {
		return nil, &ValidationError{Fields: errs}
	}

	entry, err := s.repo.Update(ctx, tableName(ct.Name), ct.Fields, id, data, adminID)
	if err != nil {
		return nil, fmt.Errorf("updating %s entry: %w", contentType, err)
	}

	s.logAudit(ctx, audit.Event{
		Action:     "entry.update",
		ActorID:    adminID,
		Resource:   contentType,
		ResourceID: id,
	})

	return entry, nil
}

// Publish sets an entry's status to 'published'.
func (s *Service) Publish(ctx context.Context, contentType, id, adminID string) (map[string]any, error) {
	ct, ok := s.getSchema(contentType)
	if !ok {
		return nil, ErrNotFound
	}

	entry, err := s.repo.Publish(ctx, tableName(ct.Name), ct.Fields, id, adminID)
	if err != nil {
		return nil, fmt.Errorf("publishing %s entry: %w", contentType, err)
	}

	s.logAudit(ctx, audit.Event{
		Action:     "entry.publish",
		ActorID:    adminID,
		Resource:   contentType,
		ResourceID: id,
	})

	return entry, nil
}
