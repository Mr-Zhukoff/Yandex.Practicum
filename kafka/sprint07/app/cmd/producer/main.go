package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"sprint07-kafka/internal/kafkaconfig"

	"github.com/segmentio/kafka-go"
)

type UserEvent struct {
	EventID   string `json:"event_id"`
	EventType string `json:"event_type"`
	UserID    string `json:"user_id"`
	EventTS   int64  `json:"event_ts"`
	Source    string `json:"source"`
}

func main() {
	cfg := kafkaconfig.Load()

	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Topic:        cfg.Topic,
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireAll,
		Transport:    cfg.Transport(),
	}
	defer writer.Close()

	ctx := context.Background()

	events := []UserEvent{
		{EventID: "evt-1", EventType: "login", UserID: "u-100", EventTS: time.Now().UnixMilli(), Source: "web"},
		{EventID: "evt-2", EventType: "view_product", UserID: "u-101", EventTS: time.Now().UnixMilli(), Source: "mobile"},
		{EventID: "evt-3", EventType: "checkout", UserID: "u-102", EventTS: time.Now().UnixMilli(), Source: "web"},
	}

	for _, e := range events {
		payload, err := json.Marshal(e)
		if err != nil {
			log.Fatalf("marshal error: %v", err)
		}

		msg := kafka.Message{
			Key:   []byte(e.UserID),
			Value: payload,
			Time:  time.Now(),
		}

		if err = writer.WriteMessages(ctx, msg); err != nil {
			log.Fatalf("write message error: %v", err)
		}

		fmt.Printf("producer: sent event_id=%s key=%s value=%s\n", e.EventID, e.UserID, string(payload))
	}
}
