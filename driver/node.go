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
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/mount-utils"
	utilexec "k8s.io/utils/exec"
)

const (
	// Current technical limit is 128
	//   - 1 for root
	//   - 1 for /var/lib/docker
	//   - 1 additional volume outside of CSI
	fallbackMaxVolumesPerNode = 125

	volumeModeBlock      = "block"
	volumeModeFilesystem = "filesystem"

	topologyZonePrefix = "csi.cloudscale.ch/zone"
)

// NodeStageVolume mounts the volume to a staging path on the node. This is
// called by the CO before NodePublishVolume and is used to temporary mount the
// volume to a staging path. Once mounted, NodePublishVolume will make sure to
// mount it to the appropriate path
func (d *Driver) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "NodeStageVolume Volume ID must be provided")
	}

	if req.StagingTargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "NodeStageVolume Staging Target Path must be provided")
	}

	if req.VolumeCapability == nil {
		return nil, status.Error(codes.InvalidArgument, "NodeStageVolume Volume Capability must be provided")
	}

	ll := d.log.WithFields(logrus.Fields{
		"volume_id":           req.VolumeId,
		"staging_target_path": req.StagingTargetPath,
		"method":              "node_stage_volume",
		"volume_context":      req.VolumeContext,
		"publish_context":     req.PublishContext,
	})
	ll.Info("node stage volume called")

	if acquired := d.volumeLocks.TryAcquire(req.VolumeId); !acquired {
		return nil, status.Errorf(codes.Aborted, "an operation with the given Volume ID %s already exists", req.VolumeId)
	}
	defer d.volumeLocks.Release(req.VolumeId)

	// Apparently sometimes we need to call udevadm trigger to get the volume
	// properly registered in /dev/disk. More information can be found here:
	// https://github.com/cloudscale-ch/csi-cloudscale/issues/9
	source, err := d.mounter.FinalizeVolumeAttachmentAndFindPath(ll, req.VolumeId)
	if err != nil {
		return nil, err
	}

	ll = ll.WithFields(logrus.Fields{
		"source": source,
	})
	ll.Info("resolved volume device path")

	publishContext := req.GetPublishContext()
	if publishContext == nil {
		return nil, status.Error(codes.InvalidArgument, "PublishContext must be provided")
	}

	volumeName, ok := publishContext[PublishInfoVolumeName]
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "Could not find the volume by name")
	}

	luksContext := getLuksContext(req.Secrets, publishContext, VolumeLifecycleNodeStageVolume)

	// If it is a block volume, we do nothing for stage volume
	// because we bind mount the absolute device path to a file
	switch req.VolumeCapability.GetAccessType().(type) {
	case *csi.VolumeCapability_Block:
		return &csi.NodeStageVolumeResponse{}, nil
	}

	stagingTargetPath := req.StagingTargetPath

	mnt := req.VolumeCapability.GetMount()
	options := mnt.MountFlags

	fsType := "ext4"
	if mnt.FsType != "" {
		fsType = mnt.FsType
	}

	ll = ll.WithFields(logrus.Fields{
		"volume_mode":    volumeModeFilesystem,
		"volume_name":    volumeName,
		"fs_type":        fsType,
		"mount_options":  options,
		"luks_encrypted": luksContext.EncryptionEnabled,
	})

	formatted, err := d.mounter.IsFormatted(source, luksContext, ll)
	if err != nil {
		return nil, err
	}

	if !formatted {
		ll.Info("formatting the volume for staging")
		if err := d.mounter.Format(source, fsType, luksContext, ll); err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	} else {
		ll.Info("source device is already formatted")
	}

	ll.Info("checking if stagingTargetPath is already mounted")

	mountInfo, err := d.mounter.GetMountInfo(stagingTargetPath, ll)
	if err != nil {
		ll.WithError(err).Error("unable to check if already mounted")
		return nil, err
	}
	if mountInfo != nil && mountInfo.Propagation != "shared" {
		return nil, fmt.Errorf("mount propagation for target %q is not enabled", stagingTargetPath)
	}

	if mountInfo == nil {
		ll.Info("not mounted yet, mounting the volume for staging")
		if err := d.mounter.Mount(source, stagingTargetPath, fsType, luksContext, ll, options...); err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	} else {
		// Something is already mounted at the staging path. Verify it is
		// mounted from the device we just resolved before declaring success —
		// otherwise a stale mount left by an earlier (failed or racing) stage
		// operation can be silently accepted, which is the same class of bug
		// as the LUKS mapping reuse in luksOpen.
		expected := source
		if luksContext.EncryptionEnabled {
			expected = "/dev/mapper/" + luksContext.VolumeName
		} else if resolved, err := filepath.EvalSymlinks(source); err == nil {
			// findmnt reports the kernel-resolved device, so compare against
			// the canonical form. Fall back to the literal source on resolve
			// failure — the mismatch will then surface as a loud error.
			expected = resolved
		}

		if strings.TrimSpace(mountInfo.Source) != expected {
			return nil, status.Errorf(codes.FailedPrecondition,
				"stage path %s is mounted from %q, expected %s, refusing to reuse stale mount",
				stagingTargetPath, mountInfo.Source, expected)
		}
		ll.Info("source device is already mounted to the stagingTargetPath path")
	}

	// Resize the filesystem if the block device is larger (e.g. snapshot
	// restored to a larger volume). Kubernetes only calls NodeExpandVolume
	// for PVC resizes, not for freshly created volumes, so we must handle
	// it here. See https://github.com/kubernetes/kubernetes/issues/94929.
	// For LUKS-encrypted volumes, the filesystem lives on the /dev/mapper
	// device, not on the raw block device. Resolve the actual device backing
	// the staging path so that both LUKS and non-LUKS volumes are handled
	// correctly.
	mounter := mount.New("")
	devicePath, err := d.mounter.GetDeviceName(mounter, stagingTargetPath)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "NodeStageVolume unable to get device path for %q: %v", stagingTargetPath, err)
	}

	ll = ll.WithFields(logrus.Fields{
		"device_path": devicePath,
	})

	// If the staged device is a LUKS mapping, grow the LUKS container first so
	// the filesystem can see the larger size.
	isLuks, _, err := isLuksMapping(devicePath)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "NodeStageVolume unable to test if volume at %q is encrypted with LUKS: %v", devicePath, err)
	}
	if isLuks {
		ll.Info("resizing LUKS container before filesystem resize")
		if err := luksResize(devicePath, ll); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to resize LUKS container on %s: %v", devicePath, err)
		}
	}

	r := mount.NewResizeFs(utilexec.New())
	needResize, err := r.NeedResize(devicePath, stagingTargetPath)
	if err != nil {
		ll.WithError(err).Warn("unable to check if filesystem needs resize")
	} else if needResize {
		ll.Info("resizing filesystem to match block device size")
		if _, err := r.Resize(devicePath, stagingTargetPath); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to resize filesystem on %s: %v", devicePath, err)
		}
		ll.Info("filesystem resized successfully")
	}

	ll.Info("formatting and mounting stage volume is finished")
	return &csi.NodeStageVolumeResponse{}, nil
}

