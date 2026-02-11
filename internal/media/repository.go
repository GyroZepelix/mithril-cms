// Package media provides media upload, processing, storage, and serving
// capabilities for the Mithril CMS.
package media

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/GyroZepelix/mithril-cms/internal/database"
)

// ErrNotFound is returned when a media record does not exist.
var ErrNotFound = errors.New("media not found")

// Media represents a stored media file with its metadata.
type Media struct {
	ID           string            `json:"id"`
	Filename     string            `json:"filename"`
	OriginalName string            `json:"original_name"`
	MimeType     string            `json:"mime_type"`
	Size         int64             `json:"size"`
	Width        *int              `json:"width,omitempty"`
	Height       *int              `json:"height,omitempty"`
	Variants     map[string]string `json:"variants"`
	UploadedBy   *string           `json:"uploaded_by,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
}

// Repository handles database operations for media records.
type Repository struct {
	db *database.DB
}

// NewRepository creates a new media Repository.
func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts a new media record. The ID field is populated from the
// database-generated UUID after insertion.
func (r *Repository) Create(ctx context.Context, m *Media) error {
	variantsJSON, err := json.Marshal(m.Variants)
	if err != nil {
		return fmt.Errorf("marshaling variants: %w", err)
	}

	err = r.db.Pool().QueryRow(ctx, `
		INSERT INTO media (filename, original_name, mime_type, size, width, height, variants, uploaded_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at`,
		m.Filename, m.OriginalName, m.MimeType, m.Size, m.Width, m.Height, variantsJSON, m.UploadedBy,
	).Scan(&m.ID, &m.CreatedAt)
	if err != nil {
		return fmt.Errorf("inserting media record: %w", err)
	}
	return nil
}

// GetByID retrieves a media record by its UUID.
func (r *Repository) GetByID(ctx context.Context, id string) (*Media, error) {
	m := &Media{}
	var variantsJSON []byte

	err := r.db.Pool().QueryRow(ctx, `
		SELECT id, filename, original_name, mime_type, size, width, height, variants, uploaded_by, created_at
		FROM media WHERE id = $1`, id,
	).Scan(&m.ID, &m.Filename, &m.OriginalName, &m.MimeType, &m.Size,
		&m.Width, &m.Height, &variantsJSON, &m.UploadedBy, &m.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("querying media by id: %w", err)
	}

	if err := json.Unmarshal(variantsJSON, &m.Variants); err != nil {
		return nil, fmt.Errorf("unmarshaling variants: %w", err)
	}
	return m, nil
}

// GetByFilename retrieves a media record by its unique generated filename.
func (r *Repository) GetByFilename(ctx context.Context, filename string) (*Media, error) {
	m := &Media{}
	var variantsJSON []byte

	err := r.db.Pool().QueryRow(ctx, `
		SELECT id, filename, original_name, mime_type, size, width, height, variants, uploaded_by, created_at
		FROM media WHERE filename = $1`, filename,
	).Scan(&m.ID, &m.Filename, &m.OriginalName, &m.MimeType, &m.Size,
		&m.Width, &m.Height, &variantsJSON, &m.UploadedBy, &m.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("querying media by filename: %w", err)
	}

	if err := json.Unmarshal(variantsJSON, &m.Variants); err != nil {
		return nil, fmt.Errorf("unmarshaling variants: %w", err)
	}
	return m, nil
}

// List retrieves a paginated list of media records ordered by created_at desc.
// Returns the records and the total count.
func (r *Repository) List(ctx context.Context, page, perPage int) ([]*Media, int, error) {
	var total int
	err := r.db.Pool().QueryRow(ctx, `SELECT count(*) FROM media`).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("counting media: %w", err)
	}

	if total == 0 {
		return []*Media{}, 0, nil
	}

	offset := (page - 1) * perPage
	rows, err := r.db.Pool().Query(ctx, `
		SELECT id, filename, original_name, mime_type, size, width, height, variants, uploaded_by, created_at
		FROM media
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("listing media: %w", err)
	}
	defer rows.Close()

	var results []*Media
	for rows.Next() {
		m := &Media{}
		var variantsJSON []byte

		if err := rows.Scan(&m.ID, &m.Filename, &m.OriginalName, &m.MimeType, &m.Size,
			&m.Width, &m.Height, &variantsJSON, &m.UploadedBy, &m.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scanning media row: %w", err)
		}

		if err := json.Unmarshal(variantsJSON, &m.Variants); err != nil {
			return nil, 0, fmt.Errorf("unmarshaling variants: %w", err)
		}
		results = append(results, m)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterating media rows: %w", err)
	}

	return results, total, nil
}

// Delete removes a media record by its UUID. Returns ErrNotFound if the
// record does not exist.
func (r *Repository) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Pool().Exec(ctx, `DELETE FROM media WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting media: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
