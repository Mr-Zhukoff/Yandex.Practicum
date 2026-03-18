package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"
)

func main() {
	topics := []string{
		"postgres-cdc.public.users",
		"postgres-cdc.public.orders",
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{"127.0.0.1:9092"},
		GroupID:        "cdc-demo-reader-go",
		GroupTopics:    topics,
		MinBytes:       1,
		MaxBytes:       10e6,
		StartOffset:    kafka.FirstOffset,
		CommitInterval: time.Second,
	})
	defer reader.Close()

	fmt.Println("Reading Debezium CDC events from topics:")
	for _, t := range topics {
		fmt.Printf(" - %s\n", t)
	}
	fmt.Println("Press Ctrl+C to stop")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nStopping consumer...")
		cancel()
	}()

	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				break
			}
			log.Printf("read error: %v", err)
			continue
		}

		fmt.Printf("\nTopic: %s | Partition: %d | Offset: %d\n", msg.Topic, msg.Partition, msg.Offset)
		fmt.Printf("Key: %s\n", string(msg.Key))
		fmt.Printf("Value: %s\n", string(msg.Value))
	}

	fmt.Println("Consumer stopped")
}

