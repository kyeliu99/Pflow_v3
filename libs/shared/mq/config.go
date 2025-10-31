package mq

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// ProducerConfig describes how to connect to a Kafka topic for publishing messages.
type ProducerConfig struct {
	Brokers   []string
	Topic     string
	ClientID  string
	BatchSize int
	Timeout   time.Duration
}

// Validate ensures the producer configuration is usable.
func (cfg ProducerConfig) Validate() error {
	if len(cfg.Brokers) == 0 {
		return errors.New("mq: at least one broker must be configured")
	}
	if strings.TrimSpace(cfg.Topic) == "" {
		return errors.New("mq: topic must be provided")
	}
	return nil
}

// ConsumerConfig defines how to consume messages from Kafka.
type ConsumerConfig struct {
	Brokers  []string
	Topic    string
	GroupID  string
	ClientID string
	MinBytes int
	MaxBytes int
}

// Validate ensures the consumer configuration is usable.
func (cfg ConsumerConfig) Validate() error {
	if len(cfg.Brokers) == 0 {
		return errors.New("mq: at least one broker must be configured")
	}
	if strings.TrimSpace(cfg.Topic) == "" {
		return errors.New("mq: topic must be provided")
	}
	if strings.TrimSpace(cfg.GroupID) == "" {
		return errors.New("mq: group id must be provided")
	}
	return nil
}

func (cfg ProducerConfig) effectiveTimeout() time.Duration {
	if cfg.Timeout <= 0 {
		return 5 * time.Second
	}
	return cfg.Timeout
}

func (cfg ProducerConfig) effectiveBatchSize() int {
	if cfg.BatchSize <= 0 {
		return 1
	}
	return cfg.BatchSize
}

func (cfg ConsumerConfig) normalize() ConsumerConfig {
	normalized := cfg
	if normalized.MinBytes <= 0 {
		normalized.MinBytes = 1e3
	}
	if normalized.MaxBytes <= 0 {
		normalized.MaxBytes = 10e6
	}
	normalized.Topic = strings.TrimSpace(normalized.Topic)
	normalized.GroupID = strings.TrimSpace(normalized.GroupID)
	normalized.ClientID = strings.TrimSpace(normalized.ClientID)
	brokers := make([]string, 0, len(normalized.Brokers))
	for _, broker := range normalized.Brokers {
		broker = strings.TrimSpace(broker)
		if broker == "" {
			continue
		}
		brokers = append(brokers, broker)
	}
	normalized.Brokers = brokers
	return normalized
}

func (cfg ProducerConfig) normalize() ProducerConfig {
	normalized := cfg
	normalized.Topic = strings.TrimSpace(normalized.Topic)
	normalized.ClientID = strings.TrimSpace(normalized.ClientID)
	brokers := make([]string, 0, len(normalized.Brokers))
	for _, broker := range normalized.Brokers {
		broker = strings.TrimSpace(broker)
		if broker == "" {
			continue
		}
		brokers = append(brokers, broker)
	}
	normalized.Brokers = brokers
	return normalized
}

func joinBrokers(brokers []string) string {
	if len(brokers) == 0 {
		return ""
	}
	return strings.Join(brokers, ",")
}

// String implements fmt.Stringer but redacts sensitive information.
func (cfg ProducerConfig) String() string {
	normalized := cfg.normalize()
	return fmt.Sprintf("ProducerConfig{brokers=%s, topic=%s, client=%s}", joinBrokers(normalized.Brokers), normalized.Topic, normalized.ClientID)
}

// String implements fmt.Stringer for ConsumerConfig.
func (cfg ConsumerConfig) String() string {
	normalized := cfg.normalize()
	return fmt.Sprintf("ConsumerConfig{brokers=%s, topic=%s, group=%s, client=%s}", joinBrokers(normalized.Brokers), normalized.Topic, normalized.GroupID, normalized.ClientID)
}
