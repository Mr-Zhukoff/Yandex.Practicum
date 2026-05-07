package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"sprint07-kafka/internal/kafkaconfig"

	"github.com/segmentio/kafka-go"
)

func main() {
	cfg := kafkaconfig.Load()
	transport := cfg.Transport()
	dialer := &kafka.Dialer{Timeout: 10 * time.Second, DualStack: true}
	if transport != nil {
		dialer.TLS = transport.TLS
		dialer.SASLMechanism = transport.SASL
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     cfg.Brokers,
		Topic:       "user-events",
		GroupID:     "user-events-group",
		MinBytes:    1,
		MaxBytes:    10e6,
		StartOffset: kafka.FirstOffset,
		Dialer:      dialer,
	})
	defer reader.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for {
		m, err := reader.ReadMessage(ctx)
		if err != nil {
			log.Printf("consumer: stop reading: %v", err)
			break
		}

		fmt.Printf("consumer: received topic=%s partition=%d offset=%d key=%s value=%s\n",
			m.Topic, m.Partition, m.Offset, string(m.Key), string(m.Value))
	}
}
