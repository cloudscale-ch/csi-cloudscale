# LUKS Encrypted Volumes Example

This example demonstrates how to create and restore LUKS-encrypted volumes from snapshots using the cloudscale.ch CSI driver.

## Prerequisites

1. **Snapshot CRDs installed**: See the [main README](../../README.md#prerequisites-for-snapshot-support)
2. **VolumeSnapshotClass created**: See the [volume-snapshots example](../volume-snapshots/)
3. **LUKS storage classes available**: `cloudscale-volume-ssd-luks` or `cloudscale-volume-bulk-luks`

## Workflow

### 1. Create Original LUKS Volume

```bash
# Create the LUKS secret (contains the encryption key)
kubectl apply -f luks-secret.yaml

# Create the PVC (this will create a LUKS-encrypted volume)
kubectl apply -f luks-pvc.yaml

# Optional: Create a pod to use the volume
kubectl apply -f luks-pod.yaml
```

**Note:** The pod will remain in `ContainerCreating` state until:
- The PVC is bound (volume provisioned)
- The LUKS volume is decrypted and mounted on the node
- This can take 30-60 seconds depending on volume size

### 2. Create Snapshot

```bash
kubectl apply -f luks-volumesnapshot.yaml
```

Wait for the snapshot to be ready:
```bash
kubectl get volumesnapshot my-luks-snapshot
# Wait until READYTOUSE is true
```

### 3. Restore from Snapshot

```bash
# Create the LUKS secret for the restored volume
# IMPORTANT: Use the SAME key as the original volume
kubectl apply -f restored-luks-secret.yaml

# Create the restored PVC (from snapshot)
kubectl apply -f restored-luks-pvc.yaml

# Optional: Create a pod to use the restored volume
kubectl apply -f restored-luks-pod.yaml
```

**Note:** Restored pods will also remain in `ContainerCreating` until:
- The volume is created from the snapshot
- The PVC is bound
- The LUKS volume is decrypted and mounted
- This can take 1-2 minutes for snapshot restore

## Verification

Check PVC status:
```bash
kubectl get pvc
# Wait until STATUS is Bound
```

Check pod status:
```bash
kubectl get pod
# Pods will be in ContainerCreating until PVCs are bound and volumes are mounted
```

Check pod events if stuck:
```bash
kubectl describe pod my-csi-app-luks
kubectl describe pod my-restored-luks-app
```

## Important Notes

1. **LUKS Key Matching**: The restored volume MUST use the same LUKS key as the original volume. The key is stored in the secret.

2. **Secret Naming**: The secret name must follow the pattern `${pvc-name}-luks-key`:
   - Original PVC `csi-pod-pvc-luks` → Secret `csi-pod-pvc-luks-luks-key`
   - Restored PVC `my-restored-luks-volume` → Secret `my-restored-luks-volume-luks-key`

3. **Storage Class**: Both original and restored volumes must use a LUKS storage class (`cloudscale-volume-ssd-luks` or `cloudscale-volume-bulk-luks`).

4. **Size Matching**: The restored volume size must match the snapshot size exactly (1Gi in this example).

5. **ContainerCreating State**: It's **expected** for pods to remain in `ContainerCreating` state for 30-120 seconds while:
   - Volumes are being provisioned/restored
   - LUKS volumes are being decrypted
   - Filesystems are being mounted

## Troubleshooting

If pods remain stuck in `ContainerCreating` for more than 5 minutes:

1. Check PVC status:
   ```bash
   kubectl get pvc
   kubectl describe pvc <pvc-name>
   ```

2. Check for events:
   ```bash
   kubectl get events --sort-by='.lastTimestamp'
   ```

3. Verify secrets exist:
   ```bash
   kubectl get secret <pvc-name>-luks-key
   ```

4. Check node logs for LUKS errors:
   ```bash
   kubectl logs -n kube-system -l app=csi-cloudscale-node
   ```

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
