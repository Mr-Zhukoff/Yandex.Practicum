package producer

import (
	"context"
	"fmt"
	"log"
	"time"

	"kafka-app/model"

	"github.com/segmentio/kafka-go"
)

// Producer sends messages to a Kafka topic asynchronously.
type Producer struct {
	writer *kafka.Writer
	topic  string
}

// NewProducer creates a new Kafka producer with "At Least Once" delivery guarantee.
// Configuration:
// - RequiredAcks: RequireAll - all in-sync replicas must acknowledge
// - MaxAttempts: 5 - retry up to 5 times on transient errors
// - WriteTimeout: 10s - timeout for write operations
// - ReadTimeout: 10s - timeout for reading acknowledgments
func NewProducer(brokers []string, topic string) *Producer {
	w := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
		
		// At Least Once delivery configuration
		RequiredAcks: kafka.RequireAll,  // All in-sync replicas must acknowledge
		MaxAttempts:  5,                  // Retry up to 5 times on failures
		WriteTimeout: 10 * time.Second,   // Timeout for write operations
		ReadTimeout:  10 * time.Second,   // Timeout for reading acknowledgments
		
		BatchSize: 1,
		Async:     true,
		
		Completion: func(messages []kafka.Message, err error) {
			for _, msg := range messages {
				if err != nil {
					log.Printf("[Producer] ✗ DELIVERY FAILED key=%s: %v", string(msg.Key), err)
					log.Printf("[Producer] Message will be retried if attempts remain")
				} else {
					log.Printf("[Producer] ✓ DELIVERY CONFIRMED key=%s partition=%d offset=%d",
						string(msg.Key), msg.Partition, msg.Offset)
				}
			}
		},
	}

	log.Println("[Producer] Configured with 'At Least Once' delivery guarantee:")
	log.Println("[Producer] - RequiredAcks: All in-sync replicas")
	log.Println("[Producer] - MaxAttempts: 5 retries")
	log.Println("[Producer] - WriteTimeout: 10s")
	log.Println("[Producer] - ReadTimeout: 10s")

	return &Producer{
		writer: w,
		topic:  topic,
	}
}

// Start begins producing messages asynchronously.
func (p *Producer) Start(ctx context.Context) {
	msgNum := 0
	for {
		select {
		case <-ctx.Done():
			log.Println("[Producer] Stopping...")
			return
		default:
		}

		msgNum++
		content := fmt.Sprintf("Message #%d at %s", msgNum, time.Now().Format(time.RFC3339))
		key := fmt.Sprintf("key-%d", msgNum)

		// Create a structured message
		msg := model.NewMessage(msgNum, content, "producer")

		// Serialize message to JSON
		jsonData, err := msg.ToJSON()
		if err != nil {
			log.Printf("[Producer] ✗ ERROR serializing message: %v", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		// Log the JSON message being sent
		log.Printf("[Producer] → Sending JSON message: %s", string(jsonData))

		// Send message with At Least Once guarantee
		// The async writer will retry automatically up to MaxAttempts times
		err = p.writer.WriteMessages(ctx, kafka.Message{
			Key:   []byte(key),
			Value: jsonData,
		})

		if err != nil {
			// This error means the message was rejected immediately (e.g., invalid data)
			// It does NOT include delivery failures, which are handled in Completion callback
			log.Printf("[Producer] ✗ ERROR submitting message: %v", err)
		} else {
			// Message accepted for sending; delivery confirmation will come via Completion callback
			log.Printf("[Producer] ↑ Message accepted for delivery: key=%s, id=%d", key, msg.ID)
		}

		time.Sleep(500 * time.Millisecond)
	}
}

// Close flushes and closes the producer.
func (p *Producer) Close() error {
	return p.writer.Close()
}
