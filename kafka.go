package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"golang.org/x/net/context"

	"github.com/Shopify/sarama"
	"github.com/cloudfoundry/sonde-go/events"
)

const (
	DefaultKafkaRepartitionMax = 5
	DefaultKafkaRetryMax       = 1
	DefaultKafkaRetryBackoff   = 100 * time.Millisecond

	DefaultChannelBufferSize  = 512 // Sarama default is 256
	DefaultSubInputBufferSize = 1024
)

func NewKafkaProducer(logger *log.Logger, stats *Stats, config *Config) (NozzleProducer, error) {
	// Setup kafka async producer (We must use sync producer)
	// TODO (tcnksm): Enable to configure more properties.
	producerConfig := sarama.NewConfig()

	if config.Kafka.EnableTLS {
		if config.Kafka.ClientCert == "" {
			return nil, errors.New("please specify client_certificate")
		}
		if config.Kafka.ClientKey == "" {
			return nil, errors.New("please specify private_key")
		}

		producerConfig.Net.TLS.Enable = true
		if producerConfig.Net.TLS.Config == nil {
			producerConfig.Net.TLS.Config = &tls.Config{}
		}

		if len(config.Kafka.CACerts) == 0 {
			var err error
			producerConfig.Net.TLS.Config.RootCAs, err = x509.SystemCertPool()
			if err != nil {
				return nil, err
			}
		} else {
			producerConfig.Net.TLS.Config.RootCAs = x509.NewCertPool()
			for _, certString := range config.Kafka.CACerts {
				if !producerConfig.Net.TLS.Config.RootCAs.AppendCertsFromPEM([]byte(certString)) {
					return nil, errors.New("no certs in ca pem")
				}
			}
		}

		cert, err := tls.X509KeyPair([]byte(config.Kafka.ClientCert), []byte(config.Kafka.ClientKey))
		if err != nil {
			return nil, err
		}

		producerConfig.Net.TLS.Config.Certificates = []tls.Certificate{cert}
		producerConfig.Net.TLS.Config.BuildNameToCertificate()
	}

	producerConfig.Producer.Partitioner = sarama.NewRoundRobinPartitioner
	producerConfig.Producer.Return.Successes = true
	producerConfig.Producer.RequiredAcks = sarama.WaitForAll

	// This is the default, but Errors are required for repartitioning
	producerConfig.Producer.Return.Errors = true

	producerConfig.Producer.Retry.Max = DefaultKafkaRetryMax
	if config.Kafka.RetryMax != 0 {
		producerConfig.Producer.Retry.Max = config.Kafka.RetryMax
	}

	producerConfig.Producer.Retry.Backoff = DefaultKafkaRetryBackoff
	if config.Kafka.RetryBackoff != 0 {
		backoff := time.Duration(config.Kafka.RetryBackoff) * time.Millisecond
		producerConfig.Producer.Retry.Backoff = backoff
	}

	producerConfig.ChannelBufferSize = DefaultChannelBufferSize

	switch config.Kafka.Compression {
	case "gzip":
		producerConfig.Producer.Compression = sarama.CompressionGZIP
	case "snappy":
		producerConfig.Producer.Compression = sarama.CompressionSnappy
	case "none":
		break
	case "":
		break
	default:
		return nil, fmt.Errorf("unknown compression codec: %s", config.Kafka.Compression)
	}

	brokers := config.Kafka.Brokers
	if len(brokers) < 1 {
		return nil, fmt.Errorf("brokers are not provided")
	}

	asyncProducer, err := sarama.NewAsyncProducer(brokers, producerConfig)
	if err != nil {
		return nil, err
	}

	repartitionMax := DefaultKafkaRepartitionMax
	if config.Kafka.RepartitionMax != 0 {
		repartitionMax = config.Kafka.RepartitionMax
	}

	subInputBuffer := DefaultSubInputBufferSize
	subInputCh := make(chan *sarama.ProducerMessage, subInputBuffer)

	return &KafkaProducer{
		AsyncProducer:           asyncProducer,
		config:                  producerConfig,
		Logger:                  logger,
		Stats:                   stats,
		logMessageTopic:         config.Kafka.Topic.LogMessage,
		logMessageTopicFmt:      config.Kafka.Topic.LogMessageFmt,
		valueMetricTopic:        config.Kafka.Topic.ValueMetric,
		containerMetricTopic:    config.Kafka.Topic.ContainerMetric,
		containerMetricTopicFmt: config.Kafka.Topic.ContainerMetricFmt,
		httpStartStopTopic:      config.Kafka.Topic.HttpStartStop,
		httpStartStopTopicFmt:   config.Kafka.Topic.HttpStartStopFmt,
		counterEventTopic:       config.Kafka.Topic.CounterEvent,
		errorTopic:              config.Kafka.Topic.Error,
		repartitionMax:          repartitionMax,
		subInputCh:              subInputCh,
		errors:                  make(chan *sarama.ProducerError),
	}, nil
}

