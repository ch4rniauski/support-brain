package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/ch4rniauski/support-brain/internal/template/api"
	"github.com/ch4rniauski/support-brain/internal/template/config"
	"github.com/ch4rniauski/support-brain/internal/template/repository"
	"github.com/ch4rniauski/support-brain/internal/template/service"
	"github.com/ch4rniauski/support-brain/pkg/kafka"
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

	templateRepo := repository.NewTemplateRepository(pool)
	templateService := service.NewTemplateService(templateRepo)

	consumer := kafka.NewConsumer(
		cfg.Kafka.Brokers,
		cfg.Kafka.TemplateTopic,
		cfg.Kafka.ConsumerGroup,
		templateService,
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
		Handler:           api.NewRouter(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("template listening on %s", server.Addr)
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
