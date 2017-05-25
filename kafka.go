package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"sync"
	"time"

	"golang.org/x/net/context"

	"github.com/Shopify/sarama"
	"github.com/cloudfoundry/sonde-go/events"
)

const (
	DefaultKafkaRetryMax     = 5
	DefaultKafkaRetryBackoff = 100 * time.Millisecond
)

func NewKafkaProducer(logger *log.Logger, stats *Stats, config *Config) (NozzleProducer, error) {
	// Setup kafka async producer (We must use sync producer)
	// TODO (tcnksm): Enable to configure more properties.
	producerConfig := sarama.NewConfig()

	producerConfig.Producer.Partitioner = sarama.NewRoundRobinPartitioner
	producerConfig.Producer.Return.Successes = true
	producerConfig.Producer.RequiredAcks = sarama.WaitForAll

	producerConfig.Producer.Retry.Max = DefaultKafkaRetryMax
	if config.Kafka.RetryMax != 0 {
		producerConfig.Producer.Retry.Max = config.Kafka.RetryMax
	}

	producerConfig.Producer.Retry.Backoff = DefaultKafkaRetryBackoff
	if config.Kafka.RetryBackoff != 0 {
		backoff := time.Duration(config.Kafka.RetryBackoff) * time.Millisecond
		producerConfig.Producer.Retry.Backoff = backoff
	}

	brokers := config.Kafka.Brokers
	if len(brokers) < 1 {
		return nil, fmt.Errorf("brokers are not provided")
	}

	asyncProducer, err := sarama.NewAsyncProducer(brokers, producerConfig)
	if err != nil {
		return nil, err
	}

	return &KafkaProducer{
		AsyncProducer:           asyncProducer,
		Logger:                  logger,
		Stats:                   stats,
		logMessageTopic:         config.Kafka.Topic.LogMessage,
		logMessageTopicFmt:      config.Kafka.Topic.LogMessageFmt,
		valueMetricTopic:        config.Kafka.Topic.ValueMetric,
		containerMetricTopic:    config.Kafka.Topic.ContainerMetric,
		containerMetricTopicFmt: config.Kafka.Topic.ContainerMetricFmt,
		httpStartTopic:          config.Kafka.Topic.HttpStart,
		httpStartTopicFmt:       config.Kafka.Topic.HttpStartFmt,
		httpStopTopic:           config.Kafka.Topic.HttpStop,
		httpStopTopicFmt:        config.Kafka.Topic.HttpStopFmt,
		httpStartStopTopic:      config.Kafka.Topic.HttpStartStop,
		httpStartStopTopicFmt:   config.Kafka.Topic.HttpStartStopFmt,
		counterEventTopic:       config.Kafka.Topic.CounterEvent,
		errorTopic:              config.Kafka.Topic.Error,
	}, nil
}

// KafkaProducer implements NozzleProducer interfaces
type KafkaProducer struct {
	sarama.AsyncProducer

	logMessageTopic         string
	logMessageTopicFmt      string
	valueMetricTopic        string
	containerMetricTopic    string
	containerMetricTopicFmt string
	httpStartTopic          string
	httpStartTopicFmt       string
	httpStopTopic           string
	httpStopTopicFmt        string
	httpStartStopTopic      string
	httpStartStopTopicFmt   string
	counterEventTopic       string
	errorTopic              string

	Logger *log.Logger
	Stats  *Stats

	once sync.Once
}

// init sets default logger
func (kp *KafkaProducer) init() {
	if kp.Logger == nil {
		kp.Logger = defaultLogger
	}
}

func fmtTopic(t, tfmt, appID string) string {
	if tfmt != "" {
		return fmt.Sprintf(tfmt, appID)
	}
	return t
}

func (kp *KafkaProducer) LogMessageTopic(appID string) string {
	return fmtTopic(kp.logMessageTopic, kp.logMessageTopicFmt, appID)
}

