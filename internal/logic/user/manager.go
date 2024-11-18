package user

import (
	"context"

	"github.com/GyroZepelix/mithril-cms/internal/storage/persistence"
)

type Manager struct {
	DB *persistence.Queries
}

func NewManager(db *persistence.Queries) Manager {
	return Manager{
		DB: db,
	}
}

func (m *Manager) GetUser(userId int32, ctx context.Context) (persistence.User, error) {
	user, err := m.DB.GetUser(ctx, userId)
	if err != nil {
		return persistence.User{}, err
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
