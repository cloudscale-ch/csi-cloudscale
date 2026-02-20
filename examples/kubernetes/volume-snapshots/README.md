# Volume Snapshots Example

Demonstrates creating and restoring volumes from snapshots.

## Prerequisites

- Snapshot CRDs and snapshot controller installed (see [main README](../../README.md#prerequisites-for-snapshot-support))

## Workflow

1. Create VolumeSnapshotClass (one-time setup):
   ```bash
   kubectl apply -f volumesnapshotclass.yaml
   ```

2. Create original volume and pod:
   ```bash
   kubectl apply -f original-pvc.yaml
   kubectl apply -f original-pod.yaml
   ```

3. Create snapshot:
   ```bash
   kubectl apply -f volumesnapshot.yaml
   kubectl get volumesnapshot my-volume-snapshot  # wait for READYTOUSE=true
   ```

4. Create restored volume and pod:
   ```bash
   kubectl apply -f restored-pvc.yaml
   kubectl apply -f restored-pod.yaml
   ```

**Note:** Restored volumes must match the snapshot size exactly (5Gi in this example).

## Cleanup

```bash
kubectl delete -f restored-pod.yaml
kubectl delete -f restored-pvc.yaml
kubectl delete -f volumesnapshot.yaml
kubectl delete -f original-pod.yaml
kubectl delete -f original-pvc.yaml
```

**LUKS volumes**: For LUKS-encrypted volumes, see the [LUKS snapshot example](../luks-encrypted-volumes/).
