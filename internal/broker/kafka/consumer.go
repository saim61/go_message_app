package kafka

import (
	"sync"

	"github.com/IBM/sarama"
)

// SaramaConsumer satisfies internal/broker.Consumer.
type SaramaConsumer struct {
	c        sarama.Consumer
	partCons []sarama.PartitionConsumer
	wg       sync.WaitGroup
}

func NewConsumer(brokers []string) (*SaramaConsumer, error) {
	c, err := sarama.NewConsumer(brokers, nil)
	if err != nil {
		return nil, err
	}
	return &SaramaConsumer{c: c}, nil
}

func (s *SaramaConsumer) Consume(topic string, handler func([]byte, []byte) error) error {
	partitions, err := s.c.Partitions(topic)
	if err != nil {
		return err
	}
	for _, p := range partitions {
		pc, err := s.c.ConsumePartition(topic, p, sarama.OffsetNewest)
		if err != nil {
			return err
		}
		s.partCons = append(s.partCons, pc)
		s.wg.Add(1)
		go func(pc sarama.PartitionConsumer) {
			defer s.wg.Done()
			for msg := range pc.Messages() {
				_ = handler(msg.Key, msg.Value) // ignore handler error for MVP
			}
		}(pc)
	}
	return nil
}

func (s *SaramaConsumer) Close() error {
	for _, pc := range s.partCons {
		_ = pc.Close()
	}
	s.wg.Wait()
	return s.c.Close()
}
