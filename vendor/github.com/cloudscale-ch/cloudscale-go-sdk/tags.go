package cloudscale

import (
	"fmt"
	"net/http"
)

type TagMap map[string]string

type TaggedResource struct {
	Tags TagMap `json:"tags"`
}

type TaggedResourceRequest struct {
	Tags TagMap `json:"tags,omitempty"`
}

func WithTagFilter(tags TagMap) ListRequestModifier {
	return func(request *http.Request) {
		query := request.URL.Query()
		for key, value := range tags {
			query.Add(fmt.Sprintf("tag:%s", key), value)
		}
		request.URL.RawQuery = query.Encode()
	}
}
