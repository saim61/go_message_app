package postgres

import (
	"context"

	"github.com/jmoiron/sqlx"
)

// MessageRepo handles INSERTs into messages.
type MessageRepo struct{ db *sqlx.DB }

func NewMessageRepo(db *sqlx.DB) *MessageRepo { return &MessageRepo{db} }

type Message struct {
	ID        string `db:"id"`
	Room      string `db:"room"`
	Author    string // username
	Body      string `db:"body"`
	CreatedAt string `db:"created_at"` // RFC3339 string from JSON
}

func (r *MessageRepo) Save(ctx context.Context, m *Message) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO messages (id, room, author_id, body, created_at)
		VALUES ($1, $2,
		        (SELECT id FROM users WHERE username=$3),
		        $4, $5)
		ON CONFLICT (id) DO NOTHING`,
		m.ID, m.Room, m.Author, m.Body, m.CreatedAt,
	)
	return err
}
