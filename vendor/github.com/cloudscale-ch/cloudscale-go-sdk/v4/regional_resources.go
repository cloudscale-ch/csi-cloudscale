package cloudscale

type RegionalResource struct {
	Region Region `json:"Region"`
}

type RegionalResourceRequest struct {
	Region string `json:"region,omitempty"`
}
