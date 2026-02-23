package main

import (
	"encoding/json"
	"fmt"
	"kafka-message-filter/models"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/IBM/sarama"
)

func getBrokers() []string {
	brokersEnv := os.Getenv("KAFKA_BROKERS")
	if brokersEnv != "" {
		return strings.Split(brokersEnv, ",")
	}
	return []string{"localhost:9092"}
}

var (
	brokers = getBrokers()
)

func main() {
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true
	config.Consumer.Offsets.Initial = sarama.OffsetOldest

	consumer, err := sarama.NewConsumer(brokers, config)
	if err != nil {
		log.Fatalf("Failed to create consumer: %v", err)
	}
	defer consumer.Close()

	partitionConsumer, err := consumer.ConsumePartition("filtered_messages", 0, sarama.OffsetOldest)
	if err != nil {
		log.Fatalf("Failed to start partition consumer: %v", err)
	}
	defer partitionConsumer.Close()

	fmt.Println("=== Filtered Messages Consumer ===")
	fmt.Println("Listening for filtered messages...")
	fmt.Println("Press Ctrl+C to stop\n")

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case msg := <-partitionConsumer.Messages():
			var message models.Message
			if err := json.Unmarshal(msg.Value, &message); err != nil {
				log.Printf("Error unmarshaling message: %v", err)
				continue
			}

			fmt.Printf("📨 [%s] From: %s → To: %s\n", message.ID, message.From, message.To)
			fmt.Printf("   Content: %s\n", message.Content)
			fmt.Printf("   Timestamp: %d\n\n", message.Timestamp)

		case err := <-partitionConsumer.Errors():
			log.Printf("Error: %v", err)

		case <-signals:
			fmt.Println("\nShutting down consumer...")
			return
		}
	}
}
