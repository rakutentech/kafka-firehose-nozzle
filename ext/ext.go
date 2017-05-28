//go:generate ffjson $GOFILE
package ext

import "github.com/cloudfoundry/sonde-go/events"

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

type Envelope struct {
	*events.Envelope
	ExtraContent
}
