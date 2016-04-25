package main

import (
	"bytes"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/cloudfoundry/sonde-go/events"
	"golang.org/x/net/context"
)

func TestLogProducer_implement(t *testing.T) {
	var _ NozzleProducer = &LogProducer{}
}

func TestLogProducer_Produce(t *testing.T) {
	buf := new(bytes.Buffer)
	bufLogger := log.New(buf, "", log.LstdFlags)
	producer := NewLogProducer(bufLogger)

	// Create test eventCh where producer gets actual messages
	doneCh := make(chan struct{}, 1)
	eventCh := make(chan *events.Envelope)

	// Start producing
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		producer.Produce(ctx, eventCh)
		doneCh <- struct{}{}
	}()

	// Create test event and send it to channel
	timestamp := time.Now().UnixNano()
	eventCh <- logMessage("", testAppId, timestamp)

	// Stop producing
	cancel()

	// Wihtout this sync, log writing is slow and buf will be just empty.
	<-doneCh

	expect := "logMessage"
	if !strings.Contains(buf.String(), expect) {
		t.Fatalf("expect %q to contain %q", buf.String(), expect)
	}
}
