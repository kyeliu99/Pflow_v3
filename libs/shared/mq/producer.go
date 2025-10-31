package mq

import (
	"context"
	"log"

	"github.com/segmentio/kafka-go"
)

// Producer wraps a Kafka writer.
type Producer struct {
	writer *kafka.Writer
	topic  string
}

// NewProducer constructs a Kafka producer using the provided configuration.
func NewProducer(cfg ProducerConfig) (*Producer, error) {
	normalized := cfg.normalize()
	if err := normalized.Validate(); err != nil {
		return nil, err
	}

	writer := &kafka.Writer{
		Addr:                   kafka.TCP(normalized.Brokers...),
		Topic:                  normalized.Topic,
		AllowAutoTopicCreation: true,
		RequiredAcks:           kafka.RequireAll,
		Balancer:               &kafka.LeastBytes{},
		BatchTimeout:           normalized.effectiveTimeout(),
		BatchSize:              normalized.effectiveBatchSize(),
	}
	if normalized.ClientID != "" {
		writer.Transport = &kafka.Transport{ClientID: normalized.ClientID}
	}

	log.Printf("mq: initialized producer %s", normalized.String())
	return &Producer{writer: writer, topic: normalized.Topic}, nil
}

// Publish sends a message to Kafka.
func (p *Producer) Publish(ctx context.Context, key string, value []byte, headers map[string]string) error {
	if p == nil {
		return nil
	}

	msg := kafka.Message{
		Key:   []byte(key),
		Value: value,
	}
	for headerKey, headerValue := range headers {
		msg.Headers = append(msg.Headers, kafka.Header{Key: headerKey, Value: []byte(headerValue)})
	}

	return p.writer.WriteMessages(ctx, msg)
}

// Close flushes and closes the underlying writer.
func (p *Producer) Close(ctx context.Context) error {
	if p == nil || p.writer == nil {
		return nil
	}
	return p.writer.Close()
}
