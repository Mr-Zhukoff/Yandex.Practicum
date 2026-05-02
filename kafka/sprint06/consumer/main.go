package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/segmentio/kafka-go"
)

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}

func main() {
	brokers := getEnv("KAFKA_BROKERS", "localhost:9093")
	topic := getEnv("KAFKA_TOPIC", "input-topic")
	groupID := getEnv("KAFKA_GROUP_ID", "go-consumer-group")

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{brokers},
		Topic:   topic,
		GroupID: groupID,
	})
	defer reader.Close()

	log.Printf("consumer started: brokers=%s topic=%s group.id=%s", brokers, topic, groupID)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("shutdown signal received")
			return
		default:
			msg, err := reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					log.Println("consumer stopped")
					return
				}
				log.Printf("read error: %v", err)
				continue
			}

			fmt.Printf(
				"topic=%s partition=%d offset=%d key=%s value=%s\n",
				msg.Topic,
				msg.Partition,
				msg.Offset,
				string(msg.Key),
				string(msg.Value),
			)

			if err = reader.CommitMessages(ctx, msg); err != nil {
				log.Printf("commit error: %v", err)
			}
		}
	}
}
