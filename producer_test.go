package main

import "testing"

func TestLogProducer_implement(t *testing.T) {
	var _ NozzleProducer = &LogProducer{}
}

func TestLogProducer_produce(t *testing.T) {
}
func TestLogProducer_Errors(t *testing.T) {
}
func TestLogProducer_Close(t *testing.T) {
}
