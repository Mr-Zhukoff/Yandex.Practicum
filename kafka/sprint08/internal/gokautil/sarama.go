package gokautil

import (
	"crypto/tls"

	"github.com/IBM/sarama"
	"github.com/lovoo/goka"

	"marketplace-analytics/internal/kafkautil"
)

func ProcessorOptions(tlsOptions kafkautil.TLSOptions) ([]goka.ProcessorOption, error) {
	if !tlsOptions.Enabled {
		return nil, nil
	}

	tlsConfig, err := kafkautil.BuildTLSConfig(tlsOptions)
	if err != nil {
		return nil, err
	}

	config := goka.DefaultConfig()
	config.Net.TLS.Enable = true
	config.Net.TLS.Config = tlsConfig.Clone()
	config.Net.TLS.Config.MinVersion = tls.VersionTLS12
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Return.Successes = true

	return []goka.ProcessorOption{
		goka.WithConsumerGroupBuilder(goka.ConsumerGroupBuilderWithConfig(config)),
		goka.WithProducerBuilder(goka.ProducerBuilderWithConfig(config)),
		goka.WithConsumerSaramaBuilder(goka.SaramaConsumerBuilderWithConfig(config)),
		goka.WithTopicManagerBuilder(goka.TopicManagerBuilderWithConfig(config, goka.NewTopicManagerConfig())),
	}, nil
}
