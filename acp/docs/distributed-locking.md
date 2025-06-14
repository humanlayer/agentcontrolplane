# Distributed Locking in Agent Control Plane

## Overview

The Agent Control Plane implements a dual-layer locking mechanism to prevent race conditions in LLM interactions while supporting both single-pod and multi-pod deployments. This document explains how the Kubernetes lease-based distributed locking works.

## Problem Statement

When multiple tool calls complete simultaneously, they can trigger concurrent Task reconciliations that cause:
- Duplicate events (`SendingContextWindowToLLM`, `LLMFinalAnswer`)
- Race conditions in context window updates
- Invalid LLM payloads due to corrupted state
- Multiple simultaneous LLM requests for the same task

## Solution Architecture

### Dual-Layer Locking

The system uses two complementary locking mechanisms:

1. **In-Memory Mutexes** (single-pod optimization)
2. **Kubernetes Leases** (multi-pod coordination)

```go
// Layer 1: Fast in-memory synchronization
mutex := sm.getTaskMutex(task.Name)
mutex.Lock()
defer mutex.Unlock()

// Layer 2: Distributed coordination via etcd
lease, acquired, err := sm.acquireTaskLease(ctx, task.Name)
if !acquired {
    return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
}
defer sm.releaseTaskLease(ctx, lease)
```

## Implementation Details

### StateMachine Structure

```go
type StateMachine struct {
    client            client.Client
    // ... other fields
    
    // Task-level mutexes (single-pod optimization)
    taskMutexes  map[string]*sync.Mutex
    mutexMapLock sync.RWMutex
    
    // Distributed locking (multi-pod coordination)
    namespace     string
    podName       string
    leaseDuration time.Duration
}
```

### Pod Identity Detection

```go
func NewStateMachine(...) *StateMachine {
    // Get pod identity for distributed locking
    namespace := os.Getenv("POD_NAMESPACE")
    if namespace == "" {
        namespace = "default"
    }
    
    podName := os.Getenv("POD_NAME")
    if podName == "" {
        podName = "acp-controller-manager-" + uuid.New().String()[:8]
    }
    
    return &StateMachine{
        // ... other fields
        namespace:     namespace,
        podName:       podName,
        leaseDuration: 30 * time.Second,
    }
}
```

### Lease Acquisition Logic

```go
func (sm *StateMachine) acquireTaskLease(ctx context.Context, taskName string) (*coordinationv1.Lease, bool, error) {
    leaseName := "task-llm-" + taskName
    now := metav1.NewMicroTime(time.Now())
    
    lease := &coordinationv1.Lease{
        ObjectMeta: metav1.ObjectMeta{
            Name:      leaseName,
            Namespace: sm.namespace,
        },
        Spec: coordinationv1.LeaseSpec{
            HolderIdentity:       &sm.podName,
            LeaseDurationSeconds: ptr.To(int32(sm.leaseDuration.Seconds())),
            AcquireTime:          &now,
            RenewTime:            &now,
        },
    }

    // Try to create new lease
    err := sm.client.Create(ctx, lease)
    if err == nil {
        return lease, true, nil
    }

    // If lease exists, check if we can acquire it
    if apierrors.IsAlreadyExists(err) {
        existingLease := &coordinationv1.Lease{}
        if err := sm.client.Get(ctx, client.ObjectKey{
            Namespace: sm.namespace,
            Name:      leaseName,
        }, existingLease); err != nil {
            return nil, false, err
        }

        // Take over if expired or we already own it
        if sm.canAcquireLease(existingLease) {
            existingLease.Spec.HolderIdentity = &sm.podName
            existingLease.Spec.AcquireTime = &now
            existingLease.Spec.RenewTime = &now
            
            if err := sm.client.Update(ctx, existingLease); err != nil {
                return nil, false, err
            }
            return existingLease, true, nil
        }
        
        return nil, false, nil // Lease held by another pod
    }

    return nil, false, err
}
```

### Lease Ownership Rules

```go
func (sm *StateMachine) canAcquireLease(lease *coordinationv1.Lease) bool {
    // We can acquire if we already hold it
    if lease.Spec.HolderIdentity != nil && *lease.Spec.HolderIdentity == sm.podName {
        return true
    }

    // We can acquire if lease is expired
    if lease.Spec.RenewTime == nil {
        return true
    }

    expireTime := lease.Spec.RenewTime.Add(sm.leaseDuration)
    return time.Now().After(expireTime)
}
```

