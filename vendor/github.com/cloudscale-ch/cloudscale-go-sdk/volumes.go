package cloudscale

import (
	"context"
	"fmt"
	"net/http"
)

const volumeBasePath = "v1/volumes"

type Volume struct {
	ZonalResource
	TaggedResource
	// Just use omitempty everywhere. This makes it easy to use restful. Errors
	// will be coming from the API if something is disabled.
	HREF        string    `json:"href,omitempty"`
	UUID        string    `json:"uuid,omitempty"`
	Name        string    `json:"name,omitempty"`
	SizeGB      int       `json:"size_gb,omitempty"`
	Type        string    `json:"type,omitempty"`
	ServerUUIDs *[]string `json:"server_uuids,omitempty"`
}

type VolumeRequest struct {
	ZonalResourceRequest
	TaggedResourceRequest
	Name        string    `json:"name,omitempty"`
	SizeGB      int       `json:"size_gb,omitempty"`
	Type        string    `json:"type,omitempty"`
	ServerUUIDs *[]string `json:"server_uuids,omitempty"`
}

type VolumeService interface {
	Create(ctx context.Context, createRequest *VolumeRequest) (*Volume, error)
	Get(ctx context.Context, volumeID string) (*Volume, error)
	List(ctx context.Context, modifiers ...ListRequestModifier) ([]Volume, error)
	Update(ctx context.Context, volumeID string, updateRequest *VolumeRequest) error
	Delete(ctx context.Context, volumeID string) error
}

type VolumeServiceOperations struct {
	client *Client
}

func (s VolumeServiceOperations) Create(ctx context.Context, createRequest *VolumeRequest) (*Volume, error) {
	path := volumeBasePath

	req, err := s.client.NewRequest(ctx, http.MethodPost, path, createRequest)
	if err != nil {
		return nil, err
	}

	volume := new(Volume)

	err = s.client.Do(ctx, req, volume)
	if err != nil {
		return nil, err
	}

	return volume, nil
}

func (f VolumeServiceOperations) Update(ctx context.Context, volumeID string, updateRequest *VolumeRequest) error {
	path := fmt.Sprintf("%s/%s", volumeBasePath, volumeID)

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

func (s VolumeServiceOperations) Get(ctx context.Context, volumeID string) (*Volume, error) {
	path := fmt.Sprintf("%s/%s", volumeBasePath, volumeID)

	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	volume := new(Volume)
	err = s.client.Do(ctx, req, volume)
	if err != nil {
		return nil, err
	}

	return volume, nil
}

func (s VolumeServiceOperations) Delete(ctx context.Context, volumeID string) error {
	path := fmt.Sprintf("%s/%s", volumeBasePath, volumeID)

	req, err := s.client.NewRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	return s.client.Do(ctx, req, nil)
}

func (s VolumeServiceOperations) List(ctx context.Context, modifiers ...ListRequestModifier) ([]Volume, error) {
	path := volumeBasePath
	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	for _, modifier := range modifiers {
		modifier(req)
	}

	volumes := []Volume{}
	err = s.client.Do(ctx, req, &volumes)
	if err != nil {
		return nil, err
	}

	return volumes, nil
}

//WithNameFilter uses an undocumented feature of the cloudscale.ch API
func WithNameFilter(name string) ListRequestModifier {
	return func(request *http.Request) {
		query := request.URL.Query()
		query.Add(fmt.Sprintf("name"), name)
		request.URL.RawQuery = query.Encode()
	}
}
