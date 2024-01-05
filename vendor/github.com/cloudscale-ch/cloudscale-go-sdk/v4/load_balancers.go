package cloudscale

import (
	"time"
)

const loadBalancerBasePath = "v1/load-balancers"

type LoadBalancerStub struct {
	HREF string `json:"href,omitempty"`
	UUID string `json:"uuid,omitempty"`
	Name string `json:"name,omitempty"`
}

type LoadBalancerFlavorStub struct {
	Slug string `json:"slug,omitempty"`
	Name string `json:"name,omitempty"`
}

type LoadBalancer struct {
	ZonalResource
	TaggedResource
	// Just use omitempty everywhere. This makes it easy to use restful. Errors
	// will be coming from the API if something is disabled.
	HREF         string                 `json:"href,omitempty"`
	UUID         string                 `json:"uuid,omitempty"`
	Name         string                 `json:"name,omitempty"`
	Flavor       LoadBalancerFlavorStub `json:"flavor,omitempty"`
	Status       string                 `json:"status,omitempty"`
	VIPAddresses []VIPAddress           `json:"vip_addresses,omitempty"`
	CreatedAt    time.Time              `json:"created_at,omitempty"`
}

type VIPAddress struct {
	Version int        `json:"version,omitempty"`
	Address string     `json:"address,omitempty"`
	Subnet  SubnetStub `json:"subnet,omitempty"`
}

type LoadBalancerRequest struct {
	ZonalResourceRequest
	TaggedResourceRequest
	Name         string               `json:"name,omitempty"`
	Flavor       string               `json:"flavor,omitempty"`
	VIPAddresses *[]VIPAddressRequest `json:"vip_addresses,omitempty"`
}

type VIPAddressRequest struct {
	Address string `json:"address,omitempty"`
	Subnet  string `json:"subnet,omitempty"`
}

type LoadBalancerService interface {
	GenericCreateService[LoadBalancer, LoadBalancerRequest, LoadBalancerRequest]
	GenericGetService[LoadBalancer, LoadBalancerRequest, LoadBalancerRequest]
	GenericListService[LoadBalancer, LoadBalancerRequest, LoadBalancerRequest]
	GenericUpdateService[LoadBalancer, LoadBalancerRequest, LoadBalancerRequest]
	GenericDeleteService[LoadBalancer, LoadBalancerRequest, LoadBalancerRequest]
}
