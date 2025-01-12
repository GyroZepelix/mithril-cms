package user

import (
	"context"
	"database/sql"
	"errors"

	"github.com/GyroZepelix/mithril-cms/internal/errs"
	"github.com/GyroZepelix/mithril-cms/internal/logging"
	"github.com/GyroZepelix/mithril-cms/internal/storage/persistence"
)

type Manager interface {
	GetUser(userId int32, ctx context.Context) (persistence.User, error)
	CreateUser(username, email, password string, ctx context.Context) (*persistence.User, error)
	ListUsers(ctx context.Context) ([]persistence.User, error)
	GetUserByUsername(username string, ctx context.Context) (persistence.User, error)
}

type UserManager struct {
	db *persistence.Queries
}

func NewManager(db *persistence.Queries) Manager {
	return &UserManager{
		db: db,
	}
}

func (m *UserManager) GetUser(userId int32, ctx context.Context) (persistence.User, error) {
	user, err := m.db.GetUser(ctx, userId)
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

func (m *UserManager) GetUserByUsername(username string, ctx context.Context) (persistence.User, error) {
	user, err := m.db.GetUserByUsername(ctx, username)
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

func (m *UserManager) ListUsers(ctx context.Context) ([]persistence.User, error) {
	users, err := m.db.ListUsers(ctx)
	if err != nil {
		return []persistence.User{}, err
	}
	return users, nil
}

func (m *UserManager) CreateUser(username, email, password string, ctx context.Context) (*persistence.User, error) {
	createdUser, err := m.db.CreateUser(ctx, persistence.CreateUserParams{
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
