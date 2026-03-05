package kafka

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/twmb/franz-go/pkg/kgo"
)

// Producer wraps a franz-go client for publishing messages to Kafka.
type Producer struct {
	client *kgo.Client
	log    *logrus.Logger
}

// NewProducer creates a new Kafka producer connected to the given brokers.
func NewProducer(brokers []string, log *logrus.Logger) (*Producer, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	log.Infof("Kafka producer connected to brokers: %v", brokers)
	return &Producer{client: client, log: log}, nil
}

// Publish sends a message to the specified Kafka topic.
func (p *Producer) Publish(topic, key string, payload []byte) error {
	rec := &kgo.Record{
		Topic: topic,
		Key:   []byte(key),
		Value: payload,
	}

	results := p.client.ProduceSync(context.Background(), rec)
	if err := results.FirstErr(); err != nil {
		return fmt.Errorf("failed to publish message to topic %q: %w", topic, err)
	}

	r := results[0].Record
	p.log.Debugf("message published to topic=%s partition=%d offset=%d", r.Topic, r.Partition, r.Offset)
	return nil
}

// Close shuts down the Kafka producer.
func (p *Producer) Close() {
	if p.client != nil {
		p.client.Close()
	}
}
