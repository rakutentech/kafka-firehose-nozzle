package main

import (
	"fmt"
	"testing"
)

func TestJsonEncoder_Encode_NoExtraContent(t *testing.T) {
	timestamp := int64(1461318380946558204)
	expect := fmt.Sprintf(
		`{"origin":"fake-origin-1","eventType":5,"timestamp":%d,"logMessage":{"message":"aGVsbG8=","message_type":1,"timestamp":1461318380946558204,"app_id":"%[2]s","source_type":"DEA"}}`,
		timestamp, testAppId)

	encoder := toJSON(enrich(logMessage("hello", testAppId, timestamp), "", 0))

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
	expect := fmt.Sprintf(
		`{"origin":"fake-origin-1","eventType":5,"timestamp":%d,"logMessage":{"message":"aGVsbG8=","message_type":1,"timestamp":1461318380946558204,"app_id":"%[2]s","source_type":"DEA"},"app_guid":"%[2]s","instance_idx":1}`,
		timestamp, testAppId)

	encoder := toJSON(enrich(logMessage("hello", testAppId, timestamp), testAppId, 1))

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
