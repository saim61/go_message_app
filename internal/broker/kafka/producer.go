package kafka

import (
	"github.com/IBM/sarama"
)

// SaramaProducer satisfies the internal/broker.Producer interface.
type SaramaProducer struct {
	p sarama.SyncProducer
}

// NewProducer connects to the given brokers and returns a sync producer.
func NewProducer(brokers []string) (*SaramaProducer, error) {
	cfg := sarama.NewConfig()
	cfg.Producer.Return.Successes = true
	cfg.Producer.Retry.Max = 3
	prod, err := sarama.NewSyncProducer(brokers, cfg)
	if err != nil {
		return nil, err
	}
	return &SaramaProducer{p: prod}, nil
}

func (s *SaramaProducer) Produce(topic string, key, value []byte) error {
	_, _, err := s.p.SendMessage(&sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.ByteEncoder(key),
		Value: sarama.ByteEncoder(value),
	})
	return err
}

func (s *SaramaProducer) Close() error { return s.p.Close() }
