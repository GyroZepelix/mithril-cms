package auth

import (
	"fmt"
	"strconv"
	"time"

	"github.com/GyroZepelix/mithril-cms/internal/config"
	"github.com/golang-jwt/jwt"
)

const (
	UserIdKey    string = "userId"
	RoleKey      string = "role"
	ExpiredAtKey string = "expiredAt"
)

func CreateJWT(userID int32, role string) (string, error) {
	expiration := time.Second * time.Duration(config.Envs.AuthJwtExpirationInSec)
	secret := config.Envs.AuthJwtSecret

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		UserIdKey:    strconv.FormatInt(int64(userID), 10),
		RoleKey:      role,
		ExpiredAtKey: time.Now().Add(expiration).Unix(),
	})

	tokenString, err := token.SignedString(secret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func ValidateJWT(t string) (*jwt.Token, error) {
	return jwt.Parse(t, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}

		return []byte(config.Envs.AuthJwtSecret), nil
	})
}
