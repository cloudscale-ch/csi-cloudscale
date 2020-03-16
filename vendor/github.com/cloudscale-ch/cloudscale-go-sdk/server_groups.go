package cloudscale

import (
	"context"
	"fmt"
	"net/http"
)

const serverGroupsBasePath = "v1/server-groups"

type ServerGroup struct {
	ZonalResource
	TaggedResource
	HREF    string       `json:"href"`
	UUID    string       `json:"uuid"`
	Name    string       `json:"name"`
	Type    string       `json:"type"`
	Servers []ServerStub `json:"servers"`
}

type ServerGroupRequest struct {
	ZonalResourceRequest
	TaggedResourceRequest
	Name    string       `json:"name,omitempty"`
	Type    string       `json:"type,omitempty"`
}

type ServerGroupService interface {
	Create(ctx context.Context, createRequest *ServerGroupRequest) (*ServerGroup, error)
	Get(ctx context.Context, serverGroupID string) (*ServerGroup, error)
	Update(ctx context.Context, networkID string, updateRequest *ServerGroupRequest) error
	Delete(ctx context.Context, serverGroupID string) error
	List(ctx context.Context, modifiers ...ListRequestModifier) ([]ServerGroup, error)
}

type ServerGroupServiceOperations struct {
	client *Client
}

func (s ServerGroupServiceOperations) Create(ctx context.Context, createRequest *ServerGroupRequest) (*ServerGroup, error) {
	req, err := s.client.NewRequest(ctx, http.MethodPost, serverGroupsBasePath, createRequest)
	if err != nil {
		return nil, err
	}

	serverGroup := new(ServerGroup)
	err = s.client.Do(ctx, req, serverGroup)
	if err != nil {
		return nil, err
	}

	return serverGroup, nil
}

func (s ServerGroupServiceOperations) Get(ctx context.Context, serverGroupID string) (*ServerGroup, error) {
	path := fmt.Sprintf("%s/%s", serverGroupsBasePath, serverGroupID)

	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	serverGroup := new(ServerGroup)
	err = s.client.Do(ctx, req, serverGroup)
	if err != nil {
		return nil, err
	}

	return serverGroup, nil
}

func (f ServerGroupServiceOperations) Update(ctx context.Context, serverGroupID string, updateRequest *ServerGroupRequest) error {
	path := fmt.Sprintf("%s/%s", serverGroupsBasePath, serverGroupID)

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

func (s ServerGroupServiceOperations) Delete(ctx context.Context, serverGroupID string) error {
	return genericDelete(s.client, ctx, serverGroupsBasePath, serverGroupID)
}

func (s ServerGroupServiceOperations) List(ctx context.Context, modifiers ...ListRequestModifier) ([]ServerGroup, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, serverGroupsBasePath, nil)
	if err != nil {
		return nil, err
	}

	for _, modifier := range modifiers {
		modifier(req)
	}

	serverGroups := []ServerGroup{}
	err = s.client.Do(ctx, req, &serverGroups)
	if err != nil {
		return nil, err
	}

	return serverGroups, nil
}