// NodeUnstageVolume unstages the volume from the staging path
func (d *Driver) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "NodeUnstageVolume Volume ID must be provided")
	}

	if req.StagingTargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "NodeUnstageVolume Staging Target Path must be provided")
	}

	if acquired := d.volumeLocks.TryAcquire(req.VolumeId); !acquired {
		return nil, status.Errorf(codes.Aborted, "an operation with the given Volume ID %s already exists", req.VolumeId)
	}
	defer d.volumeLocks.Release(req.VolumeId)

	luksContext := LuksContext{VolumeLifecycle: VolumeLifecycleNodeUnstageVolume}

	ll := d.log.WithFields(logrus.Fields{
		"volume_id":           req.VolumeId,
		"staging_target_path": req.StagingTargetPath,
		"method":              "node_unstage_volume",
	})
	ll.Info("node unstage volume called")

	mountInfo, err := d.mounter.GetMountInfo(req.StagingTargetPath, ll)
	if err != nil {
		return nil, err
	}

	if mountInfo != nil {
		ll.Info("unmounting the staging target path")
		err := d.mounter.Unmount(req.StagingTargetPath, luksContext, ll)
		if err != nil {
			return nil, err
		}
	} else {
		ll.Info("staging target path is already unmounted")
	}

	ll.Info("unmounting stage volume is finished")
	return &csi.NodeUnstageVolumeResponse{}, nil
}

