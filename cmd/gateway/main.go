package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/saim61/go_message_app/utils"

	"github.com/saim61/go_message_app/internal/auth"
	kafkabr "github.com/saim61/go_message_app/internal/broker/kafka"
)

type inbound struct {
	Room string `json:"room"`
	Body string `json:"body"`
}
type wireMessage struct {
	ID        string    `json:"id"`
	Room      string    `json:"room"`
	Author    string    `json:"author"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

var (
	producer *kafkabr.SaramaProducer
	upgrader = websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
)

func main() {
	_ = godotenv.Load()

	brokers := strings.Split(utils.GetEnv("KAFKA_BROKERS", "localhost:9092"), ",")
	p, err := kafkabr.NewProducer(brokers)
	if err != nil {
		log.Fatalf("kafka producer: %v", err)
	}
	defer p.Close()
	producer = p

	http.HandleFunc("/ws", wsHandler)

	port := utils.GetEnv("GATEWAY_PORT", "8081")
	log.Printf("[gateway] listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// ---------- handlers ----------

func wsHandler(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	claims, err := auth.ParseToken(token)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var in inbound
		if json.Unmarshal(raw, &in) != nil || in.Room == "" || in.Body == "" {
			continue
		}
		msg := wireMessage{
			ID:        uuid.NewString(),
			Room:      in.Room,
			Author:    claims.Username,
			Body:      in.Body,
			CreatedAt: time.Now().UTC(),
		}
		payload, _ := json.Marshal(msg)
		_ = producer.Produce("chat-in", []byte(in.Room), payload)
	}
}
