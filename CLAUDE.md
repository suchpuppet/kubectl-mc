# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`kubectl-mc` is a kubectl plugin for multi-cluster Kubernetes operations using sig-multicluster standards. The goal is to make commands like `kubectl mc get pods` work across multiple clusters as naturally as they work on a single cluster.

**Status**: ✅ **Working POC** - Phase 1 core functionality is implemented and tested!

**What's Working:**
- ✅ ClusterProfile-based cluster discovery (sig-multicluster v1alpha1)
- ✅ Multi-cluster `get` command (pods, deployments, services)
- ✅ Result aggregation with CLUSTER column
- ✅ Interactive setup command for context mapping
- ✅ Parallel execution with concurrency control
- ✅ Error handling for partial cluster failures
- ✅ Unit tests with 74-90% coverage
- ✅ Tested on kind + OCM environment

**Community Validation**: Discussed with sig-multicluster leadership. There is interest in hosting this plugin in the sig-multicluster GitHub organization. The sig has focused on API development and recognizes the value of standardized user experience tooling.

**Phase 2 Design**: Automatic credential configuration using ClusterProfile properties to store exec plugin metadata. This eliminates the manual mapping file while preserving the user credentials model (exec plugins run locally using user's own AWS/GCP/Azure CLI credentials).

## Core Architecture

### Hub-and-Spoke Model
- **Hub Cluster**: Central cluster running sig-multicluster APIs (Cluster Profile API) for cluster discovery
- **Member Clusters**: Individual clusters discovered via the hub and accessed directly

### Authentication Model (Critical)
The plugin uses a **dual-credential model**:
1. **Hub Authentication**: User's kubeconfig context for hub cluster (discovery only)
2. **Cluster Authentication**: User's individual kubeconfig contexts for each member cluster (actual operations)

**NEVER use privileged service account credentials** - always operate with end-user credentials and RBAC permissions.

### Command Execution Flow
1. Parse kubectl-like command from user
2. Connect to hub cluster using user's hub context
3. Discover available clusters via sig-multicluster APIs (ClusterProfile, About, or Cluster Inventory)
4. Resolve kubeconfig contexts for each discovered cluster
5. Execute command on each cluster in parallel using user's credentials
6. Aggregate results and add CLUSTER column to output
7. Format output to match standard kubectl format

## Technical Stack

- **Language**: Go 1.25.0 (using idiomatic Go patterns)
- **Kubernetes Client**: client-go v0.34.2
- **CLI Framework**: cobra v1.10.1 (standard for kubectl plugins)
- **sig-multicluster**: Uses `multicluster.x-k8s.io/v1alpha1` ClusterProfile API
  - **IMPLEMENTED**: ClusterProfile-based discovery with dynamic client
  - **CRITICAL**: Does NOT use OCM-specific APIs (ManagedCluster)
  - Maintains vendor neutrality for sig-multicluster donation
- **Configuration**: YAML-based cluster mapping file at `~/.kube/kubectl-mc-clusters.yaml`
- **Testing**: Table-driven tests with fake dynamic clients
- **Cloud CLI Integration**: Design complete, not yet implemented (Phase 2)

## Implemented Project Structure

```
kubectl-mc/
├── main.go                 # Entry point
├── cmd/                    # CLI command implementations (cobra)
│   ├── root.go            # Root command and flags
│   ├── get.go             # Multi-cluster get command
│   └── setup.go           # Interactive setup command
├── pkg/
│   ├── discovery/         # ✅ Cluster discovery (ClusterProfile API)
│   │   ├── types.go
│   │   ├── clusterprofile.go
│   │   └── clusterprofile_test.go (74.3% coverage)
│   ├── executor/          # ✅ Multi-cluster command execution
│   │   ├── types.go
│   │   └── executor.go
│   ├── aggregator/        # ✅ Result aggregation and table formatting
│   │   └── table.go
│   ├── kubeconfig/        # ✅ Context mapping management
│   │   ├── types.go
│   │   ├── manager.go
│   │   └── manager_test.go (90.5% coverage)
│   └── client/            # ✅ Kubernetes client factory
│       └── factory.go
├── docs/                  # ✅ Comprehensive documentation
│   ├── architecture.md
│   ├── design-decisions.md
│   └── llm-context.md
├── test/                  # Test directories (structure ready)
├── Makefile              # ✅ Build automation
├── QUICKSTART.md         # ✅ Getting started guide
└── go.mod                # ✅ Go 1.25.0 with latest k8s.io libs
```

## Key Implementation Details

### ClusterProfile Discovery
- **Location**: `pkg/discovery/clusterprofile.go`
- Uses dynamic client to query `clusterprofiles.multicluster.x-k8s.io` resources
- Parses health from `status.conditions` (ControlPlaneHealthy)
- Extracts cluster name, display name, namespace, and Kubernetes version
- Returns vendor-neutral `ClusterInfo` struct

### Context Mapping
- **Location**: `pkg/kubeconfig/manager.go`
- Config file: `~/.kube/kubectl-mc-clusters.yaml`
- Maps ClusterProfile names to kubeconfig context names
- Persists to disk with YAML marshaling
- Thread-safe for concurrent access

### Parallel Execution
- **Location**: `pkg/executor/executor.go`
- Goroutines with semaphore for concurrency control (default: 10 concurrent)
- Context timeouts per cluster (default: 30s)
- Collects results via channels
- Graceful error handling - partial results on failures

### Result Aggregation
- **Location**: `pkg/aggregator/table.go`
- Formats output based on resource type (pods, deployments, services)
- Adds CLUSTER column to standard kubectl output
- Sorts by cluster, then namespace, then name
- Tab-separated values matching kubectl format

### Client Factory
- **Location**: `pkg/client/factory.go`
- Creates Kubernetes clients for specific contexts
- Returns dynamic, typed, and discovery clients
- Handles kubeconfig loading and overrides

## Development Commands

```bash
# Build
make build

# Install locally as kubectl plugin
make install

# Run unit tests
make test

# Run with coverage report
make test-coverage

# Format code
make fmt

# Lint
make vet

# Clean artifacts
make clean

# Test against your cluster
./kubectl-mc get pods -n test --hub-context kind-ocm-hub
```

## Implementation Status

### Phase 1 (Read-only operations) - IN PROGRESS
1. ✅ `kubectl mc get <resource>` - Get resources across clusters (pods, deployments, services)
2. ✅ `kubectl mc setup` - Interactive mapping setup between ClusterProfile names and kubeconfig contexts
3. ✅ Error handling for partial cluster failures with clear attribution
4. ✅ Parallel execution with configurable concurrency
5. ✅ Result aggregation with CLUSTER column
6. ⏳ `kubectl mc describe <resource> <name>` - Describe specific resource (TODO)
7. ⏳ `kubectl mc logs <pod>` - Get logs from pod (TODO)
8. ⏳ Basic cluster filtering (--clusters, --exclude flags exist but not fully implemented)
9. ⏳ More resource types (currently: pods, deployments, services)

**Potential Phase 1 Additions:**
- ⏳ `kubectl mc edit <resource>` - Safe write operation (opens editor, user reviews each cluster's manifest)
- ⏳ `kubectl mc setup --provider <cloud>` - Platform-specific helpers (design complete, not implemented)
- ⏳ Wildcard cluster filtering (`--clusters=prod-*`)

### Phase 2 (Configuration Management)
- Platform-specific credential helpers (aws, gcp, azure, ali)
- Advanced cluster filtering (labels, selectors, wildcards)
- Context validation and credential checking

### Phase 3 (Write operations with safety)
- `kubectl mc apply -f <file>` - With cluster selection and confirmation
- `kubectl mc delete <resource> <name>` - With confirmation prompt
- `kubectl mc patch <resource> <name>` - With dry-run preview
- `kubectl mc create <resource>` - With cluster targeting
- Dry-run mode, progressive rollout, audit logging

### Phase 4 (Enhanced UX)
- Cluster health indicators, color-coded output, progress indicators
- Interactive cluster selection, resource diff across clusters

## Critical Design Constraints

1. **Vendor Neutrality**: Use ONLY sig-multicluster standard APIs - this is NON-NEGOTIABLE. Do not use OCM-specific APIs (ManagedCluster), Rancher APIs, or any other solution-specific APIs
2. **sig-multicluster Standards**: All code must align with sig-multicluster patterns - this project is intended for donation to the sig-multicluster community
3. **Familiar UX**: Maintain kubectl command structure and output format (add CLUSTER column)
4. **User Credentials Only**: Never use privileged hub credentials for member cluster operations
5. **No Reconciliation**: Use discovery APIs but not reconciliation-based features
6. **Graceful Degradation**: Handle partial cluster failures elegantly - show partial results with clear error attribution
7. **Parallel Execution**: Execute commands across clusters concurrently with proper goroutine management
8. **OCM Orthogonal**: Works with OCM but doesn't require it; complements existing tools, doesn't compete

## Output Format

All multi-cluster output should add a CLUSTER column while maintaining kubectl's familiar table format:

```
NAMESPACE     NAME              CLUSTER        READY   STATUS    RESTARTS   AGE
default       nginx-abc123      us-west-1      1/1     Running   0          5d
default       nginx-def456      us-east-1      1/1     Running   0          3d
kube-system   coredns-xyz789    eu-central-1   1/1     Running   0          10d
```

## Kubeconfig Management: Two-Track Approach

The plugin must solve the "bootstrap problem" using two complementary approaches:

### Track 1: Manual Context Management (Universal)
- User manually configures kubeconfig contexts for each cluster using standard kubectl commands
- Works for any cluster type: cloud-managed, on-prem, custom, etc.
- `kubectl mc setup` discovers ClusterProfile resources and creates a mapping file linking profile names to context names
- Example mapping file (`~/.kube/kubectl-mc.yaml`):
  ```yaml
  clusters:
    - profile: "production-us-west"
      context: "prod-us-west-1"
    - profile: "production-eu-central"
      context: "prod-eu-1"
  ```

### Track 2: Platform-Specific Helpers (Convenience)
- Automated credential fetching for cloud-managed Kubernetes clusters
- `kubectl mc setup --provider aws|gcp|azure|ali`
- Executes cloud-specific CLI commands to fetch credentials:
  - **AWS EKS**: `aws eks update-kubeconfig --name <name> --region <region>`
  - **Google GKE**: `gcloud container clusters get-credentials <name> --region <region>`
  - **Azure AKS**: `az aks get-credentials --name <name> --resource-group <rg>`
  - **Alibaba ACK**: `aliyun cs cluster get-kubeconfig --cluster <id>`
- Requires:
  - Cloud-specific metadata in ClusterProfile resources (cluster name, region, resource group, etc.)
  - Cloud CLI tools installed and authenticated
  - Falls back to manual configuration if metadata unavailable or CLI tools missing
- Supports mixed environments: some clusters via platform helpers, others manual

### Hybrid Approach (Most Common)
Most organizations will use both:
1. Platform helpers for cloud-managed clusters
2. Manual configuration for on-prem or custom clusters
3. Single mapping file manages all clusters

## Relationship with Open Cluster Management (OCM)

**This project is orthogonal to OCM** - it complements but does not require OCM.

### Design Principle: Vendor Neutrality
Since this plugin is intended for donation to sig-multicluster, it **must not rely on OCM-specific APIs**. The plugin uses only standard sig-multicluster APIs (ClusterProfile, About, or Cluster Inventory) that work across different multi-cluster solutions.

### OCM Integration
While the plugin doesn't require OCM, it works naturally in OCM environments:
- OCM provides hub cluster infrastructure and cluster lifecycle management
- OCM manages cluster registration and health monitoring
- `kubectl-mc` provides the user-facing kubectl-like experience for resource operations
- Both use sig-multicluster standard APIs as the integration layer

### Differences from OCM Tools
- **clusteradm**: OCM's administrative CLI for cluster lifecycle (join, accept, addon management)
- **kubectl-mc**: User-facing tool for kubectl-style resource operations across clusters
- They complement each other: clusteradm manages clusters, kubectl-mc operates on resources

### OCM-Specific Alternative
An OCM-specific variant could be developed separately that:
- Directly queries `ManagedCluster` resources
- Leverages OCM-specific features (ManifestWork, Placement, etc.)
- Provides tighter OCM integration

This separation maintains vendor neutrality for sig-multicluster donation while allowing ecosystem-specific enhancements.

## Key Design Patterns

- **Plugin Architecture**: Follow kubectl plugin conventions (kubectl-mc binary name)
- **Vendor Neutrality**: Use only sig-multicluster standard APIs - no solution-specific code
- **Error Handling**: Graceful degradation with clear error attribution (which cluster failed and why)
- **Safety First**: Read-only operations first; write operations require safety mechanisms
- **Concurrency**: Parallel execution across clusters with proper goroutine management
- **Caching**: Consider caching cluster discovery results with appropriate TTL
- **Extensibility**: Design for future sig-multicluster API evolution
- **Scale**: Design to handle 10s to 100s of clusters
- **Mixed Environments**: Support heterogeneous clusters (cloud + on-prem, different auth methods)

## Error Handling Philosophy

Multi-cluster operations should gracefully handle partial failures:

```bash
Error: Failed to query 2 clusters: cluster-3 (connection timeout), cluster-7 (unauthorized)
Showing results from 8/10 clusters:

NAMESPACE     NAME              CLUSTER        READY   STATUS    RESTARTS   AGE
default       nginx-abc123      cluster-1      1/1     Running   0          5d
...
```

Key principles:
- Show partial results rather than failing completely
- Clearly attribute errors to specific clusters
- Provide actionable error messages (permission issues, network problems, etc.)

## Safety Mechanisms for Write Operations

Write operations must have safeguards:

```bash
# Requires explicit targeting or confirmation
$ kubectl mc delete deployment nginx
Error: Use --all-clusters to delete from all clusters, or --clusters to specify targets

# With explicit confirmation
$ kubectl mc delete deployment nginx --all-clusters
Delete deployment 'nginx' from 10 clusters? [y/N]:

# With cluster filtering
$ kubectl mc apply -f deployment.yaml --clusters=prod-us-*
```

Requirements:
- Confirmation prompts for destructive operations
- Dry-run mode available for all write operations
- Cluster filtering to limit blast radius
- Progressive rollout capabilities for large-scale changes

## Phase 2: Automatic Credential Configuration

Phase 2 will eliminate the manual mapping file by using ClusterProfile properties for exec plugin configuration.

### Key Design Principles

1. **User Credentials Model**: Exec plugins run locally using the user's own cloud CLI credentials (not stored secrets)
2. **ClusterProfile Properties**: Store exec plugin **metadata** in `status.properties`, not credentials
3. **No Hardcoded Logic**: kubectl-mc reads properties generically - OCM addon provides cloud-specific metadata
4. **Fallback Compatibility**: If properties not present, fall back to Phase 1 manual mapping

### ClusterProfile Properties Schema

```yaml
status:
  accessProviders:  # Connection info
  - name: kubeconfig
    cluster:
      server: https://ABC123.eks.amazonaws.com
      certificate-authority-data: LS0t...
  properties:      # Exec plugin metadata
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

### Implementation Approach

```go
// Generic exec plugin construction from properties
func buildExecConfig(props []Property) *api.ExecConfig {
    command := getProperty(props, "auth.exec.command")
    if command == "" {
        return nil  // No exec auth, fallback to mapping file
    }
    
    return &api.ExecConfig{
        APIVersion: getProperty(props, "auth.exec.apiVersion"),
        Command:    command,
        Args:       strings.Split(getProperty(props, "auth.exec.args"), ","),
        Env:        parseEnv(getProperty(props, "auth.exec.env")),
    }
}
```

### Why Properties, Not Secrets?

The KEP-4322 suggests using Secrets for kubeconfig data, but this doesn't work for user-credential-based exec plugins:

- **Problem**: Secrets assume static credentials (tokens/certs), not dynamic user credentials
- **Solution**: Properties store exec plugin **metadata** (command + args), not credentials
- **Benefit**: Same ClusterProfile works for all users - exec plugin uses their own local cloud credentials

### OCM Addon Responsibility

An OCM addon watches ManagedCluster resources and enriches ClusterProfile with properties:

```go
// Addon detects cluster type and populates properties
if managedCluster.Labels["vendor"] == "EKS" {
    clusterProfile.Status.Properties = []Property{
        {Name: "auth.exec.command", Value: "aws"},
        {Name: "auth.exec.args", 
         Value: "eks,get-token,--cluster-name," + clusterName + ",--region," + region},
        {Name: "auth.exec.apiVersion", Value: "client.authentication.k8s.io/v1beta1"},
    }
}
```

### Supported Cloud Providers

- **AWS EKS**: `aws eks get-token` (requires `aws configure` or IAM role)
- **GCP GKE**: `gke-gcloud-auth-plugin` (requires `gcloud auth login`)
- **Azure AKS**: `kubelogin` (requires `az login`)

kubectl-mc remains cloud-agnostic - it just reads properties generically.

## Development Considerations

When implementing:
1. **Vendor neutrality is non-negotiable** - use only sig-multicluster standard APIs
2. **User credentials always** - exec plugins use user's local cloud CLI credentials, never stored secrets
3. Start with read-only operations (get, describe, logs) before write operations
4. Implement robust error handling for partial cluster failures (show partial results)
5. Consider cluster health/reachability before operations
6. Log verbosely for troubleshooting multi-cluster issues
7. Always verify sig-multicluster API compatibility
8. Think about scale (10s to 100s of clusters)
9. Maintain backward compatibility with kubectl patterns
10. Follow standard Go project layout and kubectl plugin naming conventions
11. Design for mixed environments (different cloud providers, on-prem, various auth methods)
12. OCM users should benefit, but OCM must not be required
13. Consider credential security (never log or expose sensitive tokens)
14. **Phase 2**: Read exec plugin metadata from ClusterProfile properties generically, no hardcoded cloud logic

## Resources

- [Kubernetes sig-multicluster](https://github.com/kubernetes/community/tree/master/sig-multicluster)
- [kubectl Plugin Development](https://kubernetes.io/docs/tasks/extend-kubectl/kubectl-plugins/)
- [client-go Documentation](https://github.com/kubernetes/client-go)
