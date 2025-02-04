// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0

package persistence

import (
	"context"

	"github.com/google/uuid"
)

type Querier interface {
	CreateUser(ctx context.Context, arg CreateUserParams) (User, error)
	GetContent(ctx context.Context, id uuid.UUID) (Post, error)
	GetUser(ctx context.Context, id uuid.UUID) (User, error)
	GetUserByUsername(ctx context.Context, username string) (User, error)
	ListContents(ctx context.Context) ([]Post, error)
	ListUsers(ctx context.Context) ([]User, error)
}

var _ Querier = (*Queries)(nil)
