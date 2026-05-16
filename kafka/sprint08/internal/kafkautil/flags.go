package kafkautil

import "flag"

func AddTLSFlags(fs *flag.FlagSet, options *TLSOptions) {
	fs.BoolVar(&options.Enabled, "tls", false, "enable Kafka TLS")
	fs.StringVar(&options.CACertPath, "tls-ca-cert", "", "path to CA certificate PEM file")
	fs.StringVar(&options.ClientCertPath, "tls-client-cert", "", "path to client certificate PEM file")
	fs.StringVar(&options.ClientKeyPath, "tls-client-key", "", "path to client private key PEM file")
	fs.StringVar(&options.ServerName, "tls-server-name", "", "Kafka TLS server name override")
	fs.BoolVar(&options.InsecureSkipVerify, "tls-insecure-skip-verify", false, "skip Kafka TLS certificate verification")
}
