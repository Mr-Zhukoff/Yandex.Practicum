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
		content := fmt.Sprintf("Message #%d at %s", msgNum, time.Now().Format(time.RFC3339))
		key := fmt.Sprintf("key-%d", msgNum)

		// Create a structured message
		msg := model.NewMessage(msgNum, content, "producer")

		// Serialize message to JSON
		jsonData, err := msg.ToJSON()
		if err != nil {
			log.Printf("[Producer] ERROR serializing message: %v", err)
			continue
		}

		// Log the JSON message being sent
		log.Printf("[Producer] Sending JSON message: %s", string(jsonData))

		err = p.writer.WriteMessages(ctx, kafka.Message{
			Key:   []byte(key),
			Value: jsonData,
		})

		if err != nil {
			log.Printf("[Producer] ERROR sending message: %v", err)
		} else {
			log.Printf("[Producer] Sent message with key=%s, id=%d, content=%s", key, msg.ID, msg.Content)
		}

		time.Sleep(500 * time.Millisecond)
	}
}

// Close flushes and closes the producer.
func (p *Producer) Close() error {
	return p.writer.Close()
}
