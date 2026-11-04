package config

import (
	"time"

	"github.com/ch4rniauski/support-brain/pkg/auth"
	pkgconfig "github.com/ch4rniauski/support-brain/pkg/config"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	HTTP     pkgconfig.HTTPConfig
	Postgres pkgconfig.PostgresConfig
	Auth     auth.Config
	Kafka    KafkaConfig
	Ollama   OllamaConfig
	Consumer ConsumerConfig
	Outbox   pkgconfig.OutboxConfig
}

type KafkaConfig struct {
	Brokers        []string `envconfig:"KAFKA_BROKERS" default:"localhost:9092"`
	QAPairTopic    string   `envconfig:"KAFKA_QA_PAIR_TOPIC" default:"qa-pairs"`
	EmbeddingTopic string   `envconfig:"KAFKA_EMBEDDING_TOPIC" default:"embeddings"`
	ConsumerGroup  string   `envconfig:"KAFKA_EMBEDDING_GROUP" default:"embedding-service"`
}

type OllamaConfig struct {
	BaseURL        string `envconfig:"OLLAMA_BASE_URL" default:"http://localhost:11434"`
	EmbeddingModel string `envconfig:"OLLAMA_EMBEDDING_MODEL" default:"nomic-embed-text"`
}

type ConsumerConfig struct {
	MaxAttempts    int           `envconfig:"EMBEDDING_CONSUMER_MAX_ATTEMPTS" default:"5"`
	InitialBackoff time.Duration `envconfig:"EMBEDDING_CONSUMER_INITIAL_BACKOFF" default:"1s"`
	MaxBackoff     time.Duration `envconfig:"EMBEDDING_CONSUMER_MAX_BACKOFF" default:"30s"`
}

func Load() (*Config, error) {
	var cfg Config

	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}

	cfg.Kafka.Brokers = pkgconfig.SplitAndTrim(cfg.Kafka.Brokers)

	return &cfg, nil
}
