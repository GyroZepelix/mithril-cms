package content

import (
	"context"
	"fmt"

	"github.com/GyroZepelix/mithril-cms/internal/schema"
	"github.com/GyroZepelix/mithril-cms/internal/server"
)

// Service implements the business logic for content CRUD operations.
type Service struct {
	repo    *Repository
	schemas map[string]schema.ContentType
}

// NewService creates a new content Service.
func NewService(repo *Repository, schemas map[string]schema.ContentType) *Service {
	return &Service{
		repo:    repo,
		schemas: schemas,
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
	ct, ok := s.schemas[contentType]
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
	ct, ok := s.schemas[contentType]
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
	ct, ok := s.schemas[contentType]
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

	return entry, nil
}

// Update validates and updates an existing content entry.
func (s *Service) Update(ctx context.Context, contentType, id string, data map[string]any, adminID string) (map[string]any, error) {
	ct, ok := s.schemas[contentType]
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

	return entry, nil
}

// Publish sets an entry's status to 'published'.
func (s *Service) Publish(ctx context.Context, contentType, id, adminID string) (map[string]any, error) {
	ct, ok := s.schemas[contentType]
	if !ok {
		return nil, ErrNotFound
	}

	entry, err := s.repo.Publish(ctx, tableName(ct.Name), ct.Fields, id, adminID)
	if err != nil {
		return nil, fmt.Errorf("publishing %s entry: %w", contentType, err)
	}

	return entry, nil
}
