package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/ch4rniauski/support-brain/internal/knowledge/api"
	"github.com/ch4rniauski/support-brain/internal/knowledge/client"
	"github.com/ch4rniauski/support-brain/internal/knowledge/config"
	"github.com/ch4rniauski/support-brain/internal/knowledge/handler"
	"github.com/ch4rniauski/support-brain/internal/knowledge/repository"
	"github.com/ch4rniauski/support-brain/internal/knowledge/service"
	"github.com/ch4rniauski/support-brain/pkg/kafka"
	"github.com/ch4rniauski/support-brain/pkg/outbox"
	"github.com/ch4rniauski/support-brain/pkg/postgres"
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

	clusterRepo := repository.NewClusterRepository(pool)
	outboxRepo := repository.NewOutboxRepository(pool)
	embeddingClient := client.NewEmbeddingClient(cfg.Embedding.BaseURL, cfg.Embedding.ModelName)
	clusteringService := service.NewClusteringService(
		clusterRepo,
		embeddingClient,
		service.ClusteringConfig{
			SimilarityThreshold:  cfg.Clustering.SimilarityThreshold,
			SearchLimit:          cfg.Clustering.SearchLimit,
			GenerationThresholdN: cfg.Clustering.GenerationThresholdN,
		},
	)
	clusterService := service.NewClusterService(clusterRepo)
	clusterHandler := handler.NewClusterHandler(clusterService)

	producer := kafka.NewProducer(cfg.Kafka.Brokers)
	defer func() {
		if err := producer.Close(); err != nil {
			log.Printf("close kafka producer: %v", err)
		}
	}()

	publisher := outbox.NewPublisher(
		outboxRepo,
		producer,
		cfg.Kafka.ClusterTopic,
		cfg.Outbox.PollInterval,
		cfg.Outbox.BatchSize,
	)
	go publisher.Run(ctx)

	consumer := kafka.NewConsumer(
		cfg.Kafka.Brokers,
		cfg.Kafka.EmbeddingTopic,
		cfg.Kafka.ConsumerGroup,
		clusteringService,
		kafka.RetryConfig{
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
		Handler:           api.NewRouter(clusterHandler),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("knowledge listening on %s", server.Addr)
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
