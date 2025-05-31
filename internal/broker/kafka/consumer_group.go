package kafka

import (
	"context"

	"github.com/IBM/sarama"
)

type GroupConsumer struct{ cg sarama.ConsumerGroup }

func NewGroupConsumer(brokers []string, groupID string) (*GroupConsumer, error) {
	cfg := sarama.NewConfig()
	cfg.Version = sarama.V3_6_0_0
	cfg.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRange
	cfg.Consumer.Offsets.Initial = sarama.OffsetNewest
	cg, err := sarama.NewConsumerGroup(brokers, groupID, cfg)
	if err != nil {
		return nil, err
	}
	return &GroupConsumer{cg: cg}, nil
}

func (g *GroupConsumer) Consume(ctx context.Context, topics []string, h sarama.ConsumerGroupHandler) error {
	for {
		if err := g.cg.Consume(ctx, topics, h); err != nil {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
}

func (g *GroupConsumer) Errors() <-chan error { return g.cg.Errors() }
func (g *GroupConsumer) Close() error         { return g.cg.Close() }
