package messaging

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/pflow/shared/config"
	"github.com/segmentio/kafka-go"
)

var (
	writerOnce sync.Once
	writer     *kafka.Writer

	readerMu sync.Mutex
	readers  = map[string]*kafka.Reader{}
)

// Writer returns a singleton Kafka writer.
func Writer() *kafka.Writer {
	writerOnce.Do(func() {
		cfg := config.MustGet()
		writer = &kafka.Writer{
			Addr:         kafka.TCP(cfg.KafkaBrokers),
			Topic:        cfg.KafkaTopic,
			BatchTimeout: 10 * time.Millisecond,
		}
	})
	return writer
}

// Reader returns a singleton Kafka reader.
func Reader(groupID string) *kafka.Reader {
	readerMu.Lock()
	defer readerMu.Unlock()

	if r, ok := readers[groupID]; ok {
		return r
	}

	cfg := config.MustGet()
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{cfg.KafkaBrokers},
		Topic:    cfg.KafkaTopic,
		GroupID:  groupID,
		MinBytes: 1,
		MaxBytes: 10e6,
	})
	readers[groupID] = reader

	return reader
}

// Publish sends an event payload to Kafka.
func Publish(ctx context.Context, key string, value []byte) error {
	return Writer().WriteMessages(ctx, kafka.Message{Key: []byte(key), Value: value})
}

// Consume fetches a single message from Kafka.
func Consume(ctx context.Context) (kafka.Message, error) {
	return Reader("pflow-consumer").FetchMessage(ctx)
}

// Commit acknowledges a consumed message.
func Commit(ctx context.Context, msg kafka.Message) error {
	return Reader("pflow-consumer").CommitMessages(ctx, msg)
}

// Close releases Kafka resources.
func Close() {
	if writer != nil {
		if err := writer.Close(); err != nil {
			log.Printf("kafka writer close error: %v", err)
		}
	}
	readerMu.Lock()
	defer readerMu.Unlock()
	for groupID, r := range readers {
		if err := r.Close(); err != nil {
			log.Printf("kafka reader close error (%s): %v", groupID, err)
		}
		delete(readers, groupID)
	}
}
