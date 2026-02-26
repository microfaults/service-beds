package kafka

import (
	"fmt"

	"github.com/IBM/sarama"
	"github.com/sirupsen/logrus"
)

// Producer wraps a Sarama SyncProducer for publishing messages to Kafka.
type Producer struct {
	producer sarama.SyncProducer
	log      *logrus.Logger
}

// NewProducer creates a new Kafka sync producer connected to the given brokers.
func NewProducer(brokers []string, log *logrus.Logger) (*Producer, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.RequiredAcks = sarama.WaitForLocal
	config.Producer.Retry.Max = 3

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	log.Infof("Kafka producer connected to brokers: %v", brokers)
	return &Producer{producer: producer, log: log}, nil
}

// Publish sends a message to the specified Kafka topic.
func (p *Producer) Publish(topic, key string, payload []byte) error {
	msg := &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.StringEncoder(key),
		Value: sarama.ByteEncoder(payload),
	}

	partition, offset, err := p.producer.SendMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to publish message to topic %q: %w", topic, err)
	}

	p.log.Debugf("message published to topic=%s partition=%d offset=%d", topic, partition, offset)
	return nil
}

// Close shuts down the Kafka producer.
func (p *Producer) Close() error {
	if p.producer != nil {
		return p.producer.Close()
	}
	return nil
}