// NodePublishVolume mounts the volume mounted to the staging path to the target path
func (d *Driver) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "NodePublishVolume Volume ID must be provided")
	}

	if req.StagingTargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "NodePublishVolume Staging Target Path must be provided")
	}

	if req.TargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "NodePublishVolume Target Path must be provided")
	}

	if req.VolumeCapability == nil {
		return nil, status.Error(codes.InvalidArgument, "NodePublishVolume Volume Capability must be provided")
	}

	ll := d.log.WithFields(logrus.Fields{
		"volume_id":           req.VolumeId,
		"staging_target_path": req.StagingTargetPath,
		"target_path":         req.TargetPath,
		"method":              "node_publish_volume",
	})
	ll.Info("node publish volume called")

	if acquired := d.volumeLocks.TryAcquire(req.VolumeId); !acquired {
		return nil, status.Errorf(codes.Aborted, "an operation with the given Volume ID %s already exists", req.VolumeId)
	}
	defer d.volumeLocks.Release(req.VolumeId)

	publishContext := req.GetPublishContext()
	if publishContext == nil {
		return nil, status.Error(codes.InvalidArgument, "PublishContext must be provided")
	}
	luksContext := getLuksContext(req.Secrets, publishContext, VolumeLifecycleNodePublishVolume)

	ll = ll.WithFields(logrus.Fields{
		"luks_encrypted": luksContext.EncryptionEnabled,
	})

	options := []string{"bind"}
	if req.Readonly {
		options = append(options, "ro")
	}

	var err error
	switch req.GetVolumeCapability().GetAccessType().(type) {
	case *csi.VolumeCapability_Block:
		err = d.nodePublishVolumeForBlock(req, luksContext, options, ll)
	case *csi.VolumeCapability_Mount:
		err = d.nodePublishVolumeForFileSystem(req, luksContext, options, ll)
	default:
		return nil, status.Error(codes.InvalidArgument, "Unknown access type")
	}

	if err != nil {
		return nil, err
	}

	ll.Info("bind mounting the volume is finished")
	return &csi.NodePublishVolumeResponse{}, nil
}

// NodeUnpublishVolume unmounts the volume from the target path
func (d *Driver) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "NodeUnpublishVolume Volume ID must be provided")
	}

	if req.TargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "NodeUnpublishVolume Target Path must be provided")
	}

	if acquired := d.volumeLocks.TryAcquire(req.VolumeId); !acquired {
		return nil, status.Errorf(codes.Aborted, "an operation with the given Volume ID %s already exists", req.VolumeId)
	}
	defer d.volumeLocks.Release(req.VolumeId)

	luksContext := LuksContext{VolumeLifecycle: VolumeLifecycleNodeUnpublishVolume}

	ll := d.log.WithFields(logrus.Fields{
		"volume_id":      req.VolumeId,
		"target_path":    req.TargetPath,
		"method":         "node_unpublish_volume",
		"luks_encrypted": luksContext.EncryptionEnabled,
	})
	ll.Info("node unpublish volume called")

	err := d.mounter.Unmount(req.TargetPath, luksContext, ll)
	if err != nil {
		return nil, err
	}

	ll.Info("unmounting volume is finished")
	return &csi.NodeUnpublishVolumeResponse{}, nil
}

// NodeGetCapabilities returns the supported capabilities of the node server
func (d *Driver) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	nscaps := []*csi.NodeServiceCapability{
		&csi.NodeServiceCapability{
			Type: &csi.NodeServiceCapability_Rpc{
				Rpc: &csi.NodeServiceCapability_RPC{
					Type: csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
				},
			},
		},
		&csi.NodeServiceCapability{
			Type: &csi.NodeServiceCapability_Rpc{
				Rpc: &csi.NodeServiceCapability_RPC{
					Type: csi.NodeServiceCapability_RPC_EXPAND_VOLUME,
				},
			},
		},
		&csi.NodeServiceCapability{
			Type: &csi.NodeServiceCapability_Rpc{
				Rpc: &csi.NodeServiceCapability_RPC{
					Type: csi.NodeServiceCapability_RPC_GET_VOLUME_STATS,
				},
			},
		},
	}

	d.log.WithFields(logrus.Fields{
		"node_capabilities": nscaps,
		"method":            "node_get_capabilities",
	}).Info("node get capabilities called")
	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: nscaps,
	}, nil
}

