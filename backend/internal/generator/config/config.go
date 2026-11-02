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
	LLM        LLMConfig
	Generation GenerationConfig
	Intake     IntakeConfig
	Consumer   ConsumerConfig
	Outbox     pkgconfig.OutboxConfig
}

type KafkaConfig struct {
	Brokers       []string `envconfig:"KAFKA_BROKERS" default:"localhost:9092"`
	ClusterTopic  string   `envconfig:"KAFKA_CLUSTER_TOPIC" default:"clusters"`
	TemplateTopic string   `envconfig:"KAFKA_TEMPLATE_TOPIC" default:"templates"`
	ConsumerGroup string   `envconfig:"KAFKA_GENERATOR_GROUP" default:"generator-service"`
}

type LLMConfig struct {
	BaseURL   string `envconfig:"LLM_BASE_URL" default:"http://localhost:11434"`
	ChatModel string `envconfig:"CHAT_MODEL" default:"llama3.2"`
}

type GenerationConfig struct {
	PromptVersion  string `envconfig:"PROMPT_VERSION" default:"v1"`
	MaxAttempts    int    `envconfig:"MAX_ATTEMPTS" default:"3"`
	MaxOutputChars int    `envconfig:"MAX_OUTPUT_CHARS" default:"4000"`
	MinOutputChars int    `envconfig:"MIN_OUTPUT_CHARS" default:"20"`
}

type IntakeConfig struct {
	BaseURL string `envconfig:"INTAKE_BASE_URL" default:"http://localhost:8082"`
}

type ConsumerConfig struct {
	MaxAttempts    int           `envconfig:"GENERATOR_CONSUMER_MAX_ATTEMPTS" default:"5"`
	InitialBackoff time.Duration `envconfig:"GENERATOR_CONSUMER_INITIAL_BACKOFF" default:"1s"`
	MaxBackoff     time.Duration `envconfig:"GENERATOR_CONSUMER_MAX_BACKOFF" default:"30s"`
}

func Load() (*Config, error) {
	var cfg Config

	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}

	cfg.Kafka.Brokers = pkgconfig.SplitAndTrim(cfg.Kafka.Brokers)

	return &cfg, nil
}
