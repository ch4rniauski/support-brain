package kafka

import (
	"context"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(brokers []string) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Balancer:     &kafka.LeastBytes{},
			RequiredAcks: kafka.RequireOne,
			Async:        false,
		},
	}
}

func (p *Producer) Publish(
	ctx context.Context,
	topic string,
	key string,
	eventType string,
	payload []byte,
) error {
	err := p.writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: payload,
		Headers: []kafka.Header{
			{Key: "event_type", Value: []byte(eventType)},
		},
		Time: time.Now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("publish kafka message: %w", err)
	}

	return nil
}

func (p *Producer) Close() error {
	if err := p.writer.Close(); err != nil {
		return fmt.Errorf("close kafka producer: %w", err)
	}
	return nil
}
