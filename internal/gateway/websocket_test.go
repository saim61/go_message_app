package gateway

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseToken(t *testing.T) {
	// Set test JWT secret
	os.Setenv("JWT_SECRET", "test-secret-key")

	tests := []struct {
		name       string
		token      string
		wantClaims *Claims
		wantErr    bool
	}{
		{
			name:  "valid token",
			token: createTestToken("testuser", 15*time.Minute),
			wantClaims: &Claims{
				Username: "testuser",
			},
			wantErr: false,
		},
		{
			name:       "expired token",
			token:      createTestToken("expireduser", -1*time.Hour),
			wantClaims: nil,
			wantErr:    true,
		},
		{
			name:       "invalid token",
			token:      "invalid.token.format",
			wantClaims: nil,
			wantErr:    true,
		},
		{
			name:       "empty token",
			token:      "",
			wantClaims: nil,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := parseToken(tt.token)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, claims)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, claims)
				assert.Equal(t, tt.wantClaims.Username, claims.Username)
			}
		})
	}
}

func createTestToken(username string, ttl time.Duration) string {
	claims := &Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	return tokenString
}

func TestWSHandler_Authentication(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-key")

	hub := NewHub()
	handler := WSHandler(hub, nil)

	tests := []struct {
		name           string
		token          string
		room           string
		expectedStatus int
	}{
		{
			name:           "valid token",
			token:          createTestToken("testuser", 15*time.Minute),
			room:           "testroom",
			expectedStatus: http.StatusSwitchingProtocols,
		},
		{
			name:           "missing token",
			token:          "",
			room:           "testroom",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid token",
			token:          "invalid-token",
			room:           "testroom",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "expired token",
			token:          createTestToken("expireduser", -1*time.Hour),
			room:           "testroom",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request with WebSocket upgrade headers
			req := httptest.NewRequest("GET", "/ws", nil)
			req.Header.Set("Connection", "upgrade")
			req.Header.Set("Upgrade", "websocket")
			req.Header.Set("Sec-WebSocket-Version", "13")
			req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")

			// Add query parameters
			q := req.URL.Query()
			if tt.token != "" {
				q.Add("token", tt.token)
			}
			if tt.room != "" {
				q.Add("room", tt.room)
			}
			req.URL.RawQuery = q.Encode()

			w := httptest.NewRecorder()
			handler(w, req)

			if tt.expectedStatus == http.StatusSwitchingProtocols {
				// For successful WebSocket upgrade, we expect specific headers
				assert.Equal(t, "websocket", w.Header().Get("Upgrade"))
				assert.Equal(t, "upgrade", strings.ToLower(w.Header().Get("Connection")))
			} else {
				assert.Equal(t, tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestWSHandler_DefaultRoom(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-key")

	hub := NewHub()
	handler := WSHandler(hub, nil)

	// Create request without room parameter
	req := httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("Connection", "upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")

	q := req.URL.Query()
	q.Add("token", createTestToken("testuser", 15*time.Minute))
	req.URL.RawQuery = q.Encode()

	w := httptest.NewRecorder()
	handler(w, req)

	// Should default to "general" room and succeed
	assert.Equal(t, "websocket", w.Header().Get("Upgrade"))
}

// Mock WebSocket connection for testing read/write pumps
type testWebSocketConn struct {
	readChan      chan []byte
	writeChan     chan interface{}
	closed        bool
	readLimit     int64
	readDeadline  time.Time
	writeDeadline time.Time
	pongHandler   func(string) error
}

func newTestWebSocketConn() *testWebSocketConn {
	return &testWebSocketConn{
		readChan:  make(chan []byte, 10),
		writeChan: make(chan interface{}, 10),
	}
}

func (c *testWebSocketConn) ReadMessage() (int, []byte, error) {
	if c.closed {
		return 0, nil, websocket.ErrCloseSent
	}

	select {
	case data := <-c.readChan:
		return websocket.TextMessage, data, nil
	case <-time.After(100 * time.Millisecond):
		return 0, nil, websocket.ErrReadTimeout
	}
}

func (c *testWebSocketConn) WriteJSON(v interface{}) error {
	if c.closed {
		return websocket.ErrCloseSent
	}

	select {
	case c.writeChan <- v:
		return nil
	default:
		return websocket.ErrWriteTimeout
	}
}

func (c *testWebSocketConn) WriteMessage(messageType int, data []byte) error {
	if c.closed {
		return websocket.ErrCloseSent
	}
	return nil
}

func (c *testWebSocketConn) Close() error {
	c.closed = true
	close(c.readChan)
	close(c.writeChan)
	return nil
}

func (c *testWebSocketConn) SetReadLimit(limit int64) {
	c.readLimit = limit
}

func (c *testWebSocketConn) SetReadDeadline(t time.Time) error {
	c.readDeadline = t
	return nil
}

func (c *testWebSocketConn) SetWriteDeadline(t time.Time) error {
	c.writeDeadline = t
	return nil
}

func (c *testWebSocketConn) SetPongHandler(h func(string) error) {
	c.pongHandler = h
}

func TestClientReadPump(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer close(hub.register)
	defer close(hub.unregister)

	conn := newTestWebSocketConn()
	client := &Client{
		ID:       "test-client",
		Username: "testuser",
		Room:     "testroom",
		Conn:     conn,
		Send:     make(chan WireMessage, 256),
	}

	// Start read pump in goroutine
	go client.readPump(hub, nil)

	// Send a test message
	testMessage := `{"room":"testroom","body":"hello world"}`
	conn.readChan <- []byte(testMessage)

	// Give some time for processing
	time.Sleep(10 * time.Millisecond)

	// Close connection to stop read pump
	conn.Close()

	// Verify read pump sets read limit
	assert.Equal(t, int64(512), conn.readLimit)
}

func TestClientWritePump(t *testing.T) {
	hub := NewHub()
	conn := newTestWebSocketConn()
	client := &Client{
		ID:       "test-client",
		Username: "testuser",
		Room:     "testroom",
		Conn:     conn,
		Send:     make(chan WireMessage, 256),
	}

	// Start write pump in goroutine
	go client.writePump(hub)

	// Send a test message
	testMessage := WireMessage{
		ID:        "test-id",
		Room:      "testroom",
		Author:    "testuser",
		Body:      "test message",
		CreatedAt: time.Now(),
	}

	client.Send <- testMessage

	// Check if message was written
	select {
	case written := <-conn.writeChan:
		msg, ok := written.(WireMessage)
		require.True(t, ok)
		assert.Equal(t, testMessage.ID, msg.ID)
		assert.Equal(t, testMessage.Body, msg.Body)
	case <-time.After(100 * time.Millisecond):
		t.Error("Message should have been written")
	}

	// Close send channel to stop write pump
	close(client.Send)
	time.Sleep(10 * time.Millisecond)
}

func TestClientRoomSwitching(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer close(hub.register)
	defer close(hub.unregister)

	conn := newTestWebSocketConn()
	client := &Client{
		ID:       "test-client",
		Username: "testuser",
		Room:     "room1",
		Conn:     conn,
		Send:     make(chan WireMessage, 256),
	}

	// Register client initially
	hub.Register(client)
	time.Sleep(10 * time.Millisecond)

	// Verify client is in room1
	count := hub.GetRoomUserCount("room1")
	assert.Equal(t, 1, count)

	// Start read pump
	go client.readPump(hub, nil)

	// Send message to switch to room2
	roomSwitchMessage := `{"room":"room2","body":"switching rooms"}`
	conn.readChan <- []byte(roomSwitchMessage)

	time.Sleep(20 * time.Millisecond)

	// Verify client moved to room2
	count1 := hub.GetRoomUserCount("room1")
	count2 := hub.GetRoomUserCount("room2")
	assert.Equal(t, 0, count1)
	assert.Equal(t, 1, count2)

	// Close connection
	conn.Close()
}

func TestInvalidJSONHandling(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer close(hub.register)
	defer close(hub.unregister)

	conn := newTestWebSocketConn()
	client := &Client{
		ID:       "test-client",
		Username: "testuser",
		Room:     "testroom",
		Conn:     conn,
		Send:     make(chan WireMessage, 256),
	}

	// Start read pump
	go client.readPump(hub, nil)

	// Send invalid JSON
	conn.readChan <- []byte(`{"invalid": json}`)

	// Send message with missing fields
	conn.readChan <- []byte(`{"room":""}`)

	// Send valid message to ensure pump is still working
	conn.readChan <- []byte(`{"room":"testroom","body":"valid message"}`)

	time.Sleep(20 * time.Millisecond)

	// Close connection
	conn.Close()

	// Test should pass without panics - invalid messages should be ignored
}
