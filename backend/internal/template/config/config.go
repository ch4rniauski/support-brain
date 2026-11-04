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
	Consumer ConsumerConfig
	Suggest  SuggestConfig
}

type KafkaConfig struct {
	Brokers       []string `envconfig:"KAFKA_BROKERS" default:"localhost:9092"`
	TemplateTopic string   `envconfig:"KAFKA_TEMPLATE_TOPIC" default:"templates"`
	ConsumerGroup string   `envconfig:"KAFKA_TEMPLATE_GROUP" default:"template-service"`
}

type ConsumerConfig struct {
	MaxAttempts    int           `envconfig:"TEMPLATE_CONSUMER_MAX_ATTEMPTS" default:"5"`
	InitialBackoff time.Duration `envconfig:"TEMPLATE_CONSUMER_INITIAL_BACKOFF" default:"1s"`
	MaxBackoff     time.Duration `envconfig:"TEMPLATE_CONSUMER_MAX_BACKOFF" default:"30s"`
}

type SuggestConfig struct {
	EmbeddingBaseURL     string  `envconfig:"EMBEDDING_BASE_URL" default:"http://localhost:8086"`
	KnowledgeBaseURL     string  `envconfig:"KNOWLEDGE_BASE_URL" default:"http://localhost:8085"`
	SimilarityThreshold  float64 `envconfig:"SUGGEST_SIMILARITY_THRESHOLD" default:"0.85"`
	SearchLimit          int     `envconfig:"SUGGEST_SEARCH_LIMIT" default:"10"`
}

func Load() (*Config, error) {
	var cfg Config

	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}

	cfg.Kafka.Brokers = pkgconfig.SplitAndTrim(cfg.Kafka.Brokers)

	return &cfg, nil
}
