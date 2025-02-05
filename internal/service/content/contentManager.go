package content

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"strings"

	"github.com/GyroZepelix/mithril-cms/internal/errs"
	"github.com/GyroZepelix/mithril-cms/internal/logging"
	// "github.com/GyroZepelix/mithril-cms/internal/logging"
	"github.com/GyroZepelix/mithril-cms/internal/storage/persistence"
	"github.com/google/uuid"
)

type Manager interface {
	GetContent(contentId uuid.UUID, ctx context.Context) (persistence.Post, error)
	ListContents(ctx context.Context) ([]persistence.Post, error)
	CreateContent(title, content string, userId uuid.UUID, ctx context.Context) (*persistence.Post, error)
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

func (m *contentManager) CreateContent(title, content string, userId uuid.UUID, ctx context.Context) (*persistence.Post, error) {
	slug := generateSlug(title)
	createdContent, err := m.db.CreateContent(ctx, persistence.CreateContentParams{
		Title:    title,
		Content:  content,
		Slug:     slug,
		AuthorID: userId,
	})
	if err != nil {
		return nil, errs.MapPostgresError(err)
	}
	logging.Infof("Post created - Id: %d Title: %s",
		createdContent.ID,
		createdContent.Title,
	)

	return &createdContent, nil
}

func generateSlug(title string) string {
	slug := strings.ToLower(title)
	slug = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	return slug
}
