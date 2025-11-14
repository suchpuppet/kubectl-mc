# kubectl-mc

[![CI](https://github.com/suchpuppet/kubectl-mc/actions/workflows/ci.yaml/badge.svg)](https://github.com/suchpuppet/kubectl-mc/actions/workflows/ci.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/suchpuppet/kubectl-mc)](https://goreportcard.com/report/github.com/suchpuppet/kubectl-mc)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

A kubectl plugin for seamless multi-cluster operations using sig-multicluster standards.

## Overview

`kubectl-mc` (multi-cluster) provides a familiar kubectl-like user experience for managing resources across multiple Kubernetes clusters. Instead of manually switching contexts and aggregating results, users can interact with multiple clusters as naturally as they would with a single cluster.

### Example

```bash
# Traditional single-cluster command
kubectl get pods

# Multi-cluster equivalent
kubectl mc get pods
```

The plugin automatically discovers available clusters via the hub, executes the command across all clusters using the user's own credentials, and aggregates results in the familiar kubectl output format with an additional cluster column.

## Architecture

### Design Principles

1. **Familiar UX**: Maintain kubectl's command structure and output format
2. **User Credentials**: Use end-user credentials, not privileged hub credentials
3. **sig-multicluster Native**: Leverage sig-multicluster APIs and patterns
4. **Discovery-Based**: Dynamically discover clusters via hub cluster
5. **Vendor Neutral**: Works with any sig-multicluster-compliant environment

### How It Works

1. Connect to hub cluster using user's kubeconfig context
2. Discover available clusters via sig-multicluster APIs (ClusterProfile, About, or Cluster Inventory)
3. Map discovered clusters to user's kubeconfig contexts
4. Execute command on each cluster in parallel using user's credentials
5. Aggregate results and add CLUSTER column to output

**For detailed architecture, see [docs/architecture.md](docs/architecture.md)**

## sig-multicluster Integration

This plugin is designed for donation to the [Kubernetes sig-multicluster](https://github.com/kubernetes/community/tree/master/sig-multicluster) community. The proposal has been discussed with sig-multicluster leadership, and there is interest in hosting this plugin in the sig-multicluster GitHub organization.

### Key Principles

1. **Vendor Neutrality**: Uses only sig-multicluster standard APIs (ClusterProfile, About, or Cluster Inventory)
2. **Standards Compliance**: Follows sig-multicluster patterns and conventions
3. **Complementary**: Works alongside existing multi-cluster solutions (OCM, Rancher, etc.) without requiring them
4. **User-Focused**: Provides kubectl-familiar UX for day-to-day operations

### Upstream Goals

This proof-of-concept is intended for:
1. Validation of the multi-cluster UX approach
2. Community feedback and iteration
3. Donation to sig-multicluster for official hosting
4. Reference implementation of user-facing sig-multicluster tooling

**For design rationale and OCM relationship, see [docs/design-decisions.md](docs/design-decisions.md)**

## Features

### Phase 1: Core Functionality (Read-Only) ✅
- ✅ Hub cluster connection and authentication
- ✅ Cluster discovery via ClusterProfile API (sig-multicluster standard)
- ✅ Basic resource operations (`kubectl mc get pods/deployments/services`)
- ✅ Result aggregation with CLUSTER column
- ✅ Kubeconfig context mapping (manual setup via config file)
- ✅ Error handling for partial cluster failures
- ✅ Parallel execution across clusters with concurrency control
- ✅ `kubectl mc setup` - Interactive cluster-to-context mapping
- ✅ `kubectl mc describe <resource>` - Describe resources across clusters
- [ ] `kubectl mc logs <pod>` - Get logs with cluster disambiguation
- [ ] Basic cluster filtering (`--clusters`, `--exclude`)

**Potential Phase 1 Additions:**
- [ ] `kubectl mc edit <resource>` - Multiplexed edit across clusters (with safety checks)
- [ ] Wildcard cluster filtering (`--clusters=prod-*`)

### Phase 2: Automatic Credential Configuration (Planned)

**Goal**: Eliminate manual mapping file by using ClusterProfile properties to dynamically construct exec plugin configurations.

- [ ] **ClusterProfile Properties Integration**:
  - [ ] Read `auth.exec.*` properties from ClusterProfile resources
  - [ ] Dynamically construct exec plugin configurations on-the-fly
  - [ ] Support for AWS EKS (`aws eks get-token`)
  - [ ] Support for GCP GKE (`gke-gcloud-auth-plugin`)
  - [ ] Support for Azure AKS (`kubelogin`)
  - [ ] Generic exec plugin property parsing

- [ ] **OCM Addon for ClusterProfile Enrichment**:
  - [ ] Addon watches ManagedCluster resources
  - [ ] Automatically populates ClusterProfile properties based on cluster type
  - [ ] Detects cloud provider from ManagedCluster labels/annotations
  - [ ] Generates exec plugin metadata for each cluster

- [ ] **User Credentials Model**:
  - [ ] Exec plugins use user's local cloud CLI credentials
  - [ ] No secrets storage required
  - [ ] Leverages existing `aws configure`, `gcloud auth login`, `az login`
  - [ ] Fallback to Phase 1 manual mapping if properties not present

**Example ClusterProfile with Properties**:
```yaml
status:
  accessProviders:
  - name: kubeconfig
    cluster:
      server: https://ABC123.eks.amazonaws.com
      certificate-authority-data: LS0t...
  properties:
  - name: auth.exec.command
    value: "aws"
  - name: auth.exec.args
    value: "eks,get-token,--cluster-name,production,--region,us-east-1"
  - name: auth.exec.apiVersion
    value: "client.authentication.k8s.io/v1beta1"
  - name: cloud.provider
    value: "aws"
  - name: cluster.type
    value: "eks"
```

See [docs/architecture.md](docs/architecture.md#phase-2-automatic-credential-configuration) for detailed design.

### Phase 3: Write Operations (With Safety Mechanisms)
- [ ] Write operations with safety mechanisms:
  - [ ] `kubectl mc apply -f <file>` - Apply with cluster selection
  - [ ] `kubectl mc delete <resource> <name>` - Requires confirmation prompt
  - [ ] `kubectl mc patch <resource> <name>` - With dry-run preview
  - [ ] `kubectl mc create <resource>` - With cluster targeting
- [ ] Cluster filtering for write operations (e.g., `--clusters=prod-*`, `--exclude=staging`)
- [ ] Dry-run mode for all write operations
- [ ] Progressive rollout (apply to subset, then all)
- [ ] Write operation audit logging
- [ ] Cross-cluster resource selection (labels, namespaces)
- [ ] Parallel execution with configurable concurrency

### Phase 4: Enhanced UX
- [ ] Cluster health indicators in output
- [ ] Color-coded multi-cluster output
- [ ] Progress indicators for long-running operations
- [ ] Interactive cluster selection
- [ ] Resource diff across clusters

## Quick Start

```bash
# 1. Build and install
make install

# 2. Enable ClusterProfile feature gate (if using OCM)
kubectl patch clustermanager cluster-manager --type=merge \
  -p '{"spec":{"registrationConfiguration":{"featureGates":[{"feature":"ClusterProfile","mode":"Enable"}]}}}'

# 3. Setup cluster-to-context mappings
kubectl mc setup --hub-context <your-hub-context>

# 4. Run multi-cluster commands
kubectl mc get pods -n default
kubectl mc get deployments --all-namespaces
```

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/suchpuppet/kubectl-mc.git
cd kubectl-mc

# Build and install
make install

# Verify installation
kubectl plugin list | grep kubectl-mc
```

### Manual Installation

```bash
# Build the binary
make build

# Copy to PATH
sudo cp kubectl-mc /usr/local/bin/kubectl-mc

# Or to personal bin
mkdir -p ~/bin
cp kubectl-mc ~/bin/kubectl-mc
export PATH="$HOME/bin:$PATH"
```

### Using Go Install

```bash
# Coming soon
go install github.com/suchpuppet/kubectl-mc@latest
```

## Usage

### Prerequisites

1. **Hub cluster** with ClusterProfile resources (sig-multicluster standard)
2. **Kubeconfig contexts** for hub and spoke clusters
3. **RBAC permissions** to list ClusterProfiles on hub and query resources on spoke clusters

### Setup Cluster Mappings

**Option 1: Interactive Setup**

```bash
# Discovers clusters and prompts for context names
kubectl mc setup --hub-context kind-ocm-hub

# Or use current context
kubectl config use-context kind-ocm-hub
kubectl mc setup
```

**Option 2: Manual Configuration**

Create `~/.kube/kubectl-mc-clusters.yaml`:

```yaml
apiVersion: kubectl-mc.k8s.io/v1alpha1
kind: ClusterMapping
hubContext: kind-ocm-hub
clusters:
- name: cluster1
  context: kind-cluster1
  namespace: open-cluster-management
- name: cluster2
  context: kind-cluster2
  namespace: open-cluster-management
```

### Basic Commands (Working Now)

```bash
# Get pods across all clusters
kubectl mc get pods -n default
kubectl mc get pods --all-namespaces

# Get deployments
kubectl mc get deployments -n default

# Get services
kubectl mc get services -n kube-system

# Specify hub context explicitly
kubectl mc get pods --hub-context kind-ocm-hub -n test

# Describe resources across clusters
kubectl mc describe pod nginx -n default
kubectl mc describe deployment my-app
kubectl mc describe service frontend -n production
```

**Example Output (get):**
```
$ kubectl mc get pods -n test
Discovered 2 cluster(s)
NAMESPACE   NAME                    CLUSTER      READY   STATUS    RESTARTS   AGE
test        nginx-bc7b4f464-npn2t   ocm-spoke1   1/1     Running   0          ---
test        nginx-bc7b4f464-24g52   ocm-spoke2   1/1     Running   0          ---
```

**Example Output (describe):**
```
$ kubectl mc describe pod nginx -n default
Discovered 2 cluster(s)

CLUSTER: ocm-spoke1
--------------------------------------------------------------------------------
Name:         nginx
Namespace:    default
Labels:       app=nginx
Annotations:  <none>

Spec:
  containers: [...]
  ...

Status:
  phase: Running
  ...

================================================================================

CLUSTER: ocm-spoke2
--------------------------------------------------------------------------------
Name:         nginx
Namespace:    default
Labels:       app=nginx
Annotations:  <none>

Spec:
  containers: [...]
  ...

Status:
  phase: Running
  ...
```

### Planned Commands (Not Yet Implemented)

```bash
# Logs (Phase 1)
kubectl mc logs nginx-abc123

# Cluster filtering (Phase 1)
kubectl mc get pods --clusters=prod-*
kubectl mc get pods --exclude=staging

# Write operations (Phase 3)
kubectl mc apply -f deployment.yaml --clusters=prod-*
kubectl mc delete deployment nginx --all-clusters
```

**For detailed usage and examples, see [docs/architecture.md](docs/architecture.md)**

## Development

### Prerequisites

- Go 1.23+
- kubectl installed
- Access to a Kubernetes cluster with sig-multicluster APIs

### Building

```bash
# Clone the repository
git clone https://github.com/suchpuppet/kubectl-mc.git
cd kubectl-mc

# Build the plugin
make build

# Install locally
make install
```

### Testing

```bash
# Run unit tests
make test

# Run with coverage report
make test-coverage

# Test specific packages
go test -v ./pkg/discovery/...
go test -v ./pkg/kubeconfig/...
```

**Current Test Coverage:**
- `pkg/discovery`: 74.3% coverage (ClusterProfile discovery logic)
- `pkg/kubeconfig`: 90.5% coverage (context mapping management)
- Total: 10 tests, all passing ✅

**Integration Testing:**

Test against your local kind + OCM setup:
```bash
# Ensure your clusters are running
kubectl get clusterprofiles -A --context kind-ocm-hub

# Test the plugin
./kubectl-mc get pods -n test --hub-context kind-ocm-hub
```

### Project Structure

```
kubectl-mc/
├── cmd/                    # CLI command implementations
├── pkg/
│   ├── discovery/         # Cluster discovery logic
│   ├── executor/          # Multi-cluster command execution
│   ├── aggregator/        # Result aggregation and formatting
│   ├── kubeconfig/        # Kubeconfig management
│   └── client/            # Kubernetes client wrappers
├── test/                  # Integration tests
└── docs/                  # Additional documentation
```

## Contributing

This project follows sig-multicluster contribution guidelines. Contributions are welcome!

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

### Code Standards

- Follow standard Go conventions and idioms
- Write tests for new functionality
- Document public APIs
- Ensure sig-multicluster pattern compliance

## Documentation

- **[Architecture](docs/architecture.md)** - Detailed technical architecture, component design, and data flows
- **[Design Decisions](docs/design-decisions.md)** - Rationale for key architectural and strategic decisions
- **[LLM Context](docs/llm-context.md)** - Comprehensive context for AI-assisted development

## Resources

- [Kubernetes sig-multicluster](https://github.com/kubernetes/community/tree/master/sig-multicluster)
- [kubectl Plugin Development](https://kubernetes.io/docs/tasks/extend-kubectl/kubectl-plugins/)
- [client-go Documentation](https://github.com/kubernetes/client-go)

## License

This project is governed by the Apache 2 License. See [LICENSE](LICENSE) file for details.

## Status

✅ **Proof of Concept - Working!** ✅

The POC successfully demonstrates:
- ✅ ClusterProfile-based cluster discovery (sig-multicluster standard)
- ✅ Multi-cluster resource queries (`get pods`, `get deployments`, `get services`)
- ✅ Result aggregation with CLUSTER column
- ✅ Parallel execution across clusters
- ✅ Graceful error handling for partial failures
- ✅ Unit tests with 74-90% coverage on core packages
- ✅ Tested on kind + OCM setup

**Current Phase:** Phase 1 (Read-only operations)

Not yet ready for production use. Gathering community feedback and refining the approach.

## Contact

For questions or discussions about this project, please reach out via:
- GitHub Issues
- Kubernetes sig-multicluster Slack channel
- sig-multicluster mailing list
