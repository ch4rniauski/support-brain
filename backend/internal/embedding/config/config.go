package config

import (
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	HTTP     HTTPConfig
	Postgres PostgresConfig
	Kafka    KafkaConfig
	Ollama   OllamaConfig
	Consumer ConsumerConfig
	Outbox   OutboxConfig
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

type OutboxConfig struct {
	PollInterval time.Duration `envconfig:"OUTBOX_POLL_INTERVAL" default:"1s"`
	BatchSize    int           `envconfig:"OUTBOX_BATCH_SIZE" default:"100"`
}

func Load() (*Config, error) {
	var cfg Config

	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}

	cfg.Kafka.Brokers = splitAndTrim(cfg.Kafka.Brokers)

	return &cfg, nil
}

func splitAndTrim(values []string) []string {
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
