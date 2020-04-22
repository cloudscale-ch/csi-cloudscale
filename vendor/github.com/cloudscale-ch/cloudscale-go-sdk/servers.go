package cloudscale

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
)

const serverBasePath = "v1/servers"

const ServerRunning = "running"
const ServerStopped = "stopped"
const ServerRebooted = "rebooted"

type Server struct {
	ZonalResource
	TaggedResource
	HREF            string            `json:"href"`
	UUID            string            `json:"uuid"`
	Name            string            `json:"name"`
	Status          string            `json:"status"`
	Flavor          Flavor            `json:"flavor"`
	Image           Image             `json:"image"`
	Volumes         []VolumeStub      `json:"volumes"`
	Interfaces      []Interface       `json:"interfaces"`
	SSHFingerprints []string          `json:"ssh_fingerprints"`
	SSHHostKeys     []string          `json:"ssh_host_keys"`
	AntiAfinityWith []ServerStub      `json:"anti_affinity_with"`
	ServerGroups    []ServerGroupStub `json:"server_groups"`
}

type ServerStub struct {
	HREF string `json:"href"`
	UUID string `json:"uuid"`
}

type ServerGroupStub struct {
	HREF string `json:"href"`
	UUID string `json:"uuid"`
	Name string `json:"name"`
}

type Flavor struct {
	Slug      string `json:"slug"`
	Name      string `json:"name"`
	VCPUCount int    `json:"vcpu_count"`
	MemoryGB  int    `json:"memory_gb"`
}

type Image struct {
	Slug            string `json:"slug"`
	Name            string `json:"name"`
	OperatingSystem string `json:"operating_system"`
}

type VolumeStub struct {
	Type       string `json:"type"`
	DevicePath string `json:"device_path"`
	SizeGB     int    `json:"size_gb"`
	UUID       string `json:"uuid"`
}

type Interface struct {
	Type      string      `json:"type,omitempty"`
	Network   NetworkStub `json:"network,omitempty"`
	Addresses []Address   `json:"addresses,omitempty"`
}

type Address struct {
	Version      int    `json:"version"`
	Address      string `json:"address"`
	PrefixLength int    `json:"prefix_length"`
	Gateway      string `json:"gateway"`
	ReversePtr   string `json:"reverse_ptr"`
}

type ServerRequest struct {
	ZonalResourceRequest
	TaggedResourceRequest
	Name              string              `json:"name"`
	Flavor            string              `json:"flavor"`
	Image             string              `json:"image"`
	Zone              string              `json:"zone,omitempty"`
	VolumeSizeGB      int                 `json:"volume_size_gb,omitempty"`
	Volumes           *[]Volume           `json:"volumes,omitempty"`
	Interfaces        *[]InterfaceRequest `json:"interfaces,omitempty"`
	BulkVolumeSizeGB  int                 `json:"bulk_volume_size_gb,omitempty"`
	SSHKeys           []string            `json:"ssh_keys"`
	Password          string              `json:"password,omitempty"`
	UsePublicNetwork  *bool               `json:"use_public_network,omitempty"`
	UsePrivateNetwork *bool               `json:"use_private_network,omitempty"`
	UseIPV6           *bool               `json:"use_ipv6,omitempty"`
	AntiAffinityWith  string              `json:"anti_affinity_with,omitempty"`
	ServerGroups      []string            `json:"server_groups,omitempty"`
	UserData          string              `json:"user_data,omitempty"`
}

type InterfaceRequest struct {
	Network   string   `json:"network,omitempty"`
	Addresses *[]string `json:"addresses,omitempty"`
}

type ServerService interface {
	Create(ctx context.Context, createRequest *ServerRequest) (*Server, error)
	Get(ctx context.Context, serverID string) (*Server, error)
	Update(ctx context.Context, serverID string, updateRequest *ServerUpdateRequest) error
	Delete(ctx context.Context, serverID string) error
	List(ctx context.Context, modifiers ...ListRequestModifier) ([]Server, error)
	Reboot(ctx context.Context, serverID string) error
	Start(ctx context.Context, serverID string) error
	Stop(ctx context.Context, serverID string) error
}

type ServerServiceOperations struct {
	client *Client
}

func (s ServerServiceOperations) Create(ctx context.Context, createRequest *ServerRequest) (*Server, error) {
	path := serverBasePath

	req, err := s.client.NewRequest(ctx, http.MethodPost, path, createRequest)
	if err != nil {
		return nil, err
	}

	server := new(Server)

	err = s.client.Do(ctx, req, server)
	if err != nil {
		return nil, err
	}

	return server, nil
}

type ServerUpdateRequest struct {
	TaggedResourceRequest
	Name       string              `json:"name,omitempty"`
	Status     string              `json:"status,omitempty"`
	Flavor     string              `json:"flavor,omitempty"`
	Interfaces *[]InterfaceRequest `json:"interfaces,omitempty"`
}

func (s ServerServiceOperations) Update(ctx context.Context, serverID string, updateRequest *ServerUpdateRequest) error {
	if updateRequest.Status != "" {
		err := error(nil)
		switch updateRequest.Status {
		case ServerRunning:
			err = s.Start(ctx, serverID)
		case ServerStopped:
			err = s.Stop(ctx, serverID)
		case ServerRebooted:
			err = s.Reboot(ctx, serverID)
		default:
			return fmt.Errorf("Status Not Supported %s", updateRequest.Status)
		}
		if err != nil {
			return err
		}
		// Get rid of status
		updateRequest.Status = ""
	}

	emptyRequest := ServerUpdateRequest{}
	if !reflect.DeepEqual(emptyRequest, *updateRequest) {
		path := fmt.Sprintf("%s/%s", serverBasePath, serverID)

		req, err := s.client.NewRequest(ctx, http.MethodPatch, path, updateRequest)
		if err != nil {
			return err
		}

		err = s.client.Do(ctx, req, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s ServerServiceOperations) Get(ctx context.Context, serverID string) (*Server, error) {
	path := fmt.Sprintf("%s/%s", serverBasePath, serverID)

	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	server := new(Server)
	err = s.client.Do(ctx, req, server)
	if err != nil {
		return nil, err
	}

	return server, nil
}

func (s ServerServiceOperations) Delete(ctx context.Context, serverID string) error {
	return genericDelete(s.client, ctx, serverBasePath, serverID)
}

func (s ServerServiceOperations) Reboot(ctx context.Context, serverID string) error {
	path := fmt.Sprintf("%s/%s/reboot", serverBasePath, serverID)
	req, err := s.client.NewRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return err
	}
	return s.client.Do(ctx, req, nil)
}

func (s ServerServiceOperations) Start(ctx context.Context, serverID string) error {
	path := fmt.Sprintf("%s/%s/start", serverBasePath, serverID)
	req, err := s.client.NewRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return err
	}
	return s.client.Do(ctx, req, nil)
}

func (s ServerServiceOperations) Stop(ctx context.Context, serverID string) error {
	path := fmt.Sprintf("%s/%s/stop", serverBasePath, serverID)
	req, err := s.client.NewRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return err
	}
	return s.client.Do(ctx, req, nil)
}

func (s ServerServiceOperations) List(ctx context.Context, modifiers ...ListRequestModifier) ([]Server, error) {
	path := serverBasePath

	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	for _, modifier := range modifiers {
		modifier(req)
	}

	servers := []Server{}
	err = s.client.Do(ctx, req, &servers)
	if err != nil {
		return nil, err
	}

	return servers, nil
}
