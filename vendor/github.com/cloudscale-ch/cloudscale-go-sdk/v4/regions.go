package cloudscale

import (
	"context"
	"net/http"
)

const regionsBasePath = "v1/regions"

type Zone struct {
	Slug string `json:"slug"`
}

type Region struct {
	Slug  string `json:"slug"`
	Zones []Zone `json:"zones"`
}

type RegionService interface {
	List(ctx context.Context) ([]Region, error)
}

type RegionServiceOperations struct {
	client *Client
}

func (s RegionServiceOperations) List(ctx context.Context) ([]Region, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, regionsBasePath, nil)
	if err != nil {
		return nil, err
	}
	regions := []Region{}
	err = s.client.Do(ctx, req, &regions)
	if err != nil {
		return nil, err
	}

	return regions, nil
}
