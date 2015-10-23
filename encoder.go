package main

import (
	"encoding/json"

	"github.com/cloudfoundry/sonde-go/events"
)

// JsonEncoder is implemtented sarama.Encoder interface.
// It transforms protobuf data to JSON data.
type JsonEncoder struct {
	event   *events.Envelope
	encoded []byte
	err     error
}

// Encode returns json encoded data. If any, returns error.
func (j *JsonEncoder) Encode() ([]byte, error) {
	j.encode()
	return j.encoded, j.err
}

// Length returns length json encoded data.
func (j JsonEncoder) Length() int {
	j.encode()
	return len(j.encoded)
}

// encode encodes data to json.
func (j *JsonEncoder) encode() {
	if j.encoded == nil && j.err == nil {
		j.encoded, j.err = json.Marshal(j.event)
	}
}
