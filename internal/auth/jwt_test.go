package auth

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-secret-key-for-jwt-signing"

func TestCreateAndValidateAccessToken(t *testing.T) {
	adminID := "550e8400-e29b-41d4-a716-446655440000"
	email := "admin@example.com"

	token, err := CreateAccessToken(adminID, email, testSecret)
	if err != nil {
		t.Fatalf("CreateAccessToken: unexpected error: %v", err)
	}
	if token == "" {
		t.Fatal("CreateAccessToken: returned empty token")
	}

	claims, err := ValidateAccessToken(token, testSecret)
	if err != nil {
		t.Fatalf("ValidateAccessToken: unexpected error: %v", err)
	}
	if claims.AdminID() != adminID {
		t.Errorf("AdminID() = %q, want %q", claims.AdminID(), adminID)
	}
	if claims.Subject != adminID {
		t.Errorf("Subject = %q, want %q (should be same as AdminID)", claims.Subject, adminID)
	}
	if claims.Email != email {
		t.Errorf("Email = %q, want %q", claims.Email, email)
	}
	if claims.Issuer != "mithril-cms" {
		t.Errorf("Issuer = %q, want %q", claims.Issuer, "mithril-cms")
	}
}

func TestValidateAccessToken_WrongSecret(t *testing.T) {
	token, err := CreateAccessToken("id", "email@test.com", testSecret)
	if err != nil {
		t.Fatalf("CreateAccessToken: unexpected error: %v", err)
	}

	_, err = ValidateAccessToken(token, "wrong-secret")
	if err == nil {
		t.Fatal("ValidateAccessToken: expected error for wrong secret, got nil")
	}
}

func TestValidateAccessToken_Expired(t *testing.T) {
	// Manually create an expired token.
	now := time.Now().Add(-1 * time.Hour)
	claims := Claims{
		Email: "test@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "id",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)), // expired 45min ago
			Issuer:    "mithril-cms",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("signing expired token: %v", err)
	}

	_, err = ValidateAccessToken(signed, testSecret)
	if err == nil {
		t.Fatal("ValidateAccessToken: expected error for expired token, got nil")
	}
}

func TestValidateAccessToken_Malformed(t *testing.T) {
	_, err := ValidateAccessToken("not-a-jwt", testSecret)
	if err == nil {
		t.Fatal("ValidateAccessToken: expected error for malformed token, got nil")
	}
}

func TestValidateAccessToken_WrongSigningMethod(t *testing.T) {
	// Create a token with a different signing method (none).
	claims := Claims{
		Email: "test@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "id",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	signed, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("signing none-method token: %v", err)
	}

	_, err = ValidateAccessToken(signed, testSecret)
	if err == nil {
		t.Fatal("ValidateAccessToken: expected error for none signing method, got nil")
	}
}

func TestValidatePassword_RuneCount(t *testing.T) {
	tests := []struct {
		name    string
		password string
		wantErr error
	}{
		{
			name:     "8 multibyte runes within limit",
			password: strings.Repeat("\u00e9", 8), // e-acute, 2 bytes each = 16 bytes but 8 runes
			wantErr:  nil,
		},
		{
			name:     "7 multibyte runes too short",
			password: strings.Repeat("\u00e9", 7),
			wantErr:  ErrPasswordTooShort,
		},
		{
			name:     "64 multibyte runes at max",
			password: strings.Repeat("\u00e9", 64),
			wantErr:  nil,
		},
		{
			name:     "65 multibyte runes too long",
			password: strings.Repeat("\u00e9", 65),
			wantErr:  ErrPasswordTooLong,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify our test data has the expected rune count.
			if tt.name == "8 multibyte runes within limit" {
				if n := utf8.RuneCountInString(tt.password); n != 8 {
					t.Fatalf("expected 8 runes, got %d", n)
				}
				if n := len(tt.password); n != 16 {
					t.Fatalf("expected 16 bytes, got %d", n)
				}
			}

			err := validatePassword(tt.password)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("validatePassword: unexpected error: %v", err)
				}
			} else {
				if err != tt.wantErr {
					t.Errorf("validatePassword: got %v, want %v", err, tt.wantErr)
				}
			}
		})
	}
}
