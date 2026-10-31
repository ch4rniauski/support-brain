package config

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	HTTP     HTTPConfig
	Postgres PostgresConfig
	Auth     AuthConfig
}

type HTTPConfig struct {
	Port string `envconfig:"HTTP_PORT" default:"8080"`
}

type PostgresConfig struct {
	Host     string `envconfig:"POSTGRES_HOST" default:"localhost"`
	Port     string `envconfig:"POSTGRES_PORT" default:"5432"`
	User     string `envconfig:"POSTGRES_USER" default:"postgres"`
	Password string `envconfig:"POSTGRES_PASSWORD" default:"postgres"`
	Database string `envconfig:"POSTGRES_DB" default:"support-brain"`
}

type AuthConfig struct {
	JWTSecret string        `envconfig:"AUTH_JWT_SECRET" default:"dev-secret-change-me"`
	TokenTTL  time.Duration `envconfig:"AUTH_TOKEN_TTL" default:"24h"`
}

func Load() (*Config, error) {
	var cfg Config

	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
