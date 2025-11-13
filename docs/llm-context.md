# LLM Assisted Development

This project is developed with assistance from AI coding assistants. This document provides comprehensive context for LLMs working on this codebase.

## Quick Context for LLMs

When working on kubectl-mc, use this prompt to get oriented:

```
PROJECT: kubectl-mc - Multi-cluster kubectl plugin

PURPOSE:
A kubectl plugin that extends kubectl's user experience to multi-cluster environments
following sig-multicluster standards. Commands like `kubectl mc get pods` should work
across multiple clusters as naturally as they work on a single cluster.

COMMUNITY CONTEXT:
- Discussed with sig-multicluster leadership
- Interest in hosting in sig-multicluster GitHub org
- Sig has focused on APIs; this addresses the UX gap
- Must be vendor-neutral (not OCM-specific, works with any sig-multicluster environment)

ARCHITECTURE:
1. Hub-and-Spoke Model:
   - Hub Cluster: Central cluster running sig-multicluster APIs
   - Member Clusters: Individual clusters discovered and accessed via the hub

2. Authentication Flow:
   - User authenticates to hub using their kubeconfig context (for discovery)
   - User authenticates to each member cluster using separate kubeconfig contexts
   - NO privileged service account credentials - always use end-user credentials

3. Command Execution Flow:
   a. Parse kubectl-like command from user
   b. Connect to hub cluster using user's hub context
   c. Discover available clusters via sig-multicluster APIs (ClusterProfile, About, or Cluster Inventory)
   d. Resolve kubeconfig contexts for each discovered cluster
   e. Execute command on each cluster in parallel using user's credentials
   f. Aggregate results and add CLUSTER column to output
   g. Format output to match standard kubectl format

TECHNICAL REQUIREMENTS:
- Language: Go 1.21+ (idiomatic Go patterns expected)
- Kubernetes Client: client-go library
- sig-multicluster: Use ONLY standard sig-multicluster APIs (vendor-neutral)
  - ClusterProfile API, About API, or Cluster Inventory API (TBD during implementation)
  - Do NOT use OCM-specific APIs (ManagedCluster) or other solution-specific APIs
- CLI Framework: cobra (standard for kubectl plugins)
- Output Formatting: Match kubectl's table formatting with additional CLUSTER column
- Cloud CLI Integration: aws, gcloud, az, aliyun for platform helpers

KEY DESIGN PATTERNS:
- Plugin Architecture: Follow kubectl plugin conventions (kubectl-mc binary)
- Error Handling: Graceful degradation if some clusters are unreachable
- Concurrency: Parallel execution across clusters with proper goroutine management
- Caching: Consider caching cluster discovery results with appropriate TTL
- Extensibility: Design for future sig-multicluster API evolution
- Safety First: Read-only operations first, write operations with safeguards later

KUBECONFIG MANAGEMENT - TWO-TRACK APPROACH:
Track 1: Manual Context Management (Universal)
- User manually configures kubeconfig contexts for each cluster
- `kubectl mc setup` creates mapping file between ClusterProfile names and contexts
- Works for any cluster type (cloud, on-prem, custom)

Track 2: Platform-Specific Helpers (Convenience)
- Automated credential fetching for cloud providers
- `kubectl mc setup --provider aws|gcp|azure|ali`
- Executes appropriate cloud CLI commands:
  - AWS: `aws eks update-kubeconfig --name <name> --region <region>`
  - GCP: `gcloud container clusters get-credentials <name> --region <region>`
  - Azure: `az aks get-credentials --name <name> --resource-group <rg>`
  - Ali: `aliyun cs cluster get-kubeconfig --cluster <id>`
- Requires cloud-specific metadata in ClusterProfile resources
- Falls back to manual if unavailable

OUTPUT FORMAT EXAMPLE:
```
NAMESPACE     NAME              CLUSTER        READY   STATUS    RESTARTS   AGE
default       nginx-abc123      us-west-1      1/1     Running   0          5d
default       nginx-def456      us-east-1      1/1     Running   0          3d
kube-system   coredns-xyz789    eu-central-1   1/1     Running   0          10d
```

IMPORTANT CONSTRAINTS:
1. Vendor Neutrality: Use ONLY sig-multicluster standard APIs, not OCM/Rancher/etc-specific APIs
2. sig-multicluster Standards: All code must align with sig-multicluster patterns
3. Upstream Intent: This is intended for donation to sig-multicluster community
4. User Credentials: Never use privileged hub credentials for member cluster operations
5. Familiar UX: Maintain kubectl command structure and output formats
6. No Reconciliation: Use discovery APIs but not reconciliation-based features
7. OCM Orthogonal: Works with OCM but doesn't require it; complements, doesn't compete

PHASED IMPLEMENTATION:

Phase 1 - Read-Only Operations:
- `kubectl mc get <resource>` - Get resources across clusters
- `kubectl mc describe <resource> <name>` - Describe specific resource
- `kubectl mc logs <pod>` - Get logs from pod (with cluster disambiguation)
- `kubectl mc setup` - Create mapping between ClusterProfile names and contexts (manual)
- Error handling for partial cluster failures
- Basic cluster filtering (--clusters, --exclude)

Phase 1 Potential Additions:
- `kubectl mc edit <resource> <name>` - Safe write operation (opens editor, user reviews)
- `kubectl mc setup --provider <cloud>` - Platform-specific helpers (experimental)

Phase 2 - Configuration Management:
- Platform-specific credential helpers (aws, gcp, azure, ali)
- Advanced cluster filtering (labels, selectors, wildcards)
- Context validation and credential checking

Phase 3 - Write Operations (With Safety):
- `kubectl mc apply -f <file>` - With cluster selection and confirmation
- `kubectl mc delete <resource> <name>` - With confirmation prompt
- `kubectl mc patch <resource> <name>` - With dry-run preview
- `kubectl mc create <resource>` - With cluster targeting
- Dry-run mode for all write operations
- Progressive rollout capabilities
- Write operation audit logging

Phase 4 - Enhanced UX:
- Cluster health indicators
- Color-coded output
- Progress indicators
- Interactive cluster selection
- Resource diff across clusters

DEVELOPMENT NOTES:
- Start with read-only operations (get, describe, logs)
- Implement robust error handling for partial cluster failures
- Consider cluster health/reachability before operations
- Log verbosely for troubleshooting multi-cluster issues
- Use standard Go project layout
- Follow kubectl plugin naming conventions
- Vendor neutrality is non-negotiable (no OCM/Rancher/etc-specific APIs)
- Think about mixed environments (cloud + on-prem clusters)
- Support both manual and automated kubeconfig management

ERROR HANDLING PHILOSOPHY:
- Graceful degradation: Show partial results if some clusters fail
- Clear error attribution: Which cluster failed and why
- Example output:
  ```
  Error: Failed to query 2 clusters: cluster-3 (connection timeout), cluster-7 (unauthorized)
  Showing results from 8/10 clusters:

  NAMESPACE     NAME              CLUSTER        READY   STATUS    RESTARTS   AGE
  default       nginx-abc123      cluster-1      1/1     Running   0          5d
  ...
  ```

SAFETY FOR WRITE OPERATIONS:
- Confirmation prompts for destructive operations (delete)
- Dry-run mode available for all write operations
- Cluster filtering required or explicit confirmation for multi-cluster writes
- Example:
  ```
  $ kubectl mc delete deployment nginx
  Error: Use --all-clusters to delete from all clusters, or --clusters to specify targets

  $ kubectl mc delete deployment nginx --all-clusters
  Delete deployment 'nginx' from 10 clusters? [y/N]:
  ```

FUTURE CONSIDERATIONS:
- Policy-based cluster selection
- RBAC aggregation and visualization
- Cost/usage reporting across clusters
- Cross-cluster resource dependencies
- Integration with GitOps workflows
```

