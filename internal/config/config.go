package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv      string
	HTTPPort    string
	DatabaseURL string
	// JWTKEK is the base64-encoded 32-byte key-encryption key used to encrypt
	// signing keys at rest.
	JWTKEK string
	// TrustedProxies is a comma-separated list of IPs/CIDRs allowed to set
	// X-Forwarded-For. Empty means no proxy is trusted.
	TrustedProxies string

	SMTPHost string
	SMTPPort int
	SMTPUser string
	SMTPPass string
	SMTPFrom string
}

func Load() *Config {
	_ = godotenv.Load() // safe to ignore if .env doesn't exist

	cfg := &Config{
		AppEnv:      getEnv("APP_ENV", "development"),
		HTTPPort:    getEnv("HTTP_PORT", "8082"),
		DatabaseURL:    getEnv("DATABASE_URL", ""),
		JWTKEK:         getEnv("JWT_KEK", ""),
		TrustedProxies: getEnv("TRUSTED_PROXIES", ""),
		SMTPHost:       getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPUser:       getEnv("SMTP_USER", ""),
		SMTPPass:       getEnv("SMTP_PASS", ""),
		SMTPFrom:       getEnv("SMTP_FROM", ""),
	}

	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	if cfg.JWTKEK == "" {
		log.Fatal("JWT_KEK is required (base64-encoded 32 bytes)")
	}

	port, err := strconv.Atoi(getEnv("SMTP_PORT", "587"))
	if err != nil {
		log.Fatalf("SMTP_PORT must be a number: %v", err)
	}
	cfg.SMTPPort = port

	return cfg
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
