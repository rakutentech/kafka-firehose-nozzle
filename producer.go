package main

import (
	"fmt"

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

// KafkaProducer implements the Nozzle.Producer interface.
type KafkaProducer struct {
	sarama.AsyncProducer

	// doneCh is used to stop producer process
	DoneCh chan struct{}
}

func (kp *KafkaProducer) produce(eventCh <-chan *events.Envelope) {
	for {
		select {
		case event := <-eventCh:
			kp.input(event)
		case <-kp.DoneCh:
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
