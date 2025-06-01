package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/IBM/sarama"
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

type Client struct {
	ID       string
	Username string
	Room     string
	Conn     *websocket.Conn
	Send     chan wireMessage
}

type Hub struct {
	clients    map[string]*Client
	rooms      map[string]map[string]*Client // room -> clientID -> client
	register   chan *Client
	unregister chan *Client
	broadcast  chan wireMessage
	mutex      sync.RWMutex
}

func newHub() *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		rooms:      make(map[string]map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan wireMessage),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client.ID] = client
			if h.rooms[client.Room] == nil {
				h.rooms[client.Room] = make(map[string]*Client)
			}
			h.rooms[client.Room][client.ID] = client
			h.mutex.Unlock()

			log.Printf("[gateway] User %s joined room %s", client.Username, client.Room)

			// Send join notification
			joinMsg := wireMessage{
				ID:        uuid.NewString(),
				Room:      client.Room,
				Author:    "System",
				Body:      client.Username + " joined the room",
				CreatedAt: time.Now().UTC(),
			}
			h.broadcastToRoom(client.Room, joinMsg)

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				if roomClients, exists := h.rooms[client.Room]; exists {
					delete(roomClients, client.ID)
					if len(roomClients) == 0 {
						delete(h.rooms, client.Room)
					}
				}
				close(client.Send)
			}
			h.mutex.Unlock()

			log.Printf("[gateway] User %s left room %s", client.Username, client.Room)

			// Send leave notification
			leaveMsg := wireMessage{
				ID:        uuid.NewString(),
				Room:      client.Room,
				Author:    "System",
				Body:      client.Username + " left the room",
				CreatedAt: time.Now().UTC(),
			}
			h.broadcastToRoom(client.Room, leaveMsg)

		case message := <-h.broadcast:
			h.broadcastToRoom(message.Room, message)
		}
	}
}

func (h *Hub) broadcastToRoom(room string, message wireMessage) {
	h.mutex.RLock()
	roomClients := h.rooms[room]
	h.mutex.RUnlock()

	for _, client := range roomClients {
		select {
		case client.Send <- message:
		default:
			close(client.Send)
			h.mutex.Lock()
			delete(h.clients, client.ID)
			delete(h.rooms[room], client.ID)
			h.mutex.Unlock()
		}
	}
}

func (h *Hub) getRoomUserCount(room string) int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	if roomClients, exists := h.rooms[room]; exists {
		return len(roomClients)
	}
	return 0
}

var (
	producer *kafkabr.SaramaProducer
	consumer sarama.Consumer
	hub      *Hub
	upgrader = websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
)

// ipv4Dialer forces IPv4 connections
func ipv4Dialer(network, address string) (net.Conn, error) {
	d := &net.Dialer{
		Timeout: 10 * time.Second,
	}
	return d.Dial("tcp4", address)
}

