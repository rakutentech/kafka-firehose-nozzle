package main

import (
	"sync/atomic"
	"time"
)

type StatsType int

const (
	Consume StatsType = iota
	ConsumeFail
	Publish
	PublishFail
	SlowConsumer
)

// Stats stores various stats infomation
type Stats struct {
	Consume     uint64
	ConsumeFail uint64

	Publish     uint64
	PublishFail uint64

	SlowConsumer uint64

	ConsumePerSec uint64
	PublishPerSec uint64
}

func NewStats() *Stats {
	return &Stats{}
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
	case SlowConsumer:
		atomic.AddUint64(&s.SlowConsumer, 1)
	}
}
