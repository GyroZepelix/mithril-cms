// Package config provides environment-based configuration for the Mithril CMS server.
package config

import (
	"log/slog"
	"os"
	"strconv"
)

// Config holds all configuration values for the Mithril CMS application.
// Values are loaded from environment variables with the MITHRIL_ prefix.
type Config struct {
	// Port is the HTTP server port. Default: 8080.
	Port int

	// DatabaseURL is the PostgreSQL connection string.
	// Example: postgres://user:pass@localhost:5432/mithril?sslmode=disable
	DatabaseURL string

	// SchemaDir is the path to the directory containing YAML schema files. Default: ./schema
	SchemaDir string

	// MediaDir is the path to the directory for media file storage. Default: ./media
	MediaDir string

	// JWTSecret is the secret key used for signing JWT access tokens.
	JWTSecret string

	// DevMode enables development features such as auto-applying breaking schema changes
	// and proxying the admin SPA to the Vite dev server. Default: false.
	DevMode bool

	// AdminEmail is the email for the initial admin user, required on first run.
	AdminEmail string

	// AdminPassword is the password for the initial admin user, required on first run.
	AdminPassword string
}

// Load reads configuration from environment variables and returns a Config
// with sensible defaults applied for optional values.
func Load() *Config {
	return &Config{
		Port:          getEnvInt("MITHRIL_PORT", 8080),
		DatabaseURL:   getEnv("MITHRIL_DATABASE_URL", ""),
		SchemaDir:     getEnv("MITHRIL_SCHEMA_DIR", "./schema"),
		MediaDir:      getEnv("MITHRIL_MEDIA_DIR", "./media"),
		JWTSecret:     getEnv("MITHRIL_JWT_SECRET", ""),
		DevMode:       getEnvBool("MITHRIL_DEV_MODE", false),
		AdminEmail:    getEnv("MITHRIL_ADMIN_EMAIL", ""),
		AdminPassword: getEnv("MITHRIL_ADMIN_PASSWORD", ""),
	}
}

// getEnv returns the value of the environment variable named by key,
// or the provided default if the variable is unset or empty.
func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// getEnvInt returns the value of the environment variable named by key
// parsed as an integer, or the provided default if the variable is unset,
// empty, or not a valid integer.
func getEnvInt(key string, defaultVal int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		slog.Warn("invalid integer for env var, using default",
			"key", key,
			"value", val,
			"default", defaultVal,
			"error", err,
		)
		return defaultVal
	}
	return n
}

// getEnvBool returns the value of the environment variable named by key
// parsed as a boolean, or the provided default if the variable is unset,
// empty, or not a valid boolean.
func getEnvBool(key string, defaultVal bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		slog.Warn("invalid boolean for env var, using default",
			"key", key,
			"value", val,
			"default", defaultVal,
			"error", err,
		)
		return defaultVal
	}
	return b
}
