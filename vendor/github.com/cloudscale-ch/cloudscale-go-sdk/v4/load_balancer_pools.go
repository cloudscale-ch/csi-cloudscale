package cloudscale

import (
	"time"
)

const loadBalancerPoolBasePath = "v1/load-balancers/pools"

type LoadBalancerPool struct {
	TaggedResource
	// Just use omitempty everywhere. This makes it easy to use restful. Errors
	// will be coming from the API if something is disabled.
	HREF         string           `json:"href,omitempty"`
	UUID         string           `json:"uuid,omitempty"`
	Name         string           `json:"name,omitempty"`
	CreatedAt    time.Time        `json:"created_at,omitempty"`
	LoadBalancer LoadBalancerStub `json:"load_balancer,omitempty"`
	Algorithm    string           `json:"algorithm,omitempty"`
	Protocol     string           `json:"protocol,omitempty"`
}

type LoadBalancerPoolRequest struct {
	TaggedResourceRequest
	Name         string `json:"name,omitempty"`
	LoadBalancer string `json:"load_balancer,omitempty"`
	Algorithm    string `json:"algorithm,omitempty"`
	Protocol     string `json:"protocol,omitempty"`
}

type LoadBalancerPoolService interface {
	GenericCreateService[LoadBalancerPool, LoadBalancerPoolRequest, LoadBalancerPoolRequest]
	GenericGetService[LoadBalancerPool, LoadBalancerPoolRequest, LoadBalancerPoolRequest]
	GenericListService[LoadBalancerPool, LoadBalancerPoolRequest, LoadBalancerPoolRequest]
	GenericUpdateService[LoadBalancerPool, LoadBalancerPoolRequest, LoadBalancerPoolRequest]
	GenericDeleteService[LoadBalancerPool, LoadBalancerPoolRequest, LoadBalancerPoolRequest]
}
