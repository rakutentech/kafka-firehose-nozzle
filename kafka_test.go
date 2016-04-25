package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/cloudfoundry/sonde-go/events"
	"golang.org/x/net/context"
)

func TestNewKafkaProducer(t *testing.T) {

	cases := []struct {
		topic string
		event *events.Envelope
	}{
		{
			topic: fmt.Sprintf("app-log-%s", testAppId),
			event: logMessage("", testAppId, time.Now().UnixNano()),
		},

		{
			topic: "cf-metrics",
			event: valueMetric(time.Now().UnixNano()),
		},
	}

	for _, tc := range cases {
		leader := sarama.NewMockBroker(t, int32(1))
		success := new(sarama.ProduceResponse)
		success.AddTopicPartition(tc.topic, int32(0), sarama.ErrNoError)
		leader.Returns(success)

		meta := new(sarama.MetadataResponse)
		meta.AddTopicPartition(tc.topic, int32(0), leader.BrokerID(), nil, nil, sarama.ErrNoError)
		meta.AddBroker(leader.Addr(), int32(1))

		seed := sarama.NewMockBroker(t, int32(0))
		seed.Returns(meta)

		config := &Config{
			Kafka: &Kafka{
				Brokers: []string{seed.Addr()},
			},
		}

		// Create new kafka producer
		producer, err := NewKafkaProducer(config)
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		// Create test eventCh where producer gets actual message
		eventCh := make(chan *events.Envelope)

		// Start producing
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			producer.Produce(ctx, eventCh)
		}()

		// Send event to producer
		go func() {
			// Create test event and send it to channel
			eventCh <- tc.event
		}()

		select {
		case err := <-producer.Errors():
			if err != nil {
				t.Fatalf("expect err to be nil: %s", err)
			}
		case msg := <-producer.Successes():
			if msg.Topic != tc.topic {
				t.Fatalf("expect %q to be eq %q", msg.Topic, tc.topic)
			}
		}
	}
}
