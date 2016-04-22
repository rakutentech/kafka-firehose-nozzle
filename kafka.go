package main

import (
	"fmt"

	"golang.org/x/net/context"

	"github.com/Shopify/sarama"
	"github.com/cloudfoundry/sonde-go/events"
)

const (
	// TopicAppLogTmpl is Kafka topic name template for LogMessage
	TopicAppLogTmpl = "app-log-%s"

	// TopicAppMetrics is Kafka topic name for ContainerMetric
	TopicContainerMetrics = "app-metric-%s"

	// TopicCFMetrics is Kafka topic name for ValueMetric
	TopicCFMetric = "cf-metrics"
)

func NewKafkaProducer(config *Config) (NozzleProducer, error) {
	// Setup kafka async producer (We must use sync producer)
	// TODO (tcnksm): Enable to configure more properties.
	producerConfig := sarama.NewConfig()
	producerConfig.Producer.Retry.Max = 5
	producerConfig.Producer.RequiredAcks = sarama.WaitForAll

	brokers := config.Kafka.Brokers
	if len(brokers) < 1 {
		return nil, fmt.Errorf("brokers are not provided")
	}

	asyncProducer, err := sarama.NewAsyncProducer(brokers, producerConfig)
	if err != nil {
		return nil, err
	}

	//
	return &KafkaProducer{
		AsyncProducer: asyncProducer,
	}, nil
}

// KafkaProducer implements NozzleProducer interfaces
type KafkaProducer struct {
	sarama.AsyncProducer
}

// Produce produces event to kafka
func (kp *KafkaProducer) Produce(ctx context.Context, eventCh <-chan *events.Envelope) {
	for {
		select {
		case event := <-eventCh:
			kp.input(event)
		case <-ctx.Done():
			// Stop process immediately
			return
		}
	}
}

func (kp *KafkaProducer) input(event *events.Envelope) {
	switch eventType := event.GetEventType(); eventType {
	case events.Envelope_HttpStart:
		// Do nothing
	case events.Envelope_HttpStartStop:
		// Do nothing
	case events.Envelope_HttpStop:
		// Do nothing
	case events.Envelope_LogMessage:
		appID := event.GetLogMessage().GetAppId()
		kp.Input() <- &sarama.ProducerMessage{
			Topic: fmt.Sprintf(TopicAppLogTmpl, appID),
			Value: &JsonEncoder{event: event},
		}
	case events.Envelope_ValueMetric:
		kp.Input() <- &sarama.ProducerMessage{
			Topic: TopicCFMetric,
			Value: &JsonEncoder{event: event},
		}
	case events.Envelope_CounterEvent:
		// Do nothing
	case events.Envelope_Error:
		// Do nothing
	case events.Envelope_ContainerMetric:
		// Do nothing
	}
}
