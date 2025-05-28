package storage

import "context"

// Message is the domain object persisted in Postgres.
type Message struct {
	ID     string
	Room   string
	Author string
	Body   string
	Ts     int64
}

type Repository interface {
	SaveMessage(ctx context.Context, m Message) error
	Close() error
}
