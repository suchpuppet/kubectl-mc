# Design Decisions

This document explains the key architectural and strategic decisions made for kubectl-mc.

## Why Not OCM-Specific?

While this project uses OCM infrastructure for development, the plugin is designed to work with any sig-multicluster-compliant environment. This ensures:

- **Broader adoption** across different multi-cluster solutions
- **Alignment** with sig-multicluster's vendor-neutral mission
- **Suitability** for donation to the sig-multicluster community

OCM-specific tooling (like `clusteradm`) serves OCM users. This plugin serves all sig-multicluster users.

### OCM-Specific Alternative

For OCM-only environments, a separate plugin could be developed that:
- Directly queries `ManagedCluster` resources
- Leverages OCM-specific features and metadata
- Provides tighter OCM integration

This separation maintains vendor neutrality for the sig-multicluster donation while allowing ecosystem-specific enhancements.

## Why Not Extend `clusteradm`?

`clusteradm` is OCM's administrative CLI for cluster lifecycle management (registration, acceptance, addon management). This plugin provides kubectl-style resource operations across clusters - a fundamentally different use case.

**clusteradm focuses on:**
- Cluster registration and lifecycle
- Hub installation and configuration
- Addon management
- Administrative operations

**kubectl-mc focuses on:**
- Day-to-day resource operations (get, describe, apply)
- kubectl-familiar UX for developers
- End-user operations, not admin operations
- Multi-cluster resource management

They complement each other: `clusteradm` sets up the infrastructure, `kubectl-mc` operates within it.

## Why Read-Only First?

Starting with read-only operations allows us to:

1. **Validate the UX approach** with lower risk
2. **Build confidence** in the discovery and aggregation mechanisms
3. **Get community feedback** before introducing write operations
4. **Establish patterns** for error handling and partial failures
5. **Prove the concept** without dangerous operations

Write operations will be added with appropriate safety mechanisms once the foundation is solid.

### Safety-First Philosophy

Multi-cluster write operations are inherently risky. A single command could affect dozens of clusters. We prioritize:

- Confirmation prompts for destructive operations
- Dry-run mode by default for multi-cluster writes
- Clear cluster targeting (no accidental "all clusters")
- Audit logging for accountability

## Kubeconfig Management: Two-Track Approach

The plugin supports both manual context configuration and platform-specific automation, recognizing that real-world environments are heterogeneous.

### Track 1: Manual Context Management (Universal)

**Why it's essential:**
- Works for ANY cluster (cloud, on-prem, edge, custom)
- Users retain full control over authentication
- No assumptions about infrastructure
- Standard kubectl patterns

**How it works:**
```bash
# User configures contexts themselves
kubectl config set-context cluster-1 --cluster=prod-us-west --user=my-user

# Plugin creates mapping file
kubectl mc setup
# Interactive: "Cluster 'prod-us-west-1' -> which context? cluster-1"
```

### Track 2: Platform-Specific Helpers (Convenience)

**Why it's valuable:**
- Automates repetitive cloud credential fetching
- Reduces setup friction for cloud-native users
- Leverages existing cloud CLI tools
- Optional enhancement, not a requirement

**How it works:**
```bash
kubectl mc setup --provider aws
# Plugin discovers clusters via ClusterProfile
# Extracts AWS metadata (region, cluster name)
# Executes: aws eks update-kubeconfig --name <name> --region <region>
# Creates context mapping automatically
```

### Why Both?

Real organizations use mixed environments:
- Some EKS clusters (can use AWS helper)
- Some on-prem clusters (need manual setup)
- Some GKE clusters (can use GCP helper)
- Some custom Kubernetes distributions

The two-track approach accommodates this reality without forcing users into a single authentication model.

## Vendor Neutrality: A Non-Negotiable Principle

Since this plugin is intended for donation to sig-multicluster, it **must not depend on any vendor-specific APIs**.

### What This Means

**Use:**
- ClusterProfile API (sig-multicluster standard)
- About API (sig-multicluster standard)
- Cluster Inventory API (sig-multicluster standard)

**Do NOT use:**
- OCM `ManagedCluster` resources
- Rancher-specific APIs
- Any vendor/solution-specific extensions

### Why This Matters

1. **sig-multicluster mission**: Promote interoperability and standards
2. **Broader applicability**: Works across different multi-cluster solutions
3. **Community adoption**: Not tied to any single ecosystem
4. **Upstream acceptance**: Essential for sig-multicluster hosting

### Relationship with OCM

This project is **orthogonal to OCM** and designed to complement it:

- OCM provides hub cluster infrastructure
- OCM manages cluster lifecycle and registration
- `kubectl-mc` provides the user-facing kubectl-like experience
- Both use sig-multicluster standard APIs

**OCM users benefit from kubectl-mc without kubectl-mc requiring OCM.**

## Error Handling Philosophy

Multi-cluster operations will experience partial failures. This is normal, not exceptional.

### Graceful Degradation

Show partial results rather than failing entirely:

```
Warning: Failed to query 2 clusters:
  - cluster-3: connection timeout (possible network issue)
  - cluster-7: unauthorized (check your credentials)

Showing results from 8/10 clusters:

NAMESPACE     NAME              CLUSTER        READY   STATUS    RESTARTS   AGE
default       nginx-abc123      cluster-1      1/1     Running   0          5d
...
```

### Design Principles

1. **Partial success is success** - Return available data
2. **Clear attribution** - Show which cluster failed and why
3. **Actionable errors** - Guide users to solutions
4. **Don't block on failures** - Continue processing other clusters
5. **Summary at end** - Remind user of partial results

## Phased Implementation Strategy

### Why Phases?

1. **Risk management** - Validate before expanding scope
2. **Community feedback** - Iterate based on real usage
3. **Foundation first** - Establish patterns before complexity
4. **Learning** - Understand multi-cluster challenges incrementally

### Phase Breakdown

**Phase 1: Prove the Concept**
- Read-only operations only
- Basic discovery and aggregation
- Manual kubeconfig setup
- Get community using it and providing feedback

**Phase 2: Polish the Experience**
- Automated credential helpers
- Advanced filtering
- Better error messages

**Phase 3: Carefully Add Write Operations**
- Safety mechanisms in place
- Confirmation prompts
- Dry-run mode
- Progressive rollout

**Phase 4: Enhanced UX**
- Visual improvements
- Interactive features
- Quality-of-life enhancements

## Scale Considerations

The plugin is designed for environments with **10s to 100s of clusters**.

### Performance Strategies

1. **Parallel execution** - Query clusters concurrently
2. **Timeouts** - Don't wait forever for slow clusters
3. **Caching** - Cache cluster discovery with appropriate TTL
4. **Streaming output** - Show results as they arrive
5. **Selective targeting** - Filter clusters before execution

### When NOT to Use This Plugin

- **1000+ clusters** - Consider purpose-built aggregation systems
- **Real-time monitoring** - Use dedicated monitoring solutions
- **Complex cross-cluster queries** - May need custom tooling

## Future Evolution

This is a v1 design. Future versions may explore:

- Policy-based cluster selection
- Cross-cluster resource dependencies
- RBAC aggregation and visualization
- Integration with GitOps workflows
- Cost/usage reporting across clusters
- Cluster health scoring

All future features must maintain vendor neutrality and sig-multicluster standards compliance.
