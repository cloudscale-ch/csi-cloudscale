package cloudscale

import (
	"time"
)

const loadBalancerHealthMonitorBasePath = "v1/load-balancers/health-monitors"

type LoadBalancerHealthMonitor struct {
	TaggedResource
	// Just use omitempty everywhere. This makes it easy to use restful. Errors
	// will be coming from the API if something is disabled.
	HREF          string                         `json:"href,omitempty"`
	UUID          string                         `json:"uuid,omitempty"`
	Pool          LoadBalancerPoolStub           `json:"pool,omitempty"`
	LoadBalancer  LoadBalancerStub               `json:"load_balancer,omitempty"`
	DelayS        int                            `json:"delay_s,omitempty"`
	TimeoutS      int                            `json:"timeout_s,omitempty"`
	UpThreshold   int                            `json:"up_threshold,omitempty"`
	DownThreshold int                            `json:"down_threshold,omitempty"`
	Type          string                         `json:"type,omitempty"`
	HTTP          *LoadBalancerHealthMonitorHTTP `json:"http,omitempty"`
	CreatedAt     time.Time                      `json:"created_at,omitempty"`
}

type LoadBalancerHealthMonitorHTTP struct {
	ExpectedCodes []string `json:"expected_codes,omitempty"`
	Method        string   `json:"method,omitempty"`
	UrlPath       string   `json:"url_path,omitempty"`
	Version       string   `json:"version,omitempty"`
	Host          *string  `json:"host,omitempty"`
}

type LoadBalancerHealthMonitorRequest struct {
	TaggedResourceRequest
	Pool          string                                `json:"pool,omitempty"`
	DelayS        int                                   `json:"delay_s,omitempty"`
	TimeoutS      int                                   `json:"timeout_s,omitempty"`
	UpThreshold   int                                   `json:"up_threshold,omitempty"`
	DownThreshold int                                   `json:"down_threshold,omitempty"`
	Type          string                                `json:"type,omitempty"`
	HTTP          *LoadBalancerHealthMonitorHTTPRequest `json:"http,omitempty"`
}

type LoadBalancerHealthMonitorHTTPRequest struct {
	ExpectedCodes []string `json:"expected_codes,omitempty"`
	Method        string   `json:"method,omitempty"`
	UrlPath       string   `json:"url_path,omitempty"`
	Version       string   `json:"version,omitempty"`
	Host          *string  `json:"host,omitempty"`
}

type LoadBalancerHealthMonitorService interface {
	GenericCreateService[LoadBalancerHealthMonitor, LoadBalancerHealthMonitorRequest, LoadBalancerHealthMonitorRequest]
	GenericGetService[LoadBalancerHealthMonitor, LoadBalancerHealthMonitorRequest, LoadBalancerHealthMonitorRequest]
	GenericListService[LoadBalancerHealthMonitor, LoadBalancerHealthMonitorRequest, LoadBalancerHealthMonitorRequest]
	GenericUpdateService[LoadBalancerHealthMonitor, LoadBalancerHealthMonitorRequest, LoadBalancerHealthMonitorRequest]
	GenericDeleteService[LoadBalancerHealthMonitor, LoadBalancerHealthMonitorRequest, LoadBalancerHealthMonitorRequest]
}
