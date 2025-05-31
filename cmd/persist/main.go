package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/cenkalti/backoff/v4"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/saim61/go_message_app/utils"

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
	_ = godotenv.Load()

	groupID := utils.GetEnv("KAFKA_CONSUMER_GROUP", "persist-svc")
	dlqTopic := utils.GetEnv("KAFKA_DLQ_TOPIC", "chat-dlq")
	brokers := strings.Split(utils.GetEnv("KAFKA_BROKERS", "localhost:9092"), ",")
	retryN, _ := strconv.Atoi(utils.GetEnv("DB_MAX_RETRIES", "3"))

	db, err := sqlx.Open("postgres", utils.BuildDSN())
	if err != nil {
		log.Fatal(err)
	}
	msgRepo := postgres.NewMessageRepo(db)

	dlqProducer, err := kafkabr.NewProducer(brokers)
	if err != nil {
		log.Fatal(err)
	}
	defer dlqProducer.Close()

	consumer, err := kafkabr.NewGroupConsumer(brokers, groupID)
	if err != nil {
		log.Fatal(err)
	}
	defer consumer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		for e := range consumer.Errors() {
			log.Printf("[persist] consumer error: %v", e)
		}
	}()

	handler := &persistHandler{
		repo:       msgRepo,
		dlq:        dlqProducer,
		dlqTopic:   dlqTopic,
		maxRetries: retryN,
	}
	if err := consumer.Consume(ctx, []string{"chat-in"}, handler); err != nil {
		log.Fatal(err)
	}
}

type persistHandler struct {
	repo       *postgres.MessageRepo
	dlq        *kafkabr.SaramaProducer
	dlqTopic   string
	maxRetries int
}

func (h *persistHandler) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (h *persistHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (h *persistHandler) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		if err := h.handle(msg); err != nil {
			log.Printf("[persist] DLQ push for offset %d: %v", msg.Offset, err)
			_ = h.dlq.Produce(h.dlqTopic, msg.Key, msg.Value)
		}
		sess.MarkMessage(msg, "")
	}
	return nil
}

func (h *persistHandler) handle(m *sarama.ConsumerMessage) error {
	var wm wireMessage
	if json.Unmarshal(m.Value, &wm) != nil {
		return fmt.Errorf("json decode: %w", errBadPayload)
	}
	op := func() error {
		return h.repo.Save(context.Background(), &postgres.Message{
			ID:        wm.ID,
			Room:      wm.Room,
			Author:    wm.Author,
			Body:      wm.Body,
			CreatedAt: wm.CreatedAt.Format(time.RFC3339),
		})
	}
	return retry(op, h.maxRetries)
}

var errBadPayload = fmt.Errorf("bad payload")

func retry(f func() error, max int) error {
	bo := backoff.WithMaxRetries(backoff.NewExponentialBackOff(), uint64(max))
	return backoff.RetryNotify(f, bo, func(err error, t time.Duration) {
		log.Printf("[persist] retry in %s: %v", t, err)
	})
}
