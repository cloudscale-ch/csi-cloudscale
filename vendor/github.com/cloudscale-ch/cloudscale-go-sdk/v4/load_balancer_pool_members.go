package cloudscale

import (
	"context"
	"fmt"
	"time"
)

const loadBalancerPoolMemberBasePath = "v1/load-balancers/pools/%s/members"

type LoadBalancerPoolMember struct {
	TaggedResource
	// Just use omitempty everywhere. This makes it easy to use restful. Errors
	// will be coming from the API if something is disabled.
	HREF          string               `json:"href,omitempty"`
	UUID          string               `json:"uuid,omitempty"`
	Name          string               `json:"name,omitempty"`
	Enabled       bool                 `json:"enabled,omitempty"`
	CreatedAt     time.Time            `json:"created_at,omitempty"`
	Pool          LoadBalancerPoolStub `json:"pool,omitempty"`
	LoadBalancer  LoadBalancerStub     `json:"load_balancer,omitempty"`
	ProtocolPort  int                  `json:"protocol_port,omitempty"`
	MonitorPort   int                  `json:"monitor_port,omitempty"`
	Address       string               `json:"address,omitempty"`
	Subnet        SubnetStub           `json:"subnet,omitempty"`
	MonitorStatus string               `json:"monitor_status,omitempty"`
}

type LoadBalancerPoolMemberRequest struct {
	TaggedResourceRequest
	Name         string `json:"name,omitempty"`
	Enabled      *bool  `json:"enabled,omitempty"`
	ProtocolPort int    `json:"protocol_port,omitempty"`
	MonitorPort  int    `json:"monitor_port,omitempty"`
	Address      string `json:"address,omitempty"`
	Subnet       string `json:"subnet,omitempty"`
}

type LoadBalancerPoolMemberService interface {
	Create(ctx context.Context, poolID string, createRequest *LoadBalancerPoolMemberRequest) (*LoadBalancerPoolMember, error)
	Get(ctx context.Context, poolID string, resourceID string) (*LoadBalancerPoolMember, error)
	List(ctx context.Context, poolID string, modifiers ...ListRequestModifier) ([]LoadBalancerPoolMember, error)
	Update(ctx context.Context, poolID string, resourceID string, updateRequest *LoadBalancerPoolMemberRequest) error
	Delete(ctx context.Context, poolID string, resourceID string) error
}

type LoadBalancerPoolMemberServiceOperations struct {
	client *Client
}

func (l LoadBalancerPoolMemberServiceOperations) Create(ctx context.Context, poolID string, createRequest *LoadBalancerPoolMemberRequest) (*LoadBalancerPoolMember, error) {
	g := parameterizeGenericInstance(l, poolID)
	return g.Create(ctx, createRequest)
}

func (l LoadBalancerPoolMemberServiceOperations) Get(ctx context.Context, poolID string, resourceID string) (*LoadBalancerPoolMember, error) {
	g := parameterizeGenericInstance(l, poolID)
	return g.Get(ctx, resourceID)
}

func (l LoadBalancerPoolMemberServiceOperations) List(ctx context.Context, poolID string, modifiers ...ListRequestModifier) ([]LoadBalancerPoolMember, error) {
	g := parameterizeGenericInstance(l, poolID)
	return g.List(ctx, modifiers...)
}

func (l LoadBalancerPoolMemberServiceOperations) Update(ctx context.Context, poolID string, resourceID string, updateRequest *LoadBalancerPoolMemberRequest) error {
	g := parameterizeGenericInstance(l, poolID)
	return g.Update(ctx, resourceID, updateRequest)
}

func (l LoadBalancerPoolMemberServiceOperations) Delete(ctx context.Context, poolID string, resourceID string) error {
	g := parameterizeGenericInstance(l, poolID)
	return g.Delete(ctx, resourceID)
}

func parameterizeGenericInstance(l LoadBalancerPoolMemberServiceOperations, poolID string) GenericServiceOperations[LoadBalancerPoolMember, LoadBalancerPoolMemberRequest, LoadBalancerPoolMemberRequest] {
	return GenericServiceOperations[LoadBalancerPoolMember, LoadBalancerPoolMemberRequest, LoadBalancerPoolMemberRequest]{
		client: l.client,
		path:   fmt.Sprintf(loadBalancerPoolMemberBasePath, poolID),
	}
}
