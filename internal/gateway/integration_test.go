package gateway

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/saim61/go_message_app/internal/auth"
)

func TestIntegration_WebSocketConnection(t *testing.T) {
	// Set up JWT secret for testing
	auth.SetJWTSecret("test-secret")

	// Create a hub
	hub := NewHub()
	go hub.Run()

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		WSHandler(w, r, hub, nil) // nil producer for testing
	}))
	defer server.Close()

	// Generate a test token
	token, err := auth.NewToken("testuser", time.Hour)
	require.NoError(t, err)

	// Convert HTTP URL to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "?token=" + token + "&room=testroom"

	// Connect to WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	// Give some time for connection to be established
	time.Sleep(100 * time.Millisecond)

	// Check that client was registered
	assert.Equal(t, 1, hub.GetRoomUserCount("testroom"))

	// Send a message
	testMessage := InboundMessage{
		Message: "Hello, integration test!",
		Room:    "testroom",
	}

	err = conn.WriteJSON(testMessage)
	require.NoError(t, err)

	// Read the broadcasted message
	var receivedMessage WireMessage
	err = conn.ReadJSON(&receivedMessage)
	require.NoError(t, err)

	// Verify the message
	assert.Equal(t, "testuser", receivedMessage.Username)
	assert.Equal(t, "Hello, integration test!", receivedMessage.Message)
	assert.Equal(t, "testroom", receivedMessage.Room)
	assert.NotEmpty(t, receivedMessage.ID)
	assert.False(t, receivedMessage.Timestamp.IsZero())
}

func TestIntegration_MultipleClients(t *testing.T) {
	// Set up JWT secret for testing
	auth.SetJWTSecret("test-secret")

	// Create a hub
	hub := NewHub()
	go hub.Run()

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		WSHandler(w, r, hub, nil) // nil producer for testing
	}))
	defer server.Close()

	// Generate tokens for two users
	token1, err := auth.NewToken("user1", time.Hour)
	require.NoError(t, err)

	token2, err := auth.NewToken("user2", time.Hour)
	require.NoError(t, err)

	// Convert HTTP URL to WebSocket URL
	wsURL1 := "ws" + strings.TrimPrefix(server.URL, "http") + "?token=" + token1 + "&room=testroom"
	wsURL2 := "ws" + strings.TrimPrefix(server.URL, "http") + "?token=" + token2 + "&room=testroom"

	// Connect first client
	conn1, _, err := websocket.DefaultDialer.Dial(wsURL1, nil)
	require.NoError(t, err)
	defer conn1.Close()

	// Give some time for connection to be established
	time.Sleep(100 * time.Millisecond)

	// Check that first client was registered
	assert.Equal(t, 1, hub.GetRoomUserCount("testroom"))

	// Connect second client
	conn2, _, err := websocket.DefaultDialer.Dial(wsURL2, nil)
	require.NoError(t, err)
	defer conn2.Close()

	// Give some time for connection to be established
	time.Sleep(100 * time.Millisecond)

	// Check that both clients are registered
	assert.Equal(t, 2, hub.GetRoomUserCount("testroom"))

	// First client should receive join notification for second user
	var joinMessage WireMessage
	err = conn1.ReadJSON(&joinMessage)
	require.NoError(t, err)
	assert.Equal(t, "system", joinMessage.Username)
	assert.Contains(t, joinMessage.Message, "user2 joined the room")

	// Send a message from first client
	testMessage := InboundMessage{
		Message: "Hello from user1!",
		Room:    "testroom",
	}

	err = conn1.WriteJSON(testMessage)
	require.NoError(t, err)

	// Both clients should receive the message
	var receivedMessage1, receivedMessage2 WireMessage

	err = conn1.ReadJSON(&receivedMessage1)
	require.NoError(t, err)

	err = conn2.ReadJSON(&receivedMessage2)
	require.NoError(t, err)

	// Verify both received the same message
	assert.Equal(t, receivedMessage1.ID, receivedMessage2.ID)
	assert.Equal(t, "user1", receivedMessage1.Username)
	assert.Equal(t, "user1", receivedMessage2.Username)
	assert.Equal(t, "Hello from user1!", receivedMessage1.Message)
	assert.Equal(t, "Hello from user1!", receivedMessage2.Message)
}

func TestIntegration_RoomSwitching(t *testing.T) {
	// Set up JWT secret for testing
	auth.SetJWTSecret("test-secret")

	// Create a hub
	hub := NewHub()
	go hub.Run()

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		WSHandler(w, r, hub, nil) // nil producer for testing
	}))
	defer server.Close()

	// Generate a test token
	token, err := auth.NewToken("testuser", time.Hour)
	require.NoError(t, err)

	// Convert HTTP URL to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "?token=" + token + "&room=room1"

	// Connect to WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	// Give some time for connection to be established
	time.Sleep(100 * time.Millisecond)

	// Check that client is in room1
	assert.Equal(t, 1, hub.GetRoomUserCount("room1"))
	assert.Equal(t, 0, hub.GetRoomUserCount("room2"))

	// Send a message to switch rooms
	switchMessage := InboundMessage{
		Message: "Hello from room2!",
		Room:    "room2",
	}

	err = conn.WriteJSON(switchMessage)
	require.NoError(t, err)

	// Give some time for room switch to be processed
	time.Sleep(100 * time.Millisecond)

	// Check that client switched rooms
	assert.Equal(t, 0, hub.GetRoomUserCount("room1"))
	assert.Equal(t, 1, hub.GetRoomUserCount("room2"))

	// Read the message that was sent
	var receivedMessage WireMessage
	err = conn.ReadJSON(&receivedMessage)
	require.NoError(t, err)

	// Verify the message
	assert.Equal(t, "testuser", receivedMessage.Username)
	assert.Equal(t, "Hello from room2!", receivedMessage.Message)
	assert.Equal(t, "room2", receivedMessage.Room)
}

