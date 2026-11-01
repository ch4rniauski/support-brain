package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/ch4rniauski/support-brain/internal/embedding/api"
	"github.com/ch4rniauski/support-brain/internal/embedding/config"
	embeddingkafka "github.com/ch4rniauski/support-brain/internal/embedding/kafka"
	"github.com/ch4rniauski/support-brain/internal/embedding/ollama"
	"github.com/ch4rniauski/support-brain/internal/embedding/outbox"
	"github.com/ch4rniauski/support-brain/internal/embedding/postgres"
	"github.com/ch4rniauski/support-brain/internal/embedding/repository"
	"github.com/ch4rniauski/support-brain/internal/embedding/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer cancel()

	pool, err := postgres.NewPool(ctx, &cfg.Postgres)
	if err != nil {
		log.Fatalf("pool creation: %v", err)
	}
	defer pool.Close()

	embeddingRepo := repository.NewEmbeddingRepository(pool)
	outboxRepo := repository.NewOutboxRepository(pool)
	ollamaClient := ollama.NewClient(cfg.Ollama.BaseURL, cfg.Ollama.EmbeddingModel)
	embeddingService := service.NewEmbeddingService(embeddingRepo, ollamaClient)

	producer := embeddingkafka.NewProducer(cfg.Kafka.Brokers)
	defer func() {
		if err := producer.Close(); err != nil {
			log.Printf("close kafka producer: %v", err)
		}
	}()

	publisher := outbox.NewPublisher(
		outboxRepo,
		producer,
		cfg.Kafka.EmbeddingTopic,
		cfg.Outbox.PollInterval,
		cfg.Outbox.BatchSize,
	)
	go publisher.Run(ctx)

	consumer := embeddingkafka.NewConsumer(
		cfg.Kafka.Brokers,
		cfg.Kafka.QAPairTopic,
		cfg.Kafka.ConsumerGroup,
		embeddingService,
		embeddingkafka.RetryConfig{
			MaxAttempts:    cfg.Consumer.MaxAttempts,
			InitialBackoff: cfg.Consumer.InitialBackoff,
			MaxBackoff:     cfg.Consumer.MaxBackoff,
		},
	)
	defer func() {
		if err := consumer.Close(); err != nil {
			log.Printf("close kafka consumer: %v", err)
		}
	}()

	go consumer.Run(ctx)

	server := &http.Server{
		Addr:              fmt.Sprintf(":%s", cfg.HTTP.Port),
		Handler:           api.NewRouter(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("embedding listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server shutdown: %v", err)
	}
}