func getEnvAsInt(key string, fallback int64) int64 {
	if valueStr, ok := os.LookupEnv(key); ok {
		if value, err := strconv.ParseInt(valueStr, 10, 64); err == nil {
			return value
		}
	}
	return fallback
}

// NodeGetInfo returns the supported capabilities of the node server. This
// should eventually return the droplet ID if possible. This is used so the CO
// knows where to place the workload. The result of this function will be used
// by the CO in ControllerPublishVolume.
func (d *Driver) NodeGetInfo(ctx context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	d.log.WithField("method", "node_get_info").Info("node get info called")

	maxVolumesPerNode := getEnvAsInt("CLOUDSCALE_MAX_CSI_VOLUMES_PER_NODE", fallbackMaxVolumesPerNode)

	return &csi.NodeGetInfoResponse{
		NodeId:            d.serverId,
		MaxVolumesPerNode: maxVolumesPerNode,

		// make sure that the driver works on this particular region only
		AccessibleTopology: &csi.Topology{
			Segments: map[string]string{
				topologyZonePrefix: d.zone,
			},
		},
	}, nil
}

// NodeGetVolumeStats returns the volume capacity statistics available for the
// the given volume.
func (d *Driver) NodeGetVolumeStats(ctx context.Context, req *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {

	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "NodeGetVolumeStats Volume ID must be provided")
	}

	volumePath := req.VolumePath
	if volumePath == "" {
		return nil, status.Error(codes.InvalidArgument, "NodeGetVolumeStats Volume Path must be provided")
	}

	ll := d.log.WithFields(logrus.Fields{
		"method":              "node_get_volume_stats",
		"volume_path":         volumePath,
		"volume_id":           req.VolumeId,
		"staging_target_path": req.StagingTargetPath,
	})
	ll.Info("node get volume stats called")

	mountInfo, err := d.mounter.GetMountInfo(volumePath, ll)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check if volume path %q is mounted: %s", volumePath, err)
	}
	if mountInfo != nil && mountInfo.Propagation != "shared" {
		return nil, status.Errorf(codes.Internal, "mount propagation for target %q is not enabled", volumePath)
	}

	if mountInfo == nil {
		return nil, status.Errorf(codes.NotFound, "volume path %q is not mounted", volumePath)
	}

	isBlock, err := d.mounter.IsBlockDevice(volumePath)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to determine if %q is block device: %s", volumePath, err)
	}

	stats, err := d.mounter.GetStatistics(volumePath)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to retrieve capacity statistics for volume path %q: %s", volumePath, err)
	}

	// only can retrieve total capacity for a block device
	if isBlock {
		ll.WithFields(logrus.Fields{
			"volume_mode": volumeModeBlock,
			"bytes_total": stats.totalBytes,
		}).Info("node capacity statistics retrieved")

		return &csi.NodeGetVolumeStatsResponse{
			Usage: []*csi.VolumeUsage{
				{
					Unit:  csi.VolumeUsage_BYTES,
					Total: stats.totalBytes,
				},
			},
		}, nil
	}

	ll.WithFields(logrus.Fields{
		"volume_mode":      volumeModeFilesystem,
		"bytes_available":  stats.availableBytes,
		"bytes_total":      stats.totalBytes,
		"bytes_used":       stats.usedBytes,
		"inodes_available": stats.availableInodes,
		"inodes_total":     stats.totalInodes,
		"inodes_used":      stats.usedInodes,
	}).Info("node capacity statistics retrieved")

	return &csi.NodeGetVolumeStatsResponse{
		Usage: []*csi.VolumeUsage{
			{
				Available: stats.availableBytes,
				Total:     stats.totalBytes,
				Used:      stats.usedBytes,
				Unit:      csi.VolumeUsage_BYTES,
			},
			{
				Available: stats.availableInodes,
				Total:     stats.totalInodes,
				Used:      stats.usedInodes,
				Unit:      csi.VolumeUsage_INODES,
			},
		},
	}, nil
}

