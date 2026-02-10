// Package auth provides authentication and authorization for the Mithril CMS
// admin interface, including JWT access tokens, rotating refresh tokens, and
// Argon2id password hashing.
package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/GyroZepelix/mithril-cms/internal/database"
)

// ErrTokenAlreadyUsed is returned by RotateRefreshToken when the old token has
// already been consumed, indicating a potential replay attack. When this occurs
// all refresh tokens for the affected admin are revoked as a security measure.
var ErrTokenAlreadyUsed = errors.New("refresh token already used")

// Admin represents an admin user row from the admins table.
type Admin struct {
	ID           string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

// RefreshToken represents a refresh token row from the refresh_tokens table.
type RefreshToken struct {
	ID        string
	AdminID   string
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// Repository provides database access for authentication operations.
type Repository struct {
	db *database.DB
}

// NewRepository creates a new auth Repository backed by the given database.
func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

// GetAdminByEmail returns the admin with the given email, or an error wrapping
// pgx.ErrNoRows if no admin exists with that email.
func (r *Repository) GetAdminByEmail(ctx context.Context, email string) (*Admin, error) {
	row := r.db.Pool().QueryRow(ctx,
		`SELECT id, email, password_hash, created_at FROM admins WHERE email = $1`,
		email,
	)

	var a Admin
	if err := row.Scan(&a.ID, &a.Email, &a.PasswordHash, &a.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("admin not found: %w", err)
		}
		return nil, fmt.Errorf("querying admin by email: %w", err)
	}
	return &a, nil
}

// GetAdminByID returns the admin with the given UUID, or an error wrapping
// pgx.ErrNoRows if no admin exists with that ID.
func (r *Repository) GetAdminByID(ctx context.Context, adminID string) (*Admin, error) {
	row := r.db.Pool().QueryRow(ctx,
		`SELECT id, email, password_hash, created_at FROM admins WHERE id = $1`,
		adminID,
	)

	var a Admin
	if err := row.Scan(&a.ID, &a.Email, &a.PasswordHash, &a.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("admin not found: %w", err)
		}
		return nil, fmt.Errorf("querying admin by id: %w", err)
	}
	return &a, nil
}

// CreateAdmin inserts a new admin with the given email and password hash. If an
// admin with the same email already exists, this is treated as success and the
// existing admin is returned. This eliminates the TOCTOU race in EnsureAdmin.
func (r *Repository) CreateAdmin(ctx context.Context, email, passwordHash string) (*Admin, error) {
	row := r.db.Pool().QueryRow(ctx,
		`INSERT INTO admins (email, password_hash) VALUES ($1, $2)
		 ON CONFLICT (email) DO NOTHING
		 RETURNING id, email, password_hash, created_at`,
		email, passwordHash,
	)

	var a Admin
	if err := row.Scan(&a.ID, &a.Email, &a.PasswordHash, &a.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// ON CONFLICT DO NOTHING returns no rows — admin already exists.
			// Fetch the existing admin to return consistent data.
			return r.GetAdminByEmail(ctx, email)
		}
		return nil, fmt.Errorf("creating admin: %w", err)
	}
	return &a, nil
}

// CountAdmins returns the total number of admin users in the database.
func (r *Repository) CountAdmins(ctx context.Context) (int, error) {
	var count int
	if err := r.db.Pool().QueryRow(ctx, `SELECT COUNT(*) FROM admins`).Scan(&count); err != nil {
		return 0, fmt.Errorf("counting admins: %w", err)
	}
	return count, nil
}

// CreateRefreshToken stores a new refresh token hash with the given admin ID
// and expiration time.
func (r *Repository) CreateRefreshToken(ctx context.Context, adminID, tokenHash string, expiresAt time.Time) error {
	_, err := r.db.Pool().Exec(ctx,
		`INSERT INTO refresh_tokens (admin_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		adminID, tokenHash, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("creating refresh token: %w", err)
	}
	return nil
}

// GetRefreshToken looks up a refresh token by its SHA256 hash. Returns an
// error wrapping pgx.ErrNoRows if no matching token exists.
func (r *Repository) GetRefreshToken(ctx context.Context, tokenHash string) (*RefreshToken, error) {
	row := r.db.Pool().QueryRow(ctx,
		`SELECT id, admin_id, token_hash, expires_at, created_at
		 FROM refresh_tokens WHERE token_hash = $1`,
		tokenHash,
	)

	var t RefreshToken
	if err := row.Scan(&t.ID, &t.AdminID, &t.TokenHash, &t.ExpiresAt, &t.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("refresh token not found: %w", err)
		}
		return nil, fmt.Errorf("querying refresh token: %w", err)
	}
	return &t, nil
}

// RotateRefreshToken atomically deletes the old refresh token and inserts a new
// one within a single database transaction. If the old token has already been
// consumed (0 rows deleted), all tokens for the admin are revoked as a security
// measure and ErrTokenAlreadyUsed is returned.
func (r *Repository) RotateRefreshToken(ctx context.Context, oldTokenHash, newTokenHash, adminID string, expiresAt time.Time) error {
	tx, err := r.db.Pool().Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning refresh token rotation tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback after commit is harmless

	// Delete the old token and verify it existed.
	tag, err := tx.Exec(ctx,
		`DELETE FROM refresh_tokens WHERE token_hash = $1 AND admin_id = $2`,
		oldTokenHash, adminID,
	)
	if err != nil {
		return fmt.Errorf("deleting old refresh token: %w", err)
	}

	if tag.RowsAffected() == 0 {
		// The old token was already consumed — possible replay attack.
		// Revoke all sessions for this admin as a security measure.
		if _, err := tx.Exec(ctx,
			`DELETE FROM refresh_tokens WHERE admin_id = $1`,
			adminID,
		); err != nil {
			return fmt.Errorf("revoking all tokens after replay: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("committing replay revocation: %w", err)
		}
		return ErrTokenAlreadyUsed
	}

	// Insert the new token.
	if _, err := tx.Exec(ctx,
		`INSERT INTO refresh_tokens (admin_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		adminID, newTokenHash, expiresAt,
	); err != nil {
		return fmt.Errorf("inserting new refresh token: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing refresh token rotation: %w", err)
	}
	return nil
}

// DeleteRefreshToken removes the refresh token with the given hash from the
// database. It is not an error if no matching token exists.
func (r *Repository) DeleteRefreshToken(ctx context.Context, tokenHash string) error {
	_, err := r.db.Pool().Exec(ctx,
		`DELETE FROM refresh_tokens WHERE token_hash = $1`,
		tokenHash,
	)
	if err != nil {
		return fmt.Errorf("deleting refresh token: %w", err)
	}
	return nil
}

// DeleteAllForAdmin removes all refresh tokens belonging to the given admin.
// This is used for security (suspected token reuse) and for implementing a
// "logout everywhere" feature.
func (r *Repository) DeleteAllForAdmin(ctx context.Context, adminID string) error {
	_, err := r.db.Pool().Exec(ctx,
		`DELETE FROM refresh_tokens WHERE admin_id = $1`,
		adminID,
	)
	if err != nil {
		return fmt.Errorf("deleting all tokens for admin: %w", err)
	}
	return nil
}

// DeleteExpiredTokens removes all refresh tokens that have passed their
// expiration time. This can be called periodically for cleanup.
func (r *Repository) DeleteExpiredTokens(ctx context.Context) error {
	_, err := r.db.Pool().Exec(ctx,
		`DELETE FROM refresh_tokens WHERE expires_at < now()`,
	)
	if err != nil {
		return fmt.Errorf("deleting expired tokens: %w", err)
	}
	return nil
}
