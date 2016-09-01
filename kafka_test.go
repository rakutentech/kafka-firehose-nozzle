package main

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/cloudfoundry/sonde-go/events"
	"golang.org/x/net/context"
)

func TestKafkaProducer(t *testing.T) {

	cases := []struct {
		config *Config
		topic  string
		event  *events.Envelope
	}{

		// use default topic
		{
			config: &Config{},
			topic:  DefaultLogMessageTopic,
			event:  logMessage("", testAppId, time.Now().UnixNano()),
		},

		{
			config: &Config{},
			topic:  DefaultValueMetricTopic,
			event:  valueMetric(time.Now().UnixNano()),
		},

		// use fixed topic name
		{
			config: &Config{
				Kafka: Kafka{
					Topic: Topic{
						LogMessage: "log",
					},
				},
			},
			topic: "log",
			event: logMessage("", testAppId, time.Now().UnixNano()),
		},

		{
			config: &Config{
				Kafka: Kafka{
					Topic: Topic{
						ValueMetric: "metric",
					},
				},
			},

			topic: "metric",
			event: valueMetric(time.Now().UnixNano()),
		},

		// use log-message topic format
		{
			config: &Config{
				Kafka: Kafka{
					Topic: Topic{
						LogMessageFmt: "log-%s",
					},
				},
			},
			topic: fmt.Sprintf("log-%s", testAppId),
			event: logMessage("", testAppId, time.Now().UnixNano()),
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

		tc.config.Kafka.Brokers = []string{seed.Addr()}

		// Create new kafka producer
		stats := NewStats()
		producer, err := NewKafkaProducer(nil, stats, tc.config)
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

func TestKafkaProducer_RoundRobin(t *testing.T) {

	// topic which is used in this test
	topic := DefaultLogMessageTopic

	// partition to use
	partition1 := int32(0)
	partition2 := int32(1)
	partitions := []int32{partition1, partition2}

	// Create fake brokers (1 leader and 2 seeds)
	leader1 := sarama.NewMockBroker(t, int32(0))
	leader2 := sarama.NewMockBroker(t, int32(1))
	seed := sarama.NewMockBroker(t, int32(2))

	// Create metadata response
	meta := new(sarama.MetadataResponse)
	meta.AddBroker(leader1.Addr(), leader1.BrokerID())
	meta.AddBroker(leader2.Addr(), leader2.BrokerID())
	meta.AddTopicPartition(topic, partition1, leader1.BrokerID(), nil, nil, sarama.ErrNoError)
	meta.AddTopicPartition(topic, partition2, leader2.BrokerID(), nil, nil, sarama.ErrNoError)
	seed.Returns(meta)

	// Set leader response
	var response1, response2 sarama.ProduceResponse
	response1.AddTopicPartition(topic, partition1, sarama.ErrNoError)
	response2.AddTopicPartition(topic, partition2, sarama.ErrNoError)
	leader1.Returns(&response1)
	leader2.Returns(&response2)

	// Create new test kafka producer
	stats := NewStats()
	config := &Config{}
	config.Kafka.Brokers = []string{seed.Addr()}
	producer, err := NewKafkaProducer(nil, stats, config)
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
		eventCh <- logMessage("", "", time.Now().UnixNano())
		eventCh <- logMessage("", "", time.Now().UnixNano())
	}()

	outputs := make([]int32, 0, 2)
	for i := 0; i < 2; i++ {
		select {
		case err := <-producer.Errors():
			if err != nil {
				t.Fatalf("expect no err to be occurred: %s", err)
			}
		case msg := <-producer.Successes():
			outputs = append(outputs, msg.Partition)
		}
	}

	sort.Sort(Int32Slice(outputs))
	if !reflect.DeepEqual(outputs, partitions) {
		t.Fatalf("expect %v to be eq %v", outputs, partitions)
	}
}

func TestKafkaProducer_repartition(t *testing.T) {

	// topic which is used in this test
	topic := DefaultLogMessageTopic
	partitionBroken := int32(0)
	partitionOK := int32(1)

	// Create fake brokers (1 leader and 2 seeds)
	leader := sarama.NewMockBroker(t, int32(0))
	seed := sarama.NewMockBroker(t, int32(1))

	// Create metadata response
	meta := new(sarama.MetadataResponse)
	meta.AddBroker(leader.Addr(), leader.BrokerID())
	meta.AddTopicPartition(topic, partitionBroken, leader.BrokerID(), nil, nil, sarama.ErrNoError)
	meta.AddTopicPartition(topic, partitionOK, leader.BrokerID(), nil, nil, sarama.ErrNoError)
	seed.Returns(meta)
	seed.Returns(meta)

	// Set leader response
	var resErr sarama.ProduceResponse
	resErr.AddTopicPartition(topic, partitionBroken, sarama.ErrNotLeaderForPartition)
	leader.Returns(&resErr)
	leader.Returns(&resErr)
	var resOK sarama.ProduceResponse
	resOK.AddTopicPartition(topic, partitionOK, sarama.ErrNoError)
	leader.Returns(&resOK)

	// Create new test kafka producer
	stats := NewStats()
	producer, err := NewKafkaProducer(nil, stats, &Config{
		Kafka: Kafka{
			Brokers: []string{seed.Addr()},

			RetryMax:     1,
			RetryBackoff: 10,
		},
	})
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
		eventCh <- logMessage("", "", time.Now().UnixNano())
	}()

	select {
	case err := <-producer.Errors():
		// Publish should not be success
		t.Fatalf("expected no error, got %s", err.Err)
	case <-producer.Successes():
	}
}

func TestKafkaProducer_error(t *testing.T) {

	// topic which is used in this test
	topic := DefaultLogMessageTopic
	partitionBroken0 := int32(0)
	partitionBroken1 := int32(1)

	// Create fake brokers (1 leader and 2 seeds)
	leader := sarama.NewMockBroker(t, int32(0))
	seed := sarama.NewMockBroker(t, int32(1))

	// Create metadata response
	meta := new(sarama.MetadataResponse)
	meta.AddBroker(leader.Addr(), leader.BrokerID())
	meta.AddTopicPartition(topic, partitionBroken0, leader.BrokerID(), nil, nil, sarama.ErrNoError)
	meta.AddTopicPartition(topic, partitionBroken1, leader.BrokerID(), nil, nil, sarama.ErrNoError)
	seed.Returns(meta)
	seed.Returns(meta)
	seed.Returns(meta)
	seed.Returns(meta)
	seed.Returns(meta)
	seed.Returns(meta)
	seed.Returns(meta)

	// Set leader response
	var resErr0 sarama.ProduceResponse
	resErr0.AddTopicPartition(topic, partitionBroken0, sarama.ErrNotLeaderForPartition)
	var resErr1 sarama.ProduceResponse
	resErr1.AddTopicPartition(topic, partitionBroken1, sarama.ErrNotLeaderForPartition)
	leader.Returns(&resErr0)
	leader.Returns(&resErr0)
	leader.Returns(&resErr1)
	leader.Returns(&resErr1)
	leader.Returns(&resErr0)
	leader.Returns(&resErr1)
	leader.Returns(&resErr0)
	leader.Returns(&resErr1)

	// Create new test kafka producer
	stats := NewStats()
	producer, err := NewKafkaProducer(nil, stats, &Config{
		Kafka: Kafka{
			Brokers: []string{seed.Addr()},

			RetryMax:     1,
			RetryBackoff: 10,
		},
	})
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
		eventCh <- logMessage("", "", time.Now().UnixNano())
	}()

	select {
	case err := <-producer.Errors():
		if err.Err != sarama.ErrNotLeaderForPartition {
			t.Fatalf("expected ErrNotLeaderForPartition, got %s", err.Err)
		}
	case <-producer.Successes():
		t.Fatalf("expected produce to fail")
	}
}
