package kafkaconfig

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"
)

type Config struct {
	Brokers       []string
	Topic         string
	Username      string
	Password      string
	SASLMechanism string
	CACertPath    string
}

func Load() Config {
	username := strings.TrimSpace(os.Getenv("KAFKA_USERNAME"))
	password := os.Getenv("KAFKA_PASSWORD")

	return Config{
		Brokers:       parseBrokers(os.Getenv("KAFKA_BROKERS")),
		Topic:         topicOrDefault(os.Getenv("KAFKA_TOPIC"), "user-events"),
		Username:      username,
		Password:      password,
		SASLMechanism: strings.ToUpper(strings.TrimSpace(os.Getenv("KAFKA_SASL_MECHANISM"))),
		CACertPath:    strings.TrimSpace(os.Getenv("KAFKA_CA_CERT_PATH")),
	}
}

func topicOrDefault(raw, def string) string {
	v := strings.TrimSpace(raw)
	if v == "" {
		return def
	}
	return v
}

func (c Config) Transport() *kafka.Transport {
	tlsConfig, err := c.tlsConfig()
	if err != nil {
		panic(err)
	}

	if c.Username == "" || c.Password == "" {
		return &kafka.Transport{TLS: tlsConfig}
	}

	mechanism := c.SASLMechanism
	if mechanism == "" {
		mechanism = "SCRAM-SHA-512"
	}

	var saslMechanism saslMechanism
	var mErr error
	if mechanism == "PLAIN" {
		saslMechanism = plain.Mechanism{Username: c.Username, Password: c.Password}
	} else {
		saslMechanism, mErr = scram.Mechanism(scram.SHA512, c.Username, c.Password)
		if mErr != nil {
			panic(fmt.Errorf("failed to create SCRAM mechanism: %w", mErr))
		}
	}

	return &kafka.Transport{TLS: tlsConfig, SASL: saslMechanism}
}

type saslMechanism interface {
	Name() string
	Start(context.Context) (sasl.StateMachine, []byte, error)
}

func (c Config) tlsConfig() (*tls.Config, error) {
	if c.CACertPath == "" {
		return &tls.Config{MinVersion: tls.VersionTLS12}, nil
	}

	pemData, err := os.ReadFile(c.CACertPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read CA cert %q: %w", c.CACertPath, err)
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pemData) {
		return nil, fmt.Errorf("failed to parse CA cert PEM: %s", c.CACertPath)
	}

	return &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: pool}, nil
}

func parseBrokers(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{"localhost:9092", "localhost:9093", "localhost:9094"}
	}

	parts := strings.Split(raw, ",")
	brokers := make([]string, 0, len(parts))
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v != "" {
			brokers = append(brokers, v)
		}
	}

	if len(brokers) == 0 {
		return []string{"localhost:9092", "localhost:9093", "localhost:9094"}
	}

	return brokers
}
