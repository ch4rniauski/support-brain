package postgres

import (
	"context"
	"fmt"
	"log"
	"github.com/ch4rniauski/support-brain/internal/knowledge/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, cfg *config.PostgresConfig) (*pgxpool.Pool, error) {
	connectionStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	pool, err := pgxpool.New(ctx, connectionStr)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		log.Fatalf("ping postgres: %v", err)
		return nil, err
	}

	return pool, nil
}
