package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
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

type dataLakeWriter struct {
	localDir string
	hdfs     *webHDFSClient
}

type webHDFSClient struct {
	baseURL string
	baseDir string
	user    string
	client  *http.Client
}

func main() {
	brokersCSV := flag.String("brokers", "localhost:9192", "comma-separated secondary Kafka brokers")
	allowedTopic := flag.String("allowed-topic", "shop.products.allowed", "allowed products topic")
	searchTopic := flag.String("search-topic", "client.search.requests", "search requests topic")
	recommendationRequestsTopic := flag.String("recommendation-requests-topic", "client.recommendation.requests", "recommendation requests topic")
	recommendationsTopic := flag.String("recommendations-topic", "analytics.recommendations", "recommendations output topic")
	streamingRecommendations := flag.Bool("streaming-recommendations", false, "also calculate simple in-process recommendations while ingesting")
	groupID := flag.String("group", "hdfs-ingestor", "Kafka consumer group prefix")
	dataDir := flag.String("data-dir", "/data/hdfs", "HDFS-compatible local data lake directory")
	hdfsWebURL := flag.String("hdfs-web-url", "", "WebHDFS namenode URL, for example http://namenode:9870")
	hdfsBaseDir := flag.String("hdfs-base-dir", "/marketplace-analytics", "base HDFS directory for analytics datasets")
	hdfsUser := flag.String("hdfs-user", "root", "WebHDFS user.name")
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
	dataLake := newDataLakeWriter(*dataDir, *hdfsWebURL, *hdfsBaseDir, *hdfsUser)
	var recommendationWriter *kafka.Writer
	if *streamingRecommendations {
		recommendationWriter = kafkautil.NewWriterWithTLS(kafkautil.Brokers(*brokersCSV), *recommendationsTopic, tlsConfig)
		defer func() { _ = recommendationWriter.Close() }()
	}

	var wg sync.WaitGroup
	startConsumer(ctx, &wg, *brokersCSV, *allowedTopic, *groupID+"-products", tlsConfig, func(ctx context.Context, msg kafka.Message) error {
		return handleAllowedProduct(ctx, msg, dataLake, store, recommendationWriter)
	})
	startConsumer(ctx, &wg, *brokersCSV, *searchTopic, *groupID+"-search", tlsConfig, func(ctx context.Context, msg kafka.Message) error {
		return dataLake.write(ctx, "search_requests", msg.Value)
	})
	startConsumer(ctx, &wg, *brokersCSV, *recommendationRequestsTopic, *groupID+"-recommendation-requests", tlsConfig, func(ctx context.Context, msg kafka.Message) error {
		return handleRecommendationRequest(ctx, msg, dataLake, store, recommendationWriter)
	})

	log.Printf("analytics hdfs-ingestor started: brokers=%s data_dir=%s hdfs_web_url=%s hdfs_base_dir=%s streaming_recommendations=%t recommendations_topic=%s", *brokersCSV, *dataDir, *hdfsWebURL, *hdfsBaseDir, *streamingRecommendations, *recommendationsTopic)
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

func handleAllowedProduct(ctx context.Context, msg kafka.Message, dataLake *dataLakeWriter, store *productStore, writer *kafka.Writer) error {
	if err := dataLake.write(ctx, "products_allowed", msg.Value); err != nil {
		return err
	}
	if writer == nil {
		return nil
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

	return emitRecommendation(ctx, dataLake, store, writer, "", product.Category)
}

func handleRecommendationRequest(ctx context.Context, msg kafka.Message, dataLake *dataLakeWriter, store *productStore, writer *kafka.Writer) error {
	if err := dataLake.write(ctx, "recommendation_requests", msg.Value); err != nil {
		return err
	}
	if writer == nil {
		return nil
	}

	var envelope events.EventEnvelope[events.RecommendationRequest]
	if err := json.Unmarshal(msg.Value, &envelope); err != nil {
		return fmt.Errorf("decode recommendation request: %w", err)
	}
	return emitRecommendation(ctx, dataLake, store, writer, envelope.Payload.UserID, envelope.Payload.Category)
}

func emitRecommendation(ctx context.Context, dataLake *dataLakeWriter, store *productStore, writer *kafka.Writer, userID, category string) error {
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
	if err := dataLake.write(ctx, "recommendations", value); err != nil {
		return err
	}

	key := category
	if userID != "" {
		key = userID + ":" + category
	}
	return writer.WriteMessages(ctx, kafka.Message{Key: []byte(key), Value: value, Time: recommendation.EventTime})
}

func newDataLakeWriter(localDir, hdfsWebURL, hdfsBaseDir, hdfsUser string) *dataLakeWriter {
	var hdfs *webHDFSClient
	if strings.TrimSpace(hdfsWebURL) != "" {
		hdfs = &webHDFSClient{
			baseURL: strings.TrimRight(hdfsWebURL, "/"),
			baseDir: "/" + strings.Trim(strings.TrimSpace(hdfsBaseDir), "/"),
			user:    hdfsUser,
			client:  &http.Client{Timeout: 30 * time.Second},
		}
	}
	return &dataLakeWriter{localDir: localDir, hdfs: hdfs}
}

func (w *dataLakeWriter) write(ctx context.Context, dataset string, value []byte) error {
	if err := appendJSONL(w.localDir, dataset, value); err != nil {
		return err
	}
	if w.hdfs == nil {
		return nil
	}
	if err := w.hdfs.createJSON(ctx, dataset, value); err != nil {
		return fmt.Errorf("write %s to HDFS: %w", dataset, err)
	}
	return nil
}

func (c *webHDFSClient) createJSON(ctx context.Context, dataset string, value []byte) error {
	date := time.Now().UTC().Format("2006-01-02")
	fileName := fmt.Sprintf("%d-%s.json", time.Now().UnixNano(), uuid.NewString())
	hdfsPath := filepath.ToSlash(filepath.Join(c.baseDir, dataset, date, fileName))
	createURL := c.operationURL(hdfsPath, "CREATE", url.Values{"overwrite": {"true"}, "createparent": {"true"}})

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, createURL, nil)
	if err != nil {
		return err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	location := resp.Header.Get("Location")
	if resp.StatusCode != http.StatusTemporaryRedirect || location == "" {
		return fmt.Errorf("create redirect status=%s location=%q", resp.Status, location)
	}

	putReq, err := http.NewRequestWithContext(ctx, http.MethodPut, location, bytes.NewReader(value))
	if err != nil {
		return err
	}
	putReq.Header.Set("Content-Type", "application/json")
	putResp, err := c.client.Do(putReq)
	if err != nil {
		return err
	}
	defer func() { _ = putResp.Body.Close() }()
	if putResp.StatusCode < 200 || putResp.StatusCode >= 300 {
		return fmt.Errorf("write status=%s", putResp.Status)
	}
	return nil
}

func (c *webHDFSClient) operationURL(hdfsPath, operation string, extra url.Values) string {
	values := url.Values{"op": {operation}, "user.name": {c.user}}
	for key, vals := range extra {
		for _, val := range vals {
			values.Add(key, val)
		}
	}
	return fmt.Sprintf("%s/webhdfs/v1%s?%s", c.baseURL, hdfsPath, values.Encode())
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
