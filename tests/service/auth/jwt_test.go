package auth

import (
	"strconv"
	"testing"

	"github.com/GyroZepelix/mithril-cms/internal/config"
	"github.com/GyroZepelix/mithril-cms/internal/service/auth"
	"github.com/golang-jwt/jwt"
)

func TestCreateJWT(t *testing.T) {
	jwtSecret := "secret"
	t.Setenv("JWT_SECRET", jwtSecret)
	config.ReloadConfig()

	var givenUserId int32 = 1
	givenUserRole := "testrole"

	token, err := auth.CreateJWT(givenUserId, givenUserRole)
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

		if actualUserId, ok := claims[auth.UserIdKey].(string); ok {
			if i, err := strconv.Atoi(actualUserId); i != int(givenUserId) || err != nil {
				t.Errorf("userId should be %d, but its %s", givenUserId, actualUserId)
			}
		} else {
			t.Errorf("userId claim is missing or not a number")
		}

		if actualUserRole := claims["role"]; actualUserRole != givenUserRole {
			t.Errorf("role should be %s, but its %s", givenUserRole, actualUserRole)
		}

	})
}
