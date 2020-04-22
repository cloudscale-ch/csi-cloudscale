package cloudscale

import (
	"context"
	"fmt"
	"net/http"
)

func genericDelete(client *Client, ctx context.Context, basePath string, ID string) error {
	path := fmt.Sprintf("%s/%s", basePath, ID)

	req, err := client.NewRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	return client.Do(ctx, req, nil)
}
