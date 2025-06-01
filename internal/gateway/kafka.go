package gateway

import (
	"encoding/json"
	"log"

	"github.com/IBM/sarama"
)

func ConsumeMessages(consumer sarama.Consumer, hub *Hub) {
	if consumer == nil {
		log.Printf("[gateway] Consumer is nil, skipping message consumption")
		return
	}

	partitions, err := consumer.Partitions("chat-in")
	if err != nil {
		log.Printf("[gateway] Error getting partitions: %v", err)
		return
	}

	for _, partition := range partitions {
		pc, err := consumer.ConsumePartition("chat-in", partition, sarama.OffsetNewest)
		if err != nil {
			log.Printf("[gateway] Error creating partition consumer: %v", err)
			continue
		}

		go func(pc sarama.PartitionConsumer) {
			defer pc.Close()
			for {
				select {
				case msg := <-pc.Messages():
					var wireMsg WireMessage
					if err := json.Unmarshal(msg.Value, &wireMsg); err == nil {
						hub.Broadcast(wireMsg)
					}
				case err := <-pc.Errors():
					log.Printf("[gateway] Consumer error: %v", err)
				}
			}
		}(pc)
	}
}
