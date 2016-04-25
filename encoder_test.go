package main

import (
	"fmt"
	"testing"
)

func TestJsonEncoder_Encode(t *testing.T) {
	timestamp := int64(1461318380946558204)
	expect := fmt.Sprintf(
		`{"origin":"fake-origin-1","eventType":5,"timestamp":%d,"logMessage":{"message":"aGVsbG8=","message_type":1,"timestamp":1461318380946558204,"app_id":"%s","source_type":"DEA"}}`,
		timestamp, testAppId)
	expectLength := 225

	encoder := &JsonEncoder{
		event: logMessage("hello", testAppId, timestamp),
	}

	buf, err := encoder.Encode()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if string(buf) != expect {
		t.Fatalf("expect %q to be eq %q", string(buf), expect)
	}

	if encoder.Length() != expectLength {
		t.Fatalf("expect %d to be eq %d", encoder.Length(), expectLength)
	}
}
