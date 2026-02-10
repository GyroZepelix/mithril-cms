package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const accessTokenExpiry = 15 * time.Minute

// Claims holds the JWT claims for an access token. The admin ID is stored in
// the standard "sub" (Subject) field of RegisteredClaims, and email is an
// additional custom claim.
type Claims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

// AdminID returns the authenticated admin's UUID from the JWT subject claim.
func (c *Claims) AdminID() string { return c.Subject }

// CreateAccessToken creates a signed JWT access token with the given admin ID
// as subject, email as a custom claim, and a 15-minute expiry. The token is
// signed with HMAC-SHA256.
func CreateAccessToken(adminID, email, secret string) (string, error) {
	now := time.Now()
	claims := Claims{
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   adminID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenExpiry)),
			Issuer:    "mithril-cms",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("signing access token: %w", err)
	}
	return signed, nil
}

// ValidateAccessToken parses and validates the given JWT string using the
// provided HMAC secret. It returns the extracted Claims on success, or an
// error if the token is malformed, expired, or signed with the wrong key.
func ValidateAccessToken(tokenString, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("parsing access token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid access token claims")
	}

	return claims, nil
}
