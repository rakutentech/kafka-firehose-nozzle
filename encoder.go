package main

import (
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/mailru/easyjson"
)

// JsonEncoder is implemtented sarama.Encoder interface.
// It transforms protobuf data to JSON data.
type JsonEncoder struct {
	encoded []byte
	err     error
}

func toJSON(e *events.Envelope) *JsonEncoder {
	encoded, err := easyjson.Marshal(e)
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
