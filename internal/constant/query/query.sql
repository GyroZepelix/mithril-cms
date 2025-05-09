-- name: GetUser :one
SELECT * FROM users
WHERE id = $1 LIMIT 1;

-- name: GetUserByUsername :one
SELECT * FROM users
WHERE username = $1 LIMIT 1;

-- name: ListUsers :many
SELECT * FROM users
ORDER BY username;

-- name: CreateUser :one
INSERT INTO users (
    username, 
    email, 
    password
) VALUES (
    $1, $2, $3
)
RETURNING *;

-- name: GetContent :one
SELECT * FROM posts
WHERE id = $1 LIMIT 1;

-- name: ListContents :many
SELECT * FROM posts
ORDER by updated_at;

-- name: ListContentsWithCategories :many
SELECT * FROM post_view
ORDER BY updated_at;


-- name: CreateContent :one
INSERT INTO posts (
    title,
    slug,
    content,
    author_id
) VALUES (
    $1, $2, $3, $4
)
RETURNING *;
