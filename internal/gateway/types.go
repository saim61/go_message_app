package gateway

import "time"

type InboundMessage struct {
	Room string `json:"room"`
	Body string `json:"body"`
}

type WireMessage struct {
	ID        string    `json:"id"`
	Room      string    `json:"room"`
	Author    string    `json:"author"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}
