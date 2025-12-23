package driver

import (
	"context"
	"testing"

	"github.com/cloudscale-ch/cloudscale-go-sdk/v6"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestCreateVolumeTypeSsdWithoutExplicitlySpecifyingTheType(t *testing.T) {
	driver := createDriverForTest(t)

	volumeName := randString(32)

	response, err := driver.CreateVolume(
		context.Background(),
		makeCreateVolumeRequest(volumeName, 1, "", false),
	)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.NotNil(t, response.Volume)
	assert.Equal(t, int64(1)*GB, response.Volume.CapacityBytes)
	assert.Equal(t, volumeName, response.Volume.VolumeContext[PublishInfoVolumeName])

	volumes, err := driver.cloudscaleClient.Volumes.List(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 1, len(volumes))
	assert.Equal(t, 1, volumes[0].SizeGB)
	assert.Equal(t, "ssd", volumes[0].Type)
}

func TestCreateVolumeTypeSsdExplicitlySpecifyingTheType(t *testing.T) {
	driver := createDriverForTest(t)

	volumeName := randString(32)

	response, err := driver.CreateVolume(
		context.Background(),
		makeCreateVolumeRequest(volumeName, 5, "ssd", false),
	)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.NotNil(t, response.Volume)
	assert.Equal(t, int64(5)*GB, response.Volume.CapacityBytes)
	assert.Equal(t, volumeName, response.Volume.VolumeContext[PublishInfoVolumeName])

	volumes, err := driver.cloudscaleClient.Volumes.List(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 1, len(volumes))
	assert.Equal(t, 5, volumes[0].SizeGB)
	assert.Equal(t, "ssd", volumes[0].Type)
}

func TestCreateVolumeTypeBulk(t *testing.T) {
	driver := createDriverForTest(t)

	volumeName := randString(32)

	response, err := driver.CreateVolume(
		context.Background(),
		makeCreateVolumeRequest(volumeName, 100, "bulk", false),
	)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.NotNil(t, response.Volume)
	assert.Equal(t, int64(100)*GB, response.Volume.CapacityBytes)
	assert.Equal(t, volumeName, response.Volume.VolumeContext[PublishInfoVolumeName])

	volumes, err := driver.cloudscaleClient.Volumes.List(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 1, len(volumes))
	assert.Equal(t, 100, volumes[0].SizeGB)
	assert.Equal(t, "bulk", volumes[0].Type)
}

func TestCreateVolumeInvalidType(t *testing.T) {
	driver := createDriverForTest(t)

	volumeName := randString(32)

	_, err := driver.CreateVolume(
		context.Background(),
		makeCreateVolumeRequest(volumeName, 100, "foo", false),
	)

	assert.Error(t, err)
}

func TestCreateVolumeInvalidLUKSAndRaw(t *testing.T) {
	driver := createDriverForTest(t)

	volumeName := randString(32)

	_, err := driver.CreateVolume(
		context.Background(),
		makeLuksCreateVolumeRequest(volumeName, 100, "ssd", true, true),
	)

	assert.Error(t, err)
}

func TestLuksEncryptionAttributeIsSetInContext(t *testing.T) {
	driver := createDriverForTest(t)

	// explicitly set luks encryption to false
	volumeName := randString(32)
	response, err := driver.CreateVolume(
		context.Background(),
		makeLuksCreateVolumeRequest(volumeName, 100, "bulk", false, false),
	)
	assert.NoError(t, err)
	assert.Equal(t, "false", response.Volume.VolumeContext[LuksEncryptedAttribute])

	// explicitly set luks encryption to true
	volumeName = randString(32)
	response, err = driver.CreateVolume(
		context.Background(),
		makeLuksCreateVolumeRequest(volumeName, 100, "bulk", true, false),
	)
	assert.NoError(t, err)
	assert.Equal(t, "true", response.Volume.VolumeContext[LuksEncryptedAttribute])

	// don't set the luks encryption parameter - must implicitly default to false
	volumeName = randString(32)
	response, err = driver.CreateVolume(
		context.Background(),
		makeCreateVolumeRequest(volumeName, 100, "bulk", false),
	)
	assert.NoError(t, err)
	assert.Equal(t, "false", response.Volume.VolumeContext[LuksEncryptedAttribute])
}

func makeLuksCreateVolumeRequest(volumeName string, sizeGb int, volumeType string, luksEncryptionEnabled bool, block bool) *csi.CreateVolumeRequest {
	request := makeCreateVolumeRequest(volumeName, sizeGb, volumeType, block)
	if luksEncryptionEnabled {
		request.Parameters[LuksEncryptedAttribute] = "true"
	} else {
		request.Parameters[LuksEncryptedAttribute] = "false"
	}
	return request
}

func makeCreateVolumeRequest(volumeName string, sizeGb int, volumeType string, block bool) *csi.CreateVolumeRequest {
	return &csi.CreateVolumeRequest{
		Name:               volumeName,
		VolumeCapabilities: makeVolumeCapabilityObject(block),
		CapacityRange: &csi.CapacityRange{
			RequiredBytes: int64(sizeGb) * GB,
		},
		Parameters: map[string]string{
			StorageTypeAttribute: volumeType,
		},
	}

}

func makeVolumeCapabilityObject(block bool) []*csi.VolumeCapability {
	accessMode := &csi.VolumeCapability_AccessMode{
		Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
	}
	if !block {
		return []*csi.VolumeCapability{
			{AccessType: &csi.VolumeCapability_Mount{Mount: &csi.VolumeCapability_MountVolume{}}, AccessMode: accessMode},
		}
	}
	return []*csi.VolumeCapability{
		{AccessType: &csi.VolumeCapability_Block{Block: &csi.VolumeCapability_BlockVolume{}}, AccessMode: accessMode},
	}
}

func createDriverForTest(t *testing.T) *Driver {
	initialServers := map[string]*cloudscale.Server{}
	cloudscaleClient := NewFakeClient(initialServers)

	return &Driver{
		mounter:          &fakeMounter{},
		log:              logrus.New().WithField("test_enabled", true),
		cloudscaleClient: cloudscaleClient,
		volumeLocks:      NewVolumeLocks(),
	}
}
