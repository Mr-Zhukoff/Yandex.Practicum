package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/IBM/sarama"
)

func main() {
	cfg, err := newConfig("consumer")
	if err != nil {
		panic(err)
	}

	group, err := sarama.NewConsumerGroup(getBrokers(), "consumer-group-1", cfg)
	if err != nil {
		panic(err)
	}
	defer group.Close()

	handler := &consumerHandler{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sig
		cancel()
	}()

	fmt.Println("consume topic-1 (allowed)")
	go consumeLoop(ctx, group, handler, []string{"topic-1"})

	go logConsumerErrors(ctx, group)
	verifyTopic2Denied(cfg)

	<-ctx.Done()
}

func logConsumerErrors(ctx context.Context, group sarama.ConsumerGroup) {
	for {
		select {
		case err, ok := <-group.Errors():
			if !ok {
				return
			}
			fmt.Printf("consumer-group error: %v\n", err)
		case <-ctx.Done():
			return
		}
	}
}

func verifyTopic2Denied(cfg *sarama.Config) {
	fmt.Println("consume topic-2 (must fail by ACL)")

	testGroup, err := sarama.NewConsumerGroup(getBrokers(), "consumer-group-topic2-check", cfg)
	if err != nil {
		fmt.Printf("cannot create test consumer group for topic-2: %v\n", err)
		return
	}
	defer testGroup.Close()

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	consumeDone := make(chan error, 1)
	go func() {
		consumeDone <- testGroup.Consume(timeoutCtx, []string{"topic-2"}, &consumerHandler{})
	}()

	for {
		select {
		case err := <-consumeDone:
			if err != nil {
				fmt.Printf("expected ACL error on topic-2: %v\n", err)
			} else {
				fmt.Println("topic-2 consume cycle finished, checking errors channel...")
			}
			return
		case err := <-testGroup.Errors():
			fmt.Printf("expected ACL error on topic-2: %v\n", err)
			return
		case <-timeoutCtx.Done():
			fmt.Println("topic-2 ACL check timeout (no read access expected).")
			return
		}
	}
}

func getBrokers() []string {
	value := os.Getenv("KAFKA_BROKERS")
	if value == "" {
		value = "kafka-1:9092,kafka-2:9092,kafka-3:9092"
	}
	parts := strings.Split(value, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func consumeLoop(ctx context.Context, group sarama.ConsumerGroup, handler sarama.ConsumerGroupHandler, topics []string) {
	for {
		if err := group.Consume(ctx, topics, handler); err != nil {
			fmt.Printf("consume error: %v\n", err)
		}
		if ctx.Err() != nil {
			return
		}
	}
}

type consumerHandler struct{}

func (h *consumerHandler) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (h *consumerHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }
func (h *consumerHandler) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		fmt.Printf("topic=%s partition=%d offset=%d value=%s\n", msg.Topic, msg.Partition, msg.Offset, string(msg.Value))
		sess.MarkMessage(msg, "")
	}
	return nil
}

func newConfig(clientName string) (*sarama.Config, error) {
	certsDir := os.Getenv("CERTS_DIR")
	if certsDir == "" {
		certsDir = "certs"
	}

	caPem, err := os.ReadFile(certsDir + "/ca.cert.pem")
	if err != nil {
		return nil, err
	}

	cert, err := tls.LoadX509KeyPair(certsDir+"/clients/"+clientName+".cert.pem", certsDir+"/clients/"+clientName+".key.pem")
	if err != nil {
		return nil, err
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPem) {
		return nil, fmt.Errorf("cannot append CA cert")
	}

	tlsCfg := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
		RootCAs:      pool,
	}

	cfg := sarama.NewConfig()
	cfg.Version = sarama.V2_8_0_0
	cfg.Net.TLS.Enable = true
	cfg.Net.TLS.Config = tlsCfg
	cfg.Consumer.Return.Errors = true
	cfg.Consumer.Offsets.Initial = sarama.OffsetOldest

	return cfg, nil
}
