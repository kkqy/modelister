package config

import (
	"errors"
	"os"
	"strings"
)

type Config struct {
	AdminUsername string
	AdminPassword string
	SessionSecret string
	DatabasePath  string
	HTTPAddr      string
}

func Load() (Config, error) {
	cfg := Config{
		AdminUsername: strings.TrimSpace(os.Getenv("APP_ADMIN_USERNAME")),
		AdminPassword: os.Getenv("APP_ADMIN_PASSWORD"),
		SessionSecret: os.Getenv("APP_SESSION_SECRET"),
		DatabasePath:  strings.TrimSpace(os.Getenv("APP_DATABASE_PATH")),
		HTTPAddr:      strings.TrimSpace(os.Getenv("APP_HTTP_ADDR")),
	}
	if cfg.DatabasePath == "" {
		cfg.DatabasePath = "/data/modelister.db"
	}
	if cfg.HTTPAddr == "" {
		cfg.HTTPAddr = ":8080"
	}
	if cfg.AdminUsername == "" || cfg.AdminPassword == "" || cfg.SessionSecret == "" {
		return Config{}, errors.New("APP_ADMIN_USERNAME, APP_ADMIN_PASSWORD and APP_SESSION_SECRET are required")
	}
	return cfg, nil
}