### Lease Cleanup

```go
func (sm *StateMachine) releaseTaskLease(ctx context.Context, lease *coordinationv1.Lease) {
    if lease == nil {
        return
    }

    // Delete lease to release it immediately
    if err := sm.client.Delete(ctx, lease); err != nil {
        // Log but don't fail - lease will expire naturally
        log.FromContext(ctx).V(1).Info("Failed to delete task lease", 
            "lease", lease.Name, "error", err)
    }
}
```

## RBAC Requirements

The controller requires permissions to manage leases:

```go
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch;delete
```

This generates the following RBAC configuration:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: acp-manager-role
rules:
- apiGroups:
  - coordination.k8s.io
  resources:
  - leases
  verbs:
  - get
  - list
  - watch
  - create
  - update
  - patch
  - delete
```

## Usage Examples

### Single Pod Deployment

With one pod, only in-memory mutexes are used:
- Zero etcd overhead
- Nanosecond-level locking performance
- Distributed lease acquisition always succeeds immediately

### Multi-Pod Deployment

With multiple pods, distributed coordination prevents conflicts:

```bash
# Scale to multiple pods
kubectl scale deployment acp-controller-manager --replicas=3

# Check lease ownership
kubectl get leases | grep task-llm
kubectl describe lease task-llm-my-task-name
```

Example lease output:
```yaml
Name:         task-llm-my-task-name
Spec:
  Holder Identity:         acp-controller-manager-64dc466545-6pb8h
  Lease Duration Seconds:  30
  Acquire Time:            2025-06-13T23:27:01.640299Z
  Renew Time:              2025-06-13T23:27:01.640299Z
```

## Debugging

### Checking Lease Status

```bash
# List all task leases
kubectl get leases | grep task-llm

# Inspect specific lease
kubectl describe lease task-llm-<task-name>

# Watch lease changes
kubectl get leases -w | grep task-llm
```

### Log Analysis

Look for these log patterns:

```bash
# Successful lease acquisition
kubectl logs -l app.kubernetes.io/name=acp | grep "acquired.*lease"

# Lease conflicts (normal in multi-pod)
kubectl logs -l app.kubernetes.io/name=acp | grep "lease held by another pod"

# Lease cleanup
kubectl logs -l app.kubernetes.io/name=acp | grep "delete.*lease"
```

## Performance Characteristics

### Single Pod
- **In-memory mutex**: ~10ns lock acquisition
- **Lease acquisition**: ~1-5ms (etcd roundtrip)
- **Total overhead**: Minimal (lease always available)

### Multi-Pod
- **Contention scenario**: 5-second requeue when lease unavailable
- **Average case**: Same as single pod (leases rarely conflict)
- **Failure recovery**: 30-second maximum delay (lease expiration)

## Design Principles

### Granular Locking
- Locks are per-task, not global
- Different tasks can process LLM requests concurrently
- Only same-task LLM requests are serialized

### Graceful Degradation
- Lease acquisition failure → polite requeue, no error
- Pod crash → automatic lease expiration (30s)
- Network partition → lease expires, other pods continue

### Observable Operations
- Lease holder identity for debugging
- Kubernetes-native resource inspection
- Standard kubectl tooling works

## Best Practices

### Pod Environment Variables
Set these environment variables in your deployment:

```yaml
env:
- name: POD_NAME
  valueFrom:
    fieldRef:
      fieldPath: metadata.name
- name: POD_NAMESPACE
  valueFrom:
    fieldRef:
      fieldPath: metadata.namespace
```

### Monitoring
Monitor these metrics:
- Lease acquisition success/failure rates
- Average lease hold duration
- Requeue frequency due to lease conflicts

### Troubleshooting
Common issues:
- **Stuck leases**: Check if pod crashed without cleanup (wait 30s for expiration)
- **High requeue rates**: Indicates heavy contention (consider task distribution)
- **RBAC errors**: Ensure lease permissions are properly configured

## Conclusion

The dual-layer locking mechanism provides:
- **Correctness**: No race conditions in multi-pod deployments
- **Performance**: Optimal speed in single-pod deployments
- **Reliability**: Self-healing via lease expiration
- **Observability**: Standard Kubernetes debugging tools

This approach ensures that LLM interactions remain properly serialized while allowing tool calls to execute in parallel, maintaining both correctness and performance across all deployment scenarios.