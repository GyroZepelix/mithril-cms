package config

import (
	"os"
	"strconv"
)

type Config struct {
	PublicHost             string
	Port                   string
	DBUser                 string
	DBPassword             string
	DBHost                 string
	DBPort                 string
	DBName                 string
	DBDriver               string
	DBFlags                string
	AuthJwtExpirationInSec int64
	AuthJwtSecret          []byte
}

var Envs = initConfig()

func initConfig() Config {

	return Config{
		PublicHost:             getEnv("HOST", "127.0.0.1"),
		Port:                   getEnv("PORT", "8080"),
		DBUser:                 getEnv("DB_USER", "mithril"),
		DBPassword:             getEnv("DB_PASSWORD", "S3cret"),
		DBHost:                 getEnv("DB_HOST", "127.0.0.1"),
		DBPort:                 getEnv("DB_PORT", "5432"),
		DBName:                 getEnv("DB_NAME", "mithrildb"),
		DBDriver:               getEnv("DB_DRIVER", "postgres"),
		DBFlags:                getEnv("DB_FLAGS", "sslmode=disable"),
		AuthJwtExpirationInSec: getEnvAsInt("JWT_EXP", 3600*24*7),
		AuthJwtSecret:          getEnvAsByteArr("JWT_SECRET", "not-a-secret!"),
	}
}

func ReloadConfig() {
	Envs = Config{
		PublicHost:             getEnv("HOST", "127.0.0.1"),
		Port:                   getEnv("PORT", "8080"),
		DBUser:                 getEnv("DB_USER", "mithril"),
		DBPassword:             getEnv("DB_PASSWORD", "S3cret"),
		DBHost:                 getEnv("DB_HOST", "127.0.0.1"),
		DBPort:                 getEnv("DB_PORT", "5432"),
		DBName:                 getEnv("DB_NAME", "mithrildb"),
		DBDriver:               getEnv("DB_DRIVER", "postgres"),
		DBFlags:                getEnv("DB_FLAGS", "sslmode=disable"),
		AuthJwtExpirationInSec: getEnvAsInt("JWT_EXP", 3600*24*7),
		AuthJwtSecret:          getEnvAsByteArr("JWT_SECRET", "not-a-secret!"),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvAsInt(key string, fallback int64) int64 {
	if value, ok := os.LookupEnv(key); ok {
		i, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fallback
		}

		return i
	}
	return fallback
}

func getEnvAsByteArr(key string, fallback string) []byte {
	if value, ok := os.LookupEnv(key); ok {
		return []byte(value)
	}
	return []byte(fallback)
}