func main() {
	_ = godotenv.Load()

	brokers := strings.Split(utils.GetEnv("KAFKA_BROKERS", "localhost:9092"), ",")
	log.Printf("[gateway] Connecting to Kafka brokers: %v", brokers)

	// Initialize producer
	p, err := kafkabr.NewProducer(brokers)
	if err != nil {
		log.Fatalf("kafka producer: %v", err)
	}
	defer p.Close()
	producer = p

	// Initialize consumer
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true
	config.Net.DialTimeout = 10 * time.Second
	config.Net.ReadTimeout = 10 * time.Second
	config.Net.WriteTimeout = 10 * time.Second

	c, err := sarama.NewConsumer(brokers, config)
	if err != nil {
		log.Printf("[gateway] Failed to create Kafka consumer: %v", err)
		log.Printf("[gateway] Continuing without Kafka consumer - messages won't be broadcasted")
	} else {
		defer c.Close()
		consumer = c
		// Start consuming messages from Kafka
		go consumeMessages()
	}

	// Initialize hub
	hub = newHub()
	go hub.run()

	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/chat", chatHandler)

	port := utils.GetEnv("GATEWAY_PORT", "8081")
	log.Printf("[gateway] listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func consumeMessages() {
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
					var wireMsg wireMessage
					if err := json.Unmarshal(msg.Value, &wireMsg); err == nil {
						hub.broadcast <- wireMsg
					}
				case err := <-pc.Errors():
					log.Printf("[gateway] Consumer error: %v", err)
				}
			}
		}(pc)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func chatHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)

	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Go Message App - Real-Time Chat</title>
    <style>
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            max-width: 900px;
            margin: 0 auto;
            padding: 20px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
        }
        .container {
            background: white;
            padding: 30px;
            border-radius: 15px;
            box-shadow: 0 10px 30px rgba(0,0,0,0.2);
        }
        .header {
            text-align: center;
            margin-bottom: 30px;
            color: #333;
        }
        .header h1 {
            margin: 0;
            color: #667eea;
            font-size: 2.5em;
        }
        .header p {
            margin: 10px 0 0 0;
            color: #666;
            font-size: 1.1em;
        }
        .form-group {
            margin-bottom: 20px;
        }
        label {
            display: block;
            margin-bottom: 8px;
            font-weight: 600;
            color: #333;
        }
        input, textarea, button {
            width: 100%;
            padding: 12px;
            border: 2px solid #e1e5e9;
            border-radius: 8px;
            font-size: 14px;
            transition: border-color 0.3s ease;
        }
        input:focus, textarea:focus {
            outline: none;
            border-color: #667eea;
        }
        button {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            border: none;
            cursor: pointer;
            margin-top: 10px;
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }
        button:hover:not(:disabled) {
            transform: translateY(-2px);
            box-shadow: 0 5px 15px rgba(102, 126, 234, 0.4);
        }
        button:disabled {
            background: #6c757d;
            cursor: not-allowed;
            transform: none;
            box-shadow: none;
        }
        #messages {
            height: 400px;
            overflow-y: auto;
            border: 2px solid #e1e5e9;
            padding: 15px;
            background-color: #f8f9fa;
            margin-bottom: 20px;
            border-radius: 8px;
        }
        .message {
            margin-bottom: 15px;
            padding: 12px;
            background-color: white;
            border-radius: 8px;
            border-left: 4px solid #667eea;
            box-shadow: 0 2px 5px rgba(0,0,0,0.1);
        }
        .message.system {
            border-left-color: #28a745;
            background-color: #f8fff9;
        }
        .message.own {
            border-left-color: #ffc107;
            background-color: #fffdf5;
        }
        .status {
            padding: 15px;
            margin-bottom: 20px;
            border-radius: 8px;
            font-weight: 600;
            text-align: center;
        }
        .status.connected {
            background-color: #d4edda;
            color: #155724;
            border: 2px solid #c3e6cb;
        }
        .status.disconnected {
            background-color: #f8d7da;
            color: #721c24;
            border: 2px solid #f5c6cb;
        }
        .auth-section {
            background: #f8f9fa;
            padding: 20px;
            border-radius: 8px;
            margin-bottom: 20px;
            border: 2px solid #e9ecef;
        }
        .quick-auth {
            display: flex;
            gap: 10px;
            margin-top: 10px;
        }
        .quick-auth input {
            flex: 1;
        }
        .quick-auth button {
            flex: 0 0 auto;
            width: 120px;
            margin-top: 0;
        }
        .room-users {
            display: flex;
            gap: 10px;
            align-items: center;
        }
        .room-users input {
            flex: 1;
        }
        .user-count {
            background: #667eea;
            color: white;
            padding: 8px 12px;
            border-radius: 20px;
            font-size: 12px;
            font-weight: 600;
            min-width: 60px;
            text-align: center;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🚀 Go Message App</h1>
            <p>Real-time microservices chat application</p>
        </div>
        
        <div id="status" class="status disconnected">
            Status: Disconnected - Please authenticate first
        </div>

        <div class="auth-section">
            <h3>🔐 Quick Authentication</h3>
            <div class="quick-auth">
                <input type="text" id="quickUsername" placeholder="Username" value="testuser">
                <input type="password" id="quickPassword" placeholder="Password" value="testpass123">
                <button onclick="quickRegister()">Register</button>
                <button onclick="quickLogin()">Login</button>
            </div>
            <div style="margin-top: 10px; font-size: 12px; color: #666;">
                Or paste your JWT token below if you already have one
            </div>
        </div>

        <div class="form-group">
            <label for="token">JWT Token:</label>
            <textarea id="token" rows="3" placeholder="Your JWT token will appear here after login..."></textarea>
        </div>

        <div class="form-group">
            <div class="room-users">
                <div style="flex: 1;">
                    <label for="room">Chat Room:</label>
                    <input type="text" id="room" value="general" placeholder="Enter room name">
                </div>
                <div class="user-count" id="userCount">0 users</div>
            </div>
        </div>

        <button id="connectBtn" onclick="connect()">Connect to Chat</button>
        <button id="disconnectBtn" onclick="disconnect()" disabled>Disconnect</button>

        <div id="messages"></div>

        <div class="form-group">
            <label for="messageInput">💬 Your Message:</label>
            <input type="text" id="messageInput" placeholder="Type your message and press Enter..." disabled>
        </div>

        <button id="sendBtn" onclick="sendMessage()" disabled>Send Message</button>
    </div>

    <script>
        let ws = null;
        let connected = false;
        let currentUser = '';
        let currentRoom = '';

        function updateStatus(status, isConnected) {
            const statusEl = document.getElementById('status');
            statusEl.textContent = 'Status: ' + status;
            statusEl.className = 'status ' + (isConnected ? 'connected' : 'disconnected');
            connected = isConnected;
            
            document.getElementById('connectBtn').disabled = isConnected;
            document.getElementById('disconnectBtn').disabled = !isConnected;
            document.getElementById('messageInput').disabled = !isConnected;
            document.getElementById('sendBtn').disabled = !isConnected;
        }

        function updateUserCount() {
            // This will be updated when we receive room info from server
            // For now, we'll simulate it
            const userCountEl = document.getElementById('userCount');
            if (connected) {
                // In a real implementation, the server would send user count updates
                userCountEl.textContent = '1+ users';
            } else {
                userCountEl.textContent = '0 users';
            }
        }

        function addMessage(message, type = 'normal') {
            const messagesEl = document.getElementById('messages');
            const messageEl = document.createElement('div');
            let messageClass = 'message';
            
            if (typeof message === 'string') {
                messageClass += ' system';
                messageEl.innerHTML = '<strong>System:</strong> ' + message;
            } else {
                const timestamp = new Date(message.created_at).toLocaleTimeString();
                const isOwnMessage = message.author === currentUser;
                const isSystemMessage = message.author === 'System';
                
                if (isOwnMessage) {
                    messageClass += ' own';
                } else if (isSystemMessage) {
                    messageClass += ' system';
                }
                
                messageEl.innerHTML = 
                    '<strong>' + message.author + (isOwnMessage ? ' (You)' : '') + '</strong> ' +
                    '<span style="color: #666; font-size: 12px;">[' + message.room + '] ' + timestamp + '</span><br>' +
                    '<div style="margin-top: 5px;">' + message.body + '</div>';
            }
            
            messageEl.className = messageClass;
            messagesEl.appendChild(messageEl);
            messagesEl.scrollTop = messagesEl.scrollHeight;
        }

        async function quickRegister() {
            const username = document.getElementById('quickUsername').value.trim();
            const password = document.getElementById('quickPassword').value.trim();
            
            if (!username || !password) {
                alert('Please enter username and password');
                return;
            }

            try {
                const response = await fetch('http://localhost:8080/api/v1/register', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ username, password })
                });
                
                const data = await response.json();
                if (data.success) {
                    addMessage('✅ Registration successful! You can now login.', 'system');
                } else {
                    addMessage('❌ Registration failed: ' + data.message, 'system');
                }
            } catch (error) {
                addMessage('❌ Registration error: ' + error.message, 'system');
            }
        }

        async function quickLogin() {
            const username = document.getElementById('quickUsername').value.trim();
            const password = document.getElementById('quickPassword').value.trim();
            
            if (!username || !password) {
                alert('Please enter username and password');
                return;
            }

            try {
                const response = await fetch('http://localhost:8080/api/v1/login', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ username, password })
                });
                
                const data = await response.json();
                if (data.success && data.data.token) {
                    document.getElementById('token').value = data.data.token;
                    currentUser = username;
                    addMessage('✅ Login successful! Token received. You can now connect to chat.', 'system');
                } else {
                    addMessage('❌ Login failed: ' + data.message, 'system');
                }
            } catch (error) {
                addMessage('❌ Login error: ' + error.message, 'system');
            }
        }

        function connect() {
            const token = document.getElementById('token').value.trim();
            const room = document.getElementById('room').value.trim();
            
            if (!token) {
                alert('Please login first or enter a JWT token');
                return;
            }
            
            if (!room) {
                alert('Please enter a room name');
                return;
            }

            currentRoom = room;
            updateStatus('Connecting...', false);
            addMessage('Attempting to connect to WebSocket...', 'system');

            const wsUrl = 'ws://localhost:8081/ws?token=' + encodeURIComponent(token) + '&room=' + encodeURIComponent(room);
            ws = new WebSocket(wsUrl);

            ws.onopen = function(event) {
                updateStatus('Connected to real-time chat!', true);
                updateUserCount();
                addMessage('✅ Connected to WebSocket! You can now send messages.', 'system');
            };

            ws.onmessage = function(event) {
                try {
                    const message = JSON.parse(event.data);
                    addMessage(message);
                } catch (e) {
                    addMessage('Raw message: ' + event.data, 'system');
                }
            };

            ws.onclose = function(event) {
                updateStatus('Disconnected', false);
                updateUserCount();
                addMessage('❌ Connection closed. Code: ' + event.code + ', Reason: ' + event.reason, 'system');
            };

            ws.onerror = function(error) {
                updateStatus('Connection Error', false);
                addMessage('❌ WebSocket error occurred', 'system');
            };
        }

        function disconnect() {
            if (ws) {
                ws.close();
                ws = null;
            }
        }

        function sendMessage() {
            if (!connected || !ws) {
                alert('Not connected to chat');
                return;
            }

            const room = document.getElementById('room').value.trim();
            const body = document.getElementById('messageInput').value.trim();

            if (!room || !body) {
                alert('Please enter both room and message');
                return;
            }

            const message = {
                room: room,
                body: body
            };

            ws.send(JSON.stringify(message));
            document.getElementById('messageInput').value = '';
        }

        // Allow Enter key to send message
        document.getElementById('messageInput').addEventListener('keypress', function(e) {
            if (e.key === 'Enter') {
                sendMessage();
            }
        });

        // Auto-fill token if provided in URL
        const urlParams = new URLSearchParams(window.location.search);
        const tokenParam = urlParams.get('token');
        if (tokenParam) {
            document.getElementById('token').value = tokenParam;
        }

        // Welcome message
        addMessage('Welcome to Go Message App! Register or login to start chatting.', 'system');
    </script>
</body>
</html>`

	w.Write([]byte(html))
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	room := r.URL.Query().Get("room")

	if room == "" {
		room = "general" // default room
	}

	claims, err := auth.ParseToken(token)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := &Client{
		ID:       uuid.NewString(),
		Username: claims.Username,
		Room:     room,
		Conn:     conn,
		Send:     make(chan wireMessage, 256),
	}

	hub.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		hub.unregister <- c
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
			break
		}

		var in inbound
		if json.Unmarshal(raw, &in) != nil || in.Room == "" || in.Body == "" {
			continue
		}

		// Update client room if changed
		if in.Room != c.Room {
			// Remove from old room
			hub.unregister <- c
			// Update room and re-register
			c.Room = in.Room
			hub.register <- c
		}

		msg := wireMessage{
			ID:        uuid.NewString(),
			Room:      in.Room,
			Author:    c.Username,
			Body:      in.Body,
			CreatedAt: time.Now().UTC(),
		}

		// Send to Kafka for persistence
		payload, _ := json.Marshal(msg)
		_ = producer.Produce("chat-in", []byte(in.Room), payload)

		// Broadcast immediately to connected clients
		hub.broadcast <- msg
	}
}

func (c *Client) writePump() {
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
