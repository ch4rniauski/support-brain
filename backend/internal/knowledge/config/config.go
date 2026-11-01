package config

import (
	"time"

	pkgconfig "github.com/ch4rniauski/support-brain/pkg/config"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	HTTP       pkgconfig.HTTPConfig
	Postgres   pkgconfig.PostgresConfig
	Kafka      KafkaConfig
	Embedding  EmbeddingConfig
	Clustering ClusteringConfig
	Consumer   ConsumerConfig
	Outbox     pkgconfig.OutboxConfig
}

type KafkaConfig struct {
	Brokers        []string `envconfig:"KAFKA_BROKERS" default:"localhost:9092"`
	EmbeddingTopic string   `envconfig:"KAFKA_EMBEDDING_TOPIC" default:"embeddings"`
	ClusterTopic   string   `envconfig:"KAFKA_CLUSTER_TOPIC" default:"clusters"`
	ConsumerGroup  string   `envconfig:"KAFKA_KNOWLEDGE_GROUP" default:"knowledge-service"`
}

type EmbeddingConfig struct {
	BaseURL   string `envconfig:"EMBEDDING_BASE_URL" default:"http://localhost:8086"`
	ModelName string `envconfig:"OLLAMA_EMBEDDING_MODEL" default:"nomic-embed-text"`
}

type ClusteringConfig struct {
	SimilarityThreshold  float64 `envconfig:"SIMILARITY_THRESHOLD" default:"0.85"`
	SearchLimit          int     `envconfig:"SEARCH_LIMIT" default:"10"`
	GenerationThresholdN int     `envconfig:"GENERATION_THRESHOLD_N" default:"5"`
}

type ConsumerConfig struct {
	MaxAttempts    int           `envconfig:"KNOWLEDGE_CONSUMER_MAX_ATTEMPTS" default:"5"`
	InitialBackoff time.Duration `envconfig:"KNOWLEDGE_CONSUMER_INITIAL_BACKOFF" default:"1s"`
	MaxBackoff     time.Duration `envconfig:"KNOWLEDGE_CONSUMER_MAX_BACKOFF" default:"30s"`
}

func Load() (*Config, error) {
	var cfg Config

	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}

	cfg.Kafka.Brokers = pkgconfig.SplitAndTrim(cfg.Kafka.Brokers)

	return &cfg, nil
}
