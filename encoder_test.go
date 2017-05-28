package main

import (
	"fmt"
	"testing"
)

func TestJsonEncoder_Encode_NoExtraContent(t *testing.T) {
	timestamp := int64(1461318380946558204)
	expect := fmt.Sprintf(
		`{ "origin":"fake-origin-1","eventType":5,"timestamp":%d,"logMessage":{ "message":"aGVsbG8=","message_type":1,"timestamp":1461318380946558204,"app_id":"%[2]s","source_type":"DEA"}}`,
		timestamp, testAppId)

	encoder := extEnvelopeJSON(enrich(logMessage("hello", testAppId, timestamp), "", 0))

	buf, err := encoder.Encode()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if string(buf) != expect {
		t.Fatalf("expect %q to be eq %q", string(buf), expect)
	}

	if encoder.Length() != len(expect) {
		t.Fatalf("expect %d to be eq %d", encoder.Length(), len(expect))
	}
}

func TestJsonEncoder_Encode_ExtraContent(t *testing.T) {
	timestamp := int64(1461318380946558204)
	instanceIdx := 2
	expect := fmt.Sprintf(
		`{ "origin":"fake-origin-1","eventType":5,"timestamp":%[1]d,"logMessage":{ "message":"aGVsbG8=","message_type":1,"timestamp":1461318380946558204,"app_id":"%[2]s","source_type":"DEA"},"app_guid":"%[2]s","instance_idx":%[3]d}`,
		timestamp, testAppId, instanceIdx)

	encoder := extEnvelopeJSON(enrich(logMessage("hello", testAppId, timestamp), testAppId, instanceIdx))

	buf, err := encoder.Encode()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if string(buf) != expect {
		t.Fatalf("expect %q to be eq %q", string(buf), expect)
	}

	if encoder.Length() != len(expect) {
		t.Fatalf("expect %d to be eq %d", encoder.Length(), len(expect))
	}
}
