package main

import (
	"encoding/json"
)

// JsonEncoder is implemtented sarama.Encoder interface.
// It transforms protobuf data to JSON data.
type JsonEncoder struct {
	encoded []byte
	err     error
}

func toJSON(e *Envelope) *JsonEncoder {
	encoded, err := json.Marshal(e)
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
