package cloudscale

type ZonalResource struct {
	Zone Zone `json:"zone"`
}

type ZonalResourceRequest struct {
	Zone string `json:"zone,omitempty"`
}
