package gateway

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockConsumer implements sarama.Consumer interface for testing
type MockConsumer struct {
	mock.Mock
}

func (m *MockConsumer) Topics() ([]string, error) {
	args := m.Called()
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockConsumer) Partitions(topic string) ([]int32, error) {
	args := m.Called(topic)
	return args.Get(0).([]int32), args.Error(1)
}

func (m *MockConsumer) ConsumePartition(topic string, partition int32, offset int64) (sarama.PartitionConsumer, error) {
	args := m.Called(topic, partition, offset)
	return args.Get(0).(sarama.PartitionConsumer), args.Error(1)
}

func (m *MockConsumer) HighWaterMarks() map[string]map[int32]int64 {
	args := m.Called()
	return args.Get(0).(map[string]map[int32]int64)
}

func (m *MockConsumer) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockPartitionConsumer implements sarama.PartitionConsumer interface for testing
type MockPartitionConsumer struct {
	mock.Mock
	messages chan *sarama.ConsumerMessage
	errors   chan *sarama.ConsumerError
}

func NewMockPartitionConsumer() *MockPartitionConsumer {
	return &MockPartitionConsumer{
		messages: make(chan *sarama.ConsumerMessage, 10),
		errors:   make(chan *sarama.ConsumerError, 10),
	}
}

func (m *MockPartitionConsumer) AsyncClose() {
	m.Called()
	close(m.messages)
	close(m.errors)
}

func (m *MockPartitionConsumer) Close() error {
	args := m.Called()
	close(m.messages)
	close(m.errors)
	return args.Error(0)
}

func (m *MockPartitionConsumer) Messages() <-chan *sarama.ConsumerMessage {
	return m.messages
}

func (m *MockPartitionConsumer) Errors() <-chan *sarama.ConsumerError {
	return m.errors
}

func (m *MockPartitionConsumer) HighWaterMarkOffset() int64 {
	args := m.Called()
	return args.Get(0).(int64)
}

// MockHub implements the Hub interface for testing
type MockHub struct {
	mock.Mock
	broadcastCalls []WireMessage
}

func (m *MockHub) Broadcast(message WireMessage) {
	m.Called(message)
	m.broadcastCalls = append(m.broadcastCalls, message)
}

func (m *MockHub) Register(client *Client) {
	m.Called(client)
}

func (m *MockHub) Unregister(client *Client) {
	m.Called(client)
}

func (m *MockHub) Run() {
	m.Called()
}

func (m *MockHub) GetRoomUserCount(room string) int {
	args := m.Called(room)
	return args.Int(0)
}

func TestConsumeMessages_NilConsumer(t *testing.T) {
	hub := &MockHub{}

	// Test with nil consumer
	ConsumeMessages(nil, hub)

	// Should not panic and should log a message
	// Since we can't easily test log output, we just ensure it doesn't crash
}

func TestConsumeMessages_PartitionsError(t *testing.T) {
	mockConsumer := &MockConsumer{}
	hub := &MockHub{}

	// Mock Partitions to return an error
	mockConsumer.On("Partitions", "chat-in").Return([]int32{}, assert.AnError)

	ConsumeMessages(mockConsumer, hub)

	mockConsumer.AssertExpectations(t)
}

func TestConsumeMessages_ConsumePartitionError(t *testing.T) {
	mockConsumer := &MockConsumer{}
	hub := &MockHub{}

	// Mock Partitions to return one partition
	mockConsumer.On("Partitions", "chat-in").Return([]int32{0}, nil)

	// Mock ConsumePartition to return an error
	mockConsumer.On("ConsumePartition", "chat-in", int32(0), sarama.OffsetNewest).Return((*MockPartitionConsumer)(nil), assert.AnError)

	ConsumeMessages(mockConsumer, hub)

	mockConsumer.AssertExpectations(t)
}

func TestConsumeMessages_SuccessfulConsumption(t *testing.T) {
	mockConsumer := &MockConsumer{}
	hub := &MockHub{}
	mockPartitionConsumer := NewMockPartitionConsumer()

	// Mock Partitions to return one partition
	mockConsumer.On("Partitions", "chat-in").Return([]int32{0}, nil)

	// Mock ConsumePartition to return the mock partition consumer
	mockConsumer.On("ConsumePartition", "chat-in", int32(0), sarama.OffsetNewest).Return(mockPartitionConsumer, nil)

	// Prepare test message
	testMessage := WireMessage{
		ID:        "test-id",
		Username:  "testuser",
		Message:   "Hello, World!",
		Room:      "general",
		Timestamp: time.Now(),
	}

	messageBytes, _ := json.Marshal(testMessage)

	// Create a Kafka message
	kafkaMessage := &sarama.ConsumerMessage{
		Topic:     "chat-in",
		Partition: 0,
		Offset:    1,
		Value:     messageBytes,
	}

	// Set up hub expectation
	hub.On("Broadcast", mock.MatchedBy(func(msg WireMessage) bool {
		return msg.ID == testMessage.ID &&
			msg.Username == testMessage.Username &&
			msg.Message == testMessage.Message &&
			msg.Room == testMessage.Room
	}))

	// Start consuming in a goroutine
	go ConsumeMessages(mockConsumer, hub)

	// Send the message
	mockPartitionConsumer.messages <- kafkaMessage

	// Give some time for processing
	time.Sleep(100 * time.Millisecond)

	// Close the partition consumer
	mockPartitionConsumer.AsyncClose()

	// Verify expectations
	mockConsumer.AssertExpectations(t)
	hub.AssertExpectations(t)

	// Verify the message was broadcast
	assert.Len(t, hub.broadcastCalls, 1)
	assert.Equal(t, testMessage.ID, hub.broadcastCalls[0].ID)
	assert.Equal(t, testMessage.Username, hub.broadcastCalls[0].Username)
	assert.Equal(t, testMessage.Message, hub.broadcastCalls[0].Message)
	assert.Equal(t, testMessage.Room, hub.broadcastCalls[0].Room)
}

