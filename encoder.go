package main

import (
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/pquerna/ffjson/ffjson"
	"github.com/rakutentech/kafka-firehose-nozzle/ext"
)

// JsonEncoder is implemtented sarama.Encoder interface.
// It transforms protobuf data to JSON data.
type JsonEncoder struct {
	encoded []byte
	err     error
}

func extEnvelopeJSON(e *ext.Envelope) *JsonEncoder {
	encoded, err := ffjson.Marshal(e)
	return &JsonEncoder{encoded, err}
}

func eventsEnvelopeJSON(e *events.Envelope) *JsonEncoder {
	encoded, err := ffjson.Marshal(e)
	return &JsonEncoder{encoded, err}
}

// Encode returns json encoded data. If any, returns error.
func (j *JsonEncoder) Encode() ([]byte, error) {
	return j.encoded, j.err
}

// Length returns length json encoded data.
func (j JsonEncoder) Length() int {
	return len(j.encoded)
}

func (j JsonEncoder) Recycle() {
	if j.encoded != nil {
		ffjson.Pool(j.encoded)
		j.encoded = nil
	}
}
