package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"marketplace-analytics/internal/events"
	"marketplace-analytics/internal/kafkautil"
)

type productStore struct {
	mu       sync.RWMutex
	products map[string]events.Product
}

func main() {
	brokersCSV := flag.String("brokers", "localhost:9192", "comma-separated secondary Kafka brokers")
	allowedTopic := flag.String("allowed-topic", "shop.products.allowed", "allowed products topic")
	searchTopic := flag.String("search-topic", "client.search.requests", "search requests topic")
	recommendationRequestsTopic := flag.String("recommendation-requests-topic", "client.recommendation.requests", "recommendation requests topic")
	recommendationsTopic := flag.String("recommendations-topic", "analytics.recommendations", "recommendations output topic")
	groupID := flag.String("group", "hdfs-ingestor", "Kafka consumer group prefix")
	dataDir := flag.String("data-dir", "/data/hdfs", "HDFS-compatible local data lake directory")
	tlsOptions := kafkautil.TLSOptions{}
	kafkautil.AddTLSFlags(flag.CommandLine, &tlsOptions)
	flag.Parse()

	tlsConfig, err := kafkautil.BuildTLSConfig(tlsOptions)
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store := &productStore{products: make(map[string]events.Product)}
	recommendationWriter := kafkautil.NewWriterWithTLS(kafkautil.Brokers(*brokersCSV), *recommendationsTopic, tlsConfig)
	defer func() { _ = recommendationWriter.Close() }()

	var wg sync.WaitGroup
	startConsumer(ctx, &wg, *brokersCSV, *allowedTopic, *groupID+"-products", tlsConfig, func(ctx context.Context, msg kafka.Message) error {
		return handleAllowedProduct(ctx, msg, *dataDir, store, recommendationWriter)
	})
	startConsumer(ctx, &wg, *brokersCSV, *searchTopic, *groupID+"-search", tlsConfig, func(ctx context.Context, msg kafka.Message) error {
		return appendJSONL(*dataDir, "search_requests", msg.Value)
	})
	startConsumer(ctx, &wg, *brokersCSV, *recommendationRequestsTopic, *groupID+"-recommendation-requests", tlsConfig, func(ctx context.Context, msg kafka.Message) error {
		return handleRecommendationRequest(ctx, msg, *dataDir, store, recommendationWriter)
	})

	log.Printf("analytics hdfs-ingestor started: brokers=%s data_dir=%s recommendations_topic=%s", *brokersCSV, *dataDir, *recommendationsTopic)
	<-ctx.Done()
	wg.Wait()
}

func startConsumer(ctx context.Context, wg *sync.WaitGroup, brokersCSV, topic, groupID string, tlsConfig *tls.Config, handler func(context.Context, kafka.Message) error) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		config := kafka.ReaderConfig{
			Brokers:        kafkautil.Brokers(brokersCSV),
			Topic:          topic,
			GroupID:        groupID,
			MinBytes:       1,
			MaxBytes:       10e6,
			CommitInterval: time.Second,
			StartOffset:    kafka.FirstOffset,
		}
		if tlsConfig != nil {
			config.Dialer = &kafka.Dialer{TLS: tlsConfig}
		}
		reader := kafka.NewReader(config)
		defer func() { _ = reader.Close() }()

		for {
			msg, err := reader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Printf("read %s: %v", topic, err)
				time.Sleep(2 * time.Second)
				continue
			}
			if err := handler(ctx, msg); err != nil {
				log.Printf("handle %s offset=%d: %v", topic, msg.Offset, err)
			}
		}
	}()
}

func handleAllowedProduct(ctx context.Context, msg kafka.Message, dataDir string, store *productStore, writer *kafka.Writer) error {
	if err := appendJSONL(dataDir, "products_allowed", msg.Value); err != nil {
		return err
	}

	var envelope events.EventEnvelope[events.Product]
	if err := json.Unmarshal(msg.Value, &envelope); err != nil {
		return fmt.Errorf("decode product event: %w", err)
	}
	product := envelope.Payload
	if product.ProductID == "" {
		return nil
	}

	store.mu.Lock()
	store.products[product.ProductID] = product
	store.mu.Unlock()

	return emitRecommendation(ctx, dataDir, store, writer, "", product.Category)
}

func handleRecommendationRequest(ctx context.Context, msg kafka.Message, dataDir string, store *productStore, writer *kafka.Writer) error {
	if err := appendJSONL(dataDir, "recommendation_requests", msg.Value); err != nil {
		return err
	}

	var envelope events.EventEnvelope[events.RecommendationRequest]
	if err := json.Unmarshal(msg.Value, &envelope); err != nil {
		return fmt.Errorf("decode recommendation request: %w", err)
	}
	return emitRecommendation(ctx, dataDir, store, writer, envelope.Payload.UserID, envelope.Payload.Category)
}

func emitRecommendation(ctx context.Context, dataDir string, store *productStore, writer *kafka.Writer, userID, category string) error {
	category = strings.TrimSpace(category)
	if category == "" {
		return nil
	}

	store.mu.RLock()
	products := make([]events.Product, 0, len(store.products))
	for _, product := range store.products {
		if product.Category == category {
			products = append(products, product)
		}
	}
	store.mu.RUnlock()

	sort.Slice(products, func(i, j int) bool {
		if products[i].Stock.Available == products[j].Stock.Available {
			return products[i].UpdatedAt.After(products[j].UpdatedAt)
		}
		return products[i].Stock.Available > products[j].Stock.Available
	})

	limit := min(5, len(products))
	recommended := make([]events.RecommendedProduct, 0, limit)
	for i := 0; i < limit; i++ {
		score := float64(products[i].Stock.Available-products[i].Stock.Reserved) / 100.0
		if score < 0 {
			score = 0
		}
		recommended = append(recommended, events.RecommendedProduct{
			ProductID: products[i].ProductID,
			Name:      products[i].Name,
			Score:     score,
		})
	}

	recommendation := events.EventEnvelope[events.Recommendation]{
		EventID:   uuid.NewString(),
		EventType: "recommendations_calculated",
		EventTime: time.Now().UTC(),
		Source:    "hdfs-ingestor",
		Payload: events.Recommendation{
			RecommendationID: uuid.NewString(),
			UserID:           userID,
			Category:         category,
			Products:         recommended,
			CalculatedAt:     time.Now().UTC(),
		},
	}

	value, err := json.Marshal(recommendation)
	if err != nil {
		return err
	}
	if err := appendJSONL(dataDir, "recommendations", value); err != nil {
		return err
	}

	key := category
	if userID != "" {
		key = userID + ":" + category
	}
	return writer.WriteMessages(ctx, kafka.Message{Key: []byte(key), Value: value, Time: recommendation.EventTime})
}

func appendJSONL(baseDir, dataset string, value []byte) error {
	date := time.Now().UTC().Format("2006-01-02")
	dir := filepath.Join(baseDir, dataset)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	filePath := filepath.Join(dir, date+".jsonl")
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	if _, err := file.Write(append(value, '\n')); err != nil {
		return err
	}
	return nil
}
