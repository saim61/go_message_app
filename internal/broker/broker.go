package broker

// Producer abstracts Kafka so we can mock it in tests.
type Producer interface {
	Produce(topic string, key, value []byte) error
	Close() error
}

// Consumer abstracts Kafka consumer-group behaviour.
type Consumer interface {
	Consume(topic string, handler func(key, value []byte) error) error
	Close() error
}
