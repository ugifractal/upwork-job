package config

import (
	"errors"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTPPort    int
	APIKey      string
	DBPath      string
	SMTPHost    string
	SMTPPort    int
	SMTPUser    string
	SMTPPass    string
	SMTPFrom    string
	NotifyEmail string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	port, err := strconv.Atoi(getEnv("HTTP_PORT", "8080"))
	if err != nil {
		return nil, errors.New("HTTP_PORT must be a number")
	}
	smtpPort, err := strconv.Atoi(getEnv("SMTP_PORT", "587"))
	if err != nil {
		return nil, errors.New("SMTP_PORT must be a number")
	}

	cfg := &Config{
		HTTPPort:    port,
		APIKey:      os.Getenv("API_KEY"),
		DBPath:      getEnv("DB_PATH", "./jobs.db"),
		SMTPHost:    os.Getenv("SMTP_HOST"),
		SMTPPort:    smtpPort,
		SMTPUser:    os.Getenv("SMTP_USER"),
		SMTPPass:    os.Getenv("SMTP_PASS"),
		SMTPFrom:    os.Getenv("SMTP_FROM"),
		NotifyEmail: os.Getenv("NOTIFY_EMAIL"),
	}

	if cfg.APIKey == "" {
		return nil, errors.New("API_KEY is not set in .env")
	}
	if cfg.SMTPHost == "" || cfg.SMTPUser == "" || cfg.SMTPPass == "" || cfg.SMTPFrom == "" || cfg.NotifyEmail == "" {
		return nil, errors.New("SMTP settings (SMTP_HOST, SMTP_USER, SMTP_PASS, SMTP_FROM, NOTIFY_EMAIL) are not fully set in .env")
	}

	return cfg, nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}