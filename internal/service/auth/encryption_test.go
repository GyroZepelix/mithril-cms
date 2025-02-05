package auth

import (
	"testing"
)

func TestHashPassword(t *testing.T) {
	t.Parallel()
	hash, err := HashPassword("password")
	if err != nil {
		t.Errorf("error hashing password: %v", err)
	}

	if hash == "" {
		t.Error("expected hash to be not empty")
	}

	if hash == "password" {
		t.Error("expected hash to be different from password")
	}
}

func TestCheckPasswordHash(t *testing.T) {
	t.Parallel()
	hash, err := HashPassword("password")
	if err != nil {
		t.Errorf("error hashing password: %v", err)
	}

	t.Run("Should return true for correct password", func(t *testing.T) {
		t.Parallel()
		password := "password"
		if !CheckPasswordHash(password, hash) {
			t.Error("expected CheckPasswordHash to return true for correct password and hash")
		}
	})

	t.Run("Should return false for incorrect password", func(t *testing.T) {
		t.Parallel()
		password := "12345678"
		if CheckPasswordHash(password, hash) {
			t.Error("expected CheckPasswordHash to return false for incorrect password and hash")
		}
	})
}
