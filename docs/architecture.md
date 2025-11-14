# Architecture

This document provides detailed technical architecture for kubectl-mc.

## Design Principles

1. **Familiar UX**: Maintain kubectl's command structure and output format
2. **User Credentials**: Use end-user credentials, not privileged hub credentials
3. **sig-multicluster Native**: Leverage sig-multicluster APIs and patterns
4. **Discovery-Based**: Dynamically discover clusters via hub cluster
5. **Context-Aware**: Integrate with kubeconfig contexts for authentication

## High-Level Flow

```
┌─────────────────────────────────────────────────────────────────┐
│ User runs: kubectl mc get pods                                   │
└─────────────────┬───────────────────────────────────────────────┘
                  │
                  v
┌─────────────────────────────────────────────────────────────────┐
│ 1. Connect to Hub Cluster (using user's hub kubeconfig context) │
└─────────────────┬───────────────────────────────────────────────┘
                  │
                  v
┌─────────────────────────────────────────────────────────────────┐
│ 2. Discover Available Clusters                                   │
│    - Query Cluster Profile API or equivalent sig-multicluster   │
│      resources to enumerate accessible clusters                 │
└─────────────────┬───────────────────────────────────────────────┘
                  │
                  v
┌─────────────────────────────────────────────────────────────────┐
│ 3. Resolve Kubeconfig Contexts                                   │
│    - Map discovered clusters to user's kubeconfig contexts      │
│    - Validate user has credentials for each cluster             │
└─────────────────┬───────────────────────────────────────────────┘
                  │
                  v
┌─────────────────────────────────────────────────────────────────┐
│ 4. Execute Command on Each Cluster                               │
│    - Run `kubectl get pods` (or equivalent API call) per cluster│
│    - Use user's credentials from their kubeconfig               │
└─────────────────┬───────────────────────────────────────────────┘
                  │
                  v
┌─────────────────────────────────────────────────────────────────┐
│ 5. Aggregate and Format Results                                  │
│    - Combine results from all clusters                          │
│    - Add CLUSTER column to output                               │
│    - Maintain familiar kubectl output format                    │
└─────────────────────────────────────────────────────────────────┘
```

## Component Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│                        kubectl-mc CLI                             │
│                         (cobra app)                               │
└────────────┬────────────────────────────────────────────┬────────┘
             │                                             │
             v                                             v
┌────────────────────────┐                    ┌────────────────────┐
│   Discovery Package    │                    │  Executor Package  │
│                        │                    │                    │
│ - Connect to hub       │                    │ - Parallel exec    │
│ - Query ClusterProfile │                    │ - Timeout handling │
│ - Cache results        │                    │ - Error collection │
└────────┬───────────────┘                    └──────────┬─────────┘
         │                                               │
         v                                               v
┌────────────────────────┐                    ┌────────────────────┐
│ Kubeconfig Package     │                    │ Aggregator Package │
│                        │                    │                    │
│ - Context resolution   │                    │ - Merge results    │
│ - Mapping file         │                    │ - Format output    │
│ - Platform helpers     │                    │ - Add CLUSTER col  │
└────────┬───────────────┘                    └──────────┬─────────┘
         │                                               │
         v                                               v
┌────────────────────────────────────────────────────────────────────┐
│                      Client Package                                │
│                   (client-go wrappers)                             │
│                                                                    │
│  - REST client factory                                             │
│  - Dynamic client support                                          │
│  - Typed client support                                            │
└────────────────────────────────────────────────────────────────────┘
```

## Authentication Model

The plugin uses a dual-credential model to ensure users operate with their own RBAC permissions.

### Hub Authentication

**Purpose**: Cluster discovery only

**Mechanism**:
- User's kubeconfig context for the hub cluster
- Standard kubectl authentication (certificates, tokens, etc.)
- Read-only access to ClusterProfile resources

**Example**:
```bash
# User has hub context configured
kubectl config use-context my-hub-cluster

# Plugin uses this context to discover clusters
kubectl mc get pods
```

### Cluster Authentication

**Purpose**: Actual resource operations

**Mechanism**:
- User's individual kubeconfig contexts for each member cluster
- Mapping file: `~/.kube/kubectl-mc-clusters.yaml`
- Each cluster operation uses the mapped context

**Example Mapping File**:
```yaml
apiVersion: kubectl-mc.k8s.io/v1alpha1
kind: ClusterMapping
clusters:
- name: prod-us-west-1    # ClusterProfile name
  context: eks-us-west    # kubeconfig context name
