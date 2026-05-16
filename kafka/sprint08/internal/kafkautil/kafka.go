package kafkautil

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

type TLSOptions struct {
	Enabled            bool
	CACertPath         string
	ClientCertPath     string
	ClientKeyPath      string
	ServerName         string
	InsecureSkipVerify bool
}

func Brokers(csv string) []string {
	parts := strings.Split(csv, ",")
	brokers := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			brokers = append(brokers, part)
		}
	}
	return brokers
}

func NewWriter(brokers []string, topic string) *kafka.Writer {
	return NewWriterWithTLS(brokers, topic, nil)
}

func NewWriterWithTLS(brokers []string, topic string, tlsConfig *tls.Config) *kafka.Writer {
	writer := &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Topic:                  topic,
		Balancer:               &kafka.Hash{},
		AllowAutoTopicCreation: false,
		RequiredAcks:           kafka.RequireAll,
		BatchTimeout:           50 * time.Millisecond,
	}
	if tlsConfig != nil {
		writer.Transport = &kafka.Transport{TLS: tlsConfig}
	}
	return writer
}

func NewReader(brokers []string, topic, groupID string) *kafka.Reader {
	return NewReaderWithTLS(brokers, topic, groupID, nil)
}

func NewReaderWithTLS(brokers []string, topic, groupID string, tlsConfig *tls.Config) *kafka.Reader {
	config := kafka.ReaderConfig{
		Brokers:        brokers,
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
	return kafka.NewReader(config)
}

func BuildTLSConfig(options TLSOptions) (*tls.Config, error) {
	if !options.Enabled {
		return nil, nil
	}

	config := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		ServerName:         options.ServerName,
		InsecureSkipVerify: options.InsecureSkipVerify, //nolint:gosec // local project/dev flag
	}

	if options.CACertPath != "" {
		caCert, err := os.ReadFile(options.CACertPath)
		if err != nil {
			return nil, fmt.Errorf("read CA certificate: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("append CA certificate from %s", options.CACertPath)
		}
		config.RootCAs = pool
	}

	if options.ClientCertPath != "" || options.ClientKeyPath != "" {
		if options.ClientCertPath == "" || options.ClientKeyPath == "" {
			return nil, fmt.Errorf("both client certificate and client key are required")
		}
		cert, err := tls.LoadX509KeyPair(options.ClientCertPath, options.ClientKeyPath)
		if err != nil {
			return nil, fmt.Errorf("load client certificate/key: %w", err)
		}
		config.Certificates = []tls.Certificate{cert}
	}

	return config, nil
}

func CloseWriter(ctx context.Context, writer *kafka.Writer) error {
	done := make(chan error, 1)
	go func() { done <- writer.Close() }()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
