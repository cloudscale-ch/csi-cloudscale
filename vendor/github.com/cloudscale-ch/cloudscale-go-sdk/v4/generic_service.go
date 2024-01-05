package cloudscale

import (
	"context"
	"fmt"
	"net/http"
)

type GenericCreateService[TResource any, TCreateRequest any, TUpdateRequest any] interface {
	Create(ctx context.Context, createRequest *TCreateRequest) (*TResource, error)
}

type GenericGetService[TResource any, TCreateRequest any, TUpdateRequest any] interface {
	Get(ctx context.Context, resourceID string) (*TResource, error)
}

type GenericListService[TResource any, TCreateRequest any, TUpdateRequest any] interface {
	List(ctx context.Context, modifiers ...ListRequestModifier) ([]TResource, error)
}

type GenericUpdateService[TResource any, TCreateRequest any, TUpdateRequest any] interface {
	Update(ctx context.Context, resourceID string, updateRequest *TUpdateRequest) error
}

type GenericDeleteService[TResource any, TCreateRequest any, TUpdateRequest any] interface {
	Delete(ctx context.Context, resourceID string) error
}

type GenericServiceOperations[TResource any, TCreateRequest any, TUpdateRequest any] struct {
	client *Client
	path   string
}

func (g GenericServiceOperations[TResource, TCreateRequest, TUpdateRequest]) Create(ctx context.Context, createRequest *TCreateRequest) (*TResource, error) {
	req, err := g.client.NewRequest(ctx, http.MethodPost, g.path, createRequest)
	if err != nil {
		return nil, err
	}

	resource := new(TResource)

	err = g.client.Do(ctx, req, resource)
	if err != nil {
		return nil, err
	}

	return resource, nil
}

func (g GenericServiceOperations[TResource, TCreateRequest, TUpdateRequest]) Get(ctx context.Context, resourceID string) (*TResource, error) {
	path := fmt.Sprintf("%s/%s", g.path, resourceID)

	req, err := g.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resource := new(TResource)
	err = g.client.Do(ctx, req, resource)
	if err != nil {
		return nil, err
	}

	return resource, nil
}

func (g GenericServiceOperations[TResource, TCreateRequest, TUpdateRequest]) List(ctx context.Context, modifiers ...ListRequestModifier) ([]TResource, error) {
	req, err := g.client.NewRequest(ctx, http.MethodGet, g.path, nil)
	if err != nil {
		return nil, err
	}

	for _, modifier := range modifiers {
		modifier(req)
	}

	resources := []TResource{}
	err = g.client.Do(ctx, req, &resources)
	if err != nil {
		return nil, err
	}

	return resources, nil
}

func (g GenericServiceOperations[TResource, TCreateRequest, TUpdateRequest]) Update(ctx context.Context, resourceID string, updateRequest *TUpdateRequest) error {
	path := fmt.Sprintf("%s/%s", g.path, resourceID)

	req, err := g.client.NewRequest(ctx, http.MethodPatch, path, updateRequest)
	if err != nil {
		return err
	}

	err = g.client.Do(ctx, req, nil)
	if err != nil {
		return err
	}
	return nil
}

func (g GenericServiceOperations[TResource, TCreateRequest, TUpdateRequest]) Delete(ctx context.Context, resourceID string) error {
	path := fmt.Sprintf("%s/%s", g.path, resourceID)

	req, err := g.client.NewRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	return g.client.Do(ctx, req, nil)
}
