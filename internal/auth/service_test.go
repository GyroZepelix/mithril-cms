package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
)

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  error
	}{
		{name: "valid 8 chars", password: "12345678", wantErr: nil},
		{name: "valid 64 chars", password: string(make([]byte, 64)), wantErr: nil},
		{name: "too short", password: "1234567", wantErr: ErrPasswordTooShort},
		{name: "empty", password: "", wantErr: ErrPasswordTooShort},
		{name: "too long 65 chars", password: string(make([]byte, 65)), wantErr: ErrPasswordTooLong},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For the "valid 64 chars" and "too long" cases, replace zero bytes
			// with 'a' so argon2id can handle them if needed.
			pw := tt.password
			if len(pw) > 10 {
				b := []byte(pw)
				for i := range b {
					if b[i] == 0 {
						b[i] = 'a'
					}
				}
				pw = string(b)
			}

			err := validatePassword(pw)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("validatePassword(%q): unexpected error: %v", pw, err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("validatePassword(%q): got %v, want %v", pw, err, tt.wantErr)
			}
		})
	}
}

func TestHashToken(t *testing.T) {
	token := "abc123"
	h := sha256.Sum256([]byte(token))
	want := hex.EncodeToString(h[:])

	got := hashToken(token)
	if got != want {
		t.Errorf("hashToken(%q) = %q, want %q", token, got, want)
	}

	// Different tokens produce different hashes.
	got2 := hashToken("different")
	if got == got2 {
		t.Error("hashToken: different inputs should produce different hashes")
	}
}

func TestHashAndVerifyPassword(t *testing.T) {
	svc := &Service{}
	password := "securepassword123"

	hash, err := svc.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword: unexpected error: %v", err)
	}
	if hash == "" {
		t.Fatal("HashPassword: returned empty hash")
	}

	// Correct password.
	match, err := svc.VerifyPassword(hash, password)
	if err != nil {
		t.Fatalf("VerifyPassword: unexpected error: %v", err)
	}
	if !match {
		t.Error("VerifyPassword: expected match for correct password")
	}

	// Wrong password.
	match, err = svc.VerifyPassword(hash, "wrongpassword")
	if err != nil {
		t.Fatalf("VerifyPassword: unexpected error: %v", err)
	}
	if match {
		t.Error("VerifyPassword: expected no match for wrong password")
	}
}
