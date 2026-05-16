package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"marketplace-analytics/internal/events"
	"marketplace-analytics/internal/kafkautil"
	"marketplace-analytics/internal/productio"
)

func main() {
	file := flag.String("file", "./data/products.json", "path to product JSON file")
	brokersCSV := flag.String("brokers", "localhost:9092", "comma-separated Kafka brokers")
	topic := flag.String("topic", "shop.products.raw", "Kafka topic for raw products")
	flag.Parse()

	products, err := productio.ReadProducts(*file)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	writer := kafkautil.NewWriter(kafkautil.Brokers(*brokersCSV), *topic)
	defer func() { _ = writer.Close() }()

	for _, product := range products {
		if err := events.ValidateProduct(product); err != nil {
			log.Printf("skip product %q: %v", product.ProductID, err)
			continue
		}

		envelope := events.EventEnvelope[events.Product]{
			EventID:   uuid.NewString(),
			EventType: "product_created_or_updated",
			EventTime: time.Now().UTC(),
			Source:    "shop-api",
			Payload:   product,
		}

		value, err := json.Marshal(envelope)
		if err != nil {
			log.Printf("marshal product %q: %v", product.ProductID, err)
			continue
		}

		msg := kafka.Message{
			Key:   []byte(product.ProductID),
			Value: value,
			Time:  envelope.EventTime,
		}
		if err := writer.WriteMessages(ctx, msg); err != nil {
			log.Fatalf("write product %q: %v", product.ProductID, err)
		}

		fmt.Printf("sent product %s to %s\n", product.ProductID, *topic)
	}
}