func TestIntegration_InvalidToken(t *testing.T) {
	// Set up JWT secret for testing
	auth.SetJWTSecret("test-secret")

	// Create a hub
	hub := NewHub()
	go hub.Run()

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		WSHandler(w, r, hub, nil) // nil producer for testing
	}))
	defer server.Close()

	// Convert HTTP URL to WebSocket URL with invalid token
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "?token=invalid-token&room=testroom"

	// Try to connect to WebSocket
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)

	// Connection should fail
	if conn != nil {
		conn.Close()
	}

	// Should get an HTTP error response
	assert.Error(t, err)
	if resp != nil {
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	}
}

func TestIntegration_DefaultRoom(t *testing.T) {
	// Set up JWT secret for testing
	auth.SetJWTSecret("test-secret")

	// Create a hub
	hub := NewHub()
	go hub.Run()

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		WSHandler(w, r, hub, nil) // nil producer for testing
	}))
	defer server.Close()

	// Generate a test token
	token, err := auth.NewToken("testuser", time.Hour)
	require.NoError(t, err)

	// Convert HTTP URL to WebSocket URL without room parameter
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "?token=" + token

	// Connect to WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	// Give some time for connection to be established
	time.Sleep(100 * time.Millisecond)

	// Check that client is in the default "general" room
	assert.Equal(t, 1, hub.GetRoomUserCount("general"))

	// Send a message (should go to general room)
	testMessage := InboundMessage{
		Message: "Hello, default room!",
	}

	err = conn.WriteJSON(testMessage)
	require.NoError(t, err)

	// Read the broadcasted message
	var receivedMessage WireMessage
	err = conn.ReadJSON(&receivedMessage)
	require.NoError(t, err)

	// Verify the message went to general room
	assert.Equal(t, "general", receivedMessage.Room)
}

func TestIntegration_ClientDisconnection(t *testing.T) {
	// Set up JWT secret for testing
	auth.SetJWTSecret("test-secret")

	// Create a hub
	hub := NewHub()
	go hub.Run()

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		WSHandler(w, r, hub, nil) // nil producer for testing
	}))
	defer server.Close()

	// Generate tokens for two users
	token1, err := auth.NewToken("user1", time.Hour)
	require.NoError(t, err)

	token2, err := auth.NewToken("user2", time.Hour)
	require.NoError(t, err)

	// Convert HTTP URL to WebSocket URL
	wsURL1 := "ws" + strings.TrimPrefix(server.URL, "http") + "?token=" + token1 + "&room=testroom"
	wsURL2 := "ws" + strings.TrimPrefix(server.URL, "http") + "?token=" + token2 + "&room=testroom"

	// Connect both clients
	conn1, _, err := websocket.DefaultDialer.Dial(wsURL1, nil)
	require.NoError(t, err)

	conn2, _, err := websocket.DefaultDialer.Dial(wsURL2, nil)
	require.NoError(t, err)

	// Give some time for connections to be established
	time.Sleep(100 * time.Millisecond)

	// Check that both clients are registered
	assert.Equal(t, 2, hub.GetRoomUserCount("testroom"))

	// Read join notification on first client
	var joinMessage WireMessage
	err = conn1.ReadJSON(&joinMessage)
	require.NoError(t, err)

	// Disconnect first client
	conn1.Close()

	// Give some time for disconnection to be processed
	time.Sleep(100 * time.Millisecond)

	// Check that only one client remains
	assert.Equal(t, 1, hub.GetRoomUserCount("testroom"))

	// Second client should receive leave notification
	var leaveMessage WireMessage
	err = conn2.ReadJSON(&leaveMessage)
	require.NoError(t, err)
	assert.Equal(t, "system", leaveMessage.Username)
	assert.Contains(t, leaveMessage.Message, "user1 left the room")

	conn2.Close()
}

func TestIntegration_ConcurrentConnections(t *testing.T) {
	// Set up JWT secret for testing
	auth.SetJWTSecret("test-secret")

	// Create a hub
	hub := NewHub()
	go hub.Run()

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		WSHandler(w, r, hub, nil) // nil producer for testing
	}))
	defer server.Close()

	const numClients = 10
	connections := make([]*websocket.Conn, numClients)

	// Connect multiple clients concurrently
	for i := 0; i < numClients; i++ {
		go func(clientID int) {
			token, err := auth.NewToken(fmt.Sprintf("user%d", clientID), time.Hour)
			require.NoError(t, err)

			wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "?token=" + token + "&room=testroom"

			conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
			require.NoError(t, err)

			connections[clientID] = conn
		}(i)
	}

	// Give some time for all connections to be established
	time.Sleep(500 * time.Millisecond)

	// Check that all clients are registered
	assert.Equal(t, numClients, hub.GetRoomUserCount("testroom"))

	// Clean up connections
	for _, conn := range connections {
		if conn != nil {
			conn.Close()
		}
	}
}
