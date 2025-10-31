package mq

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

// Message represents a Kafka message delivered to consumers.
type Message struct {
	Key     []byte
	Value   []byte
	Headers map[string]string
	Time    time.Time
}

// Handler processes messages from a consumer.
type Handler func(context.Context, Message) error

// Consumer wraps a Kafka reader and invokes a handler for each message.
type Consumer struct {
	reader  *kafka.Reader
	handler Handler
}

// NewConsumer constructs a Kafka consumer and prepares it for message processing.
func NewConsumer(cfg ConsumerConfig, handler Handler) (*Consumer, error) {
	normalized := cfg.normalize()
	if err := normalized.Validate(); err != nil {
		return nil, err
	}

	readerCfg := kafka.ReaderConfig{
		Brokers:  normalized.Brokers,
		Topic:    normalized.Topic,
		GroupID:  normalized.GroupID,
		MinBytes: normalized.MinBytes,
		MaxBytes: normalized.MaxBytes,
	}
	if normalized.ClientID != "" {
		readerCfg.Dialer = &kafka.Dialer{ClientID: normalized.ClientID}
	}

	log.Printf("mq: initialized consumer %s", normalized.String())
	return &Consumer{
		reader:  kafka.NewReader(readerCfg),
		handler: handler,
	}, nil
}

// Run starts consuming messages until the context is cancelled or an unrecoverable error occurs.
func (c *Consumer) Run(ctx context.Context) error {
	if c == nil || c.reader == nil {
		return nil
	}

	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return ctx.Err()
			}
			return err
		}

		payload := Message{
			Key:     msg.Key,
			Value:   msg.Value,
			Headers: make(map[string]string, len(msg.Headers)),
			Time:    msg.Time,
		}
		for _, header := range msg.Headers {
			payload.Headers[header.Key] = string(header.Value)
		}

		if c.handler != nil {
			if err := c.handler(ctx, payload); err != nil {
				log.Printf("mq: handler error for topic %s: %v", msg.Topic, err)
			}
		}
	}
}

// Close shuts down the reader.
func (c *Consumer) Close() error {
	if c == nil || c.reader == nil {
		return nil
	}
	return c.reader.Close()
}
