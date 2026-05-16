package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"

	"github.com/jackc/pgx/v5"

	"marketplace-analytics/internal/events"
	"marketplace-analytics/internal/kafkautil"
)

func main() {
	brokersCSV := flag.String("brokers", "localhost:9092", "comma-separated Kafka brokers")
	topic := flag.String("topic", "shop.products.allowed", "allowed products topic")
	groupID := flag.String("group", "postgres-sink", "Kafka consumer group")
	databaseURL := flag.String("database-url", "postgres://marketplace:marketplace@localhost:5432/marketplace?sslmode=disable", "PostgreSQL URL")
	flag.Parse()

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, *databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = conn.Close(ctx) }()

	reader := kafkautil.NewReader(kafkautil.Brokers(*brokersCSV), *topic, *groupID)
	defer func() { _ = reader.Close() }()

	log.Printf("postgres-sink started")
	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			log.Fatal(err)
		}

		var envelope events.EventEnvelope[events.Product]
		if err := json.Unmarshal(msg.Value, &envelope); err != nil {
			log.Printf("skip invalid product event key=%q: %v", string(msg.Key), err)
			continue
		}
		if err := upsertProduct(ctx, conn, envelope.Payload, msg.Value); err != nil {
			log.Printf("upsert product %q: %v", envelope.Payload.ProductID, err)
			continue
		}
		log.Printf("upserted product %s", envelope.Payload.ProductID)
	}
}

func upsertProduct(ctx context.Context, conn *pgx.Conn, product events.Product, raw []byte) error {
	_, err := conn.Exec(ctx, `
INSERT INTO products (
    product_id, name, description, category, brand, sku, store_id,
    price_amount, price_currency, stock_available, stock_reserved,
    tags, raw_payload, created_at, updated_at, ingested_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7,
    $8, $9, $10, $11,
    $12, $13, $14, $15, now()
)
ON CONFLICT (product_id) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    category = EXCLUDED.category,
    brand = EXCLUDED.brand,
    sku = EXCLUDED.sku,
    store_id = EXCLUDED.store_id,
    price_amount = EXCLUDED.price_amount,
    price_currency = EXCLUDED.price_currency,
    stock_available = EXCLUDED.stock_available,
    stock_reserved = EXCLUDED.stock_reserved,
    tags = EXCLUDED.tags,
    raw_payload = EXCLUDED.raw_payload,
    created_at = EXCLUDED.created_at,
    updated_at = EXCLUDED.updated_at,
    ingested_at = now()`,
		product.ProductID,
		product.Name,
		product.Description,
		product.Category,
		product.Brand,
		product.SKU,
		product.StoreID,
		product.Price.Amount,
		product.Price.Currency,
		product.Stock.Available,
		product.Stock.Reserved,
		product.Tags,
		raw,
		product.CreatedAt,
		product.UpdatedAt,
	)
	return err
}
