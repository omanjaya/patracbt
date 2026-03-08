package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

type Config struct {
	App      AppConfig
	DB       DBConfig
	Redis    RedisConfig
	JWT      JWTConfig
	MinIO    MinIOConfig
	AI       AIConfig
	CORS     CORSConfig
	HashID   HashIDConfig
}

type HashIDConfig struct {
	Salt      string
	MinLength int
}

type AppConfig struct {
	Env           string
	Port          string
	FlushInterval time.Duration
}

type DBConfig struct {
	Host        string
	Port        string
	Database    string
	Username    string
	Password    string
	MaxOpenConn int
	MaxIdleConn int
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type JWTConfig struct {
	AccessSecret  string
	RefreshSecret string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
}

type MinIOConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

type AIConfig struct {
	APIURL    string
	APIKey    string
	APIHeader string
}

type CORSConfig struct {
	AllowedOrigins string
}

func Load() *Config {
	return &Config{
		App: AppConfig{
			Env:           getEnv("APP_ENV", "development"),
			Port:          getEnv("APP_PORT", "8080"),
			FlushInterval: time.Duration(getEnvInt("FLUSH_INTERVAL_SECONDS", 5)) * time.Second,
		},
		DB: DBConfig{
			Host:        getEnv("DB_HOST", "localhost"),
			Port:        getEnv("DB_PORT", "5432"),
			Database:    getEnv("DB_DATABASE", "cbt_patra"),
			Username:    getEnv("DB_USERNAME", "cbt_user"),
			Password:    getEnv("DB_PASSWORD", "cbt_password"),
			MaxOpenConn: getEnvInt("DB_MAX_OPEN", 80),
			MaxIdleConn: getEnvInt("DB_MAX_IDLE", 20),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			AccessSecret:  getEnv("JWT_ACCESS_SECRET", "dev-access-secret"),
			RefreshSecret: getEnv("JWT_REFRESH_SECRET", "dev-refresh-secret"),
			AccessTTL:     time.Duration(getEnvInt("JWT_ACCESS_TTL", 900)) * time.Second,
			RefreshTTL:    time.Duration(getEnvInt("JWT_REFRESH_TTL", 604800)) * time.Second,
		},
		MinIO: MinIOConfig{
			Endpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
			AccessKey: getEnv("MINIO_ACCESS_KEY", "minioadmin"),
			SecretKey: getEnv("MINIO_SECRET_KEY", "minioadmin123"),
			Bucket:    getEnv("MINIO_BUCKET", "cbt-patra"),
			UseSSL:    getEnv("MINIO_USE_SSL", "false") == "true",
		},
		AI: AIConfig{
			APIURL:    getEnv("AI_API_URL", ""),
			APIKey:    getEnv("AI_API_KEY", ""),
			APIHeader: getEnv("AI_API_HEADER", "Authorization"),
		},
		CORS: CORSConfig{
			AllowedOrigins: getEnv("ALLOWED_ORIGINS", "http://localhost:5173"),
		},
		HashID: HashIDConfig{
			Salt:      getEnv("HASHID_SALT", "patra-default-salt"),
			MinLength: getEnvInt("HASHID_MIN_LENGTH", 8),
		},
	}
}

// jwtDefaultSecrets lists known default/placeholder JWT secrets that must not be
// used in production.
var jwtDefaultSecrets = map[string]bool{
	"dev-access-secret":  true,
	"dev-refresh-secret": true,
	"secret":             true,
	"change-me":          true,
}

// Validate checks the loaded configuration for security issues.
// In production it rejects insecure defaults; in all environments it warns
// about weak JWT secrets.
func (c *Config) Validate() {
	isProduction := c.App.Env == "production"

	// --- JWT secret validation ---
	if jwtDefaultSecrets[c.JWT.AccessSecret] || jwtDefaultSecrets[c.JWT.RefreshSecret] {
		if isProduction {
			log.Fatalf("[CONFIG] FATAL: JWT secrets must not use default values in production. " +
				"Set JWT_ACCESS_SECRET and JWT_REFRESH_SECRET environment variables.")
		}
	}

	// Enforce minimum 32 chars in all environments; 64 chars in production.
	accessLen := len(c.JWT.AccessSecret)
	refreshLen := len(c.JWT.RefreshSecret)

	if accessLen < 32 {
		log.Fatalf("[CONFIG] FATAL: JWT_ACCESS_SECRET is only %d chars — minimum 32 chars required.", accessLen)
	}
	if refreshLen < 32 {
		log.Fatalf("[CONFIG] FATAL: JWT_REFRESH_SECRET is only %d chars — minimum 32 chars required.", refreshLen)
	}

	if accessLen < 64 {
		msg := fmt.Sprintf("[CONFIG] WARNING: JWT_ACCESS_SECRET is only %d chars — recommend at least 64 chars for security.", accessLen)
		if isProduction {
			log.Fatalf("[CONFIG] FATAL: %s", msg)
		}
		log.Println(msg)
	}

	if refreshLen < 64 {
		msg := fmt.Sprintf("[CONFIG] WARNING: JWT_REFRESH_SECRET is only %d chars — recommend at least 64 chars for security.", refreshLen)
		if isProduction {
			log.Fatalf("[CONFIG] FATAL: %s", msg)
		}
		log.Println(msg)
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return fallback
}
