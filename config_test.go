package main

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	cases := []struct {
		in      string
		success bool
		errStr  string
		config  *Config
	}{
		{
			in:      "basic.toml",
			success: true,
			config: &Config{
				SubscriptionID:        "kafka-firehose-nozzle",
				InsecureSSLSkipVerify: true,
				CF: CF{
					DopplerAddr: "wss://doppler.cloudfoundry.net",
					UAAAddr:     "https://uaa.cloudfoundry.net",
					Username:    "tcnksm",
					Password:    "xyz",
					IdleTimeout: 10, // seconds
				},
				Kafka: Kafka{
					Brokers: []string{"192.168.1.1:9092", "192.168.1.2:9092", "192.168.1.3:9092"},

					RetryMax:     10,
					RetryBackoff: 500,

					Topic: Topic{
						LogMessage:    "log",
						LogMessageFmt: "log-%s",
						ValueMetric:   "metric",
					},
				},
			},
		},

		{
			in:      "not-exist.toml",
			success: false,
			errStr:  "no such file",
			config:  &Config{},
		},
	}

	for i, tc := range cases {
		path := filepath.Join("./fixtures/", tc.in)
		config, err := LoadConfig(path)
		if tc.success {
			if err != nil {
				t.Fatalf("#%d expect %q to be nil", i, err)
			}

			if !reflect.DeepEqual(config, tc.config) {
				t.Fatalf("#%d expect to %#v to be eq %#v", i, config, tc.config)
			}
			continue
		}

		if err == nil {
			t.Fatalf("#%d expect to be failed", i)
		}

		if !strings.Contains(err.Error(), tc.errStr) {
			t.Fatalf("#%d expect %s to contain %s", i, err.Error(), tc.errStr)
		}

	}

}
