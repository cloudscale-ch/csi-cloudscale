package cloudscale

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

const customImagesBasePath = "v1/custom-images"

const UserDataHandlingPassThrough = "pass-through"
const UserDataHandlingExtendCloudConfig = "extend-cloud-config"

type CustomImage struct {
	ZonalResource
	TaggedResource
	// Just use omitempty everywhere. This makes it easy to use restful. Errors
	// will be coming from the API if something is disabled.
	HREF             string            `json:"href,omitempty"`
	UUID             string            `json:"uuid,omitempty"`
	Name             string            `json:"name,omitempty"`
	Slug             string            `json:"slug,omitempty"`
	SizeGB           int               `json:"size_gb,omitempty"`
	Checksums        map[string]string `json:"checksums,omitempty"`
	UserDataHandling string            `json:"user_data_handling,omitempty"`
	FirmwareType     string            `json:"firmware_type,omitempty"`
	Zones            []Zone            `json:"zones"`
	CreatedAt        time.Time         `json:"created_at"`
}

type CustomImageRequest struct {
	TaggedResourceRequest
	Name             string `json:"name,omitempty"`
	Slug             string `json:"slug,omitempty"`
	UserDataHandling string `json:"user_data_handling,omitempty"`
}

type CustomImageService interface {
	Get(ctx context.Context, CustomImageID string) (*CustomImage, error)
	List(ctx context.Context, modifiers ...ListRequestModifier) ([]CustomImage, error)
	Update(ctx context.Context, customImageID string, updateRequest *CustomImageRequest) error
	Delete(ctx context.Context, customImageID string) error
}

type CustomImageServiceOperations struct {
	client *Client
}

func (s CustomImageServiceOperations) Get(ctx context.Context, customImageID string) (*CustomImage, error) {
	path := fmt.Sprintf("%s/%s", customImagesBasePath, customImageID)

	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	customImage := new(CustomImage)
	err = s.client.Do(ctx, req, customImage)
	if err != nil {
		return nil, err
	}

	return customImage, nil
}

func (s CustomImageServiceOperations) List(ctx context.Context, modifiers ...ListRequestModifier) ([]CustomImage, error) {
	path := customImagesBasePath
	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	for _, modifier := range modifiers {
		modifier(req)
	}

	customImages := []CustomImage{}
	err = s.client.Do(ctx, req, &customImages)
	if err != nil {
		return nil, err
	}

	return customImages, nil
}

func (s CustomImageServiceOperations) Update(ctx context.Context, customImageID string, updateRequest *CustomImageRequest) error {
	path := fmt.Sprintf("%s/%s", customImagesBasePath, customImageID)

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

func (s CustomImageServiceOperations) Delete(ctx context.Context, customImageID string) error {
	path := fmt.Sprintf("%s/%s", customImagesBasePath, customImageID)

	req, err := s.client.NewRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	return s.client.Do(ctx, req, nil)
}
