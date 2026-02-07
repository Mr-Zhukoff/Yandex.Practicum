package producer

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

// Producer sends messages to a Kafka topic asynchronously.
type Producer struct {
	writer *kafka.Writer
	topic  string
}

// NewProducer creates a new Kafka producer.
func NewProducer(brokers []string, topic string) *Producer {
	w := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    1,
		RequiredAcks: kafka.RequireAll,
		Async:        true,
		Completion: func(messages []kafka.Message, err error) {
			for _, msg := range messages {
				if err != nil {
					log.Printf("[Producer] ERROR delivering message key=%s: %v", string(msg.Key), err)
				} else {
					log.Printf("[Producer] Delivered message key=%s partition=%d offset=%d",
						string(msg.Key), msg.Partition, msg.Offset)
				}
			}
		},
	}

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
		value := fmt.Sprintf("Message #%d at %s", msgNum, time.Now().Format(time.RFC3339))
		key := fmt.Sprintf("key-%d", msgNum)

		err := p.writer.WriteMessages(ctx, kafka.Message{
			Key:   []byte(key),
			Value: []byte(value),
		})

		if err != nil {
			log.Printf("[Producer] ERROR sending message: %v", err)
		} else {
			log.Printf("[Producer] Sent: %s", value)
		}

		time.Sleep(500 * time.Millisecond)
	}
}

// Close flushes and closes the producer.
func (p *Producer) Close() error {
	return p.writer.Close()
}
