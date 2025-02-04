package content

import (
	"context"
	"database/sql"
	"errors"

	"github.com/GyroZepelix/mithril-cms/internal/errs"
	"github.com/GyroZepelix/mithril-cms/internal/storage/persistence"
	"github.com/google/uuid"
)

type Manager interface {
	GetContent(contentId uuid.UUID, ctx context.Context) (persistence.Post, error)
	ListContents(ctx context.Context) ([]persistence.Post, error)
}

type contentManager struct {
	db persistence.Querier
}

func NewManager(db persistence.Querier) Manager {
	return &contentManager{
		db,
	}
}

func (m *contentManager) GetContent(contentId uuid.UUID, ctx context.Context) (persistence.Post, error) {
	content, err := m.db.GetContent(ctx, contentId)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return persistence.Post{}, errs.ErrNotFound
		default:
			return persistence.Post{}, errors.Join(err, errs.ErrInternalServer)
		}
	}
	return content, nil
}

func (m *contentManager) ListContents(ctx context.Context) ([]persistence.Post, error) {
	contents, err := m.db.ListContents(ctx)
	if err != nil {
		return []persistence.Post{}, err
	}
	return contents, nil
}
