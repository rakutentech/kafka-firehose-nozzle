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
}

// Kafka holds Kafka related configuration
type Kafka struct {
	Brokers []string `toml:"brokers"`
	Topic   Topic    `toml:"topic"`

	RetryMax       int `toml:"retry_max"`
	RetryBackoff   int `toml:"retry_backoff_ms"`
	RepartitionMax int `toml:"repartition_max"`
}

type Topic struct {
	LogMessage    string `toml:"log_message"`
	LogMessageFmt string `toml:"log_message_fmt"`
	ValueMetric   string `toml:"value_metric"`
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
