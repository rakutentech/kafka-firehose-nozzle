package main

import "github.com/cloudfoundry/sonde-go/events"

type Envelope struct {
	*events.Envelope
	ExtraContent
}

type ExtraContent struct {
	AppGuid     string `json:"app_guid,omitempty"`
	AppName     string `json:"app_name,omitempty"`
	SpaceGuid   string `json:"space_guid,omitempty"`
	SpaceName   string `json:"space_name,omitempty"`
	OrgGuid     string `json:"org_guid,omitempty"`
	OrgName     string `json:"org_name,omitempty"`
	InstanceID  string `json:"instance_guid,omitempty"`
	InstanceIdx int    `json:"instance_idx,omitempty"`
}

func enrich(e *events.Envelope, AppGuid string, InstanceIdx int) *Envelope {
	return &Envelope{e, ExtraContent{AppGuid: AppGuid, InstanceIdx: InstanceIdx}}
}
