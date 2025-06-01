package gateway

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/IBM/sarama"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func parseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrInvalidKey
}

func WSHandler(hub *Hub, producer sarama.SyncProducer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		room := r.URL.Query().Get("room")

		if room == "" {
			room = "general" // default room
		}

		claims, err := parseToken(token)
		if err != nil {
			log.Printf("[gateway] Token parsing failed: %v", err)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("[gateway] WebSocket upgrade failed: %v", err)
			return
		}

		client := &Client{
			ID:       uuid.NewString(),
			Username: claims.Username,
			Room:     room,
			Conn:     conn,
			Send:     make(chan WireMessage, 256),
		}

		hub.Register(client)

		// Start goroutines for reading and writing
		go client.writePump(hub)
		go client.readPump(hub, producer)
	}
}

func (c *Client) readPump(hub *Hub, producer sarama.SyncProducer) {
	defer func() {
		hub.Unregister(c)
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(512)
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, raw, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[gateway] WebSocket error: %v", err)
			}
			break
		}

		var in InboundMessage
		if json.Unmarshal(raw, &in) != nil || in.Room == "" || in.Body == "" {
			continue
		}

		// Update client room if changed
		if in.Room != c.Room {
			// Remove from old room
			hub.Unregister(c)
			// Update room and re-register
			c.Room = in.Room
			hub.Register(c)
		}

		msg := WireMessage{
			ID:        uuid.NewString(),
			Room:      in.Room,
			Author:    c.Username,
			Body:      in.Body,
			CreatedAt: time.Now().UTC(),
		}

		// Send to Kafka for persistence
		if producer != nil {
			payload, _ := json.Marshal(msg)
			kafkaMsg := &sarama.ProducerMessage{
				Topic: "chat-in",
				Key:   sarama.StringEncoder(in.Room),
				Value: sarama.ByteEncoder(payload),
			}
			_, _, err := producer.SendMessage(kafkaMsg)
			if err != nil {
				log.Printf("[gateway] Failed to send message to Kafka: %v", err)
			}
		}

		// Broadcast immediately to connected clients
		hub.Broadcast(msg)
	}
}

func (c *Client) writePump(hub *Hub) {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteJSON(message); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