func TestConsumeMessages_InvalidJSON(t *testing.T) {
	mockConsumer := &MockConsumer{}
	hub := &MockHub{}
	mockPartitionConsumer := NewMockPartitionConsumer()

	// Mock Partitions to return one partition
	mockConsumer.On("Partitions", "chat-in").Return([]int32{0}, nil)

	// Mock ConsumePartition to return the mock partition consumer
	mockConsumer.On("ConsumePartition", "chat-in", int32(0), sarama.OffsetNewest).Return(mockPartitionConsumer, nil)

	// Create a Kafka message with invalid JSON
	kafkaMessage := &sarama.ConsumerMessage{
		Topic:     "chat-in",
		Partition: 0,
		Offset:    1,
		Value:     []byte("invalid json"),
	}

	// Hub should not receive any broadcast calls for invalid JSON
	// (no expectations set)

	// Start consuming in a goroutine
	go ConsumeMessages(mockConsumer, hub)

	// Send the invalid message
	mockPartitionConsumer.messages <- kafkaMessage

	// Give some time for processing
	time.Sleep(100 * time.Millisecond)

	// Close the partition consumer
	mockPartitionConsumer.AsyncClose()

	// Verify expectations
	mockConsumer.AssertExpectations(t)
	hub.AssertExpectations(t)

	// Verify no messages were broadcast
	assert.Len(t, hub.broadcastCalls, 0)
}

func TestConsumeMessages_ConsumerError(t *testing.T) {
	mockConsumer := &MockConsumer{}
	hub := &MockHub{}
	mockPartitionConsumer := NewMockPartitionConsumer()

	// Mock Partitions to return one partition
	mockConsumer.On("Partitions", "chat-in").Return([]int32{0}, nil)

	// Mock ConsumePartition to return the mock partition consumer
	mockConsumer.On("ConsumePartition", "chat-in", int32(0), sarama.OffsetNewest).Return(mockPartitionConsumer, nil)

	// Create a consumer error
	consumerError := &sarama.ConsumerError{
		Topic:     "chat-in",
		Partition: 0,
		Err:       assert.AnError,
	}

	// Start consuming in a goroutine
	go ConsumeMessages(mockConsumer, hub)

	// Send the error
	mockPartitionConsumer.errors <- consumerError

	// Give some time for processing
	time.Sleep(100 * time.Millisecond)

	// Close the partition consumer
	mockPartitionConsumer.AsyncClose()

	// Verify expectations
	mockConsumer.AssertExpectations(t)
	hub.AssertExpectations(t)

	// Verify no messages were broadcast due to error
	assert.Len(t, hub.broadcastCalls, 0)
}

func TestConsumeMessages_MultiplePartitions(t *testing.T) {
	mockConsumer := &MockConsumer{}
	hub := &MockHub{}
	mockPartitionConsumer1 := NewMockPartitionConsumer()
	mockPartitionConsumer2 := NewMockPartitionConsumer()

	// Mock Partitions to return two partitions
	mockConsumer.On("Partitions", "chat-in").Return([]int32{0, 1}, nil)

	// Mock ConsumePartition for both partitions
	mockConsumer.On("ConsumePartition", "chat-in", int32(0), sarama.OffsetNewest).Return(mockPartitionConsumer1, nil)
	mockConsumer.On("ConsumePartition", "chat-in", int32(1), sarama.OffsetNewest).Return(mockPartitionConsumer2, nil)

	// Prepare test messages for both partitions
	testMessage1 := WireMessage{
		ID:        "test-id-1",
		Username:  "testuser1",
		Message:   "Hello from partition 0!",
		Room:      "general",
		Timestamp: time.Now(),
	}

	testMessage2 := WireMessage{
		ID:        "test-id-2",
		Username:  "testuser2",
		Message:   "Hello from partition 1!",
		Room:      "general",
		Timestamp: time.Now(),
	}

	messageBytes1, _ := json.Marshal(testMessage1)
	messageBytes2, _ := json.Marshal(testMessage2)

	// Create Kafka messages
	kafkaMessage1 := &sarama.ConsumerMessage{
		Topic:     "chat-in",
		Partition: 0,
		Offset:    1,
		Value:     messageBytes1,
	}

	kafkaMessage2 := &sarama.ConsumerMessage{
		Topic:     "chat-in",
		Partition: 1,
		Offset:    1,
		Value:     messageBytes2,
	}

	// Set up hub expectations for both messages
	hub.On("Broadcast", mock.MatchedBy(func(msg WireMessage) bool {
		return msg.ID == testMessage1.ID
	}))
	hub.On("Broadcast", mock.MatchedBy(func(msg WireMessage) bool {
		return msg.ID == testMessage2.ID
	}))

	// Start consuming in a goroutine
	go ConsumeMessages(mockConsumer, hub)

	// Send messages to both partitions
	mockPartitionConsumer1.messages <- kafkaMessage1
	mockPartitionConsumer2.messages <- kafkaMessage2

	// Give some time for processing
	time.Sleep(200 * time.Millisecond)

	// Close both partition consumers
	mockPartitionConsumer1.AsyncClose()
	mockPartitionConsumer2.AsyncClose()

	// Verify expectations
	mockConsumer.AssertExpectations(t)
	hub.AssertExpectations(t)

	// Verify both messages were broadcast
	assert.Len(t, hub.broadcastCalls, 2)
}