// KafkaProducer implements NozzleProducer interfaces
type KafkaProducer struct {
	sarama.AsyncProducer

	config *sarama.Config

	repartitionMax int
	errors         chan *sarama.ProducerError

	// SubInputCh is buffer for re-partitioning
	subInputCh chan *sarama.ProducerMessage

	logMessageTopic         string
	logMessageTopicFmt      string
	valueMetricTopic        string
	containerMetricTopic    string
	containerMetricTopicFmt string
	httpStartStopTopic      string
	httpStartStopTopicFmt   string
	counterEventTopic       string
	errorTopic              string

	Logger *log.Logger
	Stats  *Stats

	once sync.Once
}

// metadata is metadata which will be injected to ProducerMessage.Metadata.
// This is used only when publish is failed and re-partitioning by ourself.
type metadata struct {
	// retires is the number of re-partitioning
	retries int
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

func (kp *KafkaProducer) Errors() <-chan *sarama.ProducerError {
	return kp.errors
}

// Produce produces event to kafka
func (kp *KafkaProducer) Produce(ctx context.Context, eventCh <-chan *events.Envelope) {
	kp.once.Do(kp.init)

	kp.Logger.Printf("[INFO] Start to watching producer error for re-partition")
	go func() {
		for producerErr := range kp.AsyncProducer.Errors() {
			// Instead of giving up, try to resubmit the message so that it can end up
			// on a different partition (we don't care about order of message)
			// This is a workaround for https://github.com/Shopify/sarama/issues/514
			meta, _ := producerErr.Msg.Metadata.(metadata)
			kp.Logger.Printf("[ERROR] Producer error %+v", producerErr)

			if meta.retries >= kp.repartitionMax {
				kp.errors <- producerErr
				continue
			}

			// NOTE: We need to re-create Message because original message
			// which producer.Error stores internal state (unexported field)
			// and it effect partitioning.
			originalMsg := producerErr.Msg
			msg := &sarama.ProducerMessage{
				Topic: originalMsg.Topic,
				Value: originalMsg.Value,

				// Update retry count
				Metadata: metadata{
					retries: meta.retries + 1,
				},
			}

			// If sarama buffer is full, then input it to nozzle side buffer
			// (subInput) and retry to produce it later. When subInput is
			// full, we drop message.
			//
			// TODO(tcnksm): Monitor subInput buffer.
			select {
			case kp.Input() <- msg:
				kp.Logger.Printf("[DEBUG] Repartitioning")
			default:
				select {
				case kp.subInputCh <- msg:
					kp.Stats.Inc(SubInputBuffer)
				default:
					// If subInput is full, then drop message.....
					kp.errors <- producerErr
				}
			}
		}
	}()

	kp.Logger.Printf("[INFO] Start to sub input (buffer for sarama input)")
	go func() {
		for msg := range kp.subInputCh {
			kp.Input() <- msg
			kp.Logger.Printf("[DEBUG] Repartitioning (from subInput)")
			kp.Stats.Dec(SubInputBuffer)
		}
	}()

	kp.Logger.Printf("[INFO] Start loop to watch events")
	for {
		select {
		case <-ctx.Done():
			// Stop process immediately
			kp.Logger.Printf("[INFO] Stop kafka producer")
			return

		case event, ok := <-eventCh:
			if !ok {
				kp.Logger.Printf("[ERROR] Nozzle consumer eventCh is closed")
				return
			}

			kp.input(event)
		}
	}
}

func (kp *KafkaProducer) input(event *events.Envelope) {
	topic := ""

	kp.Stats.Inc(Consume)

	switch event.GetEventType() {
	case events.Envelope_HttpStartStop:
		topic = kp.HttpStartStopTopic(uuid2str(event.GetHttpStartStop().GetApplicationId()))
		kp.Stats.Inc(ConsumeHttpStartStop)
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

	kp.Input() <- &sarama.ProducerMessage{Topic: topic, Value: toJSON(event), Metadata: metadata{retries: 0}}
}