## Guidelines for LLM Contributors

When working on this project:

1. **Always verify sig-multicluster API compatibility** - Check that any APIs used are standard sig-multicluster, not vendor-specific
2. **Maintain vendor neutrality** - No solution-specific APIs (OCM, Rancher, etc.)
3. **Consider multi-cluster failure scenarios** - Partial failures are normal, not exceptional
4. **Maintain backward compatibility with kubectl patterns** - Users should feel at home
5. **Document multi-cluster specific behaviors** - What's different from single-cluster kubectl?
6. **Think about scale** - 10s to 100s of clusters, not just 2-3
7. **OCM users should benefit from this** - But OCM should not be required
8. **Design for mixed environments** - Different auth methods, cloud + on-prem

## Project Structure

```
kubectl-mc/
├── cmd/
│   ├── root.go              # Root command, global flags
│   ├── get.go               # Get command (Phase 1)
│   ├── describe.go          # Describe command (Phase 1)
│   ├── logs.go              # Logs command (Phase 1)
│   ├── setup.go             # Setup command (Phase 1-2)
│   ├── apply.go             # Apply command (Phase 3)
│   └── delete.go            # Delete command (Phase 3)
├── pkg/
│   ├── discovery/
│   │   ├── discovery.go     # Cluster discovery interface
│   │   ├── clusterprofile.go # ClusterProfile implementation
│   │   ├── about.go         # About API implementation
│   │   └── cache.go         # Discovery caching
│   ├── executor/
│   │   ├── executor.go      # Parallel execution engine
│   │   ├── timeout.go       # Timeout handling
│   │   └── results.go       # Result collection
│   ├── aggregator/
│   │   ├── aggregator.go    # Result aggregation
│   │   ├── table.go         # Table formatting
│   │   └── json.go          # JSON/YAML formatting
│   ├── kubeconfig/
│   │   ├── manager.go       # Kubeconfig management
│   │   ├── mapping.go       # Cluster-to-context mapping
│   │   ├── aws.go           # AWS EKS helper
│   │   ├── gcp.go           # GCP GKE helper
│   │   ├── azure.go         # Azure AKS helper
│   │   └── alibaba.go       # Alibaba ACK helper
│   ├── client/
│   │   ├── factory.go       # Client factory
│   │   └── wrapper.go       # Client-go wrappers
│   └── config/
│       ├── types.go         # Configuration types
│       └── loader.go        # Config file loading
├── test/
│   ├── integration/         # Integration tests
│   └── fixtures/            # Test fixtures
└── docs/
    ├── architecture.md      # Detailed architecture
    ├── design-decisions.md  # Design rationale
    └── llm-context.md       # This file
```

