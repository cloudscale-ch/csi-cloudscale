package cloudscale

type Metadata struct {
	AvailabilityZone string   `json:"availability_zone,omitempty"`
	// For now don't define those, because they don't need to match the
	// official cloudscale.ch API and are part of OpenStack.
	//Hostname         string   `json:"hostname,omitempty"`
	//Name             string   `json:"user_data,omitempty"`

	Meta struct {
		CloudscaleUUID string `json:"cloudscale_uuid,omitempty"`
	} `json:"meta,omitempty"`
}
