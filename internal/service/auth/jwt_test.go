package auth

import (
	"testing"

	"github.com/GyroZepelix/mithril-cms/internal/config"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
)

func TestCreateJWT(t *testing.T) {
	jwtSecret := "secret"
	t.Setenv("JWT_SECRET", jwtSecret)
	config.ReloadConfig()

	givenUserId := uuid.MustParse("30bafa1b-a0f2-4e0b-a8b0-697258145f69")
	givenUserRole := "testrole"

	token, err := CreateJWT(givenUserId, givenUserRole)
	if err != nil {
		t.Errorf("error creating JWT: %v", err)
	}

	t.Run("should not be empty", func(t *testing.T) {
		if token == "" {
			t.Error("expected token to be not empty")
		}
	})

	t.Run("should be valid and have correct claims", func(t *testing.T) {
		parsedToken, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})
		if err != nil {
			t.Errorf("error parsing token: %v", err)
		}

		claims, ok := parsedToken.Claims.(jwt.MapClaims)
		if !ok || !parsedToken.Valid {
			t.Errorf("error extracting claims from token")
		}

		if actualUserIdParam, ok := claims[UserIdKey].(string); ok {
			actualUserId := uuid.MustParse(actualUserIdParam)
			if actualUserId != givenUserId {
				t.Errorf("userId should be %d, but its %s", givenUserId, actualUserIdParam)
			}
		} else {
			t.Errorf("userId claim is missing or not a number")
		}

		if actualUserRole := claims["role"]; actualUserRole != givenUserRole {
			t.Errorf("role should be %s, but its %s", givenUserRole, actualUserRole)
		}

	})
}
