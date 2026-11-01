package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/ch4rniauski/support-brain/internal/auth/api"
	"github.com/ch4rniauski/support-brain/internal/auth/config"
	"github.com/ch4rniauski/support-brain/internal/auth/handler"
	"github.com/ch4rniauski/support-brain/internal/auth/repository"
	"github.com/ch4rniauski/support-brain/internal/auth/service"
	"github.com/ch4rniauski/support-brain/pkg/auth"
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

	tokenManager := auth.NewTokenManager(cfg.Auth.JWTSecret, cfg.Auth.TokenTTL)
	userRepo := repository.NewUserRepository(pool)
	authService := service.NewAuthService(userRepo, tokenManager)
	authHandler := handler.NewAuthHandler(authService)

	server := &http.Server{
		Addr:              fmt.Sprintf(":%s", cfg.HTTP.Port),
		Handler:           api.NewRouter(authHandler),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("auth listening on %s", server.Addr)
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
