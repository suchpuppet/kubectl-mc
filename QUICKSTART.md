# kubectl-mc Quick Start

This guide helps you get started with kubectl-mc in your local OCM environment.

## Prerequisites

- Go 1.23+
- OCM hub cluster with ClusterProfile feature gate enabled
- Registered spoke clusters
- kubectl configured with contexts for hub and spoke clusters

## Build the Plugin

```bash
# Build the binary
make build

# Or install to GOPATH/bin
make install
```

## Setup Cluster Mappings

Create the mapping file at `~/.kube/kubectl-mc-clusters.yaml`:

```yaml
apiVersion: kubectl-mc.k8s.io/v1alpha1
kind: ClusterMapping
hubContext: kind-ocm-hub
clusters:
- name: ocm-spoke1
  context: kind-ocm-spoke1
  namespace: open-cluster-management
- name: ocm-spoke2
  context: kind-ocm-spoke2
  namespace: open-cluster-management
```

Or use the interactive setup command:

```bash
./kubectl-mc setup --hub-context kind-ocm-hub
```

## Usage Examples

### Get pods across all clusters

```bash
./kubectl-mc get pods --all-namespaces --hub-context kind-ocm-hub
```

### Get pods in a specific namespace

```bash
./kubectl-mc get pods -n test --hub-context kind-ocm-hub
```

### Get deployments

```bash
./kubectl-mc get deployments -n default --hub-context kind-ocm-hub
```

### Get services

```bash
./kubectl-mc get services -n kube-system --hub-context kind-ocm-hub
```

## Example Output

```
$ ./kubectl-mc get pods -n test --hub-context kind-ocm-hub
Discovered 2 cluster(s)
NAMESPACE       NAME                    CLUSTER         READY   STATUS    RESTARTS        AGE
test            nginx-bc7b4f464-npn2t   ocm-spoke1      1/1     Running   0               ---
test            nginx-bc7b4f464-24g52   ocm-spoke2      1/1     Running   0               ---
```

## Testing

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run specific package tests
go test -v ./pkg/discovery/...
```

## Development

```bash
# Format code
make fmt

# Run linters
make vet

# Clean build artifacts
make clean

# Tidy dependencies
make tidy
```

## Troubleshooting

### No clusters discovered

Verify ClusterProfile feature gate is enabled:

```bash
kubectl get clusterprofiles -A --context kind-ocm-hub
```

If empty, enable the feature gate on the hub:

```bash
kubectl patch clustermanager cluster-manager --type=merge --context kind-ocm-hub \
  -p '{"spec":{"registrationConfiguration":{"featureGates":[{"feature":"ClusterProfile","mode":"Enable"}]}}}'
```

### Context not found errors

Verify your kubeconfig has the correct contexts:

```bash
kubectl config get-contexts
```

Update the mapping file with the correct context names.

### Permission denied errors

Ensure your user has appropriate RBAC permissions on both hub and spoke clusters:

```bash
# Check hub access
kubectl auth can-i list clusterprofiles --context kind-ocm-hub -n open-cluster-management

# Check spoke access  
kubectl auth can-i list pods --context kind-ocm-spoke1 -n test
```

## Next Steps

- See [README.md](README.md) for full documentation
- See [docs/architecture.md](docs/architecture.md) for technical details
- See [docs/design-decisions.md](docs/design-decisions.md) for design rationale
