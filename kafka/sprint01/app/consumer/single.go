package consumer

import (
	"context"
	"log"
	"time"

	"kafka-app/model"

	"github.com/segmentio/kafka-go"
)

// SingleMessageConsumer reads one message at a time with auto-commit enabled.
// Uses ReadMessage which automatically commits the offset after reading.
type SingleMessageConsumer struct {
	reader *kafka.Reader
}

// NewSingleMessageConsumer creates a consumer that processes messages one by one
// and auto-commits offsets via CommitInterval.
func NewSingleMessageConsumer(brokers []string, topic, groupID string) *SingleMessageConsumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       1,
		MaxBytes:       10e6,
		CommitInterval: time.Second, // auto-commit every second
		StartOffset:    kafka.FirstOffset,
	})

	return &SingleMessageConsumer{
		reader: r,
	}
}

// Start subscribes to the topic and begins consuming messages one at a time.
// ReadMessage automatically commits the offset after successful read.
// On error, it logs the message and continues working.
func (s *SingleMessageConsumer) Start(ctx context.Context) {
	log.Println("[SingleMessageConsumer] Started, waiting for messages...")

	for {
		select {
		case <-ctx.Done():
			log.Println("[SingleMessageConsumer] Stopping...")
			return
		default:
		}

		msg, err := s.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("[SingleMessageConsumer] ERROR reading message: %v", err)
			continue
		}

		// Log raw JSON received
		log.Printf("[SingleMessageConsumer] Received raw JSON: %s", string(msg.Value))

		// Deserialize JSON message
		message, err := model.FromJSON(msg.Value)
		if err != nil {
			log.Printf("[SingleMessageConsumer] ERROR deserializing message: %v", err)
			log.Printf("[SingleMessageConsumer] Raw data: topic=%s partition=%d offset=%d key=%s",
				msg.Topic,
				msg.Partition,
				msg.Offset,
				string(msg.Key))
			continue
		}

		// Log deserialized message
		log.Printf("[SingleMessageConsumer] Deserialized message: topic=%s partition=%d offset=%d key=%s | ID=%d Content=%s Timestamp=%s Source=%s",
			msg.Topic,
			msg.Partition,
			msg.Offset,
			string(msg.Key),
			message.ID,
			message.Content,
			message.Timestamp.Format("2006-01-02 15:04:05"),
			message.Source)
	}
}

// Close closes the consumer.
func (s *SingleMessageConsumer) Close() error {
	return s.reader.Close()
}
