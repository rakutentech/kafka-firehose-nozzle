package main

import (
	"bytes"
	"encoding/json"
	"sync"
	"testing"
)

func TestStatsInc(t *testing.T) {

	s := NewStats()

	loop := 20
	inc := 5

	var wg sync.WaitGroup
	wg.Add(loop)
	for i := 0; i < loop; i++ {
		go func() {
			defer wg.Done()
			for i := 0; i < inc; i++ {
				s.Inc(Consume)
			}
		}()
	}

	wg.Wait()

	expect := loop * inc
	if s.Consume != uint64(expect) {
		t.Fatalf("expect %d to be eq %d", s.Consume, expect)
	}
}

func TestStatsJson(t *testing.T) {
	s := NewStats()

	s.Inc(Consume)
	s.Inc(Consume)
	s.Inc(Publish)

	expect := `{
  "consume": 2,
  "consume_per_sec": 0,
  "consume_fail": 0,
  "publish": 1,
  "publish_per_sec": 0,
  "publish_fail": 0,
  "slow_consumer_alert": 0,
  "delay": 1
}`

	b, _ := s.Json()

	var buf bytes.Buffer
	json.Indent(&buf, b, "", "  ")
	if buf.String() != expect {
		t.Fatalf("expect %v to be eq %v", buf.String(), expect)
	}
}
