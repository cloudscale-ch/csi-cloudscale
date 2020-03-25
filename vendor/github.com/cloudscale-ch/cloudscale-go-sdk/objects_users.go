package cloudscale

import (
	"context"
	"fmt"
	"net/http"
)

const objectsUsersBasePath = "v1/objects-users"

// ObjectsUser contains information
type ObjectsUser struct {
	TaggedResource
	HREF        string              `json:"href,omitempty"`
	ID          string              `json:"id,omitempty"`
	DisplayName string              `json:"display_name,omitempty"`
	Keys        []map[string]string `json:"keys,omitempty"`
}

// ObjectsUserRequest is used to create and update Objects Users
type ObjectsUserRequest struct {
	TaggedResourceRequest
	DisplayName string `json:"display_name,omitempty"`
}

// ObjectsUsersService manages users of the S3-compatible objects storage
type ObjectsUsersService interface {
	Create(ctx context.Context, createRequest *ObjectsUserRequest) (*ObjectsUser, error)
	Get(ctx context.Context, objectsUserID string) (*ObjectsUser, error)
	Update(ctx context.Context, objectsUserID string, updateRequest *ObjectsUserRequest) error
	Delete(ctx context.Context, objectsUserID string) error
	List(ctx context.Context, modifiers ...ListRequestModifier) ([]ObjectsUser, error)
}

// ObjectsUsersServiceOperations contains config for this service
type ObjectsUsersServiceOperations struct {
	client *Client
}

// Create an objects user with the specified attributes.
func (s ObjectsUsersServiceOperations) Create(ctx context.Context, createRequest *ObjectsUserRequest) (*ObjectsUser, error) {
	path := objectsUsersBasePath

	req, err := s.client.NewRequest(ctx, http.MethodPost, path, createRequest)
	if err != nil {
		return nil, err
	}

	objectsUser := new(ObjectsUser)
	err = s.client.Do(ctx, req, objectsUser)
	if err != nil {
		return nil, err
	}

	return objectsUser, nil
}

// Update the properties of an objects user
func (s ObjectsUsersServiceOperations) Update(ctx context.Context, objectsUserID string, updateRequest *ObjectsUserRequest) error {
	path := fmt.Sprintf("%s/%s", objectsUsersBasePath, objectsUserID)

	req, err := s.client.NewRequest(ctx, http.MethodPatch, path, updateRequest)
	if err != nil {
		return err
	}

	err = s.client.Do(ctx, req, nil)
	if err != nil {
		return err
	}
	return nil
}

// Get an objects user by its ID
func (s ObjectsUsersServiceOperations) Get(ctx context.Context, objectsUserID string) (*ObjectsUser, error) {
	path := fmt.Sprintf("%s/%s", objectsUsersBasePath, objectsUserID)

	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	objectsUser := new(ObjectsUser)
	err = s.client.Do(ctx, req, objectsUser)
	if err != nil {
		return nil, err
	}

	return objectsUser, nil
}

// Delete an objects user
func (s ObjectsUsersServiceOperations) Delete(ctx context.Context, objectsUserID string) error {
	return genericDelete(s.client, ctx, objectsUsersBasePath, objectsUserID)
}

// List all objects users
func (s ObjectsUsersServiceOperations) List(ctx context.Context, modifiers ...ListRequestModifier) ([]ObjectsUser, error) {
	path := objectsUsersBasePath

	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	for _, modifier := range modifiers {
		modifier(req)
	}

	ObjectsUser := []ObjectsUser{}
	err = s.client.Do(ctx, req, &ObjectsUser)
	if err != nil {
		return nil, err
	}

	return ObjectsUser, nil
}
