package main

import "testing"

func TestJsonEncoder_Encode(t *testing.T) {
	timestamp := int64(1461318380946558204)
	expect := `{"origin":"fake-origin-1","eventType":5,"timestamp":1461318380946558204,"logMessage":{"message":"aGVsbG8=","message_type":1,"timestamp":1461318380946558204,"app_id":"3356a5c7-e86c-442a-b14f-ce5cc4f80ed1","source_type":"DEA"}}`
	expectLength := 225

	encoder := &JsonEncoder{
		event: testEvent("hello", timestamp),
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
