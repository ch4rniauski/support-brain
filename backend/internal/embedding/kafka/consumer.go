package kafka

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/ch4rniauski/support-brain/internal/embedding/service"
	"github.com/segmentio/kafka-go"
)

type MessageHandler interface {
	Handle(ctx context.Context, eventType string, payload []byte) error
}

type RetryConfig struct {
	MaxAttempts    int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
}

func (c RetryConfig) withDefaults() RetryConfig {
	if c.MaxAttempts <= 0 {
		c.MaxAttempts = 5
	}
	if c.InitialBackoff <= 0 {
		c.InitialBackoff = time.Second
	}
	if c.MaxBackoff <= 0 {
		c.MaxBackoff = 30 * time.Second
	}
	if c.MaxBackoff < c.InitialBackoff {
		c.MaxBackoff = c.InitialBackoff
	}
	return c
}

type Consumer struct {
	reader  *kafka.Reader
	handler MessageHandler
	retry   RetryConfig
}

func NewConsumer(
	brokers []string,
	topic, groupID string,
	handler MessageHandler,
	retry RetryConfig,
) *Consumer {
	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:        brokers,
			Topic:          topic,
			GroupID:        groupID,
			MinBytes:       1,
			MaxBytes:       10e6,
			CommitInterval: -1,
			StartOffset:    kafka.FirstOffset,
		}),
		handler: handler,
		retry:   retry.withDefaults(),
	}
}

func (c *Consumer) Run(ctx context.Context) {
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("embedding kafka consumer fetch: %v", err)
			continue
		}

		eventType := headerValue(msg.Headers, "event_type")
		if err := c.processWithRetry(ctx, eventType, msg.Value); err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("embedding kafka consumer giving up on message: %v", err)
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			log.Printf("embedding kafka consumer commit: %v", err)
		}
	}
}

func (c *Consumer) processWithRetry(ctx context.Context, eventType string, payload []byte) error {
	backoff := c.retry.InitialBackoff

	var lastErr error
	for attempt := 1; attempt <= c.retry.MaxAttempts; attempt++ {
		err := c.handler.Handle(ctx, eventType, payload)
		if err == nil {
			return nil
		}

		lastErr = err
		if errors.Is(err, service.ErrPermanent) {
			return fmt.Errorf("permanent failure: %w", err)
		}

		log.Printf(
			"embedding kafka consumer handle attempt %d/%d: %v",
			attempt,
			c.retry.MaxAttempts,
			err,
		)

		if attempt == c.retry.MaxAttempts {
			break
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}

		backoff *= 2
		if backoff > c.retry.MaxBackoff {
			backoff = c.retry.MaxBackoff
		}
	}

	return fmt.Errorf("exhausted %d attempts: %w", c.retry.MaxAttempts, lastErr)
}

func (c *Consumer) Close() error {
	if err := c.reader.Close(); err != nil {
		return fmt.Errorf("close kafka consumer: %w", err)
	}
	return nil
}

func headerValue(headers []kafka.Header, key string) string {
	for _, header := range headers {
		if header.Key == key {
			return string(header.Value)
		}
	}
	return ""
}
