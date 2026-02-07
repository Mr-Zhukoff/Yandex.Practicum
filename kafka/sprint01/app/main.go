package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"kafka-app/consumer"
	"kafka-app/producer"
)

const (
	defaultBrokers = "localhost:9092,localhost:9093,localhost:9094"
	defaultTopic   = "my-topic"
)

func main() {
	brokersStr := getEnv("KAFKA_BROKERS", defaultBrokers)
	brokers := strings.Split(brokersStr, ",")
	topic := getEnv("KAFKA_TOPIC", defaultTopic)

	log.Printf("Starting Kafka application: brokers=%v, topic=%s", brokers, topic)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create producer
	prod := producer.NewProducer(brokers, topic)
	defer func() {
		if err := prod.Close(); err != nil {
			log.Printf("Error closing producer: %v", err)
		}
	}()

	// Create SingleMessageConsumer (auto-commit, unique group_id)
	singleConsumer := consumer.NewSingleMessageConsumer(brokers, topic, "single-message-consumer-group")
	defer func() {
		if err := singleConsumer.Close(); err != nil {
			log.Printf("Error closing SingleMessageConsumer: %v", err)
		}
	}()

	// Create BatchMessageConsumer (manual commit, unique group_id)
	batchConsumer := consumer.NewBatchMessageConsumer(brokers, topic, "batch-message-consumer-group")
	defer func() {
		if err := batchConsumer.Close(); err != nil {
			log.Printf("Error closing BatchMessageConsumer: %v", err)
		}
	}()

	// Start producer in a goroutine
	go prod.Start(ctx)

	// Start SingleMessageConsumer in a goroutine
	go singleConsumer.Start(ctx)

	// Start BatchMessageConsumer in a goroutine
	go batchConsumer.Start(ctx)

	log.Println("All components started. Press Ctrl+C to exit.")

	// Wait for termination signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigChan
	log.Printf("Received signal %v, shutting down...", sig)
	cancel()
}

// getEnv returns the value of an environment variable or a default value.
func getEnv(key, defaultVal string) string {
	if val, exists := os.LookupEnv(key); exists {
		return val
	}
	return defaultVal
}
