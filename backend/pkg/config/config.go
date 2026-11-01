package config

import (
	"strings"
	"time"
)

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

type OutboxConfig struct {
	PollInterval time.Duration `envconfig:"OUTBOX_POLL_INTERVAL" default:"1s"`
	BatchSize    int           `envconfig:"OUTBOX_BATCH_SIZE" default:"100"`
}

func SplitAndTrim(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				result = append(result, part)
			}
		}
	}
	return result
}
