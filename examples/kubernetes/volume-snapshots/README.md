# Volume Snapshots Example

This example demonstrates how to create and restore volumes from snapshots using the cloudscale.ch CSI driver.

## Prerequisites

Before using snapshots, ensure your cluster has the VolumeSnapshot CRDs and snapshot controller installed.
See the [main README](../../README.md#prerequisites-for-snapshot-support) for installation instructions.

## Workflow

1. **Create VolumeSnapshotClass** (one-time setup, required before creating snapshots):
   ```bash
   kubectl apply -f volumesnapshotclass.yaml
   ```
   
   **Note:** VolumeSnapshotClass is currently not deployed automatically with the driver. You must create it manually.
   This may change in future releases where it will be deployed automatically (similar to StorageClass).

2. **Create original volume and pod** (optional, for testing):
   ```bash
   kubectl apply -f original-pvc.yaml
   kubectl apply -f original-pod.yaml
   ```

3. **Create snapshot**:
   ```bash
   kubectl apply -f volumesnapshot.yaml
   ```

4. **Create restored volume and pod**:
   ```bash
   kubectl apply -f restored-pvc.yaml
   kubectl apply -f restored-pod.yaml
   ```

## Verification

Check snapshot status:
```bash
kubectl get volumesnapshot
kubectl describe volumesnapshot/my-snapshot
```

Check restored volume:
```bash
kubectl get pvc
kubectl get pod
```

**LUKS volumes**: For LUKS-encrypted volumes, see the [LUKS snapshot example](../luks-encrypted-volumes/).

## Cleanup

```bash
kubectl delete -f restored-pod.yaml
kubectl delete -f restored-pvc.yaml
kubectl delete -f volumesnapshot.yaml
kubectl delete -f original-pod.yaml
kubectl delete -f original-pvc.yaml
# Note: VolumeSnapshotClass is typically not deleted as it's a cluster resource
```
