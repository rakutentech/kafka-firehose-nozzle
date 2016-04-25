package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"sync"

	"github.com/Shopify/sarama"
	"github.com/cloudfoundry/sonde-go/events"
	"golang.org/x/net/context"
)

type NozzleProducer interface {
	// Produce produces firehose events
	Produce(context.Context, <-chan *events.Envelope)

	// Errors returns error channel
	Errors() <-chan *sarama.ProducerError

	// Success returns sarama.ProducerMessage
	Successes() <-chan *sarama.ProducerMessage

	// Close shuts down the producer and flushes any messages it may have buffered.
	Close() error
}

var defaultLogger = log.New(ioutil.Discard, "", log.LstdFlags)

// LogProducer implements NozzleProducer interfaces.
// This producer is mainly used for debugging reason.
type LogProducer struct {
	Logger *log.Logger

	once sync.Once
}

func NewLogProducer(logger *log.Logger) NozzleProducer {
	return &LogProducer{
		Logger: logger,
	}
}

// init sets default logger
func (p *LogProducer) init() {
	if p.Logger == nil {
		p.Logger = defaultLogger
	}
}

func (p *LogProducer) Produce(ctx context.Context, eventCh <-chan *events.Envelope) {
	p.once.Do(p.init)
	for {
		select {
		case event := <-eventCh:
			buf, _ := json.Marshal(event)
			p.Logger.Printf("[INFO] %s", string(buf))
		case <-ctx.Done():
			p.Logger.Printf("[INFO] Stop producer")
			return
		}
	}
}

func (p *LogProducer) Errors() <-chan *sarama.ProducerError {
	errCh := make(chan *sarama.ProducerError, 1)
	return errCh
}

func (p *LogProducer) Successes() <-chan *sarama.ProducerMessage {
	msgCh := make(chan *sarama.ProducerMessage)
	return msgCh
}

func (p *LogProducer) Close() error {
	// Nothing to close for thi producer
	return nil
}
