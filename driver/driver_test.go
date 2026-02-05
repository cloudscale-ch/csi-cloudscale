/*
Copyright cloudscale.ch
Copyright 2018 DigitalOcean

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package driver

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v5"
	"github.com/cloudscale-ch/cloudscale-go-sdk/v6"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/google/uuid"
	"github.com/kubernetes-csi/csi-test/v5/pkg/sanity"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/mount-utils"
)

const (
	numDroplets = 100
)

type idGenerator struct{}

var DefaultZone = cloudscale.Zone{Slug: "dev1"}

func TestDriverSuite(t *testing.T) {
	socket := "/tmp/csi.sock"
	endpoint := "unix://" + socket
	if err := os.Remove(socket); err != nil && !os.IsNotExist(err) {
		t.Fatalf("failed to remove unix domain socket file %s, error: %s", socket, err)
	}

	serverId := "987654"
	initialServers := map[string]*cloudscale.Server{
		serverId: {UUID: serverId},
	}
	cloudscaleClient := NewFakeClient(initialServers)
	fm := &fakeMounter{
		mounted: map[string]string{},
	}
	driver := &Driver{
		endpoint:         endpoint,
		serverId:         serverId,
		zone:             DefaultZone.Slug,
		cloudscaleClient: cloudscaleClient,
		mounter:          fm,
		log:              logrus.New().WithField("test_enabed", true),
		volumeLocks:      NewVolumeLocks(),
	}
	defer driver.Stop()

	go func() {
		if err := driver.Run(); err != nil {
			panic(err)
		}
	}()

	cfg := sanity.NewTestConfig()
	if err := os.RemoveAll(cfg.TargetPath); err != nil {
		t.Fatalf("failed to delete target path %s: %s", cfg.TargetPath, err)
	}
	if err := os.RemoveAll(cfg.StagingPath); err != nil {
		t.Fatalf("failed to delete staging path %s: %s", cfg.StagingPath, err)
	}
	cfg.Address = endpoint
	cfg.IDGen = &idGenerator{}
	cfg.IdempotentCount = 5
	cfg.TestNodeVolumeAttachLimit = true
	cfg.CheckPath = fm.checkMountPath
	cfg.TestNodeVolumeAttachLimit = true

	sanity.Test(t, cfg)
}

func NewFakeClient(initialServers map[string]*cloudscale.Server) *cloudscale.Client {
	userAgent := "cloudscale/" + "fake"
	fakeClient := &cloudscale.Client{BaseURL: nil, UserAgent: userAgent}

	fakeClient.Servers = &FakeServerServiceOperations{
		fakeClient: fakeClient,
		servers:    initialServers,
	}
	fakeClient.Volumes = &FakeVolumeServiceOperations{
		fakeClient: fakeClient,
		volumes:    make(map[string]*cloudscale.Volume),
	}

	fakeClient.VolumeSnapshots = &FakeVolumeSnapshotServiceOperations{
		fakeClient: fakeClient,
		snapshots:  make(map[string]*cloudscale.VolumeSnapshot),
	}

	return fakeClient
}

type fakeMounter struct {
	mounted map[string]string
	mu      sync.RWMutex
}

func (f *fakeMounter) Format(source string, fsType string, luksContext LuksContext) error {
	return nil
}

func (f *fakeMounter) Mount(source string, target string, fsType string, luksContext LuksContext, options ...string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.mounted[target] = source
	return nil
}

func (f *fakeMounter) Unmount(target string, luksContext LuksContext) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.mounted, target)
	return nil
}

func (f *fakeMounter) GetDeviceName(_ mount.Interface, mountPath string) (string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if _, ok := f.mounted[mountPath]; ok {
		return "/mnt/sda1", nil
	}

	return "", nil
}

func (f *fakeMounter) FindAbsoluteDeviceByIDPath(volumeName string) (string, error) {
	return "/dev/sdb", nil
}

func (f *fakeMounter) IsFormatted(source string, luksContext LuksContext) (bool, error) {
	return true, nil
}
func (f *fakeMounter) IsMounted(target string) (bool, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	_, ok := f.mounted[target]
	return ok, nil
}

func (f *fakeMounter) checkMountPath(path string) (sanity.PathKind, error) {
	isMounted, err := f.IsMounted(path)
	if err != nil {
		return "", err
	}
	if isMounted {
		return sanity.PathIsDir, nil
	}
	return sanity.PathIsNotFound, nil
}

func (f *fakeMounter) GetStatistics(volumePath string) (volumeStatistics, error) {
	return volumeStatistics{
		availableBytes: 3 * GB,
		totalBytes:     10 * GB,
		usedBytes:      7 * GB,

		availableInodes: 3000,
		totalInodes:     10000,
		usedInodes:      7000,
	}, nil
}

func (f *fakeMounter) HasRequiredSize(log *logrus.Entry, path string, requiredSize int64) (bool, error) {
	return true, nil
}

func (f *fakeMounter) FinalizeVolumeAttachmentAndFindPath(logger *logrus.Entry, target string) (string, error) {
	path := "SomePath"
	return path, nil
}

type FakeVolumeServiceOperations struct {
	fakeClient *cloudscale.Client
	volumes    map[string]*cloudscale.Volume
}

func (f *FakeVolumeServiceOperations) Create(ctx context.Context, createRequest *cloudscale.VolumeCreateRequest) (*cloudscale.Volume, error) {
	id := randString(32)

	// todo: CSI-test pass without this, but we could implement:
	// - check if volumeSnapshot is present. Return error if volumeSnapshot does not exist
	// - create volume with inferred values form snapshot.

	vol := &cloudscale.Volume{
		UUID:        id,
		Name:        createRequest.Name,
		SizeGB:      createRequest.SizeGB,
		Type:        createRequest.Type,
		ServerUUIDs: createRequest.ServerUUIDs,
	}
	vol.Zone = DefaultZone
	if vol.ServerUUIDs == nil {
		noservers := make([]string, 0, 1)
		vol.ServerUUIDs = &noservers
	}

	f.volumes[id] = vol

	return vol, nil
}

func (f *FakeVolumeServiceOperations) Get(ctx context.Context, volumeID string) (*cloudscale.Volume, error) {
	vol, ok := f.volumes[volumeID]
	if ok != true {
		return nil, generateNotFoundError()
	}
	return vol, nil
}

func (f *FakeVolumeServiceOperations) List(ctx context.Context, modifiers ...cloudscale.ListRequestModifier) ([]cloudscale.Volume, error) {
	var volumes []cloudscale.Volume

	for _, vol := range f.volumes {
		volumes = append(volumes, *vol)
	}

	if len(modifiers) == 0 {
		return volumes, nil
	}
	if len(modifiers) > 1 {
		panic("implement me (support for more than one modifier)")
	}

	params := extractParams(modifiers)

	if filterName := params.Get("name"); filterName != "" {
		filtered := make([]cloudscale.Volume, 0, 1)
		for _, vol := range volumes {
			if vol.Name == filterName {
				filtered = append(filtered, vol)
			}
		}
		return filtered, nil
	}

	panic("implement me (support for unknown param)")
}

func extractParams(modifiers []cloudscale.ListRequestModifier) url.Values {
	// undoing the cloudscale.WithNameFilter(volumeName) magic

	modifierFunc := modifiers[0]
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	modifierFunc(req)
	params, err := url.ParseQuery(req.URL.RawQuery)
	if err != nil {
		panic("unexpected error")
	}
	return params
}

func (f *FakeVolumeServiceOperations) Update(ctx context.Context, volumeID string, updateRequest *cloudscale.VolumeUpdateRequest) error {
	vol, ok := f.volumes[volumeID]
	if ok != true {
		return generateNotFoundError()
	}

	if updateRequest.ServerUUIDs != nil {
		if serverUUIDs := *updateRequest.ServerUUIDs; serverUUIDs != nil {
			if len(serverUUIDs) > 1 {
				return errors.New("multi attach is not implemented")
			}
			if len(serverUUIDs) == 1 {
				for _, serverUUID := range serverUUIDs {
					_, err := f.fakeClient.Servers.Get(ctx, serverUUID)
					if err != nil {
						return err
					}

					volumesCount := getVolumesPerServer(f, serverUUID)
					if volumesCount >= fallbackMaxVolumesPerNode {
						return &cloudscale.ErrorResponse{
							StatusCode: 400,
							Message:    map[string]string{"detail": "Due to internal limitations, it is currently not possible to attach more than 128 volumes"},
						}
					}
				}
			}

			vol.ServerUUIDs = &serverUUIDs
			return nil
		}
	}
	if vol.SizeGB < updateRequest.SizeGB {
		vol.SizeGB = updateRequest.SizeGB
		return nil
	}
	panic("implement me")
}

func getVolumesPerServer(f *FakeVolumeServiceOperations, serverUUID string) int {
	volumesCount := 0
	for _, v := range f.volumes {
		for _, uuid := range *v.ServerUUIDs {
			if uuid == serverUUID {
				volumesCount++
			}
		}
	}
	return volumesCount
}

func (f *FakeVolumeServiceOperations) Delete(ctx context.Context, volumeID string) error {

	// prevent deletion if snapshots exist
	snapshots, err := f.fakeClient.VolumeSnapshots.List(context.Background())

	if err != nil {
		return err
	}

	for _, snapshot := range snapshots {
		if snapshot.Volume.UUID == volumeID {
			return &cloudscale.ErrorResponse{
				StatusCode: 409,
				Message:    map[string]string{"detail": "volume has snapshots"},
			}
		}
	}
	delete(f.volumes, volumeID)
	return nil
}

type FakeServerServiceOperations struct {
	fakeClient *cloudscale.Client
	servers    map[string]*cloudscale.Server
}

func (f *fakeMounter) IsBlockDevice(volumePath string) (bool, error) {
	return false, nil
}

func (f *FakeServerServiceOperations) Create(ctx context.Context, createRequest *cloudscale.ServerRequest) (*cloudscale.Server, error) {
	panic("implement me")
}

func (f *FakeServerServiceOperations) Get(ctx context.Context, serverID string) (*cloudscale.Server, error) {
	server, ok := f.servers[serverID]
	if ok != true {
		return nil, generateNotFoundError()
	}
	return server, nil
}

func (f *FakeServerServiceOperations) Update(ctx context.Context, serverID string, updateRequest *cloudscale.ServerUpdateRequest) error {
	panic("implement me")
}

func (f *FakeServerServiceOperations) Delete(ctx context.Context, serverID string) error {
	panic("implement me")
}

func (f *FakeServerServiceOperations) List(ctx context.Context, modifiers ...cloudscale.ListRequestModifier) ([]cloudscale.Server, error) {
	panic("implement me")
}

func (f *FakeServerServiceOperations) Reboot(ctx context.Context, serverID string) error {
	panic("implement me")
}

func (f *FakeServerServiceOperations) Start(ctx context.Context, serverID string) error {
	panic("implement me")
}

func (f *FakeServerServiceOperations) Stop(ctx context.Context, serverID string) error {
	panic("implement me")
}

func (f *FakeServerServiceOperations) WaitFor(ctx context.Context, id string, condition func(*cloudscale.Server) (bool, error), opts ...backoff.RetryOption) (*cloudscale.Server, error) {
	panic("implement me")
}

func (f *FakeVolumeServiceOperations) WaitFor(ctx context.Context, id string, condition func(*cloudscale.Volume) (bool, error), opts ...backoff.RetryOption) (*cloudscale.Volume, error) {
	panic("implement me")
}

type FakeVolumeSnapshotServiceOperations struct {
	fakeClient *cloudscale.Client
	snapshots  map[string]*cloudscale.VolumeSnapshot
}

func (f FakeVolumeSnapshotServiceOperations) Create(ctx context.Context, createRequest *cloudscale.VolumeSnapshotCreateRequest) (*cloudscale.VolumeSnapshot, error) {

	vol, err := f.fakeClient.Volumes.Get(ctx, createRequest.SourceVolume)
	if err != nil {
		return nil, err
	}

	id := randString(32)
	snap := &cloudscale.VolumeSnapshot{
		UUID:      id,
		Name:      createRequest.Name,
		SizeGB:    vol.SizeGB,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Status:    "available",
		Volume: cloudscale.VolumeStub{
			UUID: createRequest.SourceVolume,
		},
	}

	f.snapshots[id] = snap
	return snap, nil
}

func (f *FakeVolumeSnapshotServiceOperations) Get(
	ctx context.Context,
	snapshotID string,
) (*cloudscale.VolumeSnapshot, error) {

	snap, ok := f.snapshots[snapshotID]
	if !ok {
		return nil, generateNotFoundError()
	}
	return snap, nil
}

func (f *FakeVolumeSnapshotServiceOperations) List(
	ctx context.Context,
	modifiers ...cloudscale.ListRequestModifier,
) ([]cloudscale.VolumeSnapshot, error) {
	var snapshots []cloudscale.VolumeSnapshot

	for _, snapshot := range f.snapshots {
		snapshots = append(snapshots, *snapshot)
	}

	if len(modifiers) == 0 {
		return snapshots, nil
	}
	if len(modifiers) > 1 {
		panic("implement me (support for more than one modifier)")
	}

	params := extractParams(modifiers)

	if filterName := params.Get("name"); filterName != "" {
		filtered := make([]cloudscale.VolumeSnapshot, 0, 1)
		for _, snapshot := range snapshots {
			if snapshot.Name == filterName {
				filtered = append(filtered, snapshot)
			}
		}
		return filtered, nil
	}

	panic("implement me (support for unknown param)")
}

func (f FakeVolumeSnapshotServiceOperations) Update(ctx context.Context, resourceID string, updateRequest *cloudscale.VolumeSnapshotUpdateRequest) error {
	panic("implement me")
}

func (f *FakeVolumeSnapshotServiceOperations) Delete(
	ctx context.Context,
	snapshotID string,
) error {
	delete(f.snapshots, snapshotID)
	return nil
}
func (f FakeVolumeSnapshotServiceOperations) WaitFor(ctx context.Context, resourceID string, condition func(resource *cloudscale.VolumeSnapshot) (bool, error), opts ...backoff.RetryOption) (*cloudscale.VolumeSnapshot, error) {
	panic("implement me")
}

func generateNotFoundError() *cloudscale.ErrorResponse {
	return &cloudscale.ErrorResponse{
		StatusCode: 404,
		Message:    map[string]string{"detail": "not found"},
	}
}

func randString(n int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func (g *idGenerator) GenerateUniqueValidVolumeID() string {
	return uuid.New().String()
}

func (g *idGenerator) GenerateInvalidVolumeID() string {
	return g.GenerateUniqueValidVolumeID()
}

func (g *idGenerator) GenerateUniqueValidNodeID() string {
	return strconv.Itoa(numDroplets * 10)
}

func (g *idGenerator) GenerateInvalidNodeID() string {
	return "not-an-integer"
}

// FakeBlockingMounter wraps fakeMounter and adds blocking capability for concurrency testing.
// It blocks at FinalizeVolumeAttachmentAndFindPath, which is called early in NodeStageVolume
// after acquiring the volume lock.
type FakeBlockingMounter struct {
	*fakeMounter
	ReadyToExecute chan chan struct{}
}

// FinalizeVolumeAttachmentAndFindPath blocks until signaled, allowing tests to control
// the order of execution for concurrency testing.
func (f *FakeBlockingMounter) FinalizeVolumeAttachmentAndFindPath(logger *logrus.Entry, volumeID string) (string, error) {
	executeOp := make(chan struct{})
	f.ReadyToExecute <- executeOp
	<-executeOp
	return f.fakeMounter.FinalizeVolumeAttachmentAndFindPath(logger, volumeID)
}

// NewFakeBlockingMounter creates a new FakeBlockingMounter with the given channel.
func NewFakeBlockingMounter(readyToExecute chan chan struct{}) *FakeBlockingMounter {
	return &FakeBlockingMounter{
		fakeMounter: &fakeMounter{
			mounted: map[string]string{},
		},
		ReadyToExecute: readyToExecute,
	}
}

// initBlockingDriver creates a Driver with a FakeBlockingMounter for concurrency testing.
func initBlockingDriver(t *testing.T, readyToExecute chan chan struct{}) *Driver {
	serverId := "987654"
	initialServers := map[string]*cloudscale.Server{
		serverId: {UUID: serverId},
	}
	cloudscaleClient := NewFakeClient(initialServers)

	return &Driver{
		endpoint:         "unix:///tmp/csi-test.sock",
		serverId:         serverId,
		zone:             DefaultZone.Slug,
		cloudscaleClient: cloudscaleClient,
		mounter:          NewFakeBlockingMounter(readyToExecute),
		log:              logrus.New().WithField("test_enabled", true),
		volumeLocks:      NewVolumeLocks(),
	}
}

// TestNodeStageVolume_ConcurrentSameVolume tests that concurrent NodeStageVolume
// operations on the same volume are properly serialized with volume locks.
// The second operation should return codes.Aborted while the first is in progress.
func TestNodeStageVolume_ConcurrentSameVolume(t *testing.T) {
	readyToExecute := make(chan chan struct{}, 1)
	driver := initBlockingDriver(t, readyToExecute)

	// Create the volume in the fake client first
	ctx := t.Context()
	vol, err := driver.cloudscaleClient.Volumes.Create(ctx, &cloudscale.VolumeCreateRequest{
		Name:   "test-volume",
		SizeGB: 10,
		Type:   "ssd",
	})
	if err != nil {
		t.Fatalf("Failed to create volume: %v", err)
	}
	volumeID := vol.UUID

	req := &csi.NodeStageVolumeRequest{
		VolumeId:          volumeID,
		StagingTargetPath: "/mnt/staging",
		VolumeCapability: &csi.VolumeCapability{
			AccessMode: &csi.VolumeCapability_AccessMode{
				Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
			},
			AccessType: &csi.VolumeCapability_Mount{
				Mount: &csi.VolumeCapability_MountVolume{
					FsType: "ext4",
				},
			},
		},
		PublishContext: map[string]string{
			LuksEncryptedAttribute: "false",
			PublishInfoVolumeName:  "test-volume",
		},
	}

	runNodeStage := func(req *csi.NodeStageVolumeRequest) <-chan error {
		response := make(chan error, 1)
		go func() {
			_, err := driver.NodeStageVolume(context.Background(), req)
			response <- err
		}()
		return response
	}

	// Start first NodeStageVolume and block until it reaches FinalizeVolumeAttachmentAndFindPath
	respA := runNodeStage(req)
	execA := <-readyToExecute

	// Start second NodeStageVolume on the same volume - should get Aborted immediately
	respB := runNodeStage(req)
	select {
	case err := <-respB:
		if err == nil {
			t.Errorf("Expected error for concurrent operation on same volume, got nil")
		} else {
			serverError, ok := status.FromError(err)
			if !ok {
				t.Fatalf("Could not get error status code from err: %v", err)
			}
			if serverError.Code() != codes.Aborted {
				t.Errorf("Expected error code: %v, got: %v. err: %v", codes.Aborted, serverError.Code(), err)
			}
		}
	case <-readyToExecute:
		t.Errorf("The operation for second NodeStageVolume should have been aborted, but was started")
	case <-time.After(time.Second):
		t.Errorf("Timeout waiting for second operation to return Aborted")
	}

	// Clean up: allow first operation to complete
	execA <- struct{}{}
	<-respA
}

// TestNodeStageVolume_ConcurrentDifferentVolumes tests that concurrent NodeStageVolume
// operations on different volumes can proceed in parallel without blocking each other.
func TestNodeStageVolume_ConcurrentDifferentVolumes(t *testing.T) {
	readyToExecute := make(chan chan struct{}, 2)
	driver := initBlockingDriver(t, readyToExecute)

	ctx := t.Context()

	// Create two different volumes
	vol1, err := driver.cloudscaleClient.Volumes.Create(ctx, &cloudscale.VolumeCreateRequest{
		Name:   "test-volume-1",
		SizeGB: 10,
		Type:   "ssd",
	})
	if err != nil {
		t.Fatalf("Failed to create volume 1: %v", err)
	}

	vol2, err := driver.cloudscaleClient.Volumes.Create(ctx, &cloudscale.VolumeCreateRequest{
		Name:   "test-volume-2",
		SizeGB: 10,
		Type:   "ssd",
	})
	if err != nil {
		t.Fatalf("Failed to create volume 2: %v", err)
	}

	makeReq := func(volumeID, stagingPath, volumeName string) *csi.NodeStageVolumeRequest {
		return &csi.NodeStageVolumeRequest{
			VolumeId:          volumeID,
			StagingTargetPath: stagingPath,
			VolumeCapability: &csi.VolumeCapability{
				AccessMode: &csi.VolumeCapability_AccessMode{
					Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
				},
				AccessType: &csi.VolumeCapability_Mount{
					Mount: &csi.VolumeCapability_MountVolume{
						FsType: "ext4",
					},
				},
			},
			PublishContext: map[string]string{
				LuksEncryptedAttribute: "false",
				PublishInfoVolumeName:  volumeName,
			},
		}
	}

	runNodeStage := func(req *csi.NodeStageVolumeRequest) <-chan error {
		response := make(chan error, 1)
		go func() {
			_, err := driver.NodeStageVolume(context.Background(), req)
			response <- err
		}()
		return response
	}

	req1 := makeReq(vol1.UUID, "/mnt/staging1", vol1.Name)
	req2 := makeReq(vol2.UUID, "/mnt/staging2", vol2.Name)

	// Start first NodeStageVolume on volume 1 - will block
	resp1 := runNodeStage(req1)
	exec1 := <-readyToExecute

	// Start second NodeStageVolume on volume 2 - should also start (different volume)
	resp2 := runNodeStage(req2)

	select {
	case exec2 := <-readyToExecute:
		// Good - operation 2 started, allow both to complete
		exec1 <- struct{}{}
		exec2 <- struct{}{}
	case err := <-resp2:
		t.Errorf("Operation 2 returned error instead of starting: %v", err)
	case <-time.After(time.Second):
		t.Errorf("Timeout waiting for second operation to start")
	}

	// Wait for both operations to complete
	if err := <-resp1; err != nil {
		t.Errorf("Unexpected error from operation 1: %v", err)
	}
	if err := <-resp2; err != nil {
		t.Errorf("Unexpected error from operation 2: %v", err)
	}
}

// TestNodeOperations_CrossOperationLocking tests that different node operations
// (e.g., NodeStageVolume and NodeUnstageVolume) on the same volume are properly
// serialized using volume locks.
func TestNodeOperations_CrossOperationLocking(t *testing.T) {
	readyToExecute := make(chan chan struct{}, 1)
	driver := initBlockingDriver(t, readyToExecute)

	ctx := t.Context()

	// Create a volume
	vol, err := driver.cloudscaleClient.Volumes.Create(ctx, &cloudscale.VolumeCreateRequest{
		Name:   "test-volume",
		SizeGB: 10,
		Type:   "ssd",
	})
	if err != nil {
		t.Fatalf("Failed to create volume: %v", err)
	}

	stageReq := &csi.NodeStageVolumeRequest{
		VolumeId:          vol.UUID,
		StagingTargetPath: "/mnt/staging",
		VolumeCapability: &csi.VolumeCapability{
			AccessMode: &csi.VolumeCapability_AccessMode{
				Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
			},
			AccessType: &csi.VolumeCapability_Mount{
				Mount: &csi.VolumeCapability_MountVolume{
					FsType: "ext4",
				},
			},
		},
		PublishContext: map[string]string{
			LuksEncryptedAttribute: "false",
		},
	}

	unstageReq := &csi.NodeUnstageVolumeRequest{
		VolumeId:          vol.UUID,
		StagingTargetPath: "/mnt/staging",
	}

	runNodeStage := func(req *csi.NodeStageVolumeRequest) <-chan error {
		response := make(chan error, 1)
		go func() {
			_, err := driver.NodeStageVolume(context.Background(), req)
			response <- err
		}()
		return response
	}

	runNodeUnstage := func(req *csi.NodeUnstageVolumeRequest) <-chan error {
		response := make(chan error, 1)
		go func() {
			_, err := driver.NodeUnstageVolume(context.Background(), req)
			response <- err
		}()
		return response
	}

	// Start NodeStageVolume and block
	respStage := runNodeStage(stageReq)
	execStage := <-readyToExecute

	// Start NodeUnstageVolume on the same volume - should get Aborted
	respUnstage := runNodeUnstage(unstageReq)

	select {
	case err := <-respUnstage:
		if err == nil {
			t.Errorf("Expected error for concurrent operation on same volume, got nil")
		} else {
			serverError, ok := status.FromError(err)
			if !ok {
				t.Fatalf("Could not get error status code from err: %v", err)
			}
			if serverError.Code() != codes.Aborted {
				t.Errorf("Expected error code: %v, got: %v. err: %v", codes.Aborted, serverError.Code(), err)
			}
		}
	case <-time.After(time.Second):
		t.Errorf("Timeout waiting for unstage operation to return Aborted")
	}

	// Clean up: allow stage operation to complete
	execStage <- struct{}{}
	<-respStage
}
