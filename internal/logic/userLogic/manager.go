package userLogic

import (
	"context"
	"database/sql"
	"errors"

	"github.com/GyroZepelix/mithril-cms/internal/storage/persistence"
)

type Manager struct {
	DB *persistence.Queries
}

func NewManager(db *persistence.Queries) *Manager {
	return &Manager{
		DB: db,
	}
}

func (m *Manager) GetUser(userId int32, ctx context.Context) (persistence.User, error) {
	user, err := m.DB.GetUser(ctx, userId)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return persistence.User{}, ErrNotFound
		default:
			return persistence.User{}, errors.Join(err, ErrInternalServer)
		}
	}
	return user, nil
}

func (m *Manager) ListUsers(ctx context.Context) ([]persistence.User, error) {
	users, err := m.DB.ListUsers(ctx)
	if err != nil {
		return []persistence.User{}, err
	}
	return users, nil
}
