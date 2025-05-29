package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/saim61/go_message_app/internal/auth"
	broker "github.com/saim61/go_message_app/internal/broker"
	kafka "github.com/saim61/go_message_app/internal/broker/kafka"
)

var (
	producer broker.Producer
	upgrader = websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
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

func main() {
	var err error
	producer, err = kafka.NewProducer([]string{"localhost:9092"})
	if err != nil {
		log.Fatalf("kafka producer: %v", err)
	}
	defer producer.Close()

	http.HandleFunc("/ws", wsHandler)
	log.Println("gateway listening on :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	tkn := r.URL.Query().Get("token")
	claims, err := auth.ParseToken(tkn)
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
