package main

import (
	"encoding/binary"
	"encoding/hex"
	"strings"

	"github.com/cloudfoundry/sonde-go/events"
	"github.com/gogo/protobuf/proto"
)

const (
	// testAppId is GUID for test application
	testAppId = "3356a5c7-e86c-442a-b14f-ce5cc4f80ed1"
)

func str2uuid(s string) *events.UUID {
	s = strings.Replace(s, "-", "", 4)
	buf, _ := hex.DecodeString(s)
	return &events.UUID{
		Low:  proto.Uint64(binary.LittleEndian.Uint64(buf[0:8])),
		High: proto.Uint64(binary.LittleEndian.Uint64(buf[8:16])),
	}
}

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

func containerMetric(appId string, timestamp int64) *events.Envelope {
	return &events.Envelope{
		EventType: events.Envelope_ContainerMetric.Enum(),
		Origin:    proto.String("fake-origin-3"),
		Timestamp: proto.Int64(timestamp),
		ContainerMetric: &events.ContainerMetric{
			ApplicationId: proto.String(appId),
			InstanceIndex: proto.Int32(0),
		},
	}
}

func httpStart(appId string, timestamp int64) *events.Envelope {
	return &events.Envelope{
		EventType: events.Envelope_HttpStart.Enum(),
		Origin:    proto.String("fake-origin-4"),
		Timestamp: proto.Int64(timestamp),
		HttpStart: &events.HttpStart{
			ApplicationId: str2uuid(appId),
		},
	}
}

func httpStop(appId string, timestamp int64) *events.Envelope {
	return &events.Envelope{
		EventType: events.Envelope_HttpStop.Enum(),
		Origin:    proto.String("fake-origin-5"),
		Timestamp: proto.Int64(timestamp),
		HttpStop: &events.HttpStop{
			ApplicationId: str2uuid(appId),
		},
	}
}

func httpStartStop(appId string, timestamp int64) *events.Envelope {
	return &events.Envelope{
		EventType: events.Envelope_HttpStartStop.Enum(),
		Origin:    proto.String("fake-origin-6"),
		Timestamp: proto.Int64(timestamp),
		HttpStartStop: &events.HttpStartStop{
			ApplicationId: str2uuid(appId),
		},
	}
}

func counterEvent(timestamp int64) *events.Envelope {
	return &events.Envelope{
		EventType: events.Envelope_CounterEvent.Enum(),
		Origin:    proto.String("fake-origin-7"),
		Timestamp: proto.Int64(timestamp),
		CounterEvent: &events.CounterEvent{
			Name: proto.String("test-event"),
		},
	}
}

func errorMsg(timestamp int64) *events.Envelope {
	return &events.Envelope{
		EventType: events.Envelope_Error.Enum(),
		Origin:    proto.String("fake-origin-8"),
		Timestamp: proto.Int64(timestamp),
		Error: &events.Error{
			Message: proto.String("test-error"),
		},
	}
}

func unknown(timestamp int64) *events.Envelope {
	return &events.Envelope{
		EventType: (*events.Envelope_EventType)(proto.Int32(-1)),
		Origin:    proto.String("fake-origin-9"),
		Timestamp: proto.Int64(timestamp),
	}
}

type Int32Slice []int32

func (p Int32Slice) Len() int           { return len(p) }
func (p Int32Slice) Less(i, j int) bool { return p[i] < p[j] }
func (p Int32Slice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