func (kp *KafkaProducer) ValueMetricTopic() string {
	return kp.valueMetricTopic
}

func (kp *KafkaProducer) ContainerMetricTopic(appID string) string {
	return fmtTopic(kp.containerMetricTopic, kp.containerMetricTopicFmt, appID)
}

func (kp *KafkaProducer) HttpStartTopic(appID string) string {
	return fmtTopic(kp.httpStartTopic, kp.httpStartTopicFmt, appID)
}

func (kp *KafkaProducer) HttpStopTopic(appID string) string {
	return fmtTopic(kp.httpStopTopic, kp.httpStopTopicFmt, appID)
}

func (kp *KafkaProducer) HttpStartStopTopic(appID string) string {
	return fmtTopic(kp.httpStartStopTopic, kp.httpStartStopTopicFmt, appID)
}

func (kp *KafkaProducer) CounterEventTopic() string {
	return kp.counterEventTopic
}

func (kp *KafkaProducer) ErrorTopic() string {
	return kp.errorTopic
}

func uuid2str(uuid *events.UUID) string {
	var buf [16]byte
	binary.LittleEndian.PutUint64(buf[0:8], uuid.GetLow())
	binary.LittleEndian.PutUint64(buf[8:16], uuid.GetHigh())
	return fmt.Sprintf("%x-%x-%x-%x-%x", buf[0:4], buf[4:6], buf[6:8], buf[8:10], buf[10:16])
}

// Produce produces event to kafka
func (kp *KafkaProducer) Produce(ctx context.Context, eventCh <-chan *events.Envelope) {
	kp.once.Do(kp.init)

	kp.Logger.Printf("[INFO] Start loop to watch events")
	for {
		select {
		case event, ok := <-eventCh:
			if !ok {
				kp.Logger.Printf("[ERROR] Nozzle consumer eventCh is closed")
				return
			}

			kp.input(event)

		case <-ctx.Done():
			// Stop process immediately
			kp.Logger.Printf("[INFO] Stop kafka producer")
			return
		}
	}
}

func (kp *KafkaProducer) input(event *events.Envelope) {
	topic := ""

	kp.Stats.Inc(Consume)

	switch event.GetEventType() {
	case events.Envelope_HttpStart:
		topic = kp.HttpStartTopic(uuid2str(event.GetHttpStart().GetApplicationId()))
		kp.Stats.Inc(ConsumeHttpStart)
	case events.Envelope_HttpStartStop:
		topic = kp.HttpStartStopTopic(uuid2str(event.GetHttpStartStop().GetApplicationId()))
		kp.Stats.Inc(ConsumeHttpStartStop)
	case events.Envelope_HttpStop:
		topic = kp.HttpStopTopic(uuid2str(event.GetHttpStart().GetApplicationId()))
		kp.Stats.Inc(ConsumeHttpStop)
	case events.Envelope_LogMessage:
		topic = kp.LogMessageTopic(event.GetLogMessage().GetAppId())
		kp.Stats.Inc(ConsumeLogMessage)
	case events.Envelope_ValueMetric:
		topic = kp.ValueMetricTopic()
		kp.Stats.Inc(ConsumeValueMetric)
	case events.Envelope_CounterEvent:
		topic = kp.CounterEventTopic()
		kp.Stats.Inc(ConsumeCounterEvent)
	case events.Envelope_Error:
		topic = kp.ErrorTopic()
		kp.Stats.Inc(ConsumeError)
	case events.Envelope_ContainerMetric:
		topic = kp.ContainerMetricTopic(event.GetContainerMetric().GetApplicationId())
		kp.Stats.Inc(ConsumeContainerMetric)
	default:
		kp.Stats.Inc(ConsumeUnknown)
	}

	if topic == "" {
		kp.Stats.Inc(Ignored)
		return
	}
	kp.Stats.Inc(Forwarded)

	kp.Input() <- &sarama.ProducerMessage{Topic: topic, Value: toJSON(event)}
}
