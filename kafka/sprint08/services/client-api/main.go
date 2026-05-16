package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/segmentio/kafka-go"

	"marketplace-analytics/internal/events"
	"marketplace-analytics/internal/kafkautil"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "search":
		runSearch(os.Args[2:])
	case "recommend":
		runRecommend(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func runSearch(args []string) {
	fs := flag.NewFlagSet("search", flag.ExitOnError)
	brokersCSV := fs.String("brokers", "localhost:9092", "comma-separated Kafka brokers")
	topic := fs.String("topic", "client.search.requests", "Kafka topic for search requests")
	databaseURL := fs.String("database-url", "postgres://marketplace:marketplace@localhost:5432/marketplace?sslmode=disable", "PostgreSQL URL")
	userID := fs.String("user-id", "", "user ID")
	query := fs.String("query", "", "search query")
	_ = fs.Parse(args)

	if *userID == "" || *query == "" {
		log.Fatal("--user-id and --query are required")
	}

	req := events.SearchRequest{UserID: *userID, Query: *query}
	publishEvent(*brokersCSV, *topic, *userID, "product_search_requested", req)

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, *databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = conn.Close(ctx) }()

	rows, err := conn.Query(ctx, `
SELECT product_id, name, category, price_amount, price_currency
FROM products
WHERE search_vector @@ plainto_tsquery('russian', $1)
   OR name ILIKE '%' || $1 || '%'
ORDER BY name
LIMIT 10`, *query)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("Search results:")
	for rows.Next() {
		var productID, name, category, currency string
		var amount float64
		if err := rows.Scan(&productID, &name, &category, &amount, &currency); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("- %s | %s | %s | %.2f %s\n", productID, name, category, amount, currency)
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
}

func runRecommend(args []string) {
	fs := flag.NewFlagSet("recommend", flag.ExitOnError)
	brokersCSV := fs.String("brokers", "localhost:9092", "comma-separated Kafka brokers")
	topic := fs.String("topic", "client.recommendation.requests", "Kafka topic for recommendation requests")
	databaseURL := fs.String("database-url", "postgres://marketplace:marketplace@localhost:5432/marketplace?sslmode=disable", "PostgreSQL URL")
	userID := fs.String("user-id", "", "user ID")
	category := fs.String("category", "", "category")
	_ = fs.Parse(args)

	if *userID == "" || *category == "" {
		log.Fatal("--user-id and --category are required")
	}

	req := events.RecommendationRequest{UserID: *userID, Category: *category}
	publishEvent(*brokersCSV, *topic, *userID, "recommendation_requested", req)

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, *databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = conn.Close(ctx) }()

	rows, err := conn.Query(ctx, `
SELECT product_id, name, price_amount, price_currency
FROM products
WHERE category = $1
ORDER BY stock_available DESC, updated_at DESC
LIMIT 5`, *category)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("Current recommendations:")
	for rows.Next() {
		var productID, name, currency string
		var amount float64
		if err := rows.Scan(&productID, &name, &amount, &currency); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("- %s | %s | %.2f %s\n", productID, name, amount, currency)
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
}

func publishEvent[T any](brokersCSV, topic, key, eventType string, payload T) {
	envelope := events.EventEnvelope[T]{
		EventID:   uuid.NewString(),
		EventType: eventType,
		EventTime: time.Now().UTC(),
		Source:    "client-api",
		Payload:   payload,
	}
	value, err := json.Marshal(envelope)
	if err != nil {
		log.Fatal(err)
	}
	writer := kafkautil.NewWriter(kafkautil.Brokers(brokersCSV), topic)
	defer func() { _ = writer.Close() }()
	if err := writer.WriteMessages(context.Background(), kafka.Message{Key: []byte(key), Value: value, Time: envelope.EventTime}); err != nil {
		log.Fatal(err)
	}
}

func usage() {
	fmt.Println("usage:")
	fmt.Println("  client-api search --user-id ID --query QUERY")
	fmt.Println("  client-api recommend --user-id ID --category CATEGORY")
}
