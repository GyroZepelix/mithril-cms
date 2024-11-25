package user

import (
	"context"
	"database/sql"
	"errors"

	"github.com/GyroZepelix/mithril-cms/internal/errs"
	"github.com/GyroZepelix/mithril-cms/internal/logging"
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
			return persistence.User{}, errs.ErrNotFound
		default:
			return persistence.User{}, errors.Join(err, errs.ErrInternalServer)
		}
	}
	return user, nil
}

func (m *Manager) GetUserByUsername(username string, ctx context.Context) (persistence.User, error) {
	user, err := m.DB.GetUserByUsername(ctx, username)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return persistence.User{}, errs.ErrNotFound
		default:
			return persistence.User{}, errors.Join(err, errs.ErrInternalServer)
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

func (m *Manager) CreateUser(username, email, password string, ctx context.Context) (*persistence.User, error) {
	createdUser, err := m.DB.CreateUser(ctx, persistence.CreateUserParams{
		Username: username,
		Email:    email,
		Password: password,
	})
	if err != nil {
		return nil, errs.MapPostgresError(err)
	}
	logging.Infof("User created - Id: %d Username: %s Email: %s",
		createdUser.ID,
		createdUser.Username,
		createdUser.Email,
	)

	return &createdUser, nil
}
