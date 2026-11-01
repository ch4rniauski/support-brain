package config

import (
	"github.com/ch4rniauski/support-brain/pkg/auth"
	pkgconfig "github.com/ch4rniauski/support-brain/pkg/config"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	HTTP     pkgconfig.HTTPConfig
	Postgres pkgconfig.PostgresConfig
	Auth     auth.Config
	Kafka    KafkaConfig
	Outbox   pkgconfig.OutboxConfig
}

type KafkaConfig struct {
	Brokers     []string `envconfig:"KAFKA_BROKERS" default:"localhost:9092"`
	QAPairTopic string   `envconfig:"KAFKA_QA_PAIR_TOPIC" default:"qa-pairs"`
}

func Load() (*Config, error) {
	var cfg Config

	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}

	cfg.Kafka.Brokers = pkgconfig.SplitAndTrim(cfg.Kafka.Brokers)

	return &cfg, nil
}
