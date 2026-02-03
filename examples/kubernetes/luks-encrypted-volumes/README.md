# LUKS Encrypted Volumes Example

Demonstrates creating and restoring LUKS-encrypted volumes from snapshots.

## Prerequisites

- Snapshot CRDs installed (see [main README](../../README.md#prerequisites-for-snapshot-support))
- VolumeSnapshotClass created (see [volume-snapshots example](../volume-snapshots/))
- LUKS storage classes available: `cloudscale-volume-ssd-luks` or `cloudscale-volume-bulk-luks`

## Workflow

1. Create secret and PVC:
   ```bash
   kubectl apply -f luks-secret.yaml
   kubectl apply -f luks-pvc.yaml
   kubectl apply -f luks-pod.yaml
   ```

2. Create snapshot:
   ```bash
   kubectl apply -f luks-volumesnapshot.yaml
   kubectl get volumesnapshot luks-snapshot  # wait for READYTOUSE=true
   ```co

3. Restore from snapshot:
   ```bash
   kubectl apply -f restored-luks-secret.yaml
   kubectl apply -f restored-luks-pvc.yaml
   kubectl apply -f restored-luks-pod.yaml
   ```

## Secret Naming Pattern

**Storage class requirement:** The LUKS storage class enforces the secret naming pattern `${pvc.name}-luks-key`.

- Original PVC `luks-volume` → requires secret `luks-volume-luks-key`
- Restored PVC `luks-volume-restored` → requires secret `luks-volume-restored-luks-key`

**Important:** Different PVC names require different secret names (storage class requirement), but the key **VALUE** must be the same because restored volumes contain the same encrypted data.

## Cleanup

```bash
kubectl delete -f restored-luks-pod.yaml
kubectl delete -f restored-luks-pvc.yaml
kubectl delete -f restored-luks-secret.yaml
kubectl delete -f luks-volumesnapshot.yaml
kubectl delete -f luks-pod.yaml
kubectl delete -f luks-pvc.yaml
kubectl delete -f luks-secret.yaml
```
