package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"

	"marketplace-analytics/internal/events"
	"marketplace-analytics/internal/kafkautil"
)

func main() {
	brokersCSV := flag.String("brokers", "localhost:9092", "comma-separated Kafka brokers")
	rawTopic := flag.String("raw-topic", "shop.products.raw", "raw products topic")
	allowedTopic := flag.String("allowed-topic", "shop.products.allowed", "allowed products topic")
	rejectedTopic := flag.String("rejected-topic", "shop.products.rejected", "rejected products topic")
	dlqTopic := flag.String("dlq-topic", "shop.products.dlq", "dead-letter topic")
	forbiddenTopic := flag.String("forbidden-topic", "forbidden.products.state", "forbidden products state topic")
	groupID := flag.String("group", "product-filter", "Kafka consumer group")
	flag.Parse()

	brokers := kafkautil.Brokers(*brokersCSV)
	ctx := context.Background()
	forbidden := &forbiddenStore{items: map[string]events.ForbiddenProduct{}}

	go consumeForbiddenState(ctx, brokers, *forbiddenTopic, *groupID+"-forbidden", forbidden)

	rawReader := kafkautil.NewReader(brokers, *rawTopic, *groupID)
	defer func() { _ = rawReader.Close() }()
	allowedWriter := kafkautil.NewWriter(brokers, *allowedTopic)
	rejectedWriter := kafkautil.NewWriter(brokers, *rejectedTopic)
	dlqWriter := kafkautil.NewWriter(brokers, *dlqTopic)
	defer func() { _ = allowedWriter.Close(); _ = rejectedWriter.Close(); _ = dlqWriter.Close() }()

	log.Printf("product-filter started; this baseline processor will be replaced with Goka state processing in the next implementation slice")
	for {
		msg, err := rawReader.ReadMessage(ctx)
		if err != nil {
			log.Fatal(err)
		}
		processProduct(ctx, msg, forbidden, allowedWriter, rejectedWriter, dlqWriter)
	}
}

type forbiddenStore struct {
	mu    sync.RWMutex
	items map[string]events.ForbiddenProduct
}

func (s *forbiddenStore) set(record events.ForbiddenProduct) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if record.Active {
		s.items[record.ProductID] = record
		return
	}
	delete(s.items, record.ProductID)
}

func (s *forbiddenStore) isForbidden(productID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.items[productID]
	return ok
}

func consumeForbiddenState(ctx context.Context, brokers []string, topic, groupID string, store *forbiddenStore) {
	reader := kafkautil.NewReader(brokers, topic, groupID)
	defer func() { _ = reader.Close() }()
	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			log.Printf("read forbidden state: %v", err)
			return
		}
		var record events.ForbiddenProduct
		if err := json.Unmarshal(msg.Value, &record); err != nil {
			log.Printf("invalid forbidden state for key %q: %v", string(msg.Key), err)
			continue
		}
		store.set(record)
		log.Printf("forbidden state updated: product_id=%s active=%t", record.ProductID, record.Active)
	}
}

func processProduct(ctx context.Context, msg kafka.Message, forbidden *forbiddenStore, allowedWriter, rejectedWriter, dlqWriter *kafka.Writer) {
	var envelope events.EventEnvelope[events.Product]
	if err := json.Unmarshal(msg.Value, &envelope); err != nil {
		writeDLQ(ctx, dlqWriter, msg, "invalid JSON: "+err.Error())
		return
	}
	if err := events.ValidateProduct(envelope.Payload); err != nil {
		writeDLQ(ctx, dlqWriter, msg, "invalid product: "+err.Error())
		return
	}

	product := envelope.Payload
	if forbidden.isForbidden(product.ProductID) {
		rejected := events.EventEnvelope[events.RejectedProduct]{
			EventID:   envelope.EventID,
			EventType: "product_rejected",
			EventTime: time.Now().UTC(),
			Source:    "product-filter",
			Payload: events.RejectedProduct{
				Product: product,
				Reason:  "product is forbidden",
			},
		}
		writeJSON(ctx, rejectedWriter, product.ProductID, rejected)
		log.Printf("rejected forbidden product %s", product.ProductID)
		return
	}

	writeJSON(ctx, allowedWriter, product.ProductID, envelope)
	log.Printf("allowed product %s", product.ProductID)
}

func writeDLQ(ctx context.Context, writer *kafka.Writer, msg kafka.Message, reason string) {
	dlq := events.EventEnvelope[events.DeadLetter]{
		EventID:   string(msg.Key),
		EventType: "dead_letter",
		EventTime: time.Now().UTC(),
		Source:    "product-filter",
		Payload: events.DeadLetter{
			RawPayload: string(msg.Value),
			Reason:     reason,
			Source:     "shop.products.raw",
		},
	}
	writeJSON(ctx, writer, string(msg.Key), dlq)
}

func writeJSON(ctx context.Context, writer *kafka.Writer, key string, value any) {
	data, err := json.Marshal(value)
	if err != nil {
		log.Printf("marshal output: %v", err)
		return
	}
	if err := writer.WriteMessages(ctx, kafka.Message{Key: []byte(key), Value: data, Time: time.Now().UTC()}); err != nil {
		log.Printf("write output: %v", err)
	}
}