## Code Style Guidelines

### Go Idioms

```go
// Use descriptive error messages with context
if err != nil {
    return fmt.Errorf("failed to discover clusters from hub: %w", err)
}

// Use structured logging
logger.Info("discovered clusters",
    "count", len(clusters),
    "hub", hubContext,
    "duration", time.Since(start))

// Prefer table-driven tests
func TestClusterDiscovery(t *testing.T) {
    tests := []struct {
        name    string
        hub     string
        want    []string
        wantErr bool
    }{
        // test cases
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test logic
        })
    }
}
```

### Kubectl Plugin Conventions

```go
// Binary name: kubectl-mc (hyphenated)
// Invoked as: kubectl mc <command>

// Support --kubeconfig flag
rootCmd.PersistentFlags().String("kubeconfig", "", "path to kubeconfig file")

// Support --context for hub
rootCmd.PersistentFlags().String("hub-context", "", "hub cluster context")

// Support --namespace
rootCmd.PersistentFlags().StringP("namespace", "n", "", "namespace scope")

// Support output formats
rootCmd.PersistentFlags().StringP("output", "o", "", "output format (json|yaml|wide)")
```

## Testing Strategy

### Unit Tests

- Test each package independently
- Mock Kubernetes clients
- Test error conditions thoroughly

### Integration Tests

- Require real clusters (or kind clusters)
- Test full command execution
- Verify output formatting

### E2E Tests

- Test complete user workflows
- Multiple cluster scenarios
- Platform helper testing (requires cloud access)

## Common Patterns

### Parallel Execution

```go
func executeOnClusters(clusters []Cluster, fn func(Cluster) Result) []Result {
    results := make(chan Result, len(clusters))
    var wg sync.WaitGroup
    
    // Limit concurrency
    sem := make(chan struct{}, maxConcurrency)
    
    for _, cluster := range clusters {
        wg.Add(1)
        go func(c Cluster) {
            defer wg.Done()
            sem <- struct{}{}        // Acquire
            defer func() { <-sem }() // Release
            
            ctx, cancel := context.WithTimeout(context.Background(), timeout)
            defer cancel()
            
            results <- fn(c)
        }(cluster)
    }
    
    wg.Wait()
    close(results)
    
    // Collect results
    var collected []Result
    for r := range results {
        collected = append(collected, r)
    }
    return collected
}
```

### Error Collection

```go
type MultiClusterError struct {
    Errors map[string]error // cluster -> error
}

func (e *MultiClusterError) Error() string {
    var errs []string
    for cluster, err := range e.Errors {
        errs = append(errs, fmt.Sprintf("%s: %v", cluster, err))
    }
    return fmt.Sprintf("failed to query %d clusters: %s",
        len(e.Errors), strings.Join(errs, ", "))
}
```

## References

- [Architecture Documentation](architecture.md)
- [Design Decisions](design-decisions.md)
- [kubectl Plugin Development Guide](https://kubernetes.io/docs/tasks/extend-kubectl/kubectl-plugins/)
- [client-go Documentation](https://github.com/kubernetes/client-go)
- [sig-multicluster Community](https://github.com/kubernetes/community/tree/master/sig-multicluster)
