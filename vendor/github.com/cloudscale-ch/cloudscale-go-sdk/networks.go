package cloudscale

import (
	"context"
	"fmt"
	"net/http"
)

const networkBasePath = "v1/networks"

type Network struct {
	ZonalResource
	TaggedResource
	// Just use omitempty everywhere. This makes it easy to use restful. Errors
	// will be coming from the API if something is disabled.
	HREF    string       `json:"href,omitempty"`
	UUID    string       `json:"uuid,omitempty"`
	Name    string       `json:"name,omitempty"`
	MTU     int          `json:"mtu,omitempty"`
	Subnets []SubnetStub `json:"subnets"`
}

type NetworkStub struct {
	HREF string `json:"href,omitempty"`
	Name string `json:"name,omitempty"`
	UUID string `json:"uuid,omitempty"`
}

type NetworkCreateRequest struct {
	ZonalResourceRequest
	TaggedResourceRequest
	Name                 string `json:"name,omitempty"`
	MTU                  int    `json:"mtu,omitempty"`
	AutoCreateIPV4Subnet *bool  `json:"auto_create_ipv4_subnet,omitempty"`
}

type NetworkUpdateRequest struct {
	ZonalResourceRequest
	TaggedResourceRequest
	Name string `json:"name,omitempty"`
	MTU  int    `json:"mtu,omitempty"`
}

type NetworkService interface {
	Create(ctx context.Context, createRequest *NetworkCreateRequest) (*Network, error)
	Get(ctx context.Context, networkID string) (*Network, error)
	List(ctx context.Context, modifiers ...ListRequestModifier) ([]Network, error)
	Update(ctx context.Context, networkID string, updateRequest *NetworkUpdateRequest) error
	Delete(ctx context.Context, networkID string) error
}

type NetworkServiceOperations struct {
	client *Client
}

func (s NetworkServiceOperations) Create(ctx context.Context, createRequest *NetworkCreateRequest) (*Network, error) {
	path := networkBasePath

	req, err := s.client.NewRequest(ctx, http.MethodPost, path, createRequest)
	if err != nil {
		return nil, err
	}

	network := new(Network)

	err = s.client.Do(ctx, req, network)
	if err != nil {
		return nil, err
	}

	return network, nil
}

func (f NetworkServiceOperations) Update(ctx context.Context, networkID string, updateRequest *NetworkUpdateRequest) error {
	path := fmt.Sprintf("%s/%s", networkBasePath, networkID)

	req, err := f.client.NewRequest(ctx, http.MethodPatch, path, updateRequest)
	if err != nil {
		return err
	}

	err = f.client.Do(ctx, req, nil)
	if err != nil {
		return err
	}
	return nil
}

func (s NetworkServiceOperations) Get(ctx context.Context, networkID string) (*Network, error) {
	path := fmt.Sprintf("%s/%s", networkBasePath, networkID)

	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	network := new(Network)
	err = s.client.Do(ctx, req, network)
	if err != nil {
		return nil, err
	}

	return network, nil
}

func (s NetworkServiceOperations) Delete(ctx context.Context, networkID string) error {
	path := fmt.Sprintf("%s/%s", networkBasePath, networkID)

	req, err := s.client.NewRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	return s.client.Do(ctx, req, nil)
}

func (s NetworkServiceOperations) List(ctx context.Context, modifiers ...ListRequestModifier) ([]Network, error) {
	path := networkBasePath

	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	for _, modifier := range modifiers {
		modifier(req)
	}

	networks := []Network{}
	err = s.client.Do(ctx, req, &networks)
	if err != nil {
		return nil, err
	}

	return networks, nil
}
