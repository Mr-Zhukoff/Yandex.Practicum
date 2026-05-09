package main

import (
	"context"
	"fmt"
	"log"
	"sync"
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

	readerUserEvents := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     cfg.Brokers,
		Topic:       "user-events",
		GroupID:     "user-events-group",
		MinBytes:    1,
		MaxBytes:    10e6,
		StartOffset: kafka.FirstOffset,
		Dialer:      dialer,
	})
	defer readerUserEvents.Close()

	readerUsers := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     cfg.Brokers,
		Topic:       "users",
		GroupID:     "users-group",
		MinBytes:    1,
		MaxBytes:    10e6,
		StartOffset: kafka.FirstOffset,
		Dialer:      dialer,
	})
	defer readerUsers.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	readLoop := func(name string, reader *kafka.Reader) {
		for {
			m, err := reader.ReadMessage(ctx)
			if err != nil {
				log.Printf("consumer[%s]: stop reading: %v", name, err)
				break
			}

			fmt.Printf("consumer[%s]: received topic=%s partition=%d offset=%d key=%s value=%s\n",
				name, m.Topic, m.Partition, m.Offset, string(m.Key), string(m.Value))
		}
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		readLoop("user-events", readerUserEvents)
	}()
	go func() {
		defer wg.Done()
		readLoop("users", readerUsers)
	}()

	wg.Wait()
}
