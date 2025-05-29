package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	kafkabr "github.com/saim61/go_message_app/internal/broker/kafka"
	"github.com/saim61/go_message_app/internal/storage/postgres"
)

type wireMessage struct {
	ID        string    `json:"id"`
	Room      string    `json:"room"`
	Author    string    `json:"author"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

func main() {
	// DB
	db, err := sqlx.Open("postgres", "postgres://postgres:postgres@localhost:5432/go_message_app?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	if err = db.Ping(); err != nil {
		log.Fatal(err)
	}
	msgRepo := postgres.NewMessageRepo(db)
	// Kafka consumer
	consumer, err := kafkabr.NewConsumer([]string{"localhost:9092"})
	if err != nil {
		log.Fatalf("kafka consumer: %v", err)
	}
	defer consumer.Close()

	handler := func(_ []byte, val []byte) error {
		var wm wireMessage
		if json.Unmarshal(val, &wm) != nil {
			return nil
		}
		return msgRepo.Save(context.Background(), &postgres.Message{
			ID:        wm.ID,
			Room:      wm.Room,
			Author:    wm.Author,
			Body:      wm.Body,
			CreatedAt: wm.CreatedAt.Format(time.RFC3339),
		})
	}

	if err := consumer.Consume("chat-in", handler); err != nil {
		log.Fatal(err)
	}
	select {}
}
