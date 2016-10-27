package main

import (
	"encoding/json"
	"os"
	"strconv"
	"sync/atomic"
	"time"
)

const (
	defaultInstanceID int = 0
)

const (
	EnvCFInstanceIndex = "CF_INSTANCE_INDEX"
)

type StatsType int

const (
	Consume StatsType = iota
	ConsumeFail
	Publish
	PublishFail
	SlowConsumerAlert
	SubInputBuffer
)

// Stats stores various stats infomation
type Stats struct {
	Consume       uint64 `json:"consume"`
	ConsumePerSec uint64 `json:"consume_per_sec"`
	ConsumeFail   uint64 `json:"consume_fail"`

	Publish       uint64 `json:"publish"`
	PublishPerSec uint64 `json:"publish_per_sec"`

	// This is same as the number of dropped message
	PublishFail uint64 `json:"publish_fail"`

	SlowConsumerAlert uint64 `json:"slow_consumer_alert"`

	// SubInputBuffer is used to count number of current
	// buffer on subInput.
	SubInputBuffer int64 `json:"subinupt_buffer"`

	// Delay is Consume - Pulish
	// This indicate how slow publish to kafka
	Delay uint64 `json:"delay"`

	// InstanceID is ID for nozzle instance.
	// This is used to identify stats from different instances.
	// By default, it's defaultInstanceID
	InstanceID int `json:"instance_id"`
}

func NewStats() *Stats {
	instanceID := defaultInstanceID
	if idStr := os.Getenv(EnvCFInstanceIndex); len(idStr) != 0 {
		var err error
		instanceID, err = strconv.Atoi(idStr)
		if err != nil {
			// If it's failed to conv str to int
			// use default var
			instanceID = defaultInstanceID
		}
	}

	return &Stats{
		InstanceID: instanceID,
	}
}

func (s *Stats) Json() ([]byte, error) {
	s.Delay = s.Consume - s.Publish - s.PublishFail
	return json.Marshal(s)
}

func (s *Stats) PerSec() {
	lastConsume, lastPublish := uint64(0), uint64(0)
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ticker.C:
			s.ConsumePerSec = s.Consume - lastConsume
			s.PublishPerSec = s.Publish - lastPublish

			lastConsume = s.Consume
			lastPublish = s.Publish
		}
	}
}

func (s *Stats) Inc(statsType StatsType) {
	switch statsType {
	case Consume:
		atomic.AddUint64(&s.Consume, 1)
	case ConsumeFail:
		atomic.AddUint64(&s.ConsumeFail, 1)
	case Publish:
		atomic.AddUint64(&s.Publish, 1)
	case PublishFail:
		atomic.AddUint64(&s.PublishFail, 1)
	case SlowConsumerAlert:
		atomic.AddUint64(&s.SlowConsumerAlert, 1)
	case SubInputBuffer:
		atomic.AddInt64(&s.SubInputBuffer, 1)
	}
}

func (s *Stats) Dec(statsType StatsType) {
	switch statsType {
	case SubInputBuffer:
		atomic.AddInt64(&s.SubInputBuffer, -1)
	}
}
