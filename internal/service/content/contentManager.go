package content

import (
	"context"

	"github.com/GyroZepelix/mithril-cms/internal/storage/persistence"
	"github.com/google/uuid"
)

type Manager interface {
	GetContent(contentId uuid.UUID, ctx context.Context) (persistence.Post, error)
}

type ContentManager struct {
	db *persistence.Queries
}

func NewManager(db *persistence.Queries) Manager {
	return &ContentManager{
		db,
	}
}

func (c *ContentManager) GetContent(contentId uuid.UUID, ctx context.Context) (persistence.Post, error) {
	panic("unimplemented")
}
