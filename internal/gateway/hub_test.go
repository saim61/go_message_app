package gateway

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

// mockConn implements a mock WebSocket connection for testing
type mockConn struct {
	closed bool
}

func (m *mockConn) Close() error {
	m.closed = true
	return nil
}

func (m *mockConn) WriteMessage(messageType int, data []byte) error {
	return nil
}

func (m *mockConn) WriteJSON(v interface{}) error {
	return nil
}

func (m *mockConn) ReadMessage() (messageType int, p []byte, err error) {
	return websocket.TextMessage, []byte("test"), nil
}

func (m *mockConn) SetReadLimit(limit int64) {}

func (m *mockConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *mockConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func (m *mockConn) SetPongHandler(h func(appData string) error) {}

func createTestClient(id, username, room string) *Client {
	return &Client{
		ID:       id,
		Username: username,
		Room:     room,
		Conn:     &mockConn{},
		Send:     make(chan WireMessage, 256),
	}
}

func TestNewHub(t *testing.T) {
	hub := NewHub()

	assert.NotNil(t, hub)
	assert.NotNil(t, hub.clients)
	assert.NotNil(t, hub.rooms)
	assert.NotNil(t, hub.register)
	assert.NotNil(t, hub.unregister)
	assert.NotNil(t, hub.broadcast)
	assert.Equal(t, 0, len(hub.clients))
	assert.Equal(t, 0, len(hub.rooms))
}

func TestHubRegisterClient(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer close(hub.register)

	client := createTestClient("client1", "user1", "room1")

	// Register client
	hub.Register(client)

	// Give some time for the goroutine to process
	time.Sleep(10 * time.Millisecond)

	hub.mutex.RLock()
	assert.Equal(t, 1, len(hub.clients))
	assert.Equal(t, 1, len(hub.rooms))
	assert.Equal(t, 1, len(hub.rooms["room1"]))
	assert.Equal(t, client, hub.clients["client1"])
	assert.Equal(t, client, hub.rooms["room1"]["client1"])
	hub.mutex.RUnlock()
}

func TestHubUnregisterClient(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer close(hub.register)
	defer close(hub.unregister)

	client := createTestClient("client1", "user1", "room1")

	// Register client first
	hub.Register(client)
	time.Sleep(10 * time.Millisecond)

	// Verify client is registered
	hub.mutex.RLock()
	assert.Equal(t, 1, len(hub.clients))
	hub.mutex.RUnlock()

	// Unregister client
	hub.Unregister(client)
	time.Sleep(10 * time.Millisecond)

	hub.mutex.RLock()
	assert.Equal(t, 0, len(hub.clients))
	assert.Equal(t, 0, len(hub.rooms))
	hub.mutex.RUnlock()

	// Verify channel is closed
	select {
	case _, ok := <-client.Send:
		assert.False(t, ok, "Send channel should be closed")
	default:
		t.Error("Send channel should be closed")
	}
}

func TestHubMultipleClientsInSameRoom(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer close(hub.register)
	defer close(hub.unregister)

	client1 := createTestClient("client1", "user1", "room1")
	client2 := createTestClient("client2", "user2", "room1")

	// Register both clients
	hub.Register(client1)
	hub.Register(client2)
	time.Sleep(10 * time.Millisecond)

	hub.mutex.RLock()
	assert.Equal(t, 2, len(hub.clients))
	assert.Equal(t, 1, len(hub.rooms))
	assert.Equal(t, 2, len(hub.rooms["room1"]))
	hub.mutex.RUnlock()

	// Unregister one client
	hub.Unregister(client1)
	time.Sleep(10 * time.Millisecond)

	hub.mutex.RLock()
	assert.Equal(t, 1, len(hub.clients))
	assert.Equal(t, 1, len(hub.rooms))
	assert.Equal(t, 1, len(hub.rooms["room1"]))
	assert.Equal(t, client2, hub.rooms["room1"]["client2"])
	hub.mutex.RUnlock()
}

func TestHubMultipleRooms(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer close(hub.register)

	client1 := createTestClient("client1", "user1", "room1")
	client2 := createTestClient("client2", "user2", "room2")
	client3 := createTestClient("client3", "user3", "room1")

	// Register clients in different rooms
	hub.Register(client1)
	hub.Register(client2)
	hub.Register(client3)
	time.Sleep(10 * time.Millisecond)

	hub.mutex.RLock()
	assert.Equal(t, 3, len(hub.clients))
	assert.Equal(t, 2, len(hub.rooms))
	assert.Equal(t, 2, len(hub.rooms["room1"]))
	assert.Equal(t, 1, len(hub.rooms["room2"]))
	hub.mutex.RUnlock()
}

func TestGetRoomUserCount(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer close(hub.register)

	// Test empty room
	count := hub.GetRoomUserCount("nonexistent")
	assert.Equal(t, 0, count)

	// Add clients to room
	client1 := createTestClient("client1", "user1", "room1")
	client2 := createTestClient("client2", "user2", "room1")

	hub.Register(client1)
	time.Sleep(10 * time.Millisecond)

	count = hub.GetRoomUserCount("room1")
	assert.Equal(t, 1, count)

	hub.Register(client2)
	time.Sleep(10 * time.Millisecond)

	count = hub.GetRoomUserCount("room1")
	assert.Equal(t, 2, count)
}

func TestHubBroadcast(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer close(hub.register)
	defer close(hub.broadcast)

	client1 := createTestClient("client1", "user1", "room1")
	client2 := createTestClient("client2", "user2", "room1")
	client3 := createTestClient("client3", "user3", "room2")

	// Register clients
	hub.Register(client1)
	hub.Register(client2)
	hub.Register(client3)
	time.Sleep(10 * time.Millisecond)

	// Create test message
	message := WireMessage{
		ID:        uuid.NewString(),
		Room:      "room1",
		Author:    "testuser",
		Body:      "test message",
		CreatedAt: time.Now(),
	}

	// Broadcast message
	hub.Broadcast(message)
	time.Sleep(10 * time.Millisecond)

	// Check that clients in room1 received the message
	select {
	case msg := <-client1.Send:
		assert.Equal(t, message.ID, msg.ID)
		assert.Equal(t, message.Body, msg.Body)
	case <-time.After(100 * time.Millisecond):
		t.Error("client1 should have received the message")
	}

	select {
	case msg := <-client2.Send:
		assert.Equal(t, message.ID, msg.ID)
		assert.Equal(t, message.Body, msg.Body)
	case <-time.After(100 * time.Millisecond):
		t.Error("client2 should have received the message")
	}

	// Check that client in room2 did not receive the message
	select {
	case <-client3.Send:
		t.Error("client3 should not have received the message")
	case <-time.After(50 * time.Millisecond):
		// Expected - no message should be received
	}
}

func TestHubJoinLeaveNotifications(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer close(hub.register)
	defer close(hub.unregister)

	// Create a client to receive notifications
	observer := createTestClient("observer", "observer", "room1")
	hub.Register(observer)
	time.Sleep(10 * time.Millisecond)

	// Clear any initial messages
	select {
	case <-observer.Send:
	default:
	}

	// Register a new client (should trigger join notification)
	newClient := createTestClient("newclient", "newuser", "room1")
	hub.Register(newClient)
	time.Sleep(10 * time.Millisecond)

	// Check for join notification
	select {
	case msg := <-observer.Send:
		assert.Equal(t, "System", msg.Author)
		assert.Contains(t, msg.Body, "newuser joined the room")
	case <-time.After(100 * time.Millisecond):
		t.Error("Should have received join notification")
	}

	// Unregister the client (should trigger leave notification)
	hub.Unregister(newClient)
	time.Sleep(10 * time.Millisecond)

	// Check for leave notification
	select {
	case msg := <-observer.Send:
		assert.Equal(t, "System", msg.Author)
		assert.Contains(t, msg.Body, "newuser left the room")
	case <-time.After(100 * time.Millisecond):
		t.Error("Should have received leave notification")
	}
}

func TestHubConcurrentOperations(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer close(hub.register)
	defer close(hub.unregister)
	defer close(hub.broadcast)

	// Test concurrent registration and unregistration
	numClients := 10
	clients := make([]*Client, numClients)

	// Register clients concurrently
	for i := 0; i < numClients; i++ {
		clients[i] = createTestClient(
			"client"+string(rune(i)),
			"user"+string(rune(i)),
			"room1",
		)
		go hub.Register(clients[i])
	}

	time.Sleep(50 * time.Millisecond)

	// Check all clients are registered
	count := hub.GetRoomUserCount("room1")
	assert.Equal(t, numClients, count)

	// Unregister clients concurrently
	for i := 0; i < numClients; i++ {
		go hub.Unregister(clients[i])
	}

	time.Sleep(50 * time.Millisecond)

	// Check all clients are unregistered
	count = hub.GetRoomUserCount("room1")
	assert.Equal(t, 0, count)
}
