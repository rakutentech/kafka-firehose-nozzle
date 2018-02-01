package main

import (
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config is kafka-firehose-nozzle configuration.
type Config struct {
	SubscriptionID        string `toml:"subscription_id"`
	InsecureSSLSkipVerify bool   `toml:"insecure_ssl_skip_verify"`
	CF                    CF     `toml:"cf"`
	Kafka                 Kafka  `toml:"kafka"`
}

// CF holds CloudFoundry related configuration.
type CF struct {
	// dopplerAddr is doppler firehose address.
	// It must start with `ws://` or `wss://` schema because this is websocket.
	DopplerAddr string `toml:"doppler_address"`

	// UAAAddr is UAA server address.
	UAAAddr string `toml:"uaa_address"`

	// Username is the username which can has scope of `doppler.firehose`.
	Username string `toml:"username"`
	Password string `toml:"password"`
	Token    string `toml:"token"`

	// Firehose configuration
	IdleTimeout int `toml:"idle_timeout"` // seconds

	// How many times consumer will retry to connect to doppler
	RetryCount int `toml:"retry_count"`
}

// Kafka holds Kafka related configuration
type Kafka struct {
	Brokers []string `toml:"brokers"`
	Topic   Topic    `toml:"topic"`

	RetryMax       int `toml:"retry_max"`
	RetryBackoff   int `toml:"retry_backoff_ms"`
	RepartitionMax int `toml:"repartition_max"`

	Compression string `toml:"compression"` // ("gzip", "snappy" or "none", default: "none")

	// EnableTLS, if set, will connect to Kafka with TLS instead of plaintext.
	EnableTLS bool `toml:"enable_tls"`

	// CACerts is a list of CAs certificates used to verify the host.
	// Usually there is only one, however multiple can be specified to allow
	// for rotation. These should be PEM encoded CERTIFICATEs.
	// If none are specified, then the system CA pool is used.
	// Ignored unless enable_tls is set.
	CACerts []string `toml:"ca_certificates"`

	// ClientKey is used with the client certificate to identify this client
	// to Kafka. This should be a PEM encoded RSA PRIVATE KEY.
	// Ignored unless enable_tls is set.
	ClientKey string `toml:"private_key"`

	// ClientCertificate is used with the client key to identify this client
	// to Kafka. This should be a PEM encoded CERTIFICATE.
	// Ignored unless enable_tls is set.
	ClientCert string `toml:"certificate"`
}

type Topic struct {
	LogMessage         string `toml:"log_message"`
	LogMessageFmt      string `toml:"log_message_fmt"`
	ValueMetric        string `toml:"value_metric"`
	ContainerMetric    string `toml:"container_metric"`
	ContainerMetricFmt string `toml:"container_metric_fmt"`
	HttpStartStop      string `toml:"http_start_stop"`
	HttpStartStopFmt   string `toml:"http_start_stop_fmt"`
	CounterEvent       string `toml:"counter_event"`
	Error              string `toml:"error"`
}

// LoadConfig reads configuration file
func LoadConfig(path string) (*Config, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	config := new(Config)
	if _, err := toml.DecodeFile(path, &config); err != nil {
		return nil, err
	}

	return config, nil
}
