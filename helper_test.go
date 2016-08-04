package main

import (
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/gogo/protobuf/proto"
)

const (
	// testAppId is GUID for test application
	testAppId = "3356a5c7-e86c-442a-b14f-ce5cc4f80ed1"
)

func logMessage(message, appId string, timestamp int64) *events.Envelope {
	logMessage := &events.LogMessage{
		Message:     []byte(message),
		MessageType: events.LogMessage_OUT.Enum(),
		AppId:       proto.String(appId),
		SourceType:  proto.String("DEA"),
		Timestamp:   proto.Int64(timestamp),
	}

	return &events.Envelope{
		LogMessage: logMessage,
		EventType:  events.Envelope_LogMessage.Enum(),
		Origin:     proto.String("fake-origin-1"),
		Timestamp:  proto.Int64(timestamp),
	}
}

func valueMetric(timestamp int64) *events.Envelope {
	valueMetric := &events.ValueMetric{
		Name:  proto.String("df"),
		Value: proto.Float64(0.99),
	}
	return &events.Envelope{
		ValueMetric: valueMetric,
		EventType:   events.Envelope_ValueMetric.Enum(),
		Origin:      proto.String("fake-origin-2"),
		Timestamp:   proto.Int64(timestamp),
	}
}

type Int32Slice []int32

func (p Int32Slice) Len() int           { return len(p) }
func (p Int32Slice) Less(i, j int) bool { return p[i] < p[j] }
func (p Int32Slice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
