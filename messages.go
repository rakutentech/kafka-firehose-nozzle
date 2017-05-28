package main

import (
	"github.com/rakutentech/kafka-firehose-nozzle/ext"

	"github.com/cloudfoundry/sonde-go/events"
)

func enrich(e *events.Envelope, AppGuid string, InstanceIdx int) *ext.Envelope {
	return &ext.Envelope{e, ext.ExtraContent{AppGuid: AppGuid, InstanceIdx: InstanceIdx}}
}
