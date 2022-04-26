package cloudscale

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const metricsBasePath = "v1/metrics"

type BucketMetricsRequest struct {
	// Interpreted as midnight in the Europe/Zurich time zone at the start of
	// the day represented by the day of the passed value in the UTC time zone.
	Start time.Time
	// Interpreted as midnight in the Europe/Zurich time zone at the end of
	// the day represented by the day of the passed value in the UTC time zone.
	End            time.Time
	BucketNames    []string
	ObjectsUserIDs []string
}

type BucketMetrics struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
	Data  []BucketMetricsData
}

type BucketMetricsDataSubject struct {
	BucketName    string `json:"name"`
	ObjectsUserID string `json:"objects_user_id"`
}

type BucketMetricsIntervalUsage struct {
	Requests      int `json:"requests"`
	ObjectCount   int `json:"object_count"`
	StorageBytes  int `json:"storage_bytes"`
	ReceivedBytes int `json:"received_bytes"`
	SentBytes     int `json:"sent_bytes"`
}

type BucketMetricsInterval struct {
	Start time.Time                  `json:"start"`
	End   time.Time                  `json:"end"`
	Usage BucketMetricsIntervalUsage `json:"usage"`
}

type BucketMetricsData struct {
	Subject    BucketMetricsDataSubject `json:"subject"`
	TimeSeries []BucketMetricsInterval  `json:"time_series"`
}

type MetricsService interface {
	GetBucketMetrics(ctx context.Context, request *BucketMetricsRequest) (*BucketMetrics, error)
}

type MetricsServiceOperations struct {
	client *Client
}

func encodeGetBucketParameters(request *BucketMetricsRequest) string {
	builder := strings.Builder{}
	separator := "?"

	appendArg := func(name string, value string) {
		builder.WriteString(separator)
		builder.WriteString(name)
		builder.WriteString("=")
		builder.WriteString(url.QueryEscape(value))
		separator = "&"
	}

	appendArg("start", request.Start.Format("2006-01-02"))
	appendArg("end", request.End.Format("2006-01-02"))

	for _, i := range request.BucketNames {
		appendArg("bucket_name", i)
	}

	for _, i := range request.ObjectsUserIDs {
		appendArg("objects_user_id", i)
	}

	return builder.String()
}

func (s MetricsServiceOperations) GetBucketMetrics(ctx context.Context, request *BucketMetricsRequest) (*BucketMetrics, error) {
	path := fmt.Sprintf("%s/buckets%s", metricsBasePath, encodeGetBucketParameters(request))

	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	result := new(BucketMetrics)
	err = s.client.Do(ctx, req, result)
	if err != nil {
		return nil, err
	}

	return result, nil
}