func (d *Driver) NodeExpandVolume(ctx context.Context, req *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	volumeID := req.VolumeId
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "NodeExpandVolume volume ID not provided")
	}

	volumePath := req.VolumePath
	if len(volumePath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "NodeExpandVolume volume path not provided")
	}

	ll := d.log.WithFields(logrus.Fields{
		"volume_id":   req.VolumeId,
		"volume_path": req.VolumePath,
		"method":      "node_expand_volume",
	})
	ll.Info("node expand volume called")

	source, err := d.mounter.FindAbsoluteDeviceByIDPath(volumeID, ll)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to find device path for volume %s. %v", volumeID, err)
	}

	ll = ll.WithField("source", source)

	if req.GetVolumeCapability() != nil {
		switch req.GetVolumeCapability().GetAccessType().(type) {
		case *csi.VolumeCapability_Block:
			ll.Info("filesystem expansion is skipped for block volumes")
			return &csi.NodeExpandVolumeResponse{}, nil
		}
	}

	mountInfo, err := d.mounter.GetMountInfo(volumePath, ll)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "NodeExpandVolume failed to check if volume path %q is mounted: %s", volumePath, err)
	}
	if mountInfo != nil && mountInfo.Propagation != "shared" {
		return nil, status.Errorf(codes.Internal, "NodeExpandVolume mount propagation for target %q is not enabled", volumePath)
	}

	if mountInfo == nil {
		return nil, status.Errorf(codes.NotFound, "NodeExpandVolume volume path %q is not mounted", volumePath)
	}

	mounter := mount.New("")
	devicePath, err := d.mounter.GetDeviceName(mounter, volumePath)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "NodeExpandVolume unable to get device path for %q: %v", volumePath, err)
	}

	isLuks, _, err := isLuksMapping(devicePath)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "NodeExpandVolume unable to test if volume %q at %q is encrypted with luks: %v", volumePath, devicePath, err)
	}

	ll = ll.WithFields(logrus.Fields{
		"device_path": devicePath,
	})
	hasRequiredSize, err := d.mounter.HasRequiredSize(ll, source, req.CapacityRange.RequiredBytes)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "NodeExpandVolume unable to test if volume %q at %q has required size: %v", volumePath, source, err)
	}
	if !hasRequiredSize {
		// Volume does not yet see the new size of the volume, expanding the filesystem would result in a noop.
		// Returning a UNAVAILABLE will cause a retry.
		return nil, status.Errorf(codes.Unavailable, "Not yet required size.")
	}

	// the luks container must be resized if the volume was resized while the disk was mounted
	if isLuks {
		ll.Info("resizing luks container")
		err := luksResize(devicePath, ll)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "NodeExpandVolume unable resize luks container for volume %q at %q: %v", volumePath, devicePath, err)
		}
	}

	r := mount.NewResizeFs(utilexec.New())
	ll.Info("resizing volume")
	if _, err := r.Resize(devicePath, volumePath); err != nil {
		return nil, status.Errorf(codes.Internal, "NodeExpandVolume could not resize volume %q (%q):  %v", volumeID, req.GetVolumePath(), err)
	}

	ll.Info("volume was resized")
	return &csi.NodeExpandVolumeResponse{}, nil
}

func (d *Driver) nodePublishVolumeForFileSystem(req *csi.NodePublishVolumeRequest, luksContext LuksContext, mountOptions []string, ll *logrus.Entry) error {
	source := req.StagingTargetPath
	target := req.TargetPath

	mnt := req.VolumeCapability.GetMount()
	mountOptions = append(mountOptions, mnt.MountFlags...)

	fsType := "ext4"
	if mnt.FsType != "" {
		fsType = mnt.FsType
	}

	ll = ll.WithFields(logrus.Fields{
		"source_path":   source,
		"volume_mode":   volumeModeFilesystem,
		"fs_type":       fsType,
		"mount_options": mountOptions,
	})

	ll.Info("mounting the volume")
	if err := d.mounter.Mount(source, target, fsType, luksContext, ll, mountOptions...); err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	return nil
}

func (d *Driver) nodePublishVolumeForBlock(req *csi.NodePublishVolumeRequest, luksContext LuksContext, mountOptions []string, ll *logrus.Entry) error {
	volumeId := req.VolumeId

	source, err := d.mounter.FindAbsoluteDeviceByIDPath(volumeId, ll)
	if err != nil {
		return status.Errorf(codes.Internal, "Failed to find device path for volume %s. %v", volumeId, err)
	}

	target := req.TargetPath

	ll = ll.WithFields(logrus.Fields{
		"source_path":   source,
		"volume_mode":   volumeModeBlock,
		"mount_options": mountOptions,
	})

	ll.Info("mounting the volume")
	if err := d.mounter.Mount(source, target, "", luksContext, ll, mountOptions...); err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	return nil
}
