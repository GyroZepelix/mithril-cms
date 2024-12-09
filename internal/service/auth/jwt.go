package auth

import (
	"strconv"
	"time"

	"github.com/GyroZepelix/mithril-cms/internal/config"
	"github.com/golang-jwt/jwt"
)

func CreateJWT(userID int32, role string) (string, error) {
	expiration := time.Second * time.Duration(config.Envs.AuthJwtExpirationInSec)
	secret := config.Envs.AuthJwtSecret

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userID":    strconv.FormatInt(int64(userID), 10),
		"role":      role,
		"expiredAt": time.Now().Add(expiration).Unix(),
	})

	tokenString, err := token.SignedString(secret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
