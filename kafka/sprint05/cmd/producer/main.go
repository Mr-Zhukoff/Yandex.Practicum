package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"

	"github.com/IBM/sarama"
)

func main() {
	cfg, err := newConfig("producer")
	if err != nil {
		panic(err)
	}

	producer, err := sarama.NewSyncProducer(getBrokers(), cfg)
	if err != nil {
		panic(err)
	}
	defer producer.Close()

	for _, topic := range []string{"topic-1", "topic-2"} {
		msg := &sarama.ProducerMessage{Topic: topic, Value: sarama.StringEncoder("secure message to " + topic)}
		partition, offset, err := producer.SendMessage(msg)
		if err != nil {
			panic(err)
		}
		fmt.Printf("sent to %s partition=%d offset=%d\n", topic, partition, offset)
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
	cfg.Producer.RequiredAcks = sarama.WaitForAll
	cfg.Producer.Retry.Max = 5
	cfg.Producer.Return.Successes = true

	return cfg, nil
}
