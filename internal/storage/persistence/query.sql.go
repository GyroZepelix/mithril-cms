// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: query.sql

package persistence

import (
	"context"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

const createUser = `-- name: CreateUser :one
INSERT INTO users (
    username, 
    email, 
    password
) VALUES (
    $1, $2, $3
)
RETURNING id, username, email, password, role, created_at, posts
`

type CreateUserParams struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (q *Queries) CreateUser(ctx context.Context, arg CreateUserParams) (User, error) {
	row := q.db.QueryRowContext(ctx, createUser, arg.Username, arg.Email, arg.Password)
	var i User
	err := row.Scan(
		&i.ID,
		&i.Username,
		&i.Email,
		&i.Password,
		&i.Role,
		&i.CreatedAt,
		pq.Array(&i.Posts),
	)
	return i, err
}

const getContent = `-- name: GetContent :one
SELECT id, title, slug, content, author_id, status, created_at, updated_at, published_at FROM posts
WHERE id = $1 LIMIT 1
`

func (q *Queries) GetContent(ctx context.Context, id uuid.UUID) (Post, error) {
	row := q.db.QueryRowContext(ctx, getContent, id)
	var i Post
	err := row.Scan(
		&i.ID,
		&i.Title,
		&i.Slug,
		&i.Content,
		&i.AuthorID,
		&i.Status,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.PublishedAt,
	)
	return i, err
}

const getUser = `-- name: GetUser :one
SELECT id, username, email, password, role, created_at, posts FROM users
WHERE id = $1 LIMIT 1
`

func (q *Queries) GetUser(ctx context.Context, id uuid.UUID) (User, error) {
	row := q.db.QueryRowContext(ctx, getUser, id)
	var i User
	err := row.Scan(
		&i.ID,
		&i.Username,
		&i.Email,
		&i.Password,
		&i.Role,
		&i.CreatedAt,
		pq.Array(&i.Posts),
	)
	return i, err
}

const getUserByUsername = `-- name: GetUserByUsername :one
SELECT id, username, email, password, role, created_at, posts FROM users
WHERE username = $1 LIMIT 1
`

func (q *Queries) GetUserByUsername(ctx context.Context, username string) (User, error) {
	row := q.db.QueryRowContext(ctx, getUserByUsername, username)
	var i User
	err := row.Scan(
		&i.ID,
		&i.Username,
		&i.Email,
		&i.Password,
		&i.Role,
		&i.CreatedAt,
		pq.Array(&i.Posts),
	)
	return i, err
}

const listContents = `-- name: ListContents :many
SELECT id, title, slug, content, author_id, status, created_at, updated_at, published_at FROM posts
ORDER by updated_at
`

func (q *Queries) ListContents(ctx context.Context) ([]Post, error) {
	rows, err := q.db.QueryContext(ctx, listContents)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Post
	for rows.Next() {
		var i Post
		if err := rows.Scan(
			&i.ID,
			&i.Title,
			&i.Slug,
			&i.Content,
			&i.AuthorID,
			&i.Status,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.PublishedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listUsers = `-- name: ListUsers :many
SELECT id, username, email, password, role, created_at, posts FROM users
ORDER BY username
`

func (q *Queries) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := q.db.QueryContext(ctx, listUsers)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []User
	for rows.Next() {
		var i User
		if err := rows.Scan(
			&i.ID,
			&i.Username,
			&i.Email,
			&i.Password,
			&i.Role,
			&i.CreatedAt,
			pq.Array(&i.Posts),
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
