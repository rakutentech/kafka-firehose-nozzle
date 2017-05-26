package main

import "github.com/cloudfoundry/sonde-go/events"

type Envelope struct {
	*events.Envelope
	ExtraContent
}

type ExtraContent struct {
	AppGuid   string `json:"app_guid,omitempty"`
	AppName   string `json:"app_name,omitempty"`
	SpaceGuid string `json:"space_guid,omitempty"`
	SpaceName string `json:"space_name,omitempty"`
	OrgGuid   string `json:"org_guid,omitempty"`
	OrgName   string `json:"org_name,omitempty"`
}

func enrich(e *events.Envelope, AppGuid string) *Envelope {
	return &Envelope{e, ExtraContent{AppGuid: AppGuid}}
}
