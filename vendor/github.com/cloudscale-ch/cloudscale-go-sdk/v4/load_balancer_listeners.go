package cloudscale

import (
	"time"
)

const loadBalancerListenerBasePath = "v1/load-balancers/listeners"

type LoadBalancerPoolStub struct {
	HREF string `json:"href,omitempty"`
	UUID string `json:"uuid,omitempty"`
	Name string `json:"name,omitempty"`
}

type LoadBalancerListener struct {
	TaggedResource
	// Just use omitempty everywhere. This makes it easy to use restful. Errors
	// will be coming from the API if something is disabled.
	HREF                   string                `json:"href,omitempty"`
	UUID                   string                `json:"uuid,omitempty"`
	Name                   string                `json:"name,omitempty"`
	Pool                   *LoadBalancerPoolStub `json:"pool,omitempty"`
	LoadBalancer           LoadBalancerStub      `json:"load_balancer,omitempty"`
	Protocol               string                `json:"protocol,omitempty"`
	ProtocolPort           int                   `json:"protocol_port,omitempty"`
	AllowedCIDRs           []string              `json:"allowed_cidrs,omitempty"`
	TimeoutClientDataMS    int                   `json:"timeout_client_data_ms,omitempty"`
	TimeoutMemberConnectMS int                   `json:"timeout_member_connect_ms,omitempty"`
	TimeoutMemberDataMS    int                   `json:"timeout_member_data_ms,omitempty"`
	CreatedAt              time.Time             `json:"created_at,omitempty"`
}

type LoadBalancerListenerRequest struct {
	TaggedResourceRequest
	Name                   string   `json:"name,omitempty"`
	Pool                   string   `json:"pool,omitempty"`
	Protocol               string   `json:"protocol,omitempty"`
	ProtocolPort           int      `json:"protocol_port,omitempty"`
	AllowedCIDRs           []string `json:"allowed_cidrs,omitempty"`
	TimeoutClientDataMS    int      `json:"timeout_client_data_ms,omitempty"`
	TimeoutMemberConnectMS int      `json:"timeout_member_connect_ms,omitempty"`
	TimeoutMemberDataMS    int      `json:"timeout_member_data_ms,omitempty"`
}

type LoadBalancerListenerService interface {
	GenericCreateService[LoadBalancerListener, LoadBalancerListenerRequest, LoadBalancerListenerRequest]
	GenericGetService[LoadBalancerListener, LoadBalancerListenerRequest, LoadBalancerListenerRequest]
	GenericListService[LoadBalancerListener, LoadBalancerListenerRequest, LoadBalancerListenerRequest]
	GenericUpdateService[LoadBalancerListener, LoadBalancerListenerRequest, LoadBalancerListenerRequest]
	GenericDeleteService[LoadBalancerListener, LoadBalancerListenerRequest, LoadBalancerListenerRequest]
}
