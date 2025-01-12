package content

import (
	"context"

	"github.com/GyroZepelix/mithril-cms/internal/storage/persistence"
)

type Manager interface {
	GetContent(contentId int32, ctx context.Context) (persistence.Post, error)
}

type ContentManager struct {
	db *persistence.Queries
}

func NewManager(db *persistence.Queries) Manager {
	return &ContentManager{
		db,
	}
}

func (c *ContentManager) GetContent(contentId int32, ctx context.Context) (persistence.Post, error) {
	panic("unimplemented")
}
