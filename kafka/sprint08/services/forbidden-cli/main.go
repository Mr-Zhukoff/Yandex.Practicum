package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
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
	productID := fs.String("product-id", "", "product ID")
	reason := fs.String("reason", "", "reason for forbidding the product")
	if err := fs.Parse(os.Args[2:]); err != nil {
		log.Fatal(err)
	}

	if command != "add" && command != "remove" {
		usage()
		os.Exit(2)
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

func usage() {
	fmt.Println("usage:")
	fmt.Println("  forbidden-cli add --product-id ID --reason REASON")
	fmt.Println("  forbidden-cli remove --product-id ID")
}
