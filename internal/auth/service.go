package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"
	"unicode/utf8"

	"github.com/alexedwards/argon2id"
	"github.com/jackc/pgx/v5"
)

const (
	refreshTokenExpiry = 7 * 24 * time.Hour // 7 days
	refreshTokenBytes  = 32
	minPasswordLength  = 8
	maxPasswordLength  = 64
)

// Sentinel errors for authentication failures.
var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidToken       = errors.New("invalid or expired refresh token")
	ErrPasswordTooShort   = fmt.Errorf("password must be at least %d characters", minPasswordLength)
	ErrPasswordTooLong    = fmt.Errorf("password must be at most %d characters", maxPasswordLength)
)

// Service provides authentication business logic including password hashing,
// JWT token creation, and refresh token management.
type Service struct {
	repo      *Repository
	jwtSecret string
}

// NewService creates a new auth Service with the given repository and JWT signing secret.
func NewService(repo *Repository, jwtSecret string) *Service {
	return &Service{
		repo:      repo,
		jwtSecret: jwtSecret,
	}
}

// EnsureAdmin creates the initial admin user if one with the given email does
// not yet exist. Uses INSERT ... ON CONFLICT to avoid TOCTOU races between
// checking and creating.
func (s *Service) EnsureAdmin(ctx context.Context, email, password string) error {
	if err := validatePassword(password); err != nil {
		return fmt.Errorf("initial admin password: %w", err)
	}

	hash, err := s.HashPassword(password)
	if err != nil {
		return fmt.Errorf("hashing initial admin password: %w", err)
	}

	admin, err := s.repo.CreateAdmin(ctx, email, hash)
	if err != nil {
		return fmt.Errorf("creating initial admin: %w", err)
	}

	slog.Info("initial admin ensured", "email", admin.Email, "id", admin.ID)
	return nil
}

// HashPassword hashes a password using Argon2id with secure default parameters.
func (s *Service) HashPassword(password string) (string, error) {
	hash, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return "", fmt.Errorf("hashing password: %w", err)
	}
	return hash, nil
}

// VerifyPassword checks whether the given plain-text password matches the
// provided Argon2id hash.
func (s *Service) VerifyPassword(hash, password string) (bool, error) {
	match, err := argon2id.ComparePasswordAndHash(password, hash)
	if err != nil {
		return false, fmt.Errorf("verifying password: %w", err)
	}
	return match, nil
}

// Login validates the given credentials and, on success, returns a signed JWT
// access token and a raw refresh token (to be sent to the client in a cookie).
// The refresh token's SHA256 hash is stored in the database.
func (s *Service) Login(ctx context.Context, email, password string) (accessToken string, refreshToken string, err error) {
	admin, err := s.repo.GetAdminByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", ErrInvalidCredentials
		}
		return "", "", fmt.Errorf("looking up admin: %w", err)
	}

	match, err := s.VerifyPassword(admin.PasswordHash, password)
	if err != nil {
		return "", "", fmt.Errorf("verifying password: %w", err)
	}
	if !match {
		return "", "", ErrInvalidCredentials
	}

	accessToken, err = CreateAccessToken(admin.ID, admin.Email, s.jwtSecret)
	if err != nil {
		return "", "", err
	}

	refreshToken, err = s.createRefreshToken(ctx, admin.ID)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

// Refresh validates the given raw refresh token, atomically rotates it (deletes
// the old token and creates a new one in a single transaction), and returns new
// access and refresh tokens. If the old token was already consumed (replay
// attack), all sessions for the admin are revoked.
func (s *Service) Refresh(ctx context.Context, oldToken string) (accessToken string, newRefreshToken string, err error) {
	oldTokenHash := hashToken(oldToken)

	stored, err := s.repo.GetRefreshToken(ctx, oldTokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", ErrInvalidToken
		}
		return "", "", fmt.Errorf("looking up refresh token: %w", err)
	}

	// Check expiry.
	if time.Now().After(stored.ExpiresAt) {
		// Clean up the expired token.
		_ = s.repo.DeleteRefreshToken(ctx, oldTokenHash)
		return "", "", ErrInvalidToken
	}

	// Generate the new token material before the transaction.
	raw := make([]byte, refreshTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", "", fmt.Errorf("generating refresh token: %w", err)
	}
	newToken := hex.EncodeToString(raw)
	newTokenHash := hashToken(newToken)
	expiresAt := time.Now().Add(refreshTokenExpiry)

	// Atomic rotation: delete old + insert new in one transaction.
	if err := s.repo.RotateRefreshToken(ctx, oldTokenHash, newTokenHash, stored.AdminID, expiresAt); err != nil {
		if errors.Is(err, ErrTokenAlreadyUsed) {
			slog.Warn("refresh token replay detected, all sessions revoked",
				"admin_id", stored.AdminID)
			return "", "", ErrInvalidToken
		}
		return "", "", fmt.Errorf("rotating refresh token: %w", err)
	}

	// Look up the admin to get their email for the JWT claims.
	admin, err := s.repo.GetAdminByID(ctx, stored.AdminID)
	if err != nil {
		return "", "", fmt.Errorf("looking up admin for refresh: %w", err)
	}

	accessToken, err = CreateAccessToken(admin.ID, admin.Email, s.jwtSecret)
	if err != nil {
		return "", "", err
	}

	return accessToken, newToken, nil
}

// Logout deletes the refresh token identified by the given raw token value
// from the database. It is not an error if the token does not exist.
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := hashToken(refreshToken)
	if err := s.repo.DeleteRefreshToken(ctx, tokenHash); err != nil {
		return fmt.Errorf("deleting refresh token on logout: %w", err)
	}
	return nil
}

// createRefreshToken generates a cryptographically random token, stores its
// SHA256 hash in the database, and returns the raw hex-encoded token.
func (s *Service) createRefreshToken(ctx context.Context, adminID string) (string, error) {
	raw := make([]byte, refreshTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generating refresh token: %w", err)
	}

	token := hex.EncodeToString(raw)
	tokenHash := hashToken(token)
	expiresAt := time.Now().Add(refreshTokenExpiry)

	if err := s.repo.CreateRefreshToken(ctx, adminID, tokenHash, expiresAt); err != nil {
		return "", err
	}
	return token, nil
}

// hashToken returns the hex-encoded SHA256 hash of a raw token string.
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// validatePassword checks that the password meets the length policy. Uses rune
// count rather than byte length so that multi-byte UTF-8 characters are counted
// correctly.
func validatePassword(password string) error {
	n := utf8.RuneCountInString(password)
	if n < minPasswordLength {
		return ErrPasswordTooShort
	}
	if n > maxPasswordLength {
		return ErrPasswordTooLong
	}
	return nil
}
