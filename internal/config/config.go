package config

import (
	"os"
)

type Config struct {
	PublicHost string
	Port       string
	DBUser     string
	DBPassword string
	DBHost     string
	DBPort     string
	DBName     string
	DBDriver   string
	DBFlags    string
}

var Envs = initConfig()

func initConfig() Config {

	return Config{
		PublicHost: getEnv("HOST", "127.0.0.1"),
		Port:       getEnv("PORT", "8080"),
		DBUser:     getEnv("DB_USER", "mithril"),
		DBPassword: getEnv("DB_PASSWORD", "S3cret"),
		DBHost:     getEnv("DB_HOST", "127.0.0.1"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBName:     getEnv("DB_NAME", "mithrildb"),
		DBDriver:   getEnv("DB_DRIVER", "postgres"),
		DBFlags:    getEnv("DB_FLAGS", "sslmode=disable"),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
