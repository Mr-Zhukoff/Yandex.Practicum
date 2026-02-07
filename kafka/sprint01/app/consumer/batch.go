package consumer

import (
	"context"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

// BatchMessageConsumer reads at least 10 messages per cycle,
// processes them in a loop, and commits offsets once after the batch.
type BatchMessageConsumer struct {
	reader    *kafka.Reader
	batchSize int
}

// NewBatchMessageConsumer creates a consumer that processes messages in batches.
// It uses MinBytes and MaxWait to ensure data accumulates before fetching.
// Auto-commit is disabled; offsets are committed manually after processing each batch.
func NewBatchMessageConsumer(brokers []string, topic, groupID string) *BatchMessageConsumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		Topic:       topic,
		GroupID:     groupID,
		MinBytes:    1024,              // fetch.min.bytes — minimum 1KB per fetch
		MaxBytes:    10e6,              // max 10MB
		MaxWait:     3 * time.Second,   // fetch.max.wait.ms — wait up to 3s for data
		StartOffset: kafka.FirstOffset,
	})

	return &BatchMessageConsumer{
		reader:    r,
		batchSize: 10,
	}
}

// Start subscribes to the topic and begins consuming messages in batches.
// It collects at least batchSize messages, processes them, and then commits.
// On error, it logs the message and continues working.
func (b *BatchMessageConsumer) Start(ctx context.Context) {
	log.Printf("[BatchMessageConsumer] Started, collecting batches of %d messages...", b.batchSize)

	for {
		select {
		case <-ctx.Done():
			log.Println("[BatchMessageConsumer] Stopping...")
			return
		default:
		}

		batch := make([]kafka.Message, 0, b.batchSize)

		// Collect messages until we have at least batchSize
		for len(batch) < b.batchSize {
			select {
			case <-ctx.Done():
				log.Println("[BatchMessageConsumer] Stopping mid-batch...")
				return
			default:
			}

			// FetchMessage does NOT auto-commit — we commit manually later
			msg, err := b.reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Printf("[BatchMessageConsumer] ERROR fetching message: %v", err)
				continue
			}
			batch = append(batch, msg)
		}

		// Process the batch
		for i, msg := range batch {
			log.Printf("[BatchMessageConsumer] Batch message [%d/%d]: topic=%s partition=%d offset=%d key=%s value=%s",
				i+1, len(batch),
				msg.Topic,
				msg.Partition,
				msg.Offset,
				string(msg.Key),
				string(msg.Value))
		}

		// Commit offsets once after processing the entire batch
		if err := b.reader.CommitMessages(ctx, batch...); err != nil {
			log.Printf("[BatchMessageConsumer] ERROR committing offsets: %v", err)
		} else {
			log.Printf("[BatchMessageConsumer] Committed offsets for %d messages", len(batch))
		}
	}
}

// Close closes the consumer.
func (b *BatchMessageConsumer) Close() error {
	return b.reader.Close()
}
