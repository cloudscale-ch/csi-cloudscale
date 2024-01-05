package cloudscale

import (
	"context"
	"fmt"
	"net/http"
)

const customImageImportsBasePath = "v1/custom-images/import"

type CustomImageStub struct {
	HREF string `json:"href,omitempty"`
	UUID string `json:"uuid,omitempty"`
	Name string `json:"name,omitempty"`
}

type CustomImageImport struct {
	TaggedResource
	// Just use omitempty everywhere. This makes it easy to use restful. Errors
	// will be coming from the API if something is disabled.
	HREF         string          `json:"href,omitempty"`
	UUID         string          `json:"uuid,omitempty"`
	CustomImage  CustomImageStub `json:"custom_image,omitempty"`
	URL          string          `json:"url,omitempty"`
	Status       string          `json:"status,omitempty"`
	ErrorMessage string          `json:"error_message,omitempty"`
}

type CustomImageImportRequest struct {
	TaggedResourceRequest
	URL              string   `json:"url,omitempty"`
	Name             string   `json:"name,omitempty"`
	Slug             string   `json:"slug,omitempty"`
	UserDataHandling string   `json:"user_data_handling,omitempty"`
	FirmwareType     string   `json:"firmware_type,omitempty"`
	SourceFormat     string   `json:"source_format,omitempty"`
	Zones            []string `json:"zones,omitempty"`
}

type CustomImageImportsService interface {
	Create(ctx context.Context, createRequest *CustomImageImportRequest) (*CustomImageImport, error)
	Get(ctx context.Context, CustomImageImportID string) (*CustomImageImport, error)
	List(ctx context.Context, modifiers ...ListRequestModifier) ([]CustomImageImport, error)
}

type CustomImageImportsServiceOperations struct {
	client *Client
}

func (s CustomImageImportsServiceOperations) Create(ctx context.Context, createRequest *CustomImageImportRequest) (*CustomImageImport, error) {
	path := customImageImportsBasePath

	req, err := s.client.NewRequest(ctx, http.MethodPost, path, createRequest)
	if err != nil {
		return nil, err
	}

	customImageImport := new(CustomImageImport)

	err = s.client.Do(ctx, req, customImageImport)
	if err != nil {
		return nil, err
	}

	return customImageImport, nil
}

func (s CustomImageImportsServiceOperations) Get(ctx context.Context, CustomImageImportID string) (*CustomImageImport, error) {
	path := fmt.Sprintf("%s/%s", customImageImportsBasePath, CustomImageImportID)

	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	customImageImport := new(CustomImageImport)
	err = s.client.Do(ctx, req, customImageImport)
	if err != nil {
		return nil, err
	}

	return customImageImport, nil
}

func (s CustomImageImportsServiceOperations) List(ctx context.Context, modifiers ...ListRequestModifier) ([]CustomImageImport, error) {
	path := customImageImportsBasePath
	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	for _, modifier := range modifiers {
		modifier(req)
	}

	customImageImports := []CustomImageImport{}
	err = s.client.Do(ctx, req, &customImageImports)
	if err != nil {
		return nil, err
	}

	return customImageImports, nil
}
