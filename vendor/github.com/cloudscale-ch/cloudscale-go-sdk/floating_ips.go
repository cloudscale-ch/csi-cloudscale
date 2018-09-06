package cloudscale

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

const floatingIPsBasePath = "v1/floating-ips"

type FloatingIP struct {
	HREF           string     `json:"href"`
	Network        string     `json:"network"`
	NextHop        string     `json:"next_hop"`
	Server         ServerStub `json:"server"`
	ReversePointer string     `json:"reverse_ptr,omitempty"`
}

type FloatingIPCreateRequest struct {
	IPVersion      int    `json:"ip_version"`
	Server         string `json:"server"`
	PrefixLength   int    `json:"prefix_length,omitempty"`
	ReversePointer string `json:"reverse_ptr,omitempty"`
}

func (f FloatingIP) IP() string {
	return strings.Split(f.Network, "/")[0]
}

type FloatingIPUpdateRequest struct {
	Server string `json:"server"`
}

type FloatingIPsService interface {
	Create(ctx context.Context, floatingIPRequest *FloatingIPCreateRequest) (*FloatingIP, error)
	Get(ctx context.Context, ip string) (*FloatingIP, error)
	Update(ctx context.Context, ip string, FloatingIPRequest *FloatingIPUpdateRequest) (*FloatingIP, error)
	Delete(ctx context.Context, ip string) error
	List(ctx context.Context) ([]FloatingIP, error)
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
func (f FloatingIPsServiceOperations) Update(ctx context.Context, ip string, floatingIPUpdateRequest *FloatingIPUpdateRequest) (*FloatingIP, error) {
	path := fmt.Sprintf("%s/%s", floatingIPsBasePath, ip)

	req, err := f.client.NewRequest(ctx, http.MethodPost, path, floatingIPUpdateRequest)
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
func (f FloatingIPsServiceOperations) Delete(ctx context.Context, ip string) error {
	path := fmt.Sprintf("%s/%s", floatingIPsBasePath, ip)

	req, err := f.client.NewRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	return f.client.Do(ctx, req, nil)
}
func (f FloatingIPsServiceOperations) List(ctx context.Context) ([]FloatingIP, error) {
	path := floatingIPsBasePath

	req, err := f.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	floatingIps := []FloatingIP{}
	err = f.client.Do(ctx, req, &floatingIps)
	if err != nil {
		return nil, err
	}

	return floatingIps, nil
}
