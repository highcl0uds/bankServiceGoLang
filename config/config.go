package config

import (
	"errors"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	JWTSecret string

	PGPPublicKey  string
	PGPPrivateKey string

	HMACSecret string

	SMTPHost string
	SMTPPort int
	SMTPUser string
	SMTPPass string

	LogLevel string
}

func Load() (Config, error) {
	godotenv.Load()

	smtpPortStr := getEnv("SMTP_PORT", "587")
	smtpPort, err := strconv.Atoi(smtpPortStr)
	if err != nil {
		return Config{}, errors.New("SMTP_PORT must be a number")
	}

	cfg := Config{
		DBHost:        getEnv("DB_HOST", "localhost"),
		DBPort:        getEnv("DB_PORT", "5432"),
		DBUser:        getEnv("DB_USER", "postgres"),
		DBPassword:    getEnv("DB_PASSWORD", ""),
		DBName:        getEnv("DB_NAME", "bank_db"),
		JWTSecret:     getEnv("JWT_SECRET", ""),
		PGPPublicKey:  getEnv("PGP_PUBLIC_KEY", ""),
		PGPPrivateKey: getEnv("PGP_PRIVATE_KEY", ""),
		HMACSecret:    getEnv("HMAC_SECRET", ""),
		SMTPHost:      getEnv("SMTP_HOST", ""),
		SMTPPort:      smtpPort,
		SMTPUser:      getEnv("SMTP_USER", ""),
		SMTPPass:      getEnv("SMTP_PASS", ""),
		LogLevel:      getEnv("LOG_LEVEL", "info"),
	}

	if cfg.JWTSecret == "" {
		return Config{}, errors.New("JWT_SECRET is required")
	}
	if len(cfg.JWTSecret) < 32 {
		return Config{}, errors.New("JWT_SECRET must be at least 32 characters")
	}
	if cfg.PGPPublicKey == "" {
		return Config{}, errors.New("PGP_PUBLIC_KEY is required")
	}
	if cfg.PGPPrivateKey == "" {
		return Config{}, errors.New("PGP_PRIVATE_KEY is required")
	}
	if cfg.HMACSecret == "" {
		return Config{}, errors.New("HMAC_SECRET is required")
	}

	return cfg, nil
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
