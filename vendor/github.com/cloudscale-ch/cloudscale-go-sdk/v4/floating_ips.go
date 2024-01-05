package cloudscale

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const floatingIPsBasePath = "v1/floating-ips"

type FloatingIP struct {
	Region *Region `json:"region"` // not using RegionalResource here, as FloatingIP can be regional or global
	TaggedResource
	HREF           string            `json:"href"`
	Network        string            `json:"network"`
	IPVersion      int               `json:"ip_version"`
	NextHop        string            `json:"next_hop"`
	Server         *ServerStub       `json:"server"`
	LoadBalancer   *LoadBalancerStub `json:"load_balancer"`
	Type           string            `json:"type"`
	ReversePointer string            `json:"reverse_ptr,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
}

type FloatingIPCreateRequest struct {
	RegionalResourceRequest
	TaggedResourceRequest
	IPVersion      int    `json:"ip_version"`
	Server         string `json:"server,omitempty"`
	LoadBalancer   string `json:"load_balancer,omitempty"`
	Type           string `json:"type,omitempty"`
	PrefixLength   int    `json:"prefix_length,omitempty"`
	ReversePointer string `json:"reverse_ptr,omitempty"`
}

func (f FloatingIP) IP() string {
	return strings.Split(f.Network, "/")[0]
}

func (f FloatingIP) PrefixLength() int {
	result, _ := strconv.Atoi(strings.Split(f.Network, "/")[1])
	return result
}

type FloatingIPUpdateRequest struct {
	TaggedResourceRequest
	Server         string `json:"server,omitempty"`
	LoadBalancer   string `json:"load_balancer,omitempty"`
	ReversePointer string `json:"reverse_ptr,omitempty"`
}

type FloatingIPsService interface {
	Create(ctx context.Context, floatingIPRequest *FloatingIPCreateRequest) (*FloatingIP, error)
	Get(ctx context.Context, ip string) (*FloatingIP, error)
	Update(ctx context.Context, ip string, FloatingIPRequest *FloatingIPUpdateRequest) error
	Delete(ctx context.Context, ip string) error
	List(ctx context.Context, modifiers ...ListRequestModifier) ([]FloatingIP, error)
}

type FloatingIPsServiceOperations struct {
	client *Client
}

func (f FloatingIPsServiceOperations) Create(ctx context.Context, floatingIPRequest *FloatingIPCreateRequest) (*FloatingIP, error) {
	path := floatingIPsBasePath

	req, err := f.client.NewRequest(ctx, http.MethodPost, path, floatingIPRequest)
	if err != nil {
		return nil, err
	}

	floatingIP := new(FloatingIP)

	err = f.client.Do(ctx, req, floatingIP)
	if err != nil {
		return nil, err
	}

	return floatingIP, nil
}

func (f FloatingIPsServiceOperations) Get(ctx context.Context, ip string) (*FloatingIP, error) {
	path := fmt.Sprintf("%s/%s", floatingIPsBasePath, ip)

	req, err := f.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	floatingIP := new(FloatingIP)
	err = f.client.Do(ctx, req, floatingIP)
	if err != nil {
		return nil, err
	}

	return floatingIP, nil
}

func (f FloatingIPsServiceOperations) Update(ctx context.Context, ip string, floatingIPUpdateRequest *FloatingIPUpdateRequest) error {
	path := fmt.Sprintf("%s/%s", floatingIPsBasePath, ip)

	req, err := f.client.NewRequest(ctx, http.MethodPatch, path, floatingIPUpdateRequest)
	if err != nil {
		return err
	}

	return f.client.Do(ctx, req, nil)
}

func (f FloatingIPsServiceOperations) Delete(ctx context.Context, ip string) error {
	return genericDelete(f.client, ctx, floatingIPsBasePath, ip)
}

func (f FloatingIPsServiceOperations) List(ctx context.Context, modifiers ...ListRequestModifier) ([]FloatingIP, error) {
	path := floatingIPsBasePath

	req, err := f.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	for _, modifier := range modifiers {
		modifier(req)
	}

	floatingIps := []FloatingIP{}
	err = f.client.Do(ctx, req, &floatingIps)
	if err != nil {
		return nil, err
	}

	return floatingIps, nil
}
