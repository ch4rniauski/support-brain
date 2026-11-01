package config

import (
	pkgconfig "github.com/ch4rniauski/support-brain/pkg/config"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	HTTP     pkgconfig.HTTPConfig
	Postgres pkgconfig.PostgresConfig
}

func Load() (*Config, error) {
	var cfg Config

	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
