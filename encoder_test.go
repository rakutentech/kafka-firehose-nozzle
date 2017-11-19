package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/mailru/easyjson"
)

func TestJsonEncoder_Encode(t *testing.T) {
	timestamp := int64(1461318380946558204)
	expect := fmt.Sprintf(
		`{"origin":"fake-origin-1","eventType":5,"timestamp":%d,"logMessage":{"message":"aGVsbG8=","message_type":1,"timestamp":1461318380946558204,"app_id":"%s","source_type":"DEA"}}`,
		timestamp, testAppId)
	expectLength := 225

	encoder := toJSON(logMessage("hello", testAppId, timestamp))
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

var (
	msgLog  = logMessage(strings.Repeat("x", 100), testAppId, int64(1461318380946558204))
	msgHttp = httpStartStop(testAppId, int64(1461318380946558204))
	msgMet  = containerMetric(testAppId, int64(1461318380946558204))
)

func BenchmarkJsonEncode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		json.Marshal(msgLog)
		json.Marshal(msgHttp)
		json.Marshal(msgMet)
	}
}

func BenchmarkEasyJsonEncodeBuffer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		easyjson.Marshal(msgLog)
		easyjson.Marshal(msgHttp)
		easyjson.Marshal(msgMet)
	}
}

func BenchmarkEasyJsonEncodeWriter(b *testing.B) {
	buf := &bytes.Buffer{}
	for i := 0; i < b.N; i++ {
		buf.Reset()
		easyjson.MarshalToWriter(msgLog, buf)
		buf.Reset()
		easyjson.MarshalToWriter(msgHttp, buf)
		buf.Reset()
		easyjson.MarshalToWriter(msgMet, buf)
	}
}
