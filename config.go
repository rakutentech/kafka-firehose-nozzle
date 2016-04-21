package main

import (
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config is kafka-firehose-nozzle configuration.
type Config struct {
	CF    *CF    `toml:"cf"`
	Kafka *Kafka `toml:"kafka"`
}

// CF holds CloudFoundry related configuration.
type CF struct {
	// Domain is CloudFoundry domain. It does't require schema.
	Domain string `toml:"domain"`

	// dopplerAddr is doppler firehose address.
	// It must start with `ws://` or `wss://` schema because this is websocket.
	DopplerAddr string `toml:"doppler_address"`

	// UAAAddr is UAA server address.
	UAAAddr string `toml:"uaa_address"`

	// Username is the username which can has scope of `doppler.firehose`.
	Username string `toml:"username"`
	Password string `toml:"password"`
	Token    string `toml:"token"`
}

// Kafka holds Kafka related configuration
type Kafka struct {
	Brokers []string `toml:"brokers"`
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