- name: prod-eu-central-1
  context: eks-eu-central
- name: on-prem-dc1
  context: on-prem-1
```

### Security Benefits

1. **No Privileged Credentials**: Never use hub service account credentials for member cluster operations
2. **User-Level RBAC**: Each user operates with their own permissions
3. **Audit Trail**: Actions are attributable to individual users
4. **Least Privilege**: Users only access what they're authorized for

## Discovery Mechanisms

### sig-multicluster APIs

The plugin will use standard sig-multicluster APIs for cluster discovery. The exact API will be determined during implementation based on stability and community preference.

#### Option 1: ClusterProfile API

```yaml
apiVersion: multicluster.x-k8s.io/v1alpha1
kind: ClusterProfile
metadata:
  name: prod-us-west-1
spec:
  clusterName: prod-us-west-1
  displayName: "Production US West"
  cloudProvider: aws
  region: us-west-2
  # Additional metadata for platform helpers
status:
  conditions:
  - type: Ready
    status: "True"
```

#### Option 2: About API

Alternative cluster discovery mechanism providing cluster metadata.

#### Option 3: Cluster Inventory API

Another potential source for cluster enumeration.

### Discovery Caching

To minimize hub cluster load and improve performance:

```go
type ClusterCache struct {
    clusters []Cluster
    lastFetch time.Time
    ttl       time.Duration  // Default: 5 minutes
}
```

Cache invalidation triggers:
- TTL expiration
- Explicit refresh (`kubectl mc --refresh`)
- Setup command execution

## Execution Model

### Parallel Execution

Commands execute across clusters in parallel with proper concurrency control:

```go
type ExecutorConfig struct {
    MaxConcurrency int           // Default: 10
    Timeout        time.Duration // Default: 30s per cluster
    ContinueOnError bool         // Default: true
}
```

### Error Handling

```go
type ClusterResult struct {
    ClusterName string
    Success     bool
    Data        interface{}
    Error       error
}

type AggregatedResult struct {
    Results []ClusterResult
    Summary ResultSummary
}

type ResultSummary struct {
    Total      int
    Successful int
    Failed     int
    Errors     map[string]error  // cluster -> error
}
```

### Timeout Strategy

```
Per-cluster timeout: 30s (configurable)
Total operation timeout: max(60s, num_clusters * 5s)

Example: 20 clusters
  Per-cluster: 30s
  Total: max(60s, 100s) = 100s
```

## Output Formatting

### Table Format

Standard kubectl table format with additional CLUSTER column:

```
NAMESPACE     NAME              CLUSTER           READY   STATUS    RESTARTS   AGE
default       nginx-abc123      prod-us-west-1    1/1     Running   0          5d
default       nginx-def456      prod-eu-central-1 1/1     Running   0          3d
kube-system   coredns-xyz789    on-prem-dc1       1/1     Running   0          10d
```

### JSON/YAML Format

Multi-cluster output wraps individual cluster results:

```json
{
  "apiVersion": "kubectl-mc.k8s.io/v1alpha1",
  "kind": "MultiClusterResult",
  "clusters": [
    {
      "cluster": "prod-us-west-1",
      "items": [
        { "metadata": { "name": "nginx-abc123" }, ... }
      ]
    },
    {
      "cluster": "prod-eu-central-1",
      "items": [
        { "metadata": { "name": "nginx-def456" }, ... }
      ]
    }
  ],
  "summary": {
    "total": 2,
    "successful": 2,
    "failed": 0
  }
}
```

## Platform-Specific Credential Helpers

For cloud-managed Kubernetes clusters, the plugin can automate credential fetching:

| Provider | CLI Command Used | Metadata Required |
|----------|------------------|-------------------|
| AWS EKS  | `aws eks update-kubeconfig --name <name> --region <region>` | cluster name, region |
| Google GKE | `gcloud container clusters get-credentials <name> --region <region> --project <project>` | cluster name, region, project |
| Azure AKS | `az aks get-credentials --name <name> --resource-group <rg>` | cluster name, resource group |
| Alibaba ACK | `aliyun cs cluster get-kubeconfig --cluster <id>` | cluster ID |

### Helper Implementation Flow

```
1. Discover clusters from hub (ClusterProfile resources)
2. For each cluster with cloud provider metadata:
   a. Check if cloud CLI is available
   b. Extract required metadata (name, region, etc.)
   c. Execute cloud CLI command to fetch credentials
   d. Add context to kubeconfig
   e. Update mapping file
