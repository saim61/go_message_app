package main

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/IBM/sarama"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/saim61/go_message_app/internal/gateway"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}

	// Initialize Kafka producer
	producer, err := createKafkaProducer()
	if err != nil {
		log.Fatal("Failed to create Kafka producer:", err)
	}
	defer producer.Close()

	// Initialize Kafka consumer
	consumer, err := createKafkaConsumer()
	if err != nil {
		log.Fatal("Failed to create Kafka consumer:", err)
	}
	defer consumer.Close()

	// Create hub for managing WebSocket connections
	hub := gateway.NewHub()
	go hub.Run()

	// Start consuming messages from Kafka
	go gateway.ConsumeMessages(consumer, hub)

	// Setup HTTP routes
	router := mux.NewRouter()
	router.HandleFunc("/health", gateway.HealthHandler).Methods("GET")
	router.HandleFunc("/", gateway.ChatHandler).Methods("GET")
	router.HandleFunc("/ws", gateway.WSHandler(hub, producer)).Methods("GET")

	// Start HTTP server
	server := &http.Server{
		Addr:    ":8081",
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		log.Println("Gateway server starting on :8081")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server failed to start:", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}

func createKafkaProducer() (sarama.SyncProducer, error) {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5
	config.Producer.Return.Successes = true

	// TLS configuration
	config.Net.TLS.Enable = false
	config.Net.TLS.Config = &tls.Config{
		InsecureSkipVerify: true,
	}

	brokers := []string{getEnv("KAFKA_BROKERS", "kafka:9092")}
	log.Printf("Connecting to Kafka brokers: %v", brokers)

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, err
	}

	log.Println("Kafka producer connected successfully")
	return producer, nil
}

func createKafkaConsumer() (sarama.Consumer, error) {
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true

	// TLS configuration
	config.Net.TLS.Enable = false
	config.Net.TLS.Config = &tls.Config{
		InsecureSkipVerify: true,
	}

	brokers := []string{getEnv("KAFKA_BROKERS", "kafka:9092")}
	log.Printf("Connecting to Kafka consumer brokers: %v", brokers)

	consumer, err := sarama.NewConsumer(brokers, config)
	if err != nil {
		return nil, err
	}

	log.Println("Kafka consumer connected successfully")
	return consumer, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
