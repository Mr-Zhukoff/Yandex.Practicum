package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"github.com/segmentio/kafka-go"

	"marketplace-analytics/internal/events"
	"marketplace-analytics/internal/kafkautil"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	brokersCSV := "localhost:9092"
	topic := "forbidden.products.state"
	command := os.Args[1]

	fs := flag.NewFlagSet(command, flag.ExitOnError)
	fs.StringVar(&brokersCSV, "brokers", brokersCSV, "comma-separated Kafka brokers")
	fs.StringVar(&topic, "topic", topic, "Kafka compacted topic for forbidden products")
	partitions := fs.Int("partitions", 3, "number of partitions to scan for list command")
	productID := fs.String("product-id", "", "product ID")
	reason := fs.String("reason", "", "reason for forbidding the product")
	if err := fs.Parse(os.Args[2:]); err != nil {
		log.Fatal(err)
	}

	if command != "add" && command != "remove" && command != "list" {
		usage()
		os.Exit(2)
	}
	if command == "list" {
		listForbiddenProducts(kafkautil.Brokers(brokersCSV), topic, *partitions)
		return
	}
	if *productID == "" {
		log.Fatal("--product-id is required")
	}

	active := command == "add"
	record := events.ForbiddenProduct{
		ProductID: *productID,
		Reason:    *reason,
		CreatedAt: time.Now().UTC(),
		Active:    active,
	}
	value, err := json.Marshal(record)
	if err != nil {
		log.Fatal(err)
	}

	writer := kafkautil.NewWriter(kafkautil.Brokers(brokersCSV), topic)
	defer func() { _ = writer.Close() }()

	msg := kafka.Message{Key: []byte(*productID), Value: value, Time: time.Now().UTC()}
	if err := writer.WriteMessages(context.Background(), msg); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("forbidden product state updated: product_id=%s active=%t\n", *productID, active)
}

func listForbiddenProducts(brokers []string, topic string, partitions int) {
	states := map[string]events.ForbiddenProduct{}
	for partition := 0; partition < partitions; partition++ {
		reader := kafka.NewReader(kafka.ReaderConfig{
			Brokers:     brokers,
			Topic:       topic,
			Partition:   partition,
			MinBytes:    1,
			MaxBytes:    10e6,
			StartOffset: kafka.FirstOffset,
			MaxWait:     500 * time.Millisecond,
		})
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		for {
			msg, err := reader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					break
				}
				cancel()
				_ = reader.Close()
				log.Fatal(err)
			}
			var record events.ForbiddenProduct
			if err := json.Unmarshal(msg.Value, &record); err != nil {
				log.Printf("skip invalid forbidden record key=%q: %v", string(msg.Key), err)
				continue
			}
			if record.Active {
				states[record.ProductID] = record
				continue
			}
			delete(states, record.ProductID)
		}
		cancel()
		_ = reader.Close()
	}

	ids := make([]string, 0, len(states))
	for id := range states {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	if len(ids) == 0 {
		fmt.Println("No active forbidden products found")
		return
	}

	fmt.Println("Active forbidden products:")
	for _, id := range ids {
		record := states[id]
		fmt.Printf("- %s | active=%t | reason=%s\n", record.ProductID, record.Active, record.Reason)
	}
}

func usage() {
	fmt.Println("usage:")
	fmt.Println("  forbidden-cli add --product-id ID --reason REASON")
	fmt.Println("  forbidden-cli remove --product-id ID")
	fmt.Println("  forbidden-cli list")
}