3. For clusters without metadata:
   a. Prompt user for manual context mapping
   b. Update mapping file
```

### Fallback Behavior

```
IF cloud CLI not installed:
  WARN: "AWS CLI not found. Install it or configure context manually."
  FALLBACK: Manual context mapping

IF cloud CLI not authenticated:
  WARN: "Not authenticated to AWS. Run 'aws configure' or set up manually."
  FALLBACK: Manual context mapping

IF metadata missing from ClusterProfile:
  INFO: "Cloud metadata not available for cluster-3"
  FALLBACK: Manual context mapping
```

## Configuration Files

### Mapping File

Location: `~/.kube/kubectl-mc-clusters.yaml`

```yaml
apiVersion: kubectl-mc.k8s.io/v1alpha1
kind: ClusterMapping
hubContext: my-hub-cluster  # Optional: default hub context
clusters:
- name: prod-us-west-1
  context: eks-us-west
  cloudProvider: aws        # Optional: for platform helper support
  region: us-west-2         # Optional: for platform helper support
- name: prod-eu-central-1
  context: eks-eu-central
  cloudProvider: aws
  region: eu-central-1
- name: on-prem-dc1
  context: on-prem-1
  # No cloud metadata - manually managed
```

### Plugin Configuration

Location: `~/.kube/kubectl-mc-config.yaml`

```yaml
apiVersion: kubectl-mc.k8s.io/v1alpha1
kind: Config
discovery:
  cacheTTL: 5m
  api: clusterprofile  # or 'about' or 'inventory'
execution:
  maxConcurrency: 10
  timeout: 30s
  continueOnError: true
output:
  colorize: true
  showClusterColumn: true
```

## Data Flow Example: `kubectl mc get pods`

```
1. Parse command: resource=pods, namespace=default

2. Load configuration:
   - Read hub context from mapping file or current context
   - Read execution settings from config file

3. Discover clusters:
   - Check cache (valid for 5m)
   - If stale: query hub for ClusterProfile resources
   - Update cache

4. Resolve contexts:
   - Load mapping file
   - For each discovered cluster, find corresponding context
   - Skip clusters without valid context

5. Execute in parallel (max 10 concurrent):
   For each cluster:
     a. Create client from context
     b. Call GET /api/v1/namespaces/default/pods
     c. Collect results or error
     d. Timeout after 30s

6. Aggregate results:
   - Merge pod lists from all clusters
   - Add cluster name to each item
   - Sort by cluster, then namespace, then name

7. Format output:
   - Detect output format (table, json, yaml)
   - Add CLUSTER column to table format
   - Print to stdout

8. Print summary (if errors):
   "Warning: Failed to query 2/10 clusters. Run with -v for details."
```

## Scaling Considerations

### Performance Targets

- **Discovery**: < 1s for cached, < 5s for fresh
- **Execution**: < 30s for 10 clusters, < 120s for 50 clusters
- **Memory**: < 100MB baseline, + 10MB per cluster

### Concurrency Tuning

```bash
# Default: 10 concurrent
kubectl mc get pods

# High concurrency for fast clusters
kubectl mc get pods --concurrency=50

# Low concurrency for slow networks
kubectl mc get pods --concurrency=3
```

### Filtering for Performance

```bash
# Target specific clusters only
kubectl mc get pods --clusters=prod-*

# Exclude clusters
kubectl mc get pods --exclude=dev-*,test-*
```

## Future Architecture Enhancements

- **gRPC-based aggregation**: For very large cluster counts
- **Streaming results**: Show results as they arrive
- **Cluster health checks**: Skip unhealthy clusters automatically
- **Smart routing**: Prefer regional hubs for geo-distributed clusters
- **Plugin system**: Allow custom aggregation strategies
